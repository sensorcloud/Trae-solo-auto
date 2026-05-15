package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Executor struct {
	mu             sync.RWMutex
	executions     map[uuid.UUID]*Execution
	
	sandboxManager *SandboxManager
	containerMgr   *ContainerManager
	
	config          *ExecutorConfig
	
	outputDir       string
	scriptDir       string
	
	runningExecs    map[uuid.UUID]context.CancelFunc
}

type ExecutorConfig struct {
	DefaultTimeout      int           `json:"default_timeout"`
	MaxConcurrent       int           `json:"max_concurrent"`
	MaxOutputSize       int64         `json:"max_output_size"`
	OutputDir           string        `json:"output_dir"`
	ScriptDir           string        `json:"script_dir"`
	
	EnableCodeExecution bool          `json:"enable_code_execution"`
	EnableShellExecution bool         `json:"enable_shell_execution"`
	EnableHTTPExecution bool          `json:"enable_http_execution"`
	
	AllowedLanguages    []string      `json:"allowed_languages"`
	ForbiddenCommands   []string      `json:"forbidden_commands"`
	
	MemoryLimit         int64         `json:"memory_limit"`
	CPULimit            string        `json:"cpu_limit"`
}

func NewExecutor(cfg *ExecutorConfig, sandboxManager *SandboxManager, containerMgr *ContainerManager) *Executor {
	return &Executor{
		executions:     make(map[uuid.UUID]*Execution),
		sandboxManager: sandboxManager,
		containerMgr:   containerMgr,
		config:         cfg,
		outputDir:      cfg.OutputDir,
		scriptDir:      cfg.ScriptDir,
		runningExecs:   make(map[uuid.UUID]context.CancelFunc),
	}
}

func (e *Executor) Initialize(ctx context.Context) error {
	if err := os.MkdirAll(e.outputDir, 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %w", err)
	}
	
	if err := os.MkdirAll(e.scriptDir, 0755); err != nil {
		return fmt.Errorf("创建脚本目录失败: %w", err)
	}
	
	return nil
}

func (e *Executor) Execute(ctx context.Context, req *ExecutionRequest) (*ExecutionResult, error) {
	if err := e.validateRequest(req); err != nil {
		return nil, fmt.Errorf("验证请求失败: %w", err)
	}
	
	executionID := uuid.New()
	now := time.Now()
	
	execution := &Execution{
		ID:         executionID,
		AgentID:    req.AgentID,
		SandboxID:  req.SandboxID,
		Type:       req.Type,
		Status:     ExecutionStatusPending,
		Input:      req.Input,
		Command:    req.Command,
		Code:       req.Code,
		Language:   req.Language,
		Timeout:    req.Timeout,
		MaxMemory:  req.MaxMemory,
		Labels:     req.Labels,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	
	if execution.Timeout == 0 {
		execution.Timeout = e.config.DefaultTimeout
	}
	if execution.MaxMemory == 0 {
		execution.MaxMemory = e.config.MemoryLimit
	}
	
	e.mu.Lock()
	e.executions[executionID] = execution
	e.mu.Unlock()
	
	execCtx, cancel := context.WithTimeout(ctx, time.Duration(execution.Timeout)*time.Second)
	defer cancel()
	
	e.mu.Lock()
	e.runningExecs[executionID] = cancel
	e.mu.Unlock()
	
	defer func() {
		e.mu.Lock()
		delete(e.runningExecs, executionID)
		e.mu.Unlock()
	}()
	
	execution.Status = ExecutionStatusRunning
	startedAt := time.Now()
	execution.StartedAt = &startedAt
	
	var result *ExecutionResult
	var err error
	
	switch req.Type {
	case ExecutionTypeCode:
		result, err = e.executeCode(execCtx, execution, req)
	case ExecutionTypeShell:
		result, err = e.executeShell(execCtx, execution, req)
	case ExecutionTypeTool:
		result, err = e.executeTool(execCtx, execution, req)
	case ExecutionTypeHTTP:
		result, err = e.executeHTTP(execCtx, execution, req)
	case ExecutionTypeScript:
		result, err = e.executeScript(execCtx, execution, req)
	default:
		err = fmt.Errorf("不支持的执行类型: %s", req.Type)
	}
	
	finishedAt := time.Now()
	execution.FinishedAt = &finishedAt
	execution.Duration = finishedAt.Sub(startedAt).Milliseconds()
	
	if err != nil {
		execution.Status = ExecutionStatusFailed
		execution.Error = err.Error()
	} else {
		execution.Status = result.Status
		execution.Output = result.Output
		execution.ExitCode = result.ExitCode
		execution.Metrics = result.Metrics
	}
	
	if execCtx.Err() == context.DeadlineExceeded {
		execution.Status = ExecutionStatusTimeout
		execution.Error = "执行超时"
	}
	
	e.mu.Lock()
	execution.UpdatedAt = time.Now()
	e.mu.Unlock()
	
	if err := e.saveExecutionResult(execution); err != nil {
		log.Printf("保存执行结果失败: %v", err)
	}
	
	return &ExecutionResult{
		ExecutionID: executionID,
		Status:      execution.Status,
		Output:      execution.Output,
		Error:       execution.Error,
		ExitCode:    execution.ExitCode,
		Duration:    execution.Duration,
		Metrics:     execution.Metrics,
	}, nil
}

type ExecutionRequest struct {
	AgentID   uuid.UUID
	SandboxID uuid.UUID
	Type      ExecutionType
	
	Code     string
	Language string
	Command  []string
	Input    string
	
	Timeout  int
	MaxMemory int64
	
	Labels   map[string]string
}

type ExecutionResult struct {
	ExecutionID uuid.UUID
	Status      ExecutionStatus
	Output      string
	Error       string
	ExitCode    *int
	Duration    int64
	Metrics     ExecutionMetrics
}

func (e *Executor) validateRequest(req *ExecutionRequest) error {
	if req.AgentID == uuid.Nil {
		return fmt.Errorf("AgentID不能为空")
	}
	
	switch req.Type {
	case ExecutionTypeCode:
		if !e.config.EnableCodeExecution {
			return fmt.Errorf("代码执行功能未启用")
		}
		if req.Code == "" {
			return fmt.Errorf("代码不能为空")
		}
		if !e.isLanguageAllowed(req.Language) {
			return fmt.Errorf("语言 %s 不被允许", req.Language)
		}
		
	case ExecutionTypeShell:
		if !e.config.EnableShellExecution {
			return fmt.Errorf("Shell执行功能未启用")
		}
		if len(req.Command) == 0 {
			return fmt.Errorf("命令不能为空")
		}
		if e.isCommandForbidden(req.Command[0]) {
			return fmt.Errorf("命令 %s 被禁止", req.Command[0])
		}
		
	case ExecutionTypeHTTP:
		if !e.config.EnableHTTPExecution {
			return fmt.Errorf("HTTP执行功能未启用")
		}
	}
	
	return nil
}

func (e *Executor) isLanguageAllowed(lang string) bool {
	if len(e.config.AllowedLanguages) == 0 {
		return true
	}
	
	for _, allowed := range e.config.AllowedLanguages {
		if strings.EqualFold(allowed, lang) {
			return true
		}
	}
	return false
}

func (e *Executor) isCommandForbidden(cmd string) bool {
	for _, forbidden := range e.config.ForbiddenCommands {
		if strings.EqualFold(forbidden, cmd) {
			return true
		}
	}
	return false
}

func (e *Executor) executeCode(ctx context.Context, execution *Execution, req *ExecutionRequest) (*ExecutionResult, error) {
	sandbox, err := e.sandboxManager.GetSandbox(ctx, req.SandboxID)
	if err != nil {
		return nil, fmt.Errorf("获取沙箱失败: %w", err)
	}
	
	scriptPath, err := e.prepareScriptFile(execution.ID, req.Language, req.Code)
	if err != nil {
		return nil, fmt.Errorf("准备脚本文件失败: %w", err)
	}
	defer os.Remove(scriptPath)
	
	cmd := e.buildCodeCommand(req.Language, scriptPath)
	if len(cmd) == 0 {
		return nil, fmt.Errorf("不支持的语言: %s", req.Language)
	}
	
	return e.runInSandbox(ctx, sandbox, cmd, req.Input, execution)
}

func (e *Executor) prepareScriptFile(executionID uuid.UUID, language, code string) (string, error) {
	extensions := map[string]string{
		"python":    ".py",
		"python3":   ".py",
		"javascript": ".js",
		"node":      ".js",
		"typescript": ".ts",
		"go":        ".go",
		"java":      ".java",
		"ruby":      ".rb",
		"php":       ".php",
		"perl":      ".pl",
		"bash":      ".sh",
		"sh":        ".sh",
		"lua":       ".lua",
		"r":         ".r",
		"rust":      ".rs",
		"c":         ".c",
		"cpp":       ".cpp",
	}
	
	ext := ".txt"
	if e, ok := extensions[strings.ToLower(language)]; ok {
		ext = e
	}
	
	filename := fmt.Sprintf("script_%s%s", executionID.String(), ext)
	scriptPath := filepath.Join(e.scriptDir, filename)
	
	if err := os.WriteFile(scriptPath, []byte(code), 0644); err != nil {
		return "", err
	}
	
	return scriptPath, nil
}

func (e *Executor) buildCodeCommand(language, scriptPath string) []string {
	commands := map[string][]string{
		"python":     {"python3", scriptPath},
		"python3":    {"python3", scriptPath},
		"javascript": {"node", scriptPath},
		"node":       {"node", scriptPath},
		"typescript": {"ts-node", scriptPath},
		"go":         {"go", "run", scriptPath},
		"ruby":       {"ruby", scriptPath},
		"php":        {"php", scriptPath},
		"perl":       {"perl", scriptPath},
		"bash":       {"bash", scriptPath},
		"sh":         {"sh", scriptPath},
		"lua":        {"lua", scriptPath},
		"r":          {"Rscript", scriptPath},
	}
	
	if cmd, ok := commands[strings.ToLower(language)]; ok {
		return cmd
	}
	
	return nil
}

func (e *Executor) executeShell(ctx context.Context, execution *Execution, req *ExecutionRequest) (*ExecutionResult, error) {
	sandbox, err := e.sandboxManager.GetSandbox(ctx, req.SandboxID)
	if err != nil {
		return nil, fmt.Errorf("获取沙箱失败: %w", err)
	}
	
	return e.runInSandbox(ctx, sandbox, req.Command, req.Input, execution)
}

func (e *Executor) executeTool(ctx context.Context, execution *Execution, req *ExecutionRequest) (*ExecutionResult, error) {
	var toolInput ToolExecutionInput
	if err := json.Unmarshal([]byte(req.Input), &toolInput); err != nil {
		return nil, fmt.Errorf("解析工具输入失败: %w", err)
	}
	
	agent, err := e.sandboxManager.GetAgent(ctx, req.AgentID)
	if err != nil {
		return nil, fmt.Errorf("获取Agent失败: %w", err)
	}
	
	var targetTool *Tool
	for i := range agent.Tools {
		if agent.Tools[i].Name == toolInput.Name {
			targetTool = &agent.Tools[i]
			break
		}
	}
	
	if targetTool == nil {
		return nil, fmt.Errorf("工具 %s 不存在", toolInput.Name)
	}
	
	if !targetTool.Enabled {
		return nil, fmt.Errorf("工具 %s 未启用", toolInput.Name)
	}
	
	return e.executeToolInternal(ctx, execution, targetTool, toolInput.Parameters)
}

type ToolExecutionInput struct {
	Name       string                 `json:"name"`
	Parameters map[string]interface{} `json:"parameters"`
}

func (e *Executor) executeToolInternal(ctx context.Context, execution *Execution, tool *Tool, params map[string]interface{}) (*ExecutionResult, error) {
	switch tool.Type {
	case ToolTypeHTTP:
		return e.executeHTTPTool(ctx, execution, tool, params)
	case ToolTypeShell:
		return e.executeShellTool(ctx, execution, tool, params)
	case ToolTypeFunction:
		return e.executeFunctionTool(ctx, execution, tool, params)
	case ToolTypeCode:
		return e.executeCodeTool(ctx, execution, tool, params)
	default:
		return nil, fmt.Errorf("不支持的工具类型: %s", tool.Type)
	}
}

func (e *Executor) executeHTTPTool(ctx context.Context, execution *Execution, tool *Tool, params map[string]interface{}) (*ExecutionResult, error) {
	endpoint := tool.Config.Endpoint
	if endpoint == "" {
		return nil, fmt.Errorf("HTTP工具未配置端点")
	}
	
	method := tool.Config.Method
	if method == "" {
		method = "GET"
	}
	
	body, _ := json.Marshal(params)
	
	cmd := []string{"curl", "-s", "-X", method}
	
	for k, v := range tool.Config.Headers {
		cmd = append(cmd, "-H", fmt.Sprintf("%s: %s", k, v))
	}
	
	if method != "GET" && len(body) > 0 {
		cmd = append(cmd, "-H", "Content-Type: application/json")
		cmd = append(cmd, "-d", string(body))
	}
	
	cmd = append(cmd, endpoint)
	
	sandbox, err := e.sandboxManager.GetSandbox(ctx, execution.SandboxID)
	if err != nil {
		return nil, err
	}
	
	return e.runInSandbox(ctx, sandbox, cmd, "", execution)
}

func (e *Executor) executeShellTool(ctx context.Context, execution *Execution, tool *Tool, params map[string]interface{}) (*ExecutionResult, error) {
	command := tool.Config.Command
	if len(command) == 0 {
		return nil, fmt.Errorf("Shell工具未配置命令")
	}
	
	for k, v := range params {
		placeholder := fmt.Sprintf("{{%s}}", k)
		value := fmt.Sprintf("%v", v)
		for i, arg := range command {
			command[i] = strings.ReplaceAll(arg, placeholder, value)
		}
	}
	
	sandbox, err := e.sandboxManager.GetSandbox(ctx, execution.SandboxID)
	if err != nil {
		return nil, err
	}
	
	return e.runInSandbox(ctx, sandbox, command, "", execution)
}

func (e *Executor) executeFunctionTool(ctx context.Context, execution *Execution, tool *Tool, params map[string]interface{}) (*ExecutionResult, error) {
	script := tool.Config.Script
	if script == "" {
		return nil, fmt.Errorf("Function工具未配置脚本")
	}
	
	paramsJSON, _ := json.Marshal(params)
	
	sandbox, err := e.sandboxManager.GetSandbox(ctx, execution.SandboxID)
	if err != nil {
		return nil, err
	}
	
	interpreter := tool.Config.Interpreter
	if interpreter == "" {
		interpreter = "python3"
	}
	
	scriptPath := filepath.Join(e.scriptDir, fmt.Sprintf("tool_%s.py", tool.Name))
	if err := os.WriteFile(scriptPath, []byte(script), 0644); err != nil {
		return nil, err
	}
	defer os.Remove(scriptPath)
	
	cmd := []string{interpreter, scriptPath}
	
	return e.runInSandbox(ctx, sandbox, cmd, string(paramsJSON), execution)
}

func (e *Executor) executeCodeTool(ctx context.Context, execution *Execution, tool *Tool, params map[string]interface{}) (*ExecutionResult, error) {
	code, ok := params["code"].(string)
	if !ok {
		return nil, fmt.Errorf("缺少code参数")
	}
	
	language, _ := params["language"].(string)
	if language == "" {
		language = "python"
	}
	
	return e.executeCode(ctx, execution, &ExecutionRequest{
		AgentID:   execution.AgentID,
		SandboxID: execution.SandboxID,
		Type:      ExecutionTypeCode,
		Code:      code,
		Language:  language,
		Timeout:   execution.Timeout,
	})
}

func (e *Executor) executeHTTP(ctx context.Context, execution *Execution, req *ExecutionRequest) (*ExecutionResult, error) {
	var httpReq HTTPExecutionRequest
	if err := json.Unmarshal([]byte(req.Input), &httpReq); err != nil {
		return nil, fmt.Errorf("解析HTTP请求失败: %w", err)
	}
	
	cmd := []string{"curl", "-s", "-X", httpReq.Method}
	
	for k, v := range httpReq.Headers {
		cmd = append(cmd, "-H", fmt.Sprintf("%s: %s", k, v))
	}
	
	if httpReq.Body != "" {
		cmd = append(cmd, "-d", httpReq.Body)
	}
	
	cmd = append(cmd, httpReq.URL)
	
	sandbox, err := e.sandboxManager.GetSandbox(ctx, execution.SandboxID)
	if err != nil {
		return nil, err
	}
	
	return e.runInSandbox(ctx, sandbox, cmd, "", execution)
}

type HTTPExecutionRequest struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

func (e *Executor) executeScript(ctx context.Context, execution *Execution, req *ExecutionRequest) (*ExecutionResult, error) {
	sandbox, err := e.sandboxManager.GetSandbox(ctx, execution.SandboxID)
	if err != nil {
		return nil, err
	}
	
	scriptPath, err := e.prepareScriptFile(execution.ID, "bash", req.Code)
	if err != nil {
		return nil, err
	}
	defer os.Remove(scriptPath)
	
	cmd := []string{"bash", scriptPath}
	
	return e.runInSandbox(ctx, sandbox, cmd, req.Input, execution)
}

func (e *Executor) runInSandbox(ctx context.Context, sandbox *Sandbox, command []string, stdin string, execution *Execution) (*ExecutionResult, error) {
	result := &ExecutionResult{
		ExecutionID: execution.ID,
	}
	
	startTime := time.Now()
	
	if sandbox.ContainerID != "" && e.containerMgr != nil {
		output, err := e.containerMgr.ExecInContainer(ctx, sandbox.ContainerID, command, []byte(stdin))
		if err != nil {
			result.Status = ExecutionStatusFailed
			result.Error = err.Error()
			return result, nil
		}
		
		result.Status = ExecutionStatusCompleted
		result.Output = string(output)
		result.Duration = time.Since(startTime).Milliseconds()
		return result, nil
	}
	
	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	err := cmd.Run()
	
	result.Output = stdout.String()
	if stderr.Len() > 0 {
		result.Output += "\n" + stderr.String()
	}
	
	if len(result.Output) > int(e.config.MaxOutputSize) {
		result.Output = result.Output[:e.config.MaxOutputSize] + "\n... (输出被截断)"
	}
	
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode := exitErr.ExitCode()
			result.ExitCode = &exitCode
			result.Status = ExecutionStatusFailed
			result.Error = exitErr.Error()
		} else if ctx.Err() == context.DeadlineExceeded {
			result.Status = ExecutionStatusTimeout
			result.Error = "执行超时"
		} else {
			result.Status = ExecutionStatusFailed
			result.Error = err.Error()
		}
	} else {
		exitCode := 0
		result.ExitCode = &exitCode
		result.Status = ExecutionStatusCompleted
	}
	
	result.Duration = time.Since(startTime).Milliseconds()
	result.Metrics = ExecutionMetrics{
		CPUSeconds: float64(result.Duration) / 1000.0,
	}
	
	return result, nil
}

func (e *Executor) GetExecution(ctx context.Context, executionID uuid.UUID) (*Execution, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	execution, ok := e.executions[executionID]
	if !ok {
		return nil, fmt.Errorf("执行 %s 不存在", executionID)
	}
	
	return execution, nil
}

func (e *Executor) ListExecutions(ctx context.Context, agentID *uuid.UUID, status *ExecutionStatus) ([]*Execution, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	executions := make([]*Execution, 0)
	for _, exec := range e.executions {
		if agentID != nil && exec.AgentID != *agentID {
			continue
		}
		if status != nil && exec.Status != *status {
			continue
		}
		executions = append(executions, exec)
	}
	
	return executions, nil
}

func (e *Executor) CancelExecution(ctx context.Context, executionID uuid.UUID) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	cancel, ok := e.runningExecs[executionID]
	if !ok {
		return fmt.Errorf("执行 %s 不在运行中", executionID)
	}
	
	cancel()
	
	execution, ok := e.executions[executionID]
	if ok {
		execution.Status = ExecutionStatusCancelled
		now := time.Now()
		execution.FinishedAt = &now
	}
	
	return nil
}

func (e *Executor) saveExecutionResult(execution *Execution) error {
	resultFile := filepath.Join(e.outputDir, fmt.Sprintf("%s.json", execution.ID))
	
	data, err := json.MarshalIndent(execution, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(resultFile, data, 0644)
}

func (e *Executor) StreamOutput(ctx context.Context, executionID uuid.UUID) (<-chan string, error) {
	e.mu.RLock()
	_, ok := e.executions[executionID]
	e.mu.RUnlock()
	
	if !ok {
		return nil, fmt.Errorf("执行 %s 不存在", executionID)
	}
	
	outputChan := make(chan string, 100)
	
	go func() {
		defer close(outputChan)
		
		outputFile := filepath.Join(e.outputDir, fmt.Sprintf("%s.log", executionID))
		
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(100 * time.Millisecond):
				file, err := os.Open(outputFile)
				if err != nil {
					if os.IsNotExist(err) {
						continue
					}
					return
				}
				
				content, err := io.ReadAll(file)
				file.Close()
				if err != nil {
					return
				}
				
				outputChan <- string(content)
				
				e.mu.RLock()
				exec, ok := e.executions[executionID]
				e.mu.RUnlock()
				
				if ok && (exec.Status == ExecutionStatusCompleted || 
					exec.Status == ExecutionStatusFailed ||
					exec.Status == ExecutionStatusTimeout ||
					exec.Status == ExecutionStatusCancelled) {
					return
				}
			}
		}
	}()
	
	return outputChan, nil
}

func (e *Executor) GetExecutionMetrics(ctx context.Context, executionID uuid.UUID) (*ExecutionMetrics, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	execution, ok := e.executions[executionID]
	if !ok {
		return nil, fmt.Errorf("执行 %s 不存在", executionID)
	}
	
	metrics := &execution.Metrics
	return metrics, nil
}

func (e *Executor) CleanupOldExecutions(ctx context.Context, maxAge time.Duration) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	cutoff := time.Now().Add(-maxAge)
	
	for id, execution := range e.executions {
		if execution.CreatedAt.Before(cutoff) {
			switch execution.Status {
			case ExecutionStatusCompleted, ExecutionStatusFailed, 
				ExecutionStatusTimeout, ExecutionStatusCancelled:
				delete(e.executions, id)
				
				resultFile := filepath.Join(e.outputDir, fmt.Sprintf("%s.json", id))
				os.Remove(resultFile)
			}
		}
	}
	
	return nil
}

func (e *Executor) GetStatistics(ctx context.Context) (*ExecutorStatistics, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	stats := &ExecutorStatistics{
		TotalExecutions:    len(e.executions),
		RunningExecutions:  0,
		CompletedExecutions: 0,
		FailedExecutions:   0,
		TimeoutExecutions:  0,
		CancelledExecutions: 0,
	}
	
	for _, exec := range e.executions {
		switch exec.Status {
		case ExecutionStatusRunning:
			stats.RunningExecutions++
		case ExecutionStatusCompleted:
			stats.CompletedExecutions++
		case ExecutionStatusFailed:
			stats.FailedExecutions++
		case ExecutionStatusTimeout:
			stats.TimeoutExecutions++
		case ExecutionStatusCancelled:
			stats.CancelledExecutions++
		}
	}
	
	return stats, nil
}

type ExecutorStatistics struct {
	TotalExecutions     int `json:"total_executions"`
	RunningExecutions   int `json:"running_executions"`
	CompletedExecutions int `json:"completed_executions"`
	FailedExecutions    int `json:"failed_executions"`
	TimeoutExecutions   int `json:"timeout_executions"`
	CancelledExecutions int `json:"cancelled_executions"`
}

type ToolRegistry struct {
	mu    sync.RWMutex
	tools map[string]*ToolDefinition
}

type ToolDefinition struct {
	Name        string
	Type        ToolType
	Description string
	Schema      ToolSchema
	Handler     ToolHandler
}

type ToolHandler func(ctx context.Context, params map[string]interface{}) (interface{}, error)

func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]*ToolDefinition),
	}
}

func (tr *ToolRegistry) RegisterTool(def *ToolDefinition) error {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	
	if _, exists := tr.tools[def.Name]; exists {
		return fmt.Errorf("工具 %s 已注册", def.Name)
	}
	
	tr.tools[def.Name] = def
	return nil
}

func (tr *ToolRegistry) UnregisterTool(name string) {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	
	delete(tr.tools, name)
}

func (tr *ToolRegistry) GetTool(name string) (*ToolDefinition, bool) {
	tr.mu.RLock()
	defer tr.mu.RUnlock()
	
	tool, ok := tr.tools[name]
	return tool, ok
}

func (tr *ToolRegistry) ListTools() []*ToolDefinition {
	tr.mu.RLock()
	defer tr.mu.RUnlock()
	
	tools := make([]*ToolDefinition, 0, len(tr.tools))
	for _, tool := range tr.tools {
		tools = append(tools, tool)
	}
	return tools
}

func (tr *ToolRegistry) ExecuteTool(ctx context.Context, name string, params map[string]interface{}) (interface{}, error) {
	tr.mu.RLock()
	tool, ok := tr.tools[name]
	tr.mu.RUnlock()
	
	if !ok {
		return nil, fmt.Errorf("工具 %s 不存在", name)
	}
	
	if tool.Handler == nil {
		return nil, fmt.Errorf("工具 %s 未配置处理器", name)
	}
	
	return tool.Handler(ctx, params)
}
