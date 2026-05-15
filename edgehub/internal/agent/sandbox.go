package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type SandboxManager struct {
	mu              sync.RWMutex
	sandboxes       map[uuid.UUID]*Sandbox
	agents          map[uuid.UUID]*Agent
	
	runtimeManager  *RuntimeManager
	containerMgr    *ContainerManager
	k8sClient       kubernetes.Interface
	
	config          *SandboxManagerConfig
	eventChan       chan *SandboxEventRecord
	
	stateDir        string
	storageDir      string
}

type SandboxManagerConfig struct {
	DefaultRuntime     SandboxRuntime `json:"default_runtime"`
	DefaultNamespace   string         `json:"default_namespace"`
	
	MaxSandboxesPerTenant int         `json:"max_sandboxes_per_tenant"`
	MaxAgentsPerSandbox   int         `json:"max_agents_per_sandbox"`
	
	DefaultMemoryLimit   int64        `json:"default_memory_limit"`
	DefaultCPULimit      string       `json:"default_cpu_limit"`
	DefaultTimeout       int          `json:"default_timeout"`
	
	EnableDRA            bool         `json:"enable_dra"`
	DRAResourceClass     string       `json:"dra_resource_class"`
	
	StateDir             string       `json:"state_dir"`
	StorageDir           string       `json:"storage_dir"`
	
	MetricsInterval      time.Duration `json:"metrics_interval"`
	HealthCheckInterval  time.Duration `json:"health_check_interval"`
}

func NewSandboxManager(cfg *SandboxManagerConfig, k8sClient kubernetes.Interface) *SandboxManager {
	return &SandboxManager{
		sandboxes:      make(map[uuid.UUID]*Sandbox),
		agents:         make(map[uuid.UUID]*Agent),
		config:         cfg,
		eventChan:      make(chan *SandboxEventRecord, 1000),
		stateDir:       cfg.StateDir,
		storageDir:     cfg.StorageDir,
		k8sClient:      k8sClient,
	}
}

func (sm *SandboxManager) Initialize(ctx context.Context) error {
	if err := os.MkdirAll(sm.stateDir, 0755); err != nil {
		return fmt.Errorf("创建状态目录失败: %w", err)
	}
	
	if err := os.MkdirAll(sm.storageDir, 0755); err != nil {
		return fmt.Errorf("创建存储目录失败: %w", err)
	}
	
	sm.runtimeManager = NewRuntimeManager("")
	if err := sm.runtimeManager.Initialize(ctx); err != nil {
		return fmt.Errorf("初始化运行时管理器失败: %w", err)
	}
	
	containerStateDir := filepath.Join(sm.stateDir, "containers")
	containerRootDir := filepath.Join(sm.storageDir, "containers")
	sm.containerMgr = NewContainerManager(sm.runtimeManager, containerStateDir, containerRootDir)
	if err := sm.containerMgr.Initialize(ctx); err != nil {
		return fmt.Errorf("初始化容器管理器失败: %w", err)
	}
	
	if err := sm.restoreState(ctx); err != nil {
		log.Printf("警告: 恢复状态失败: %v", err)
	}
	
	go sm.eventLoop(ctx)
	go sm.metricsLoop(ctx)
	go sm.healthCheckLoop(ctx)
	
	return nil
}

func (sm *SandboxManager) restoreState(ctx context.Context) error {
	sandboxFile := filepath.Join(sm.stateDir, "sandboxes.json")
	data, err := os.ReadFile(sandboxFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	
	var sandboxes []*Sandbox
	if err := json.Unmarshal(data, &sandboxes); err != nil {
		return err
	}
	
	for _, sandbox := range sandboxes {
		sm.sandboxes[sandbox.ID] = sandbox
	}
	
	agentFile := filepath.Join(sm.stateDir, "agents.json")
	data, err = os.ReadFile(agentFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	
	var agents []*Agent
	if err := json.Unmarshal(data, &agents); err != nil {
		return err
	}
	
	for _, agent := range agents {
		sm.agents[agent.ID] = agent
	}
	
	return nil
}

func (sm *SandboxManager) saveState() error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	sandboxes := make([]*Sandbox, 0, len(sm.sandboxes))
	for _, s := range sm.sandboxes {
		sandboxes = append(sandboxes, s)
	}
	
	data, err := json.MarshalIndent(sandboxes, "", "  ")
	if err != nil {
		return err
	}
	
	sandboxFile := filepath.Join(sm.stateDir, "sandboxes.json")
	if err := os.WriteFile(sandboxFile, data, 0644); err != nil {
		return err
	}
	
	agents := make([]*Agent, 0, len(sm.agents))
	for _, a := range sm.agents {
		agents = append(agents, a)
	}
	
	data, err = json.MarshalIndent(agents, "", "  ")
	if err != nil {
		return err
	}
	
	agentFile := filepath.Join(sm.stateDir, "agents.json")
	return os.WriteFile(agentFile, data, 0644)
}

func (sm *SandboxManager) CreateSandbox(ctx context.Context, req *CreateSandboxRequest) (*Sandbox, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	sandboxCount := sm.countSandboxesByTenant(req.TenantID)
	if sandboxCount >= sm.config.MaxSandboxesPerTenant {
		return nil, fmt.Errorf("租户 %s 已达到最大沙箱数量限制 %d", req.TenantID, sm.config.MaxSandboxesPerTenant)
	}
	
	sandboxID := uuid.New()
	now := time.Now()
	
	sandbox := &Sandbox{
		ID:        sandboxID,
		Name:      req.Name,
		Status:    AgentStatusCreating,
		TenantID:  req.TenantID,
		NodeID:    req.NodeID,
		Runtime:   req.Runtime,
		Namespace: req.Namespace,
		Config:    req.Config,
		Resources: req.Resources,
		Network:   req.Network,
		Security:  req.Security,
		Labels:    req.Labels,
		Annotations: req.Annotations,
	}
	
	if sandbox.Namespace == "" {
		sandbox.Namespace = sm.config.DefaultNamespace
	}
	if sandbox.Runtime == "" {
		sandbox.Runtime = sm.config.DefaultRuntime
	}
	
	sm.sandboxes[sandboxID] = sandbox
	
	sm.emitEvent(&SandboxEventRecord{
		ID:        uuid.New(),
		SandboxID: sandboxID,
		Type:      "creating",
		Reason:    "CreateRequested",
		Message:   fmt.Sprintf("沙箱 %s 创建请求已接收", req.Name),
		Source:    "sandbox-manager",
		Timestamp: now,
	})
	
	if err := sm.createSandboxResources(ctx, sandbox); err != nil {
		sandbox.Status = AgentStatusError
		sandbox.LastError = err.Error()
		sm.emitEvent(&SandboxEventRecord{
			ID:        uuid.New(),
			SandboxID: sandboxID,
			Type:      "error",
			Reason:    "CreateFailed",
			Message:   fmt.Sprintf("沙箱创建失败: %v", err),
			Source:    "sandbox-manager",
			Timestamp: time.Now(),
		})
		return nil, fmt.Errorf("创建沙箱资源失败: %w", err)
	}
	
	sandbox.Status = AgentStatusRunning
	startedAt := time.Now()
	sandbox.StartedAt = &startedAt
	
	sm.saveState()
	
	sm.emitEvent(&SandboxEventRecord{
		ID:        uuid.New(),
		SandboxID: sandboxID,
		Type:      "created",
		Reason:    "CreateSucceeded",
		Message:   fmt.Sprintf("沙箱 %s 创建成功", req.Name),
		Source:    "sandbox-manager",
		Timestamp: time.Now(),
	})
	
	return sandbox, nil
}

type CreateSandboxRequest struct {
	Name        string
	TenantID    uuid.UUID
	NodeID      *uuid.UUID
	Runtime     SandboxRuntime
	Namespace   string
	Config      SandboxConfig
	Resources   ResourceSpec
	Network     NetworkConfig
	Security    SecurityConfig
	Labels      map[string]string
	Annotations map[string]string
}

func (sm *SandboxManager) createSandboxResources(ctx context.Context, sandbox *Sandbox) error {
	if sm.k8sClient != nil {
		return sm.createKubernetesResources(ctx, sandbox)
	}
	return sm.createLocalResources(ctx, sandbox)
}

func (sm *SandboxManager) createKubernetesResources(ctx context.Context, sandbox *Sandbox) error {
	pod := sm.buildPodSpec(sandbox)
	
	createdPod, err := sm.k8sClient.CoreV1().Pods(sandbox.Namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("创建Pod失败: %w", err)
	}
	
	sandbox.PodName = createdPod.Name
	
	if err := sm.waitForPodReady(ctx, sandbox.Namespace, createdPod.Name, 5*time.Minute); err != nil {
		return fmt.Errorf("等待Pod就绪失败: %w", err)
	}
	
	return nil
}

func (sm *SandboxManager) buildPodSpec(sandbox *Sandbox) *corev1.Pod {
	podName := fmt.Sprintf("sandbox-%s", sandbox.ID.String()[:8])
	
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        podName,
			Namespace:   sandbox.Namespace,
			Labels:      sm.buildPodLabels(sandbox),
			Annotations: sandbox.Annotations,
		},
		Spec: corev1.PodSpec{
			RuntimeClassName: sm.getRuntimeClassName(sandbox.Runtime),
			Containers: []corev1.Container{
				{
					Name:    "sandbox",
					Image:   sandbox.Config.RootfsImage,
					Command: []string{"/bin/sh", "-c", "sleep infinity"},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse(sandbox.Resources.CPURequest),
							corev1.ResourceMemory: *resource.NewQuantity(sandbox.Resources.MemoryRequest, resource.BinarySI),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse(sandbox.Resources.CPULimit),
							corev1.ResourceMemory: *resource.NewQuantity(sandbox.Resources.MemoryLimit, resource.BinarySI),
						},
					},
					SecurityContext: sm.buildSecurityContext(&sandbox.Security),
					VolumeMounts:    sm.buildVolumeMounts(sandbox),
				},
			},
			Volumes:            sm.buildVolumes(sandbox),
			SecurityContext:    sm.buildPodSecurityContext(&sandbox.Security),
			ServiceAccountName: "sandbox-sa",
			PriorityClassName:  "sandbox-priority",
		},
	}
	
	if sandbox.Network.Policy != NetworkPolicyHost {
		pod.Spec.HostNetwork = false
	}
	
	if sm.config.EnableDRA && sandbox.Resources.GPURequest > 0 {
		sm.addDRAResources(pod, sandbox)
	}
	
	return pod
}

func (sm *SandboxManager) getRuntimeClassName(runtime SandboxRuntime) *string {
	classNames := map[SandboxRuntime]string{
		RuntimeRunsc: "runsc",
		RuntimeKata:  "kata",
		RuntimeRunc:  "runc",
	}
	
	if name, ok := classNames[runtime]; ok {
		return &name
	}
	return nil
}

func (sm *SandboxManager) buildPodLabels(sandbox *Sandbox) map[string]string {
	labels := map[string]string{
		"app":                   "agent-sandbox",
		"sandbox-id":            sandbox.ID.String(),
		"sandbox-name":          sandbox.Name,
		"tenant-id":             sandbox.TenantID.String(),
		"edgehub.io/runtime":    string(sandbox.Runtime),
		"edgehub.io/managed":    "true",
	}
	
	for k, v := range sandbox.Labels {
		labels[k] = v
	}
	
	return labels
}

func (sm *SandboxManager) buildSecurityContext(security *SecurityConfig) *corev1.SecurityContext {
	sc := &corev1.SecurityContext{
		RunAsNonRoot:             &[]bool{true}[0],
		ReadOnlyRootFilesystem:   &security.ReadOnlyRootFS,
		AllowPrivilegeEscalation: &[]bool{false}[0],
	}
	
	if len(security.DropCapabilities) > 0 {
		sc.Capabilities = &corev1.Capabilities{
			Drop: make([]corev1.Capability, len(security.DropCapabilities)),
		}
		for i, cap := range security.DropCapabilities {
			sc.Capabilities.Drop[i] = corev1.Capability(cap)
		}
	}
	
	return sc
}

func (sm *SandboxManager) buildPodSecurityContext(security *SecurityConfig) *corev1.PodSecurityContext {
	sc := &corev1.PodSecurityContext{
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
	}
	
	if security.SeccompProfile != "" {
		sc.SeccompProfile = &corev1.SeccompProfile{
			Type:             corev1.SeccompProfileTypeLocalhost,
			LocalhostProfile: &security.SeccompProfile,
		}
	}
	
	if security.AppArmorProfile != "" {
		annotations := make(map[string]string)
		annotations["container.apparmor.security.beta.kubernetes.io/sandbox"] = security.AppArmorProfile
	}
	
	return sc
}

func (sm *SandboxManager) buildVolumeMounts(sandbox *Sandbox) []corev1.VolumeMount {
	mounts := []corev1.VolumeMount{
		{
			Name:      "sandbox-storage",
			MountPath: "/sandbox",
		},
		{
			Name:      "tmp",
			MountPath: "/tmp",
		},
	}
	
	return mounts
}

func (sm *SandboxManager) buildVolumes(sandbox *Sandbox) []corev1.Volume {
	volumes := []corev1.Volume{
		{
			Name: "sandbox-storage",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					SizeLimit: resource.NewQuantity(sandbox.Resources.EphemeralStorage, resource.BinarySI),
				},
			},
		},
		{
			Name: "tmp",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}
	
	return volumes
}

func (sm *SandboxManager) addDRAResources(pod *corev1.Pod, sandbox *Sandbox) {
	// DRA (Dynamic Resource Assignment) 支持
	// 注意: DRA API 在不同 Kubernetes 版本中可能有所不同
	// 这里使用设备插件方式作为备选方案
	
	// 为GPU资源添加limits
	if sandbox.Resources.GPURequest > 0 {
		for i := range pod.Spec.Containers {
			if pod.Spec.Containers[i].Resources.Limits == nil {
				pod.Spec.Containers[i].Resources.Limits = make(corev1.ResourceList)
			}
			pod.Spec.Containers[i].Resources.Limits[corev1.ResourceName("nvidia.com/gpu")] = *resource.NewQuantity(int64(sandbox.Resources.GPURequest), resource.DecimalSI)
		}
	}
}

func (sm *SandboxManager) waitForPodReady(ctx context.Context, namespace, podName string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
			pod, err := sm.k8sClient.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
			if err != nil {
				continue
			}
			
			if pod.Status.Phase == corev1.PodRunning {
				for _, cond := range pod.Status.Conditions {
					if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
						return nil
					}
				}
			}
			
			if pod.Status.Phase == corev1.PodFailed {
				return fmt.Errorf("Pod启动失败: %s", pod.Status.Message)
			}
		}
	}
}

func (sm *SandboxManager) createLocalResources(ctx context.Context, sandbox *Sandbox) error {
	containerSpec := &ContainerSpec{
		ID:         sandbox.ID.String(),
		Name:       sandbox.Name,
		Image:      sandbox.Config.RootfsImage,
		Runtime:    RuntimeType(sandbox.Runtime),
		Resources:  sandbox.Resources,
		Network:    sandbox.Network,
		Security:   sandbox.Security,
		Stdin:      sandbox.Config.Stdin,
		Tty:        sandbox.Config.Tty,
		Labels:     sandbox.Labels,
	}
	
	status, err := sm.containerMgr.CreateContainer(ctx, containerSpec)
	if err != nil {
		return fmt.Errorf("创建容器失败: %w", err)
	}
	
	sandbox.ContainerID = status.ID
	
	if err := sm.containerMgr.StartContainer(ctx, status.ID); err != nil {
		return fmt.Errorf("启动容器失败: %w", err)
	}
	
	return nil
}

func (sm *SandboxManager) GetSandbox(ctx context.Context, sandboxID uuid.UUID) (*Sandbox, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	sandbox, ok := sm.sandboxes[sandboxID]
	if !ok {
		return nil, fmt.Errorf("沙箱 %s 不存在", sandboxID)
	}
	
	return sandbox, nil
}

func (sm *SandboxManager) ListSandboxes(ctx context.Context, tenantID *uuid.UUID, status *AgentStatus) ([]*Sandbox, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	sandboxes := make([]*Sandbox, 0)
	for _, s := range sm.sandboxes {
		if tenantID != nil && s.TenantID != *tenantID {
			continue
		}
		if status != nil && s.Status != *status {
			continue
		}
		sandboxes = append(sandboxes, s)
	}
	
	return sandboxes, nil
}

func (sm *SandboxManager) PauseSandbox(ctx context.Context, sandboxID uuid.UUID) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	sandbox, ok := sm.sandboxes[sandboxID]
	if !ok {
		return fmt.Errorf("沙箱 %s 不存在", sandboxID)
	}
	
	if sandbox.Status != AgentStatusRunning {
		return fmt.Errorf("沙箱状态为 %s，无法暂停", sandbox.Status)
	}
	
	if sm.k8sClient != nil {
		if err := sm.pauseKubernetesSandbox(ctx, sandbox); err != nil {
			return err
		}
	} else {
		if err := sm.containerMgr.PauseContainer(ctx, sandbox.ContainerID); err != nil {
			return err
		}
	}
	
	sandbox.Status = AgentStatusPaused
	sm.saveState()
	
	sm.emitEvent(&SandboxEventRecord{
		ID:        uuid.New(),
		SandboxID: sandboxID,
		Type:      "paused",
		Reason:    "PauseRequested",
		Message:   fmt.Sprintf("沙箱 %s 已暂停", sandbox.Name),
		Source:    "sandbox-manager",
		Timestamp: time.Now(),
	})
	
	return nil
}

func (sm *SandboxManager) pauseKubernetesSandbox(ctx context.Context, sandbox *Sandbox) error {
	// 对于Pod，暂停操作需要通过其他方式实现
	// 可以通过设置Pod的activeDeadlineSeconds或使用pause容器
	return nil
}

func (sm *SandboxManager) ResumeSandbox(ctx context.Context, sandboxID uuid.UUID) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	sandbox, ok := sm.sandboxes[sandboxID]
	if !ok {
		return fmt.Errorf("沙箱 %s 不存在", sandboxID)
	}
	
	if sandbox.Status != AgentStatusPaused {
		return fmt.Errorf("沙箱状态为 %s，无法恢复", sandbox.Status)
	}
	
	if sm.k8sClient != nil {
		if err := sm.resumeKubernetesSandbox(ctx, sandbox); err != nil {
			return err
		}
	} else {
		if err := sm.containerMgr.ResumeContainer(ctx, sandbox.ContainerID); err != nil {
			return err
		}
	}
	
	sandbox.Status = AgentStatusRunning
	sm.saveState()
	
	sm.emitEvent(&SandboxEventRecord{
		ID:        uuid.New(),
		SandboxID: sandboxID,
		Type:      "resumed",
		Reason:    "ResumeRequested",
		Message:   fmt.Sprintf("沙箱 %s 已恢复", sandbox.Name),
		Source:    "sandbox-manager",
		Timestamp: time.Now(),
	})
	
	return nil
}

func (sm *SandboxManager) resumeKubernetesSandbox(ctx context.Context, sandbox *Sandbox) error {
	return nil
}

func (sm *SandboxManager) StopSandbox(ctx context.Context, sandboxID uuid.UUID, force bool) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	sandbox, ok := sm.sandboxes[sandboxID]
	if !ok {
		return fmt.Errorf("沙箱 %s 不存在", sandboxID)
	}
	
	if sandbox.Status == AgentStatusStopped {
		return nil
	}
	
	sm.emitEvent(&SandboxEventRecord{
		ID:        uuid.New(),
		SandboxID: sandboxID,
		Type:      "stopping",
		Reason:    "StopRequested",
		Message:   fmt.Sprintf("沙箱 %s 正在停止", sandbox.Name),
		Source:    "sandbox-manager",
		Timestamp: time.Now(),
	})
	
	if sm.k8sClient != nil {
		if err := sm.stopKubernetesSandbox(ctx, sandbox, force); err != nil {
			return err
		}
	} else {
		timeout := 30
		if force {
			timeout = 5
		}
		if err := sm.containerMgr.StopContainer(ctx, sandbox.ContainerID, timeout); err != nil {
			return err
		}
	}
	
	sandbox.Status = AgentStatusStopped
	stoppedAt := time.Now()
	sandbox.StoppedAt = &stoppedAt
	sm.saveState()
	
	sm.emitEvent(&SandboxEventRecord{
		ID:        uuid.New(),
		SandboxID: sandboxID,
		Type:      "stopped",
		Reason:    "StopSucceeded",
		Message:   fmt.Sprintf("沙箱 %s 已停止", sandbox.Name),
		Source:    "sandbox-manager",
		Timestamp: time.Now(),
	})
	
	return nil
}

func (sm *SandboxManager) stopKubernetesSandbox(ctx context.Context, sandbox *Sandbox, force bool) error {
	gracePeriod := int64(30)
	if force {
		gracePeriod = 5
	}
	
	err := sm.k8sClient.CoreV1().Pods(sandbox.Namespace).Delete(ctx, sandbox.PodName, metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriod,
	})
	if err != nil {
		return fmt.Errorf("删除Pod失败: %w", err)
	}
	
	return nil
}

func (sm *SandboxManager) DeleteSandbox(ctx context.Context, sandboxID uuid.UUID) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	sandbox, ok := sm.sandboxes[sandboxID]
	if !ok {
		return fmt.Errorf("沙箱 %s 不存在", sandboxID)
	}
	
	if sandbox.Status == AgentStatusRunning {
		if err := sm.stopKubernetesSandbox(ctx, sandbox, true); err != nil {
			log.Printf("警告: 停止沙箱失败: %v", err)
		}
	}
	
	if sm.k8sClient != nil {
		if sandbox.PodName != "" {
			_ = sm.k8sClient.CoreV1().Pods(sandbox.Namespace).Delete(ctx, sandbox.PodName, metav1.DeleteOptions{})
		}
	} else {
		if sandbox.ContainerID != "" {
			_ = sm.containerMgr.DeleteContainer(ctx, sandbox.ContainerID)
		}
	}
	
	delete(sm.sandboxes, sandboxID)
	sm.saveState()
	
	sm.emitEvent(&SandboxEventRecord{
		ID:        uuid.New(),
		SandboxID: sandboxID,
		Type:      "deleted",
		Reason:    "DeleteSucceeded",
		Message:   fmt.Sprintf("沙箱 %s 已删除", sandbox.Name),
		Source:    "sandbox-manager",
		Timestamp: time.Now(),
	})
	
	return nil
}

func (sm *SandboxManager) CreateAgent(ctx context.Context, sandboxID uuid.UUID, req *CreateAgentRequest) (*Agent, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	sandbox, ok := sm.sandboxes[sandboxID]
	if !ok {
		return nil, fmt.Errorf("沙箱 %s 不存在", sandboxID)
	}
	
	if sandbox.Status != AgentStatusRunning {
		return nil, fmt.Errorf("沙箱状态为 %s，无法创建Agent", sandbox.Status)
	}
	
	agentCount := sm.countAgentsBySandbox(sandboxID)
	if agentCount >= sm.config.MaxAgentsPerSandbox {
		return nil, fmt.Errorf("沙箱 %s 已达到最大Agent数量限制 %d", sandboxID, sm.config.MaxAgentsPerSandbox)
	}
	
	agentID := uuid.New()
	now := time.Now()
	
	agent := &Agent{
		ID:          agentID,
		Name:        req.Name,
		Description: req.Description,
		Status:      AgentStatusCreating,
		TenantID:    sandbox.TenantID,
		UserID:      req.UserID,
		SandboxID:   &sandboxID,
		Runtime:     sandbox.Runtime,
		Spec:        req.Spec,
		Config:      req.Config,
		MemoryLimit: req.MemoryLimit,
		CPULimit:    req.CPULimit,
		Timeout:     req.Timeout,
		Environment: req.Environment,
		Labels:      req.Labels,
	}
	
	if agent.MemoryLimit == 0 {
		agent.MemoryLimit = sm.config.DefaultMemoryLimit
	}
	if agent.CPULimit == "" {
		agent.CPULimit = sm.config.DefaultCPULimit
	}
	if agent.Timeout == 0 {
		agent.Timeout = sm.config.DefaultTimeout
	}
	
	sm.agents[agentID] = agent
	
	if err := sm.createAgentResources(ctx, sandbox, agent); err != nil {
		agent.Status = AgentStatusError
		agent.LastError = err.Error()
		return nil, fmt.Errorf("创建Agent资源失败: %w", err)
	}
	
	agent.Status = AgentStatusRunning
	agent.StartedAt = &now
	sm.saveState()
	
	return agent, nil
}

func (sm *SandboxManager) createAgentResources(ctx context.Context, sandbox *Sandbox, agent *Agent) error {
	return nil
}

func (sm *SandboxManager) GetAgent(ctx context.Context, agentID uuid.UUID) (*Agent, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	agent, ok := sm.agents[agentID]
	if !ok {
		return nil, fmt.Errorf("Agent %s 不存在", agentID)
	}
	
	return agent, nil
}

func (sm *SandboxManager) ListAgents(ctx context.Context, sandboxID *uuid.UUID, status *AgentStatus) ([]*Agent, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	agents := make([]*Agent, 0)
	for _, a := range sm.agents {
		if sandboxID != nil && a.SandboxID != sandboxID {
			continue
		}
		if status != nil && a.Status != *status {
			continue
		}
		agents = append(agents, a)
	}
	
	return agents, nil
}

func (sm *SandboxManager) DeleteAgent(ctx context.Context, agentID uuid.UUID) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	agent, ok := sm.agents[agentID]
	if !ok {
		return fmt.Errorf("Agent %s 不存在", agentID)
	}
	
	agent.Status = AgentStatusTerminated
	now := time.Now()
	agent.FinishedAt = &now
	
	delete(sm.agents, agentID)
	sm.saveState()
	
	return nil
}

func (sm *SandboxManager) countSandboxesByTenant(tenantID uuid.UUID) int {
	count := 0
	for _, s := range sm.sandboxes {
		if s.TenantID == tenantID {
			count++
		}
	}
	return count
}

func (sm *SandboxManager) countAgentsBySandbox(sandboxID uuid.UUID) int {
	count := 0
	for _, a := range sm.agents {
		if a.SandboxID != nil && *a.SandboxID == sandboxID {
			count++
		}
	}
	return count
}

func (sm *SandboxManager) emitEvent(event *SandboxEventRecord) {
	select {
	case sm.eventChan <- event:
	default:
		log.Printf("警告: 事件通道已满，丢弃事件: %s", event.Type)
	}
}

func (sm *SandboxManager) eventLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-sm.eventChan:
			sm.handleEvent(event)
		}
	}
}

func (sm *SandboxManager) handleEvent(event *SandboxEventRecord) {
	log.Printf("[SandboxEvent] %s - %s: %s", event.Type, event.Reason, event.Message)
}

func (sm *SandboxManager) metricsLoop(ctx context.Context) {
	ticker := time.NewTicker(sm.config.MetricsInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sm.collectMetrics(ctx)
		}
	}
}

func (sm *SandboxManager) collectMetrics(ctx context.Context) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	for _, sandbox := range sm.sandboxes {
		if sandbox.Status != AgentStatusRunning {
			continue
		}
		
		metrics, err := sm.getSandboxMetrics(ctx, sandbox)
		if err != nil {
			log.Printf("获取沙箱 %s 指标失败: %v", sandbox.ID, err)
			continue
		}
		
		sandbox.Metrics = *metrics
	}
}

func (sm *SandboxManager) getSandboxMetrics(ctx context.Context, sandbox *Sandbox) (*SandboxMetrics, error) {
	metrics := &SandboxMetrics{
		LastUpdated: time.Now(),
	}
	
	if sm.k8sClient != nil && sandbox.PodName != "" {
		pod, err := sm.k8sClient.CoreV1().Pods(sandbox.Namespace).Get(ctx, sandbox.PodName, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		
		for _, container := range pod.Status.ContainerStatuses {
			if container.Name == "sandbox" {
				_ = container.State.Running
			}
		}
	}
	
	return metrics, nil
}

func (sm *SandboxManager) healthCheckLoop(ctx context.Context) {
	ticker := time.NewTicker(sm.config.HealthCheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sm.performHealthCheck(ctx)
		}
	}
}

func (sm *SandboxManager) performHealthCheck(ctx context.Context) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	for _, sandbox := range sm.sandboxes {
		if sandbox.Status != AgentStatusRunning {
			continue
		}
		
		if err := sm.checkSandboxHealth(ctx, sandbox); err != nil {
			log.Printf("沙箱 %s 健康检查失败: %v", sandbox.ID, err)
			sandbox.Status = AgentStatusError
			sandbox.LastError = err.Error()
			
			sm.emitEvent(&SandboxEventRecord{
				ID:        uuid.New(),
				SandboxID: sandbox.ID,
				Type:      "error",
				Reason:    "HealthCheckFailed",
				Message:   fmt.Sprintf("沙箱健康检查失败: %v", err),
				Source:    "health-check",
				Timestamp: time.Now(),
			})
		}
	}
}

func (sm *SandboxManager) checkSandboxHealth(ctx context.Context, sandbox *Sandbox) error {
	if sm.k8sClient != nil && sandbox.PodName != "" {
		pod, err := sm.k8sClient.CoreV1().Pods(sandbox.Namespace).Get(ctx, sandbox.PodName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		
		if pod.Status.Phase != corev1.PodRunning {
			return fmt.Errorf("Pod状态异常: %s", pod.Status.Phase)
		}
	}
	
	return nil
}

func (sm *SandboxManager) GetSandboxStatus(ctx context.Context, sandboxID uuid.UUID) (*SandboxStatusResponse, error) {
	sandbox, err := sm.GetSandbox(ctx, sandboxID)
	if err != nil {
		return nil, err
	}
	
	agentCount := 0
	executionCount := 0
	
	sm.mu.RLock()
	for _, agent := range sm.agents {
		if agent.SandboxID != nil && *agent.SandboxID == sandboxID {
			agentCount++
		}
	}
	sm.mu.RUnlock()
	
	return &SandboxStatusResponse{
		ID:             sandbox.ID,
		Name:           sandbox.Name,
		Status:         sandbox.Status,
		Runtime:        sandbox.Runtime,
		ContainerID:    sandbox.ContainerID,
		PodName:        sandbox.PodName,
		Namespace:      sandbox.Namespace,
		Metrics:        sandbox.Metrics,
		StartedAt:      sandbox.StartedAt,
		AgentCount:     agentCount,
		ExecutionCount: executionCount,
	}, nil
}

func (sm *SandboxManager) ApplySecurityPolicy(ctx context.Context, sandboxID uuid.UUID, policy *SecurityPolicy) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	sandbox, ok := sm.sandboxes[sandboxID]
	if !ok {
		return fmt.Errorf("沙箱 %s 不存在", sandboxID)
	}
	
	sandbox.Security = SecurityConfig{
		SeccompProfile:    policy.SeccompProfile,
		AppArmorProfile:   policy.AppArmorProfile,
		NoNewPrivileges:   policy.NoNewPrivileges,
		ReadOnlyRootFS:    policy.ReadOnlyRootFS,
		DropCapabilities:  policy.DropCapabilities,
		ForbiddenSyscalls: policy.ForbiddenSyscalls,
		UserNamespace:     policy.UserNamespace,
		PIDNamespace:      policy.PIDNamespace,
		NetworkNamespace:  policy.NetworkNamespace,
	}
	
	sm.saveState()
	
	return nil
}

type SecurityPolicy struct {
	SeccompProfile    string
	AppArmorProfile   string
	NoNewPrivileges   bool
	ReadOnlyRootFS    bool
	DropCapabilities  []string
	ForbiddenSyscalls []string
	UserNamespace     bool
	PIDNamespace      bool
	NetworkNamespace  bool
}

func (sm *SandboxManager) ApplyNetworkPolicy(ctx context.Context, sandboxID uuid.UUID, policy *NetworkPolicySpec) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	sandbox, ok := sm.sandboxes[sandboxID]
	if !ok {
		return fmt.Errorf("沙箱 %s 不存在", sandboxID)
	}
	
	sandbox.Network.Policy = policy.Policy
	sandbox.Network.IngressRules = policy.IngressRules
	sandbox.Network.EgressRules = policy.EgressRules
	
	sm.saveState()
	
	return nil
}

type NetworkPolicySpec struct {
	Policy       NetworkPolicy
	IngressRules []NetworkRule
	EgressRules  []NetworkRule
}

func (sm *SandboxManager) Shutdown(ctx context.Context) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	for _, sandbox := range sm.sandboxes {
		if sandbox.Status == AgentStatusRunning {
			if err := sm.stopKubernetesSandbox(ctx, sandbox, true); err != nil {
				log.Printf("停止沙箱 %s 失败: %v", sandbox.ID, err)
			}
		}
	}
	
	if err := sm.saveState(); err != nil {
		log.Printf("保存状态失败: %v", err)
	}
	
	return nil
}
