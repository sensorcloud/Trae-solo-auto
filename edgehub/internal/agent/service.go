package agent

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"k8s.io/client-go/kubernetes"
)

type AgentService interface {
	CreateSandbox(ctx context.Context, req *CreateSandboxRequest) (*Sandbox, error)
	GetSandbox(ctx context.Context, sandboxID uuid.UUID) (*Sandbox, error)
	ListSandboxes(ctx context.Context, req *ListSandboxesRequest) ([]*Sandbox, error)
	PauseSandbox(ctx context.Context, sandboxID uuid.UUID) error
	ResumeSandbox(ctx context.Context, sandboxID uuid.UUID) error
	StopSandbox(ctx context.Context, sandboxID uuid.UUID, force bool) error
	DeleteSandbox(ctx context.Context, sandboxID uuid.UUID) error
	
	CreateAgent(ctx context.Context, sandboxID uuid.UUID, req *CreateAgentRequest) (*Agent, error)
	GetAgent(ctx context.Context, agentID uuid.UUID) (*Agent, error)
	ListAgents(ctx context.Context, req *ListAgentsRequest) ([]*Agent, error)
	DeleteAgent(ctx context.Context, agentID uuid.UUID) error
	
	Execute(ctx context.Context, req *ExecuteRequest) (*ExecuteResponse, error)
	GetExecution(ctx context.Context, executionID uuid.UUID) (*Execution, error)
	CancelExecution(ctx context.Context, executionID uuid.UUID) error
	
	RegisterTool(ctx context.Context, agentID uuid.UUID, tool *CreateToolRequest) (*Tool, error)
	UnregisterTool(ctx context.Context, agentID uuid.UUID, toolName string) error
	ListTools(ctx context.Context, agentID uuid.UUID) ([]*Tool, error)
	
	ApplySecurityPolicy(ctx context.Context, sandboxID uuid.UUID, policy *SecurityPolicy) error
	ApplyNetworkPolicy(ctx context.Context, sandboxID uuid.UUID, policy *NetworkPolicySpec) error
	
	GetMetrics(ctx context.Context, sandboxID uuid.UUID) (*SandboxMetrics, error)
	GetStatistics(ctx context.Context) (*ServiceStatistics, error)
}

type ServiceStatistics struct {
	TotalSandboxes      int            `json:"total_sandboxes"`
	RunningSandboxes    int            `json:"running_sandboxes"`
	PausedSandboxes     int            `json:"paused_sandboxes"`
	StoppedSandboxes    int            `json:"stopped_sandboxes"`
	
	TotalAgents         int            `json:"total_agents"`
	RunningAgents       int            `json:"running_agents"`
	
	TotalExecutions     int            `json:"total_executions"`
	RunningExecutions   int            `json:"running_executions"`
	CompletedExecutions int            `json:"completed_executions"`
	FailedExecutions    int            `json:"failed_executions"`
	
	RuntimeStatus       map[RuntimeType]RuntimeStatus `json:"runtime_status"`
}

type agentService struct {
	mu              sync.RWMutex
	
	sandboxManager  *SandboxManager
	executor        *Executor
	runtimeManager  *RuntimeManager
	containerMgr    *ContainerManager
	toolRegistry    *ToolRegistry
	
	config          *ServiceConfig
}

type ServiceConfig struct {
	SandboxManagerConfig *SandboxManagerConfig `json:"sandbox_manager"`
	ExecutorConfig       *ExecutorConfig       `json:"executor"`
	
	EnableMetrics       bool                  `json:"enable_metrics"`
	EnableAudit         bool                  `json:"enable_audit"`
	
	DefaultTenantQuota  int                   `json:"default_tenant_quota"`
	MaxConcurrentOps    int                   `json:"max_concurrent_ops"`
}

func NewAgentService(cfg *ServiceConfig, k8sClient kubernetes.Interface) (AgentService, error) {
	sandboxManager := NewSandboxManager(cfg.SandboxManagerConfig, k8sClient)
	
	runtimeManager := NewRuntimeManager("")
	containerStateDir := cfg.SandboxManagerConfig.StateDir + "/containers"
	containerRootDir := cfg.SandboxManagerConfig.StorageDir + "/containers"
	containerMgr := NewContainerManager(runtimeManager, containerStateDir, containerRootDir)
	
	executor := NewExecutor(cfg.ExecutorConfig, sandboxManager, containerMgr)
	toolRegistry := NewToolRegistry()
	
	return &agentService{
		sandboxManager: sandboxManager,
		executor:       executor,
		runtimeManager: runtimeManager,
		containerMgr:   containerMgr,
		toolRegistry:   toolRegistry,
		config:         cfg,
	}, nil
}

func (s *agentService) Initialize(ctx context.Context) error {
	if err := s.runtimeManager.Initialize(ctx); err != nil {
		return fmt.Errorf("初始化运行时管理器失败: %w", err)
	}
	
	if err := s.containerMgr.Initialize(ctx); err != nil {
		return fmt.Errorf("初始化容器管理器失败: %w", err)
	}
	
	if err := s.sandboxManager.Initialize(ctx); err != nil {
		return fmt.Errorf("初始化沙箱管理器失败: %w", err)
	}
	
	if err := s.executor.Initialize(ctx); err != nil {
		return fmt.Errorf("初始化执行器失败: %w", err)
	}
	
	s.registerBuiltinTools()
	
	return nil
}

func (s *agentService) registerBuiltinTools() {
	s.toolRegistry.RegisterTool(&ToolDefinition{
		Name:        "http_request",
		Type:        ToolTypeHTTP,
		Description: "执行HTTP请求",
		Schema: ToolSchema{
			Type: "object",
			Properties: map[string]PropertySchema{
				"url": {
					Type:        "string",
					Description: "请求URL",
				},
				"method": {
					Type:        "string",
					Description: "HTTP方法",
					Enum:        []string{"GET", "POST", "PUT", "DELETE", "PATCH"},
					Default:     "GET",
				},
				"headers": {
					Type:        "object",
					Description: "请求头",
				},
				"body": {
					Type:        "string",
					Description: "请求体",
				},
			},
			Required: []string{"url"},
		},
		Handler: s.handleHTTPRequest,
	})
	
	s.toolRegistry.RegisterTool(&ToolDefinition{
		Name:        "shell_execute",
		Type:        ToolTypeShell,
		Description: "执行Shell命令",
		Schema: ToolSchema{
			Type: "object",
			Properties: map[string]PropertySchema{
				"command": {
					Type:        "string",
					Description: "要执行的命令",
				},
				"args": {
					Type:        "array",
					Description: "命令参数",
				},
			},
			Required: []string{"command"},
		},
		Handler: s.handleShellExecute,
	})
	
	s.toolRegistry.RegisterTool(&ToolDefinition{
		Name:        "code_execute",
		Type:        ToolTypeCode,
		Description: "执行代码",
		Schema: ToolSchema{
			Type: "object",
			Properties: map[string]PropertySchema{
				"code": {
					Type:        "string",
					Description: "要执行的代码",
				},
				"language": {
					Type:        "string",
					Description: "编程语言",
					Enum:        []string{"python", "javascript", "bash"},
					Default:     "python",
				},
			},
			Required: []string{"code"},
		},
		Handler: s.handleCodeExecute,
	})
	
	s.toolRegistry.RegisterTool(&ToolDefinition{
		Name:        "file_read",
		Type:        ToolTypeFile,
		Description: "读取文件内容",
		Schema: ToolSchema{
			Type: "object",
			Properties: map[string]PropertySchema{
				"path": {
					Type:        "string",
					Description: "文件路径",
				},
			},
			Required: []string{"path"},
		},
		Handler: s.handleFileRead,
	})
	
	s.toolRegistry.RegisterTool(&ToolDefinition{
		Name:        "file_write",
		Type:        ToolTypeFile,
		Description: "写入文件内容",
		Schema: ToolSchema{
			Type: "object",
			Properties: map[string]PropertySchema{
				"path": {
					Type:        "string",
					Description: "文件路径",
				},
				"content": {
					Type:        "string",
					Description: "文件内容",
				},
			},
			Required: []string{"path", "content"},
		},
		Handler: s.handleFileWrite,
	})
}

func (s *agentService) handleHTTPRequest(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	url, _ := params["url"].(string)
	method, _ := params["method"].(string)
	if method == "" {
		method = "GET"
	}
	
	return map[string]interface{}{
		"status": "success",
		"url":    url,
		"method": method,
	}, nil
}

func (s *agentService) handleShellExecute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	command, _ := params["command"].(string)
	
	return map[string]interface{}{
		"status":  "success",
		"command": command,
		"output":  "",
	}, nil
}

func (s *agentService) handleCodeExecute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	language, _ := params["language"].(string)
	if language == "" {
		language = "python"
	}
	
	return map[string]interface{}{
		"status":   "success",
		"language": language,
		"output":   "",
	}, nil
}

func (s *agentService) handleFileRead(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	path, _ := params["path"].(string)
	
	return map[string]interface{}{
		"status":  "success",
		"path":    path,
		"content": "",
	}, nil
}

func (s *agentService) handleFileWrite(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	path, _ := params["path"].(string)
	
	return map[string]interface{}{
		"status": "success",
		"path":   path,
	}, nil
}

func (s *agentService) CreateSandbox(ctx context.Context, req *CreateSandboxRequest) (*Sandbox, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	sandboxReq := &CreateSandboxRequest{
		Name:        req.Name,
		TenantID:    req.TenantID,
		NodeID:      req.NodeID,
		Runtime:     req.Runtime,
		Namespace:   req.Namespace,
		Config:      req.Config,
		Resources:   req.Resources,
		Network:     req.Network,
		Security:    req.Security,
		Labels:      req.Labels,
		Annotations: req.Annotations,
	}
	
	return s.sandboxManager.CreateSandbox(ctx, sandboxReq)
}

func (s *agentService) GetSandbox(ctx context.Context, sandboxID uuid.UUID) (*Sandbox, error) {
	return s.sandboxManager.GetSandbox(ctx, sandboxID)
}

func (s *agentService) ListSandboxes(ctx context.Context, req *ListSandboxesRequest) ([]*Sandbox, error) {
	var tenantID *uuid.UUID
	var status *AgentStatus
	
	if req.TenantID != uuid.Nil {
		tenantID = &req.TenantID
	}
	if req.Status != "" {
		st := AgentStatus(req.Status)
		status = &st
	}
	
	return s.sandboxManager.ListSandboxes(ctx, tenantID, status)
}

func (s *agentService) PauseSandbox(ctx context.Context, sandboxID uuid.UUID) error {
	return s.sandboxManager.PauseSandbox(ctx, sandboxID)
}

func (s *agentService) ResumeSandbox(ctx context.Context, sandboxID uuid.UUID) error {
	return s.sandboxManager.ResumeSandbox(ctx, sandboxID)
}

func (s *agentService) StopSandbox(ctx context.Context, sandboxID uuid.UUID, force bool) error {
	return s.sandboxManager.StopSandbox(ctx, sandboxID, force)
}

func (s *agentService) DeleteSandbox(ctx context.Context, sandboxID uuid.UUID) error {
	return s.sandboxManager.DeleteSandbox(ctx, sandboxID)
}

func (s *agentService) CreateAgent(ctx context.Context, sandboxID uuid.UUID, req *CreateAgentRequest) (*Agent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	agentReq := &CreateAgentRequest{
		Name:        req.Name,
		Description: req.Description,
		UserID:      req.UserID,
		Spec:        req.Spec,
		Config:      req.Config,
		MemoryLimit: req.MemoryLimit,
		CPULimit:    req.CPULimit,
		Timeout:     req.Timeout,
		Environment: req.Environment,
		Labels:      req.Labels,
	}
	
	return s.sandboxManager.CreateAgent(ctx, sandboxID, agentReq)
}

func (s *agentService) GetAgent(ctx context.Context, agentID uuid.UUID) (*Agent, error) {
	return s.sandboxManager.GetAgent(ctx, agentID)
}

func (s *agentService) ListAgents(ctx context.Context, req *ListAgentsRequest) ([]*Agent, error) {
	var sandboxID *uuid.UUID
	var status *AgentStatus
	
	if req.Status != "" {
		st := AgentStatus(req.Status)
		status = &st
	}
	
	return s.sandboxManager.ListAgents(ctx, sandboxID, status)
}

func (s *agentService) DeleteAgent(ctx context.Context, agentID uuid.UUID) error {
	return s.sandboxManager.DeleteAgent(ctx, agentID)
}

func (s *agentService) Execute(ctx context.Context, req *ExecuteRequest) (*ExecuteResponse, error) {
	agent, err := s.sandboxManager.GetAgent(ctx, req.AgentID)
	if err != nil {
		return nil, fmt.Errorf("获取Agent失败: %w", err)
	}
	
	if agent.SandboxID == nil {
		return nil, fmt.Errorf("Agent未关联沙箱")
	}
	
	execReq := &ExecutionRequest{
		AgentID:   req.AgentID,
		SandboxID: *agent.SandboxID,
		Type:      req.Type,
		Code:      req.Code,
		Language:  req.Language,
		Command:   req.Command,
		Input:     req.Input,
		Timeout:   req.Timeout,
		MaxMemory: req.MaxMemory,
	}
	
	result, err := s.executor.Execute(ctx, execReq)
	if err != nil {
		return nil, fmt.Errorf("执行失败: %w", err)
	}
	
	return &ExecuteResponse{
		ExecutionID: result.ExecutionID,
		Status:      result.Status,
		Output:      result.Output,
		Error:       result.Error,
		ExitCode:    result.ExitCode,
		Duration:    result.Duration,
		Metrics:     result.Metrics,
	}, nil
}

func (s *agentService) GetExecution(ctx context.Context, executionID uuid.UUID) (*Execution, error) {
	return s.executor.GetExecution(ctx, executionID)
}

func (s *agentService) CancelExecution(ctx context.Context, executionID uuid.UUID) error {
	return s.executor.CancelExecution(ctx, executionID)
}

func (s *agentService) RegisterTool(ctx context.Context, agentID uuid.UUID, toolReq *CreateToolRequest) (*Tool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	agent, err := s.sandboxManager.GetAgent(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("获取Agent失败: %w", err)
	}
	
	tool := &Tool{
		ID:          uuid.New(),
		AgentID:     agentID,
		Name:        toolReq.Name,
		Type:        toolReq.Type,
		Description: toolReq.Description,
		Schema:      toolReq.Schema,
		Config:      toolReq.Config,
		Enabled:     toolReq.Enabled,
		Timeout:     toolReq.Timeout,
		MaxRetries:  toolReq.MaxRetries,
		Permissions: toolReq.Permissions,
		Labels:      make(map[string]string),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	
	if !tool.Enabled {
		tool.Enabled = true
	}
	if tool.Timeout == 0 {
		tool.Timeout = 60
	}
	if tool.MaxRetries == 0 {
		tool.MaxRetries = 3
	}
	
	agent.Tools = append(agent.Tools, *tool)
	
	return tool, nil
}

func (s *agentService) UnregisterTool(ctx context.Context, agentID uuid.UUID, toolName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	agent, err := s.sandboxManager.GetAgent(ctx, agentID)
	if err != nil {
		return fmt.Errorf("获取Agent失败: %w", err)
	}
	
	for i, tool := range agent.Tools {
		if tool.Name == toolName {
			agent.Tools = append(agent.Tools[:i], agent.Tools[i+1:]...)
			return nil
		}
	}
	
	return fmt.Errorf("工具 %s 不存在", toolName)
}

func (s *agentService) ListTools(ctx context.Context, agentID uuid.UUID) ([]*Tool, error) {
	agent, err := s.sandboxManager.GetAgent(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("获取Agent失败: %w", err)
	}
	
	tools := make([]*Tool, len(agent.Tools))
	for i := range agent.Tools {
		tools[i] = &agent.Tools[i]
	}
	
	return tools, nil
}

func (s *agentService) ApplySecurityPolicy(ctx context.Context, sandboxID uuid.UUID, policy *SecurityPolicy) error {
	return s.sandboxManager.ApplySecurityPolicy(ctx, sandboxID, policy)
}

func (s *agentService) ApplyNetworkPolicy(ctx context.Context, sandboxID uuid.UUID, policy *NetworkPolicySpec) error {
	return s.sandboxManager.ApplyNetworkPolicy(ctx, sandboxID, policy)
}

func (s *agentService) GetMetrics(ctx context.Context, sandboxID uuid.UUID) (*SandboxMetrics, error) {
	sandbox, err := s.sandboxManager.GetSandbox(ctx, sandboxID)
	if err != nil {
		return nil, err
	}
	
	return &sandbox.Metrics, nil
}

func (s *agentService) GetStatistics(ctx context.Context) (*ServiceStatistics, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	stats := &ServiceStatistics{
		RuntimeStatus: make(map[RuntimeType]RuntimeStatus),
	}
	
	sandboxes, _ := s.sandboxManager.ListSandboxes(ctx, nil, nil)
	stats.TotalSandboxes = len(sandboxes)
	
	for _, sb := range sandboxes {
		switch sb.Status {
		case AgentStatusRunning:
			stats.RunningSandboxes++
		case AgentStatusPaused:
			stats.PausedSandboxes++
		case AgentStatusStopped:
			stats.StoppedSandboxes++
		}
	}
	
	agents, _ := s.sandboxManager.ListAgents(ctx, nil, nil)
	stats.TotalAgents = len(agents)
	
	for _, agent := range agents {
		if agent.Status == AgentStatusRunning {
			stats.RunningAgents++
		}
	}
	
	execStats, _ := s.executor.GetStatistics(ctx)
	if execStats != nil {
		stats.TotalExecutions = execStats.TotalExecutions
		stats.RunningExecutions = execStats.RunningExecutions
		stats.CompletedExecutions = execStats.CompletedExecutions
		stats.FailedExecutions = execStats.FailedExecutions
	}
	
	for _, rt := range s.runtimeManager.GetAvailableRuntimes() {
		stats.RuntimeStatus[rt] = RuntimeStatusAvailable
	}
	
	return stats, nil
}

func (s *agentService) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	log.Println("正在关闭Agent服务...")
	
	if err := s.sandboxManager.Shutdown(ctx); err != nil {
		log.Printf("关闭沙箱管理器失败: %v", err)
	}
	
	log.Println("Agent服务已关闭")
	return nil
}

type SandboxEventHandler interface {
	OnSandboxCreated(sandbox *Sandbox)
	OnSandboxStarted(sandbox *Sandbox)
	OnSandboxStopped(sandbox *Sandbox)
	OnSandboxPaused(sandbox *Sandbox)
	OnSandboxResumed(sandbox *Sandbox)
	OnSandboxDeleted(sandbox *Sandbox)
	OnSandboxError(sandbox *Sandbox, err error)
}

type ExecutionEventHandler interface {
	OnExecutionStarted(execution *Execution)
	OnExecutionCompleted(execution *Execution)
	OnExecutionFailed(execution *Execution, err error)
	OnExecutionTimeout(execution *Execution)
	OnExecutionCancelled(execution *Execution)
}

type EventDispatcher struct {
	mu               sync.RWMutex
	sandboxHandlers  []SandboxEventHandler
	executionHandlers []ExecutionEventHandler
}

func NewEventDispatcher() *EventDispatcher {
	return &EventDispatcher{
		sandboxHandlers:   make([]SandboxEventHandler, 0),
		executionHandlers: make([]ExecutionEventHandler, 0),
	}
}

func (ed *EventDispatcher) RegisterSandboxHandler(handler SandboxEventHandler) {
	ed.mu.Lock()
	defer ed.mu.Unlock()
	ed.sandboxHandlers = append(ed.sandboxHandlers, handler)
}

func (ed *EventDispatcher) RegisterExecutionHandler(handler ExecutionEventHandler) {
	ed.mu.Lock()
	defer ed.mu.Unlock()
	ed.executionHandlers = append(ed.executionHandlers, handler)
}

func (ed *EventDispatcher) DispatchSandboxEvent(event string, sandbox *Sandbox, err error) {
	ed.mu.RLock()
	defer ed.mu.RUnlock()
	
	for _, handler := range ed.sandboxHandlers {
		switch event {
		case "created":
			handler.OnSandboxCreated(sandbox)
		case "started":
			handler.OnSandboxStarted(sandbox)
		case "stopped":
			handler.OnSandboxStopped(sandbox)
		case "paused":
			handler.OnSandboxPaused(sandbox)
		case "resumed":
			handler.OnSandboxResumed(sandbox)
		case "deleted":
			handler.OnSandboxDeleted(sandbox)
		case "error":
			handler.OnSandboxError(sandbox, err)
		}
	}
}

func (ed *EventDispatcher) DispatchExecutionEvent(event string, execution *Execution, err error) {
	ed.mu.RLock()
	defer ed.mu.RUnlock()
	
	for _, handler := range ed.executionHandlers {
		switch event {
		case "started":
			handler.OnExecutionStarted(execution)
		case "completed":
			handler.OnExecutionCompleted(execution)
		case "failed":
			handler.OnExecutionFailed(execution, err)
		case "timeout":
			handler.OnExecutionTimeout(execution)
		case "cancelled":
			handler.OnExecutionCancelled(execution)
		}
	}
}

type LoggingEventHandler struct{}

func (h *LoggingEventHandler) OnSandboxCreated(sandbox *Sandbox) {
	log.Printf("[Sandbox] Created: %s (%s)", sandbox.Name, sandbox.ID)
}

func (h *LoggingEventHandler) OnSandboxStarted(sandbox *Sandbox) {
	log.Printf("[Sandbox] Started: %s (%s)", sandbox.Name, sandbox.ID)
}

func (h *LoggingEventHandler) OnSandboxStopped(sandbox *Sandbox) {
	log.Printf("[Sandbox] Stopped: %s (%s)", sandbox.Name, sandbox.ID)
}

func (h *LoggingEventHandler) OnSandboxPaused(sandbox *Sandbox) {
	log.Printf("[Sandbox] Paused: %s (%s)", sandbox.Name, sandbox.ID)
}

func (h *LoggingEventHandler) OnSandboxResumed(sandbox *Sandbox) {
	log.Printf("[Sandbox] Resumed: %s (%s)", sandbox.Name, sandbox.ID)
}

func (h *LoggingEventHandler) OnSandboxDeleted(sandbox *Sandbox) {
	log.Printf("[Sandbox] Deleted: %s (%s)", sandbox.Name, sandbox.ID)
}

func (h *LoggingEventHandler) OnSandboxError(sandbox *Sandbox, err error) {
	log.Printf("[Sandbox] Error: %s (%s) - %v", sandbox.Name, sandbox.ID, err)
}

func (h *LoggingEventHandler) OnExecutionStarted(execution *Execution) {
	log.Printf("[Execution] Started: %s (%s)", execution.ID, execution.Type)
}

func (h *LoggingEventHandler) OnExecutionCompleted(execution *Execution) {
	log.Printf("[Execution] Completed: %s, Duration: %dms", execution.ID, execution.Duration)
}

func (h *LoggingEventHandler) OnExecutionFailed(execution *Execution, err error) {
	log.Printf("[Execution] Failed: %s - %v", execution.ID, err)
}

func (h *LoggingEventHandler) OnExecutionTimeout(execution *Execution) {
	log.Printf("[Execution] Timeout: %s", execution.ID)
}

func (h *LoggingEventHandler) OnExecutionCancelled(execution *Execution) {
	log.Printf("[Execution] Cancelled: %s", execution.ID)
}

type HealthChecker struct {
	service AgentService
}

func NewHealthChecker(service AgentService) *HealthChecker {
	return &HealthChecker{service: service}
}

func (hc *HealthChecker) Check(ctx context.Context) (*HealthStatus, error) {
	stats, err := hc.service.GetStatistics(ctx)
	if err != nil {
		return &HealthStatus{
			Status:  "unhealthy",
			Message: err.Error(),
		}, err
	}
	
	status := &HealthStatus{
		Status:            "healthy",
		Message:           "Agent service is running",
		TotalSandboxes:    stats.TotalSandboxes,
		RunningSandboxes:  stats.RunningSandboxes,
		TotalAgents:       stats.TotalAgents,
		TotalExecutions:   stats.TotalExecutions,
		RuntimeStatus:     stats.RuntimeStatus,
	}
	
	return status, nil
}

type HealthStatus struct {
	Status           string                        `json:"status"`
	Message          string                        `json:"message"`
	TotalSandboxes   int                           `json:"total_sandboxes"`
	RunningSandboxes int                           `json:"running_sandboxes"`
	TotalAgents      int                           `json:"total_agents"`
	TotalExecutions  int                           `json:"total_executions"`
	RuntimeStatus    map[RuntimeType]RuntimeStatus `json:"runtime_status"`
}
