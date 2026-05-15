package agent

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AgentStatus string

const (
	AgentStatusPending    AgentStatus = "pending"
	AgentStatusCreating   AgentStatus = "creating"
	AgentStatusRunning    AgentStatus = "running"
	AgentStatusPaused     AgentStatus = "paused"
	AgentStatusStopping   AgentStatus = "stopping"
	AgentStatusStopped    AgentStatus = "stopped"
	AgentStatusError      AgentStatus = "error"
	AgentStatusTerminated AgentStatus = "terminated"
)

type SandboxRuntime string

const (
	RuntimeRunsc SandboxRuntime = "runsc"
	RuntimeKata  SandboxRuntime = "kata"
	RuntimeRunc  SandboxRuntime = "runc"
)

type RuntimeStatus string

const (
	RuntimeStatusAvailable   RuntimeStatus = "available"
	RuntimeStatusUnavailable RuntimeStatus = "unavailable"
	RuntimeStatusError       RuntimeStatus = "error"
)

type RuntimeType string

const (
	RuntimeTypeRunsc RuntimeType = "runsc"
	RuntimeTypeKata  RuntimeType = "kata"
	RuntimeTypeRunc  RuntimeType = "runc"
)

type ExecutionStatus string

const (
	ExecutionStatusPending   ExecutionStatus = "pending"
	ExecutionStatusRunning   ExecutionStatus = "running"
	ExecutionStatusCompleted ExecutionStatus = "completed"
	ExecutionStatusFailed    ExecutionStatus = "failed"
	ExecutionStatusTimeout   ExecutionStatus = "timeout"
	ExecutionStatusCancelled ExecutionStatus = "cancelled"
)

type ToolType string

const (
	ToolTypeFunction    ToolType = "function"
	ToolTypeHTTP        ToolType = "http"
	ToolTypeShell       ToolType = "shell"
	ToolTypeCode        ToolType = "code"
	ToolTypeFile        ToolType = "file"
	ToolTypeDatabase    ToolType = "database"
	ToolTypeAPI         ToolType = "api"
	ToolTypeCustom      ToolType = "custom"
)

type NetworkPolicy string

const (
	NetworkPolicyIsolated   NetworkPolicy = "isolated"
	NetworkPolicyRestricted NetworkPolicy = "restricted"
	NetworkPolicyBridge     NetworkPolicy = "bridge"
	NetworkPolicyHost       NetworkPolicy = "host"
)

type Agent struct {
	ID          uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	Name        string         `gorm:"size:100;not null;uniqueIndex" json:"name"`
	Description string         `gorm:"type:text" json:"description"`
	Status      AgentStatus    `gorm:"size:20;default:'pending';index" json:"status"`

	TenantID    uuid.UUID      `gorm:"type:uuid;index;not null" json:"tenant_id"`
	UserID      uuid.UUID      `gorm:"type:uuid;index;not null" json:"user_id"`

	SandboxID   *uuid.UUID     `gorm:"type:uuid;index" json:"sandbox_id,omitempty"`
	Sandbox     *Sandbox       `gorm:"foreignKey:SandboxID" json:"sandbox,omitempty"`

	Runtime     SandboxRuntime `gorm:"size:20;default:'runsc'" json:"runtime"`
	
	Spec        AgentSpec      `gorm:"type:jsonb" json:"spec"`
	Config      AgentConfig    `gorm:"type:jsonb" json:"config"`
	
	Tools       []Tool         `gorm:"foreignKey:AgentID;constraint:OnDelete:CASCADE" json:"tools,omitempty"`
	Executions  []Execution    `gorm:"foreignKey:AgentID;constraint:OnDelete:CASCADE" json:"executions,omitempty"`

	MemoryLimit    int64  `gorm:"default:536870912" json:"memory_limit"`
	CPULimit       string `gorm:"size:20;default:'1'" json:"cpu_limit"`
	Timeout        int    `gorm:"default:300" json:"timeout"`
	MaxExecutions  int    `gorm:"default:100" json:"max_executions"`
	
	Environment    map[string]string `gorm:"type:jsonb" json:"environment"`
	Labels         map[string]string `gorm:"type:jsonb" json:"labels"`
	Annotations    map[string]string `gorm:"type:jsonb" json:"annotations"`

	LastActiveAt   *time.Time `json:"last_active_at,omitempty"`
	StartedAt      *time.Time `json:"started_at,omitempty"`
	FinishedAt     *time.Time `json:"finished_at,omitempty"`
	
	ErrorCount     int    `gorm:"default:0" json:"error_count"`
	LastError      string `gorm:"type:text" json:"last_error,omitempty"`
}

func (a *Agent) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	if a.Environment == nil {
		a.Environment = make(map[string]string)
	}
	if a.Labels == nil {
		a.Labels = make(map[string]string)
	}
	if a.Annotations == nil {
		a.Annotations = make(map[string]string)
	}
	return nil
}

type AgentSpec struct {
	Image       string            `json:"image"`
	Command     []string          `json:"command,omitempty"`
	Args        []string          `json:"args,omitempty"`
	WorkingDir  string            `json:"working_dir,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
	Volumes     []VolumeMount     `json:"volumes,omitempty"`
	
	SecurityContext SecurityContext `json:"security_context,omitempty"`
	
	LivenessProbe  *Probe `json:"liveness_probe,omitempty"`
	ReadinessProbe *Probe `json:"readiness_probe,omitempty"`
}

type AgentConfig struct {
	MaxMemoryMB      int    `json:"max_memory_mb"`
	MaxCPUCores      string `json:"max_cpu_cores"`
	MaxDiskMB        int    `json:"max_disk_mb"`
	MaxNetworkMbps   int    `json:"max_network_mbps"`
	MaxProcesses     int    `json:"max_processes"`
	MaxOpenFiles     int    `json:"max_open_files"`
	ExecutionTimeout int    `json:"execution_timeout"`
	IdleTimeout      int    `json:"idle_timeout"`
	
	EnableNetwork    bool   `json:"enable_network"`
	EnableGPU        bool   `json:"enable_gpu"`
	EnableDebug      bool   `json:"enable_debug"`
}

type SecurityContext struct {
	RunAsUser           *int64   `json:"run_as_user,omitempty"`
	RunAsGroup          *int64   `json:"run_as_group,omitempty"`
	RunAsNonRoot        bool     `json:"run_as_non_root"`
	ReadOnlyRootFS      bool     `json:"read_only_root_fs"`
	AllowPrivilegeEscalation bool `json:"allow_privilege_escalation"`
	Capabilities        *Capabilities `json:"capabilities,omitempty"`
	SeccompProfile      string   `json:"seccomp_profile,omitempty"`
	AppArmorProfile     string   `json:"apparmor_profile,omitempty"`
	NoNewPrivileges     bool     `json:"no_new_privileges"`
}

type Capabilities struct {
	Add  []string `json:"add,omitempty"`
	Drop []string `json:"drop,omitempty"`
}

type VolumeMount struct {
	Name      string `json:"name"`
	MountPath string `json:"mount_path"`
	SubPath   string `json:"sub_path,omitempty"`
	ReadOnly  bool   `json:"read_only"`
	
	EmptyDir  *EmptyDirVolumeSource  `json:"empty_dir,omitempty"`
	ConfigMap *ConfigMapVolumeSource `json:"config_map,omitempty"`
	Secret    *SecretVolumeSource    `json:"secret,omitempty"`
	PVC       *PVCVolumeSource       `json:"pvc,omitempty"`
}

type EmptyDirVolumeSource struct {
	Medium    string `json:"medium,omitempty"`
	SizeLimit string `json:"size_limit,omitempty"`
}

type ConfigMapVolumeSource struct {
	Name       string            `json:"name"`
	Items      map[string]string `json:"items,omitempty"`
	DefaultMode *int32           `json:"default_mode,omitempty"`
}

type SecretVolumeSource struct {
	Name       string            `json:"name"`
	Items      map[string]string `json:"items,omitempty"`
	DefaultMode *int32           `json:"default_mode,omitempty"`
	Optional   bool              `json:"optional,omitempty"`
}

type PVCVolumeSource struct {
	ClaimName string `json:"claim_name"`
	ReadOnly  bool   `json:"read_only"`
}

type Probe struct {
	Exec      *ExecAction    `json:"exec,omitempty"`
	HTTPGet   *HTTPGetAction `json:"http_get,omitempty"`
	TCPSocket *TCPAction     `json:"tcp_socket,omitempty"`
	
	InitialDelaySeconds int32 `json:"initial_delay_seconds"`
	TimeoutSeconds      int32 `json:"timeout_seconds"`
	PeriodSeconds       int32 `json:"period_seconds"`
	SuccessThreshold    int32 `json:"success_threshold"`
	FailureThreshold    int32 `json:"failure_threshold"`
}

type ExecAction struct {
	Command []string `json:"command"`
}

type HTTPGetAction struct {
	Path       string        `json:"path"`
	Port       int32         `json:"port"`
	Host       string        `json:"host,omitempty"`
	Scheme     string        `json:"scheme,omitempty"`
	HTTPHeaders []HTTPHeader `json:"http_headers,omitempty"`
}

type HTTPHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type TCPAction struct {
	Port int32  `json:"port"`
	Host string `json:"host,omitempty"`
}

type Sandbox struct {
	ID          uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-""`

	Name        string         `gorm:"size:100;not null;uniqueIndex" json:"name"`
	Status      AgentStatus    `gorm:"size:20;default:'pending';index" json:"status"`

	TenantID    uuid.UUID      `gorm:"type:uuid;index;not null" json:"tenant_id"`
	NodeID      *uuid.UUID     `gorm:"type:uuid;index" json:"node_id,omitempty"`

	Runtime     SandboxRuntime `gorm:"size:20;default:'runsc'" json:"runtime"`
	RuntimeInfo RuntimeInfo    `gorm:"type:jsonb" json:"runtime_info"`

	Config      SandboxConfig  `gorm:"type:jsonb" json:"config"`
	Resources   ResourceSpec   `gorm:"type:jsonb" json:"resources"`
	Network     NetworkConfig  `gorm:"type:jsonb" json:"network"`
	Security    SecurityConfig `gorm:"type:jsonb" json:"security"`

	PodName     string         `gorm:"size:100" json:"pod_name"`
	Namespace   string         `gorm:"size:100;default:'agent-sandbox'" json:"namespace"`
	ContainerID string         `gorm:"size:100" json:"container_id"`

	StartedAt   *time.Time     `json:"started_at,omitempty"`
	StoppedAt   *time.Time     `json:"stopped_at,omitempty"`
	
	Metrics     SandboxMetrics `gorm:"type:jsonb" json:"metrics"`
	LastError   string         `gorm:"type:text" json:"last_error,omitempty"`
	
	Labels      map[string]string `gorm:"type:jsonb" json:"labels"`
	Annotations map[string]string `gorm:"type:jsonb" json:"annotations"`
}

func (s *Sandbox) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	if s.Labels == nil {
		s.Labels = make(map[string]string)
	}
	if s.Annotations == nil {
		s.Annotations = make(map[string]string)
	}
	return nil
}

type RuntimeConfig struct {
	Debug           bool              `json:"debug"`
	DebugLog        string            `json:"debug_log"`
	Root            string            `json:"root"`
	State           string            `json:"state"`
	Network         NetworkRuntimeConfig `json:"network"`
	Security        SecurityRuntimeConfig `json:"security"`
	Platform        string            `json:"platform"`
	Rootless        bool              `json:"rootless"`
	ExtraArgs       map[string]string `json:"extra_args"`
}

type NetworkRuntimeConfig struct {
	MTU           int      `json:"mtu"`
	Enable        bool     `json:"enable"`
	NetworkMode   string   `json:"network_mode"`
	DNSServers    []string `json:"dns_servers"`
}

type SecurityRuntimeConfig struct {
	SeccompProfile   string   `json:"seccomp_profile"`
	AppArmorProfile  string   `json:"apparmor_profile"`
	NoNewPrivileges  bool     `json:"no_new_privileges"`
	DropCapabilities []string `json:"drop_capabilities"`
}

type RuntimeInfo struct {
	Type         RuntimeType     `json:"type"`
	Path         string          `json:"path"`
	Version      string          `json:"version"`
	Status       RuntimeStatus   `json:"status"`
	Config       RuntimeConfig   `json:"config"`
	Capabilities []string        `json:"capabilities"`
	Platform     string          `json:"platform"`
	RuntimePath  string          `json:"runtime_path"`
	RuntimeType  string          `json:"runtime_type"`
	APIVersion   string          `json:"api_version"`
}

type SandboxConfig struct {
	RootfsImage   string `json:"rootfs_image"`
	RootfsSize    int64  `json:"rootfs_size"`
	SharedFS      bool   `json:"shared_fs"`
	OverlayFS     bool   `json:"overlay_fs"`
	
	Stdin         bool   `json:"stdin"`
	Tty           bool   `json:"tty"`
	
	LogDriver     string `json:"log_driver"`
	LogOpts       map[string]string `json:"log_opts,omitempty"`
	
	RestartPolicy string `json:"restart_policy"`
	AutoRemove    bool   `json:"auto_remove"`
}

type ResourceSpec struct {
	CPURequest    string `json:"cpu_request"`
	CPULimit      string `json:"cpu_limit"`
	MemoryRequest int64  `json:"memory_request"`
	MemoryLimit   int64  `json:"memory_limit"`
	
	GPURequest    int    `json:"gpu_request"`
	GPUType       string `json:"gpu_type,omitempty"`
	
	EphemeralStorage int64 `json:"ephemeral_storage"`
	
	ProcessLimit int `json:"process_limit"`
	FileLimit    int `json:"file_limit"`
}

type NetworkConfig struct {
	Policy       NetworkPolicy `json:"policy"`
	NetworkName  string        `json:"network_name,omitempty"`
	IPAddress    string        `json:"ip_address,omitempty"`
	MacAddress   string        `json:"mac_address,omitempty"`
	
	DNSServers   []string      `json:"dns_servers,omitempty"`
	DNSSearch    []string      `json:"dns_search,omitempty"`
	
	PortMappings []PortMapping `json:"port_mappings,omitempty"`
	
	IngressRules []NetworkRule `json:"ingress_rules,omitempty"`
	EgressRules  []NetworkRule `json:"egress_rules,omitempty"`
	
	BandwidthIn  int64         `json:"bandwidth_in,omitempty"`
	BandwidthOut int64         `json:"bandwidth_out,omitempty"`
}

type PortMapping struct {
	HostPort      int32  `json:"host_port"`
	ContainerPort int32  `json:"container_port"`
	Protocol      string `json:"protocol"`
	HostIP        string `json:"host_ip,omitempty"`
}

type NetworkRule struct {
	FromPort    int32    `json:"from_port,omitempty"`
	ToPort      int32    `json:"to_port,omitempty"`
	Protocol    string   `json:"protocol"`
	Source      string   `json:"source,omitempty"`
	Destination string   `json:"destination,omitempty"`
	Action      string   `json:"action"`
}

type SecurityConfig struct {
	SeccompProfile   string   `json:"seccomp_profile,omitempty"`
	AppArmorProfile  string   `json:"apparmor_profile,omitempty"`
	SelinuxContext   string   `json:"selinux_context,omitempty"`
	
	NoNewPrivileges  bool     `json:"no_new_privileges"`
	ReadOnlyRootFS   bool     `json:"read_only_root_fs"`
	
	DropCapabilities []string `json:"drop_capabilities,omitempty"`
	AddCapabilities  []string `json:"add_capabilities,omitempty"`
	
	ForbiddenSyscalls []string `json:"forbidden_syscalls,omitempty"`
	AllowedSyscalls   []string `json:"allowed_syscalls,omitempty"`
	
	UserNamespace    bool     `json:"user_namespace"`
	PIDNamespace     bool     `json:"pid_namespace"`
	NetworkNamespace bool     `json:"network_namespace"`
	IPCNamespace     bool     `json:"ipc_namespace"`
	UTSNamespace     bool     `json:"uts_namespace"`
}

type SandboxMetrics struct {
	CPUUsage     float64   `json:"cpu_usage"`
	MemoryUsage  int64     `json:"memory_usage"`
	DiskUsage    int64     `json:"disk_usage"`
	NetworkIn    int64     `json:"network_in"`
	NetworkOut   int64     `json:"network_out"`
	
	ProcessCount int       `json:"process_count"`
	FileCount    int       `json:"file_count"`
	
	LastUpdated  time.Time `json:"last_updated"`
}

type Execution struct {
	ID          uuid.UUID        `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
	DeletedAt   gorm.DeletedAt   `gorm:"index" json:"-""`

	AgentID     uuid.UUID        `gorm:"type:uuid;index;not null" json:"agent_id"`
	SandboxID   uuid.UUID        `gorm:"type:uuid;index;not null" json:"sandbox_id"`

	Type        ExecutionType    `gorm:"size:30;not null" json:"type"`
	Status      ExecutionStatus  `gorm:"size:20;default:'pending';index" json:"status"`

	Input       string           `gorm:"type:text" json:"input"`
	Output      string           `gorm:"type:text" json:"output"`
	Error       string           `gorm:"type:text" json:"error,omitempty"`

	Command     []string         `gorm:"type:jsonb" json:"command,omitempty"`
	Code        string           `gorm:"type:text" json:"code,omitempty"`
	Language    string           `gorm:"size:20" json:"language,omitempty"`

	ExitCode    *int             `json:"exit_code,omitempty"`
	Signal      *int             `json:"signal,omitempty"`

	StartedAt   *time.Time       `json:"started_at,omitempty"`
	FinishedAt  *time.Time       `json:"finished_at,omitempty"`
	Duration    int64            `json:"duration"`

	Timeout     int              `gorm:"default:300" json:"timeout"`
	MaxMemory   int64            `json:"max_memory,omitempty"`
	MaxCPU      string           `gorm:"size:20" json:"max_cpu,omitempty"`

	Metrics     ExecutionMetrics `gorm:"type:jsonb" json:"metrics"`
	
	Labels      map[string]string `gorm:"type:jsonb" json:"labels"`
	Annotations map[string]string `gorm:"type:jsonb" json:"annotations"`
}

type ExecutionType string

const (
	ExecutionTypeCode   ExecutionType = "code"
	ExecutionTypeTool   ExecutionType = "tool"
	ExecutionTypeShell  ExecutionType = "shell"
	ExecutionTypeHTTP   ExecutionType = "http"
	ExecutionTypeScript ExecutionType = "script"
)

func (e *Execution) BeforeCreate(tx *gorm.DB) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	if e.Labels == nil {
		e.Labels = make(map[string]string)
	}
	if e.Annotations == nil {
		e.Annotations = make(map[string]string)
	}
	return nil
}

type ExecutionMetrics struct {
	CPUSeconds     float64 `json:"cpu_seconds"`
	MemoryMaxBytes int64   `json:"memory_max_bytes"`
	DiskReadBytes  int64   `json:"disk_read_bytes"`
	DiskWriteBytes int64   `json:"disk_write_bytes"`
	NetworkInBytes int64   `json:"network_in_bytes"`
	NetworkOutBytes int64  `json:"network_out_bytes"`
	
	ProcessCount   int     `json:"process_count"`
	ThreadCount    int     `json:"thread_count"`
	FileDescriptorCount int `json:"file_descriptor_count"`
}

type Tool struct {
	ID          uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-""`

	AgentID     uuid.UUID      `gorm:"type:uuid;index;not null" json:"agent_id"`

	Name        string         `gorm:"size:100;not null;index" json:"name"`
	Type        ToolType       `gorm:"size:20;not null" json:"type"`
	Description string         `gorm:"type:text" json:"description"`

	Schema      ToolSchema     `gorm:"type:jsonb" json:"schema"`
	Config      ToolConfig     `gorm:"type:jsonb" json:"config"`

	Enabled     bool           `gorm:"default:true" json:"enabled"`
	Priority    int            `gorm:"default:50" json:"priority"`
	
	Timeout     int            `gorm:"default:60" json:"timeout"`
	MaxRetries  int            `gorm:"default:3" json:"max_retries"`
	
	Permissions []string       `gorm:"type:jsonb" json:"permissions"`
	
	Labels      map[string]string `gorm:"type:jsonb" json:"labels"`
}

func (t *Tool) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	if t.Labels == nil {
		t.Labels = make(map[string]string)
	}
	return nil
}

type ToolSchema struct {
	Type       string                    `json:"type"`
	Properties map[string]PropertySchema `json:"properties,omitempty"`
	Required   []string                  `json:"required,omitempty"`
	
	Input      string                    `json:"input,omitempty"`
	Output     string                    `json:"output,omitempty"`
	Examples   []ToolExample             `json:"examples,omitempty"`
}

type PropertySchema struct {
	Type        string   `json:"type"`
	Description string   `json:"description,omitempty"`
	Enum        []string `json:"enum,omitempty"`
	Default     interface{} `json:"default,omitempty"`
	Minimum     *float64 `json:"minimum,omitempty"`
	Maximum     *float64 `json:"maximum,omitempty"`
	MinLength   *int     `json:"min_length,omitempty"`
	MaxLength   *int     `json:"max_length,omitempty"`
	Pattern     string   `json:"pattern,omitempty"`
}

type ToolExample struct {
	Input  interface{} `json:"input"`
	Output interface{} `json:"output"`
}

type ToolConfig struct {
	Endpoint    string            `json:"endpoint,omitempty"`
	Method      string            `json:"method,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Auth        *ToolAuth         `json:"auth,omitempty"`
	
	Command     []string          `json:"command,omitempty"`
	Script      string            `json:"script,omitempty"`
	Interpreter string            `json:"interpreter,omitempty"`
	
	Environment map[string]string `json:"environment,omitempty"`
	WorkingDir  string            `json:"working_dir,omitempty"`
	
	RateLimit   int               `json:"rate_limit,omitempty"`
	BurstLimit  int               `json:"burst_limit,omitempty"`
}

type ToolAuth struct {
	Type        string `json:"type"`
	Token       string `json:"token,omitempty"`
	Username    string `json:"username,omitempty"`
	Password    string `json:"password,omitempty"`
	APIKey      string `json:"api_key,omitempty"`
	APIKeyHeader string `json:"api_key_header,omitempty"`
}

type DRAConfig struct {
	Enabled       bool     `json:"enabled"`
	DriverName    string   `json:"driver_name"`
	ResourceClass string   `json:"resource_class"`
	
	GPUDevices    []string `json:"gpu_devices,omitempty"`
	FPGADevices   []string `json:"fpga_devices,omitempty"`
	TPUDevices    []string `json:"tpu_devices,omitempty"`
	
	NodeSelector  map[string]string `json:"node_selector,omitempty"`
}

type SandboxEventRecord struct {
	ID          uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`

	SandboxID   uuid.UUID      `gorm:"type:uuid;index;not null" json:"sandbox_id"`
	AgentID     *uuid.UUID     `gorm:"type:uuid;index" json:"agent_id,omitempty"`

	Type        string         `gorm:"size:50;not null;index" json:"type"`
	Reason      string         `gorm:"size:100" json:"reason"`
	Message     string         `gorm:"type:text" json:"message"`
	
	Source      string         `gorm:"size:50" json:"source"`
	Timestamp   time.Time      `json:"timestamp"`
	
	Data        map[string]interface{} `gorm:"type:jsonb" json:"data,omitempty"`
}

type SandboxEventType string

const (
	SandboxEventTypeCreated    SandboxEventType = "created"
	SandboxEventTypeStarted    SandboxEventType = "started"
	SandboxEventTypeStopped    SandboxEventType = "stopped"
	SandboxEventTypePaused     SandboxEventType = "paused"
	SandboxEventTypeResumed    SandboxEventType = "resumed"
	SandboxEventTypeDeleted    SandboxEventType = "deleted"
	SandboxEventTypeError      SandboxEventType = "error"
	SandboxEventTypeOOMKilled  SandboxEventType = "oom_killed"
	SandboxEventTypeEvicted    SandboxEventType = "evicted"
)

type CreateAgentRequest struct {
	Name        string            `json:"name" binding:"required"`
	Description string            `json:"description"`
	UserID      uuid.UUID         `json:"user_id"`
	
	Runtime     SandboxRuntime    `json:"runtime"`
	
	Spec        AgentSpec         `json:"spec"`
	Config      AgentConfig       `json:"config"`
	
	MemoryLimit int64             `json:"memory_limit"`
	CPULimit    string            `json:"cpu_limit"`
	Timeout     int               `json:"timeout"`
	
	Tools       []CreateToolRequest `json:"tools,omitempty"`
	
	Environment map[string]string `json:"environment,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
}

type CreateToolRequest struct {
	Name        string     `json:"name" binding:"required"`
	Type        ToolType   `json:"type" binding:"required"`
	Description string     `json:"description"`
	
	Schema      ToolSchema `json:"schema"`
	Config      ToolConfig `json:"config"`
	
	Enabled     bool       `json:"enabled"`
	Timeout     int        `json:"timeout"`
	MaxRetries  int        `json:"max_retries"`
	Permissions []string   `json:"permissions,omitempty"`
}

type ExecuteRequest struct {
	AgentID   uuid.UUID `json:"agent_id" binding:"required"`
	
	Type      ExecutionType `json:"type" binding:"required"`
	
	Code      string   `json:"code,omitempty"`
	Language  string   `json:"language,omitempty"`
	Command   []string `json:"command,omitempty"`
	Input     string   `json:"input,omitempty"`
	
	Timeout   int      `json:"timeout,omitempty"`
	MaxMemory int64    `json:"max_memory,omitempty"`
}

type ExecuteResponse struct {
	ExecutionID uuid.UUID        `json:"execution_id"`
	Status      ExecutionStatus  `json:"status"`
	Output      string           `json:"output,omitempty"`
	Error       string           `json:"error,omitempty"`
	ExitCode    *int             `json:"exit_code,omitempty"`
	Duration    int64            `json:"duration"`
	Metrics     ExecutionMetrics `json:"metrics"`
}

type SandboxStatusResponse struct {
	ID            uuid.UUID      `json:"id"`
	Name          string         `json:"name"`
	Status        AgentStatus    `json:"status"`
	Runtime       SandboxRuntime `json:"runtime"`
	
	ContainerID   string         `json:"container_id,omitempty"`
	PodName       string         `json:"pod_name,omitempty"`
	Namespace     string         `json:"namespace"`
	
	Metrics       SandboxMetrics `json:"metrics"`
	
	StartedAt     *time.Time     `json:"started_at,omitempty"`
	LastActiveAt  *time.Time     `json:"last_active_at,omitempty"`
	
	AgentCount    int            `json:"agent_count"`
	ExecutionCount int           `json:"execution_count"`
}

type ListSandboxesRequest struct {
	TenantID uuid.UUID   `form:"tenant_id" binding:"required"`
	Status   AgentStatus `form:"status"`
	Runtime  SandboxRuntime `form:"runtime"`
	
	Page     int         `form:"page" binding:"min=1"`
	PageSize int         `form:"page_size" binding:"min=1,max=100"`
	
	Labels   map[string]string `form:"labels"`
}

type ListSandboxesResponse struct {
	Sandboxes []Sandbox `json:"sandboxes"`
	Total     int64     `json:"total"`
	Page      int       `json:"page"`
	PageSize  int       `json:"page_size"`
}

type ListAgentsRequest struct {
	TenantID uuid.UUID   `form:"tenant_id" binding:"required"`
	Status   AgentStatus `form:"status"`
	Runtime  SandboxRuntime `form:"runtime"`
	
	Page     int         `form:"page" binding:"min=1"`
	PageSize int         `form:"page_size" binding:"min=1,max=100"`
	
	Labels   map[string]string `form:"labels"`
}

type ListAgentsResponse struct {
	Agents     []Agent `json:"agents"`
	Total      int64   `json:"total"`
	Page       int     `json:"page"`
	PageSize   int     `json:"page_size"`
}
