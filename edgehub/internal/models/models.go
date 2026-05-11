package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BaseModel struct {
	ID        uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (b *BaseModel) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}

type User struct {
	BaseModel
	Name            string    `gorm:"size:100;not null" json:"name"`
	Email           string    `gorm:"size:255;uniqueIndex;not null" json:"email"`
	PasswordHash    string    `gorm:"size:255;not null" json:"-"`
	Phone           string    `gorm:"size:20" json:"phone"`
	Avatar          string    `gorm:"size:500" json:"avatar"`
	Status          string    `gorm:"size:20;default:'active'" json:"status"`
	Role            string    `gorm:"size:20;default:'user'" json:"role"`
	TenantID        *uuid.UUID `gorm:"type:uuid;index" json:"tenant_id,omitempty"`
	LastLoginAt     time.Time  `json:"last_login_at"`
	EmailVerified   bool      `gorm:"default:false" json:"email_verified"`
	TwoFactorEnabled bool     `gorm:"default:false" json:"two_factor_enabled"`
}

type Tenant struct {
	BaseModel
	Name          string  `gorm:"size:100;not null" json:"name"`
	OwnerID       uuid.UUID `gorm:"type:uuid;not null" json:"owner_id"`
	Quota         Quota    `gorm:"type:jsonb" json:"quota"`
	Status        string  `gorm:"size:20;default:'active'" json:"status"`
}

type Quota struct {
	MaxNodes    int `json:"max_nodes"`
	MaxJobs     int `json:"max_jobs"`
	MaxStorage  int `json:"max_storage_gb"`
	MaxCPU      int `json:"max_cpu_cores"`
	MaxGPU      int `json:"max_gpu_count"`
	MonthlyLimit int `json:"monthly_limit"`
}

type Cluster struct {
	BaseModel
	Name          string `gorm:"size:100;not null;index" json:"name"`
	Provider      string `gorm:"size:50;not null" json:"provider"`
	Region        string `gorm:"size:50;not null;index" json:"region"`
	Zone          string `gorm:"size:50" json:"zone"`
	Status        string `gorm:"size:20;default:'active'" json:"status"`
	TenantID      uuid.UUID `gorm:"type:uuid;index" json:"tenant_id"`
	KubeConfig    string    `gorm:"type:text" json:"-"`
	APIEndpoint   string    `gorm:"size:500" json:"api_endpoint"`
	Capacity      Capacity  `gorm:"type:jsonb" json:"capacity"`
	IsEdge        bool      `gorm:"default:false" json:"is_edge"`
	Labels        StringMap `gorm:"type:jsonb" json:"labels"`
}

type Capacity struct {
	CPU         float64 `json:"cpu"`
	Memory      int64   `json:"memory"`
	GPU         int     `json:"gpu"`
	Storage     int64   `json:"storage"`
	NetworkBandwidth int64 `json:"network_bandwidth"`
}

type Node struct {
	BaseModel
	Name          string    `gorm:"size:100;not null;index" json:"name"`
	ClusterID     uuid.UUID `gorm:"type:uuid;index" json:"cluster_id"`
	NodePoolID    *uuid.UUID `gorm:"type:uuid;index" json:"node_pool_id,omitempty"`
	Status        string    `gorm:"size:20;default:'pending'" json:"status"`
	Labels        StringMap `gorm:"type:jsonb" json:"labels"`
	Annotations   StringMap `gorm:"type:jsonb" json:"annotations"`
	
	HardwareInfo  HardwareInfo `gorm:"type:jsonb" json:"hardware_info"`
	NetworkInfo   NetworkInfo  `gorm:"type:jsonb" json:"network_info"`
	Allocatable   Allocatable  `gorm:"type:jsonb" json:"allocatable"`
	Allocated     Allocatable  `gorm:"type:jsonb" json:"allocated"`
	
	HeartbeatAt   time.Time `json:"heartbeat_at"`
	LastSeenAt    time.Time `json:"last_seen_at"`
	
	ExternalIP    string `gorm:"size:50" json:"external_ip"`
	InternalIP    string `gorm:"size:50" json:"internal_ip"`
	Hostname      string `gorm:"size:255" json:"hostname"`
	OS            string `gorm:"size:50" json:"os"`
	KernelVersion string `gorm:"size:50" json:"kernel_version"`
	DockerVersion  string `gorm:"size:50" json:"docker_version"`
	KubeletVersion string `gorm:"size:50" json:"kubelet_version"`
	
	Taints        []Taint `gorm:"type:jsonb" json:"taints"`
	Conditions    []Condition `gorm:"type:jsonb" json:"conditions"`
}

type HardwareInfo struct {
	CPUModel       string `json:"cpu_model"`
	CPUCores       int    `json:"cpu_cores"`
	CPUMhz         float64 `json:"cpu_mhz"`
	MemoryTotal    int64   `json:"memory_total"`
	DiskTotal      int64   `json:"disk_total"`
	GPUCount       int     `json:"gpu_count"`
	GPUDevices     []GPUDevice `gorm:"-" json:"gpu_devices,omitempty"`
}

type GPUDevice struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Vendor   string `json:"vendor"`
	Memory   int64  `json:"memory"`
	Status   string `json:"status"`
}

type NetworkInfo struct {
	Hostname       string `json:"hostname"`
	IPAddresses    map[string]string `json:"ip_addresses"`
	DefaultGateway string `json:"default_gateway"`
	DNS            []string `json:"dns"`
	BandwidthIn    int64   `json:"bandwidth_in"`
	BandwidthOut   int64   `json:"bandwidth_out"`
	Latency        float64 `json:"latency"`
}

type Allocatable struct {
	CPU         float64 `json:"cpu"`
	Memory      int64   `json:"memory"`
	GPU         int     `json:"gpu"`
	Storage     int64   `json:"storage"`
	EphemeralStorage int64 `json:"ephemeral_storage"`
}

type Taint struct {
	Key       string `json:"key"`
	Value     string `json:"value"`
	Effect    string `json:"effect"`
}

type Condition struct {
	Type       string `json:"type"`
	Status     string `json:"status"`
	LastUpdate time.Time `json:"last_update"`
	Reason     string `json:"reason"`
	Message    string `json:"message"`
}

type NodePool struct {
	BaseModel
	ClusterID     uuid.UUID `gorm:"type:uuid;index" json:"cluster_id"`
	Name          string    `gorm:"size:100;not null" json:"name"`
	Labels        StringMap `gorm:"type:jsonb" json:"labels"`
	Annotations   StringMap `gorm:"type:jsonb" json:"annotations"`
	MinSize       int       `json:"min_size"`
	MaxSize       int       `json:"max_size"`
	DesiredSize   int       `json:"desired_size"`
	InstanceType  string    `gorm:"size:50" json:"instance_type"`
	LabelsSelector StringMap `gorm:"type:jsonb" json:"label_selector"`
	Capacity      Allocatable `gorm:"type:jsonb" json:"capacity"`
}

type Job struct {
	BaseModel
	Name          string    `gorm:"size:100;index" json:"name"`
	ClusterID     uuid.UUID `gorm:"type:uuid;index" json:"cluster_id"`
	NodeID        *uuid.UUID `gorm:"type:uuid;index" json:"node_id,omitempty"`
	TenantID      uuid.UUID `gorm:"type:uuid;index" json:"tenant_id"`
	UserID        uuid.UUID `gorm:"type:uuid;index" json:"user_id"`
	Type          string    `gorm:"size:30;not null;index" json:"type"`
	Status        string    `gorm:"size:20;default:'pending';index" json:"status"`
	Priority      int       `gorm:"default:50" json:"priority"`
	
	Spec          JobSpec   `gorm:"type:jsonb" json:"spec"`
	Result        *JobResult `gorm:"type:jsonb" json:"result,omitempty"`
	
	CreatedAt     time.Time `gorm:"index" json:"created_at"`
	ScheduledAt   *time.Time `json:"scheduled_at,omitempty"`
	StartedAt     *time.Time `json:"started_at,omitempty"`
	FinishedAt    *time.Time `json:"finished_at,omitempty"`
	FailedAt      *time.Time `json:"failed_at,omitempty"`
	
	RetryCount    int `gorm:"default:0" json:"retry_count"`
	MaxRetries    int `gorm:"default:3" json:"max_retries"`
	
	DependsOn     []uuid.UUID `gorm:"-" json:"depends_on,omitempty"`
	Namespace     string `gorm:"size:100" json:"namespace"`
	Queue         string `gorm:"size:50;index" json:"queue"`
	Labels        StringMap `gorm:"type:jsonb" json:"labels"`
	Annotations   StringMap `gorm:"type:jsonb" json:"annotations"`
}

type JobSpec struct {
	Image         string            `json:"image"`
	Command       []string          `json:"command"`
	Args          []string          `json:"args"`
	Env           map[string]string `json:"env"`
	WorkingDir    string            `json:"working_dir"`
	Resources     ResourceRequirements `json:"resources"`
	Volumes       []Volume          `json:"volumes"`
	NodeSelector   map[string]string `json:"node_selector"`
	Affinity      *Affinity         `json:"affinity"`
	Tolerations   []Taint           `json:"tolerations"`
	
	BackoffLimit  int `json:"backoff_limit"`
	ActiveDeadlineSeconds int64 `json:"active_deadline_seconds"`
	TTLSecondsAfterFinished int `json:"ttl_seconds_after_finished"`
	
	MaxRunningTime int `json:"max_running_time"`
	MaxIdleTime   int `json:"max_idle_time"`
}

type ResourceRequirements struct {
	Requests ResourceList `json:"requests"`
	Limits   ResourceList `json:"limits"`
}

type ResourceList struct {
	CPU         string `json:"cpu"`
	Memory      string `json:"memory"`
	GPU         string `json:"gpu,omitempty"`
	Storage     string `json:"storage,omitempty"`
}

type Volume struct {
	Name       string `json:"name"`
	MountPath  string `json:"mount_path"`
	ReadOnly   bool   `json:"read_only"`
	ConfigMap  *ConfigMapVolume `json:"config_map,omitempty"`
	Secret     *SecretVolume    `json:"secret,omitempty"`
	EmptyDir   *EmptyDirVolume  `json:"empty_dir,omitempty"`
	PVC        *PVCVolume       `json:"pvc,omitempty"`
	HostPath   *HostPathVolume  `json:"host_path,omitempty"`
}

type ConfigMapVolume struct {
	Name    string            `json:"name"`
	Items   map[string]string `json:"items"`
	DefaultMode *int32        `json:"default_mode,omitempty"`
}

type SecretVolume struct {
	Name    string            `json:"name"`
	Items   map[string]string `json:"items"`
	DefaultMode *int32        `json:"default_mode,omitempty"`
	Optional   bool           `json:"optional,omitempty"`
}

type EmptyDirVolume struct {
	Medium    string `json:"medium"`
	SizeLimit string `json:"size_limit"`
}

type PVCVolume struct {
	ClaimName string `json:"claim_name"`
	ReadOnly  bool   `json:"read_only"`
}

type HostPathVolume struct {
	Path      string `json:"path"`
	Type      string `json:"type"`
}

type Affinity struct {
	NodeAffinity    *NodeAffinity    `json:"node_affinity,omitempty"`
	PodAffinity    *PodAffinity     `json:"pod_affinity,omitempty"`
	PodAntiAffinity *PodAntiAffinity `json:"pod_anti_affinity,omitempty"`
}

type NodeAffinity struct {
	RequiredDuringSchedulingIgnoredDuringExecution  *NodeSelector `json:"required_during_scheduling_ignored_during_execution,omitempty"`
	PreferredDuringSchedulingIgnoredDuringExecution []PreferredTerm `json:"preferred_during_scheduling_ignored_during_execution,omitempty"`
}

type NodeSelector struct {
	NodeSelectorTerms []NodeSelectorTerm `json:"node_selector_terms"`
}

type NodeSelectorTerm struct {
	MatchExpressions []NodeSelectorRequirement `json:"match_expressions,omitempty"`
	MatchFields      []NodeSelectorRequirement `json:"match_fields,omitempty"`
}

type NodeSelectorRequirement struct {
	Key      string   `json:"key"`
	Operator string   `json:"operator"`
	Values   []string `json:"values,omitempty"`
}

type PreferredTerm struct {
	Weight     int         `json:"weight"`
	Preference NodeSelectorTerm `json:"preference"`
}

type PodAffinity struct {
	RequiredDuringSchedulingIgnoredDuringExecution  []PodAffinityTerm `json:"required_during_scheduling_ignored_during_execution,omitempty"`
	PreferredDuringSchedulingIgnoredDuringExecution []WeightedPodAffinityTerm `json:"preferred_during_scheduling_ignored_during_execution,omitempty"`
}

type PodAntiAffinity struct {
	RequiredDuringSchedulingIgnoredDuringExecution  []PodAffinityTerm `json:"required_during_scheduling_ignored_during_execution,omitempty"`
	PreferredDuringSchedulingIgnoredDuringExecution []WeightedPodAffinityTerm `json:"preferred_during_scheduling_ignored_during_execution,omitempty"`
}

type PodAffinityTerm struct {
	LabelSelector *LabelSelector `json:"label_selector,omitempty"`
	Namespaces    []string       `json:"namespaces,omitempty"`
	TopologyKey   string         `json:"topology_key"`
}

type WeightedPodAffinityTerm struct {
	Weight int           `json:"weight"`
	PodAffinityTerm PodAffinityTerm `json:"pod_affinity_term"`
}

type LabelSelector struct {
	MatchLabels      map[string]string        `json:"match_labels,omitempty"`
	MatchExpressions []LabelSelectorRequirement `json:"match_expressions,omitempty"`
}

type LabelSelectorRequirement struct {
	Key      string   `json:"key"`
	Operator string   `json:"operator"`
	Values   []string `json:"values,omitempty"`
}

type JobResult struct {
	ExitCode     int               `json:"exit_code"`
	ErrorMessage string            `json:"error_message,omitempty"`
	PodName      string            `json:"pod_name"`
	NodeIP       string            `json:"node_ip"`
	Metrics      map[string]float64 `json:"metrics,omitempty"`
	ArtifactURL  string            `json:"artifact_url,omitempty"`
}

type MarketOffer struct {
	BaseModel
	ProviderID     uuid.UUID `gorm:"type:uuid;index;not null" json:"provider_id"`
	TenantID       uuid.UUID `gorm:"type:uuid;index" json:"tenant_id"`
	Status         string    `gorm:"size:20;default:'active'" json:"status"`
	ResourceSpec   ResourceSpec `gorm:"type:jsonb" json:"resource_spec"`
	Prices         Prices      `gorm:"type:jsonb" json:"prices"`
	Constraints    Constraints `gorm:"type:jsonb" json:"constraints"`
	Available      bool       `gorm:"default:true" json:"available"`
	ValidFrom      time.Time  `json:"valid_from"`
	ValidUntil     time.Time  `json:"valid_until"`
	Description    string     `gorm:"type:text" json:"description"`
	Region         string     `gorm:"size:50;index" json:"region"`
	ClusterID      uuid.UUID  `gorm:"type:uuid" json:"cluster_id"`
}

type ResourceSpec struct {
	CPU        float64 `json:"cpu"`
	Memory     int64   `json:"memory"`
	GPU        int     `json:"gpu"`
	GPUType    string  `json:"gpu_type,omitempty"`
	Storage    int64   `json:"storage"`
	Bandwidth  int64   `json:"bandwidth"`
	DiskType   string  `json:"disk_type,omitempty"`
}

type Prices struct {
	OnDemand   float64 `json:"on_demand"`
	Reserved1M float64 `json:"reserved_1m"`
	Reserved3M float64 `json:"reserved_3m"`
	Reserved6M float64 `json:"reserved_6m"`
	Reserved12M float64 `json:"reserved_12m"`
	Spot       float64 `json:"spot"`
}

type Constraints struct {
	MinDuration int `json:"min_duration"`
	MaxDuration int `json:"max_duration"`
	MinCPU      float64 `json:"min_cpu"`
	MinMemory   int64   `json:"min_memory"`
	MinGPU      int     `json:"min_gpu"`
}

type MarketOrder struct {
	BaseModel
	OfferID       uuid.UUID `gorm:"type:uuid;index;not null" json:"offer_id"`
	ConsumerID    uuid.UUID `gorm:"type:uuid;index;not null" json:"consumer_id"`
	ProviderID    uuid.UUID `gorm:"type:uuid;index" json:"provider_id"`
	TenantID      uuid.UUID `gorm:"type:uuid;index" json:"tenant_id"`
	Type         string    `gorm:"size:20;not null" json:"type"`
	Status        string    `gorm:"size:20;default:'pending';index" json:"status"`
	ResourceSpec ResourceSpec `gorm:"type:jsonb" json:"resource_spec"`
	Price         float64   `json:"price"`
	Quantity      int       `json:"quantity"`
	Duration      int       `json:"duration"`
	TotalAmount   float64   `json:"total_amount"`
	Currency      string    `gorm:"size:10;default:'USD'" json:"currency"`
	JobID         *uuid.UUID `gorm:"type:uuid;index" json:"job_id,omitempty"`
	FulfilledAt   *time.Time `json:"fulfilled_at,omitempty"`
	CancelledAt   *time.Time `json:"cancelled_at,omitempty"`
	CancelReason  string     `gorm:"type:text" json:"cancel_reason,omitempty"`
}

type Bill struct {
	BaseModel
	TenantID      uuid.UUID `gorm:"type:uuid;index;not null" json:"tenant_id"`
	UserID        uuid.UUID `gorm:"type:uuid;index" json:"user_id"`
	OrderID       uuid.UUID `gorm:"type:uuid;index" json:"order_id"`
	JobID         *uuid.UUID `gorm:"type:uuid;index" json:"job_id,omitempty"`
	ResourceType  string    `gorm:"size:30" json:"resource_type"`
	ResourceSpec  ResourceSpec `gorm:"type:jsonb" json:"resource_spec"`
	Quantity      float64   `json:"quantity"`
	Unit          string    `gorm:"size:20" json:"unit"`
	UnitPrice     float64   `json:"unit_price"`
	TotalAmount   float64   `json:"total_amount"`
	Currency      string    `gorm:"size:10;default:'USD'" json:"currency"`
	PeriodStart   time.Time `json:"period_start"`
	PeriodEnd     time.Time `json:"period_end"`
	Status        string    `gorm:"size:20;default:'pending'" json:"status"`
	PaidAt        *time.Time `json:"paid_at,omitempty"`
	InvoicedAt    *time.Time `json:"invoiced_at,omitempty"`
	ProjectID     *uuid.UUID `gorm:"type:uuid;index" json:"project_id,omitempty"`
}

type Alert struct {
	BaseModel
	TenantID      uuid.UUID `gorm:"type:uuid;index" json:"tenant_id"`
	Name          string    `gorm:"size:100;not null" json:"name"`
	Type          string    `gorm:"size:50" json:"type"`
	Condition     AlertCondition `gorm:"type:jsonb" json:"condition"`
	Level         string    `gorm:"size:20" json:"level"`
	Status        string    `gorm:"size:20;default:'active'" json:"status"`
	Channels      []string  `gorm:"type:jsonb" json:"channels"`
	Cooldown      int       `json:"cooldown"`
	LastTriggered *time.Time `json:"last_triggered,omitempty"`
	Enabled       bool      `gorm:"default:true" json:"enabled"`
}

type AlertCondition struct {
	Metric     string   `json:"metric"`
	Operator   string   `json:"operator"`
	Threshold  float64  `json:"threshold"`
	Duration   int      `json:"duration"`
	FilterLabels map[string]string `json:"filter_labels,omitempty"`
}

type BenchmarkResult struct {
	BaseModel
	NodeID        uuid.UUID `gorm:"type:uuid;index" json:"node_id"`
	TenantID      uuid.UUID `gorm:"type:uuid;index" json:"tenant_id"`
	Type          string    `gorm:"size:50;not null" json:"type"`
	Status        string    `gorm:"size:20" json:"status"`
	Score         float64   `json:"score"`
	Details       BenchmarkDetails `gorm:"type:jsonb" json:"details"`
	RawData       string    `gorm:"type:text" json:"raw_data,omitempty"`
	StartedAt     time.Time `json:"started_at"`
	FinishedAt    *time.Time `json:"finished_at,omitempty"`
}

type BenchmarkDetails struct {
	CPUCost      float64 `json:"cpu_cost,omitempty"`
	MemoryBandwidth float64 `json:"memory_bandwidth,omitempty"`
	NetworkBandwidth float64 `json:"network_bandwidth,omitempty"`
	StorageIOPS  float64 `json:"storage_iops,omitempty"`
	GPUPerformance float64 `json:"gpu_performance,omitempty"`
	Latency      float64 `json:"latency,omitempty"`
}

type NodeMetrics struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	NodeID      uuid.UUID `gorm:"type:uuid;index;not null" json:"node_id"`
	Timestamp   time.Time `gorm:"index;not null" json:"timestamp"`
	
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage float64 `json:"memory_usage"`
	DiskUsage   float64 `json:"disk_usage"`
	NetworkIn   int64   `json:"network_in"`
	NetworkOut  int64   `json:"network_out"`
	
	GPUUsage    []GPUUsage `gorm:"-" json:"gpu_usage,omitempty"`
	GPUUtilRaw  string     `gorm:"type:text" json:"-"`
}

type GPUUsage struct {
	ID       string  `json:"id"`
	Util     float64 `json:"utilization"`
	Memory   float64 `json:"memory"`
	Temperature float64 `json:"temperature"`
	Power    float64 `json:"power"`
}

type StringMap map[string]interface{}

type APIKey struct {
	BaseModel
	UserID      uuid.UUID `gorm:"type:uuid;index;not null" json:"user_id"`
	Name        string    `gorm:"size:100" json:"name"`
	KeyHash     string    `gorm:"size:255;not null" json:"-"`
	Prefix      string    `gorm:"size:20" json:"prefix"`
	Scopes      []string `gorm:"type:jsonb" json:"scopes"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	Status      string    `gorm:"size:20;default:'active'" json:"status"`
}
