package coordination

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CoordinationBaseModel struct {
	ID        uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (b *CoordinationBaseModel) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}

type TaskType string

const (
	TaskTypeRealtime    TaskType = "realtime"
	TaskTypeDelayable   TaskType = "delayable"
	TaskTypeBatch       TaskType = "batch"
	TaskTypeInterruptible TaskType = "interruptible"
)

type TaskPriority string

const (
	TaskPriorityHigh   TaskPriority = "high"
	TaskPriorityMedium TaskPriority = "medium"
	TaskPriorityLow    TaskPriority = "low"
)

type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusScheduled  TaskStatus = "scheduled"
	TaskStatusRunning    TaskStatus = "running"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusFailed     TaskStatus = "failed"
	TaskStatusCancelled  TaskStatus = "cancelled"
)

type ComputeTask struct {
	CoordinationBaseModel
	Name            string            `gorm:"size:100;not null;index" json:"name"`
	Type            TaskType          `gorm:"size:20;not null;index" json:"type"`
	Status          TaskStatus        `gorm:"size:20;default:'pending';index" json:"status"`
	Priority        TaskPriority      `gorm:"size:20;default:'medium'" json:"priority"`
	
	ClusterID       *uuid.UUID        `gorm:"type:uuid;index" json:"cluster_id,omitempty"`
	NodeID          *uuid.UUID        `gorm:"type:uuid;index" json:"node_id,omitempty"`
	TenantID        uuid.UUID         `gorm:"type:uuid;index" json:"tenant_id"`
	
	ResourceSpec    TaskResourceSpec  `gorm:"type:jsonb" json:"resource_spec"`
	TimeConstraint  TimeConstraint    `gorm:"type:jsonb" json:"time_constraint"`
	EnergyPreference EnergyPreference  `gorm:"type:jsonb" json:"energy_preference"`
	
	EstimatedPower  float64           `json:"estimated_power"`
	EstimatedDuration int             `json:"estimated_duration"`
	ActualPower     float64           `json:"actual_power"`
	ActualDuration  int               `json:"actual_duration"`
	
	ScheduledStart  *time.Time        `json:"scheduled_start,omitempty"`
	ScheduledEnd    *time.Time        `json:"scheduled_end,omitempty"`
	ActualStart     *time.Time        `json:"actual_start,omitempty"`
	ActualEnd       *time.Time        `json:"actual_end,omitempty"`
	
	EnergyCost      float64           `json:"energy_cost"`
	CarbonEmission  float64           `json:"carbon_emission"`
	GreenRatio      float64           `json:"green_ratio"`
	OptimizationScore float64         `json:"optimization_score"`
	
	RetryCount      int               `json:"retry_count"`
	MaxRetries      int               `json:"max_retries"`
	
	Labels          map[string]string `gorm:"type:jsonb" json:"labels"`
	Annotations     map[string]string `gorm:"type:jsonb" json:"annotations"`
}

type TaskResourceSpec struct {
	CPU           float64 `json:"cpu"`
	Memory        int64   `json:"memory"`
	GPU           int     `json:"gpu"`
	GPUType       string  `json:"gpu_type,omitempty"`
	Storage       int64   `json:"storage"`
	PowerCapacity float64 `json:"power_capacity"`
}

type TimeConstraint struct {
	EarliestStart    *time.Time `json:"earliest_start,omitempty"`
	LatestStart      *time.Time `json:"latest_start,omitempty"`
	Deadline         *time.Time `json:"deadline,omitempty"`
	MaxDelayMinutes  int        `json:"max_delay_minutes"`
	PreferredHours   []int      `json:"preferred_hours,omitempty"`
	AvoidHours       []int      `json:"avoid_hours,omitempty"`
	IsFlexible       bool       `json:"is_flexible"`
	Interruptible    bool       `json:"interruptible"`
	MaxInterruptions int        `json:"max_interruptions"`
}

type EnergyPreference struct {
	MaxPricePerKWh   float64 `json:"max_price_per_kwh"`
	MinGreenRatio    float64 `json:"min_green_ratio"`
	MaxCarbonIntensity float64 `json:"max_carbon_intensity"`
	PreferGreen      bool    `json:"prefer_green"`
	PreferLowPrice   bool    `json:"prefer_low_price"`
	AllowStorage     bool    `json:"allow_storage"`
}

type EnergyResourceType string

const (
	EnergyResourceGrid    EnergyResourceType = "grid"
	EnergyResourceSolar   EnergyResourceType = "solar"
	EnergyResourceWind    EnergyResourceType = "wind"
	EnergyResourceHydro   EnergyResourceType = "hydro"
	EnergyResourceStorage EnergyResourceType = "storage"
	EnergyResourceMixed   EnergyResourceType = "mixed"
)

type EnergyResourceStatus string

const (
	EnergyResourceStatusOnline      EnergyResourceStatus = "online"
	EnergyResourceStatusOffline     EnergyResourceStatus = "offline"
	EnergyResourceStatusMaintenance EnergyResourceStatus = "maintenance"
	EnergyResourceStatusLimited     EnergyResourceStatus = "limited"
)

type EnergyResource struct {
	CoordinationBaseModel
	Name              string              `gorm:"size:100;not null;index" json:"name"`
	Type              EnergyResourceType  `gorm:"size:20;not null;index" json:"type"`
	Status            EnergyResourceStatus `gorm:"size:20;default:'online'" json:"status"`
	
	ClusterID         *uuid.UUID          `gorm:"type:uuid;index" json:"cluster_id,omitempty"`
	NodeID            *uuid.UUID          `gorm:"type:uuid;index" json:"node_id,omitempty"`
	Region            string              `gorm:"size:50;index" json:"region"`
	
	Capacity          float64             `json:"capacity"`
	AvailableCapacity float64             `json:"available_capacity"`
	CurrentOutput     float64             `json:"current_output"`
	
	PricePerKWh       float64             `json:"price_per_kwh"`
	CarbonIntensity   float64             `json:"carbon_intensity"`
	GreenRatio        float64             `json:"green_ratio"`
	
	Reliability       float64             `json:"reliability"`
	ResponseTime      int                 `json:"response_time"`
	
	Forecast          []EnergyForecastPoint `gorm:"type:jsonb" json:"forecast"`
	Schedule          []ResourceSchedule   `gorm:"type:jsonb" json:"schedule"`
	
	Labels            map[string]string   `gorm:"type:jsonb" json:"labels"`
}

type EnergyForecastPoint struct {
	Timestamp       time.Time `json:"timestamp"`
	AvailablePower  float64   `json:"available_power"`
	Price           float64   `json:"price"`
	CarbonIntensity float64   `json:"carbon_intensity"`
	Confidence      float64   `json:"confidence"`
}

type ResourceSchedule struct {
	StartTime string  `json:"start_time"`
	EndTime   string  `json:"end_time"`
	Power     float64 `json:"power"`
	Priority  int     `json:"priority"`
	TaskID    string  `json:"task_id,omitempty"`
}

type StorageResource struct {
	CoordinationBaseModel
	Name             string  `gorm:"size:100;not null;index" json:"name"`
	Status           string  `gorm:"size:20;default:'idle'" json:"status"`
	
	Capacity         float64 `json:"capacity"`
	AvailableEnergy  float64 `json:"available_energy"`
	SOC              float64 `json:"soc"`
	MinSOC           float64 `json:"min_soc"`
	MaxSOC           float64 `json:"max_soc"`
	
	MaxChargeRate    float64 `json:"max_charge_rate"`
	MaxDischargeRate float64 `json:"max_discharge_rate"`
	CurrentPower     float64 `json:"current_power"`
	
	Efficiency       float64 `json:"efficiency"`
	HealthState      float64 `json:"health_state"`
	CycleCount       int     `json:"cycle_count"`
	MaxCycles        int     `json:"max_cycles"`
	
	ClusterID        *uuid.UUID `gorm:"type:uuid;index" json:"cluster_id,omitempty"`
	NodeID           *uuid.UUID `gorm:"type:uuid;index" json:"node_id,omitempty"`
	
	Strategy         StorageCoordinationStrategy `gorm:"type:jsonb" json:"strategy"`
	Schedule         []StorageScheduleItem       `gorm:"type:jsonb" json:"schedule"`
}

type StorageCoordinationStrategy struct {
	Mode               string  `json:"mode"`
	PeakPriceThreshold float64 `json:"peak_price_threshold"`
	ValleyPriceThreshold float64 `json:"valley_price_threshold"`
	ReserveForCritical bool    `json:"reserve_for_critical"`
	MinProfitMargin    float64 `json:"min_profit_margin"`
	MaxDailyCycles     int     `json:"max_daily_cycles"`
}

type StorageScheduleItem struct {
	StartTime string  `json:"start_time"`
	EndTime   string  `json:"end_time"`
	Action    string  `json:"action"`
	Power     float64 `json:"power"`
	TaskID    string  `json:"task_id,omitempty"`
}

type OptimizationObjective string

const (
	ObjectiveMinCost         OptimizationObjective = "min_cost"
	ObjectiveMinCarbon       OptimizationObjective = "min_carbon"
	ObjectiveMaxGreen        OptimizationObjective = "max_green"
	ObjectiveMaxReliability  OptimizationObjective = "max_reliability"
	ObjectiveMinLatency      OptimizationObjective = "min_latency"
	ObjectiveBalanced        OptimizationObjective = "balanced"
)

type OptimizationConfig struct {
	Objectives       []OptimizationObjective `json:"objectives"`
	Weights          OptimizationWeights     `json:"weights"`
	Constraints      OptimizationConstraints `json:"constraints"`
	
	Horizon          int     `json:"horizon"`
	TimeResolution   int     `json:"time_resolution"`
	MaxIterations    int     `json:"max_iterations"`
	ConvergenceThreshold float64 `json:"convergence_threshold"`
}

type OptimizationWeights struct {
	CostWeight        float64 `json:"cost_weight"`
	CarbonWeight      float64 `json:"carbon_weight"`
	GreenWeight       float64 `json:"green_weight"`
	ReliabilityWeight float64 `json:"reliability_weight"`
	LatencyWeight     float64 `json:"latency_weight"`
}

type OptimizationConstraints struct {
	MaxCostPerTask      float64 `json:"max_cost_per_task"`
	MaxCarbonPerTask    float64 `json:"max_carbon_per_task"`
	MinGreenRatio       float64 `json:"min_green_ratio"`
	MinReliability      float64 `json:"min_reliability"`
	MaxLatency          int     `json:"max_latency"`
	MaxStorageUsage     float64 `json:"max_storage_usage"`
}

type OptimizationResult struct {
	TaskID            uuid.UUID           `json:"task_id"`
	ScheduledStart    time.Time           `json:"scheduled_start"`
	ScheduledEnd      time.Time           `json:"scheduled_end"`
	AssignedResources []AssignedResource  `json:"assigned_resources"`
	
	TotalCost         float64             `json:"total_cost"`
	TotalCarbon       float64             `json:"total_carbon"`
	GreenRatio        float64             `json:"green_ratio"`
	Reliability       float64             `json:"reliability"`
	
	Score             float64             `json:"score"`
	ObjectiveValues   map[string]float64  `json:"objective_values"`
	Confidence        float64             `json:"confidence"`
	
	Reason            string              `json:"reason"`
	Alternatives      []AlternativeResult `json:"alternatives,omitempty"`
}

type AssignedResource struct {
	ResourceID   uuid.UUID          `json:"resource_id"`
	ResourceType EnergyResourceType `json:"resource_type"`
	Power        float64            `json:"power"`
	Duration     int                `json:"duration"`
	Cost         float64            `json:"cost"`
	Carbon       float64            `json:"carbon"`
	GreenRatio   float64            `json:"green_ratio"`
}

type AlternativeResult struct {
	ScheduledStart time.Time `json:"scheduled_start"`
	ScheduledEnd   time.Time `json:"scheduled_end"`
	TotalCost      float64   `json:"total_cost"`
	TotalCarbon    float64   `json:"total_carbon"`
	GreenRatio     float64   `json:"green_ratio"`
	Score          float64   `json:"score"`
}

type ScheduleDecision struct {
	TaskID          uuid.UUID          `json:"task_id"`
	Decision        string             `json:"decision"`
	TargetNodeID    *uuid.UUID         `json:"target_node_id,omitempty"`
	TargetClusterID *uuid.UUID         `json:"target_cluster_id,omitempty"`
	
	ScheduledStart  time.Time          `json:"scheduled_start"`
	ScheduledEnd    time.Time          `json:"scheduled_end"`
	
	EnergySource    EnergyResourceType `json:"energy_source"`
	StorageAction   *StorageAction     `json:"storage_action,omitempty"`
	
	EstimatedCost   float64            `json:"estimated_cost"`
	EstimatedCarbon float64            `json:"estimated_carbon"`
	
	Reason          string             `json:"reason"`
	Priority        int                `json:"priority"`
}

type StorageAction struct {
	StorageID   uuid.UUID `json:"storage_id"`
	Action      string    `json:"action"`
	Power       float64   `json:"power"`
	StartTime   time.Time `json:"start_time"`
	Duration    int       `json:"duration"`
}

type PredictionType string

const (
	PredictionTypeLoad        PredictionType = "load"
	PredictionTypePrice       PredictionType = "price"
	PredictionTypeRenewable   PredictionType = "renewable"
	PredictionTypeCarbon      PredictionType = "carbon"
)

type PredictionRequest struct {
	Type          PredictionType `json:"type"`
	Region        string         `json:"region"`
	ResourceID    *uuid.UUID     `json:"resource_id,omitempty"`
	StartTime     time.Time      `json:"start_time"`
	Horizon       int            `json:"horizon"`
	Resolution    int            `json:"resolution"`
}

type PredictionResult struct {
	Type          PredictionType     `json:"type"`
	Region        string             `json:"region"`
	ResourceID    *uuid.UUID         `json:"resource_id,omitempty"`
	GeneratedAt   time.Time          `json:"generated_at"`
	
	Points        []PredictionPoint  `json:"points"`
	Accuracy      float64            `json:"accuracy"`
	Model         string             `json:"model"`
	Confidence    float64            `json:"confidence"`
}

type PredictionPoint struct {
	Timestamp  time.Time `json:"timestamp"`
	Value      float64   `json:"value"`
	LowerBound float64   `json:"lower_bound"`
	UpperBound float64   `json:"upper_bound"`
	Confidence float64   `json:"confidence"`
}

type CoordinationPolicy struct {
	CoordinationBaseModel
	Name              string                  `gorm:"size:100;not null" json:"name"`
	Description       string                  `gorm:"type:text" json:"description"`
	Enabled           bool                    `gorm:"default:true" json:"enabled"`
	
	TaskMatching      TaskMatchingPolicy      `gorm:"type:jsonb" json:"task_matching"`
	PriceOptimization PriceOptimizationPolicy `gorm:"type:jsonb" json:"price_optimization"`
	StorageStrategy   StorageStrategyPolicy   `gorm:"type:jsonb" json:"storage_strategy"`
	CarbonOptimization CarbonOptimizationPolicy `gorm:"type:jsonb" json:"carbon_optimization"`
	
	Priority          int                     `json:"priority"`
	TenantID          uuid.UUID               `gorm:"type:uuid;index" json:"tenant_id"`
}

type TaskMatchingPolicy struct {
	RealtimeTaskStrategy   string  `json:"realtime_task_strategy"`
	DelayableTaskStrategy  string  `json:"delayable_task_strategy"`
	BatchTaskStrategy      string  `json:"batch_task_strategy"`
	
	LocalPriorityThreshold float64 `json:"local_priority_threshold"`
	RemoteFallbackEnabled  bool    `json:"remote_fallback_enabled"`
	MaxQueueTime           int     `json:"max_queue_time"`
}

type PriceOptimizationPolicy struct {
	Enabled                bool    `json:"enabled"`
	PriceThresholdLow      float64 `json:"price_threshold_low"`
	PriceThresholdHigh     float64 `json:"price_threshold_high"`
	MaxDelayForLowPrice    int     `json:"max_delay_for_low_price"`
	PriceForecastWindow    int     `json:"price_forecast_window"`
	DynamicRescheduling     bool    `json:"dynamic_rescheduling"`
}

type StorageStrategyPolicy struct {
	Enabled                bool    `json:"enabled"`
	ChargeThreshold        float64 `json:"charge_threshold"`
	DischargeThreshold     float64 `json:"discharge_threshold"`
	ReserveCapacity        float64 `json:"reserve_capacity"`
	ArbitrageEnabled       bool    `json:"arbitrage_enabled"`
	MinProfitMargin        float64 `json:"min_profit_margin"`
}

type CarbonOptimizationPolicy struct {
	Enabled                bool    `json:"enabled"`
	MaxCarbonIntensity     float64 `json:"max_carbon_intensity"`
	PreferGreenHours       bool    `json:"prefer_green_hours"`
	GreenCertEnabled       bool    `json:"green_cert_enabled"`
	CarbonOffsetEnabled    bool    `json:"carbon_offset_enabled"`
}

type CoordinationMetrics struct {
	Timestamp           time.Time `json:"timestamp"`
	
	TotalTasks          int       `json:"total_tasks"`
	ScheduledTasks      int       `json:"scheduled_tasks"`
	RunningTasks        int       `json:"running_tasks"`
	CompletedTasks      int       `json:"completed_tasks"`
	
	AvgSchedulingDelay  float64   `json:"avg_scheduling_delay"`
	AvgEnergyCost       float64   `json:"avg_energy_cost"`
	AvgCarbonEmission   float64   `json:"avg_carbon_emission"`
	TotalGreenRatio     float64   `json:"total_green_ratio"`
	
	StorageUtilization  float64   `json:"storage_utilization"`
	StorageCycles       int       `json:"storage_cycles"`
	
	CostSavings         float64   `json:"cost_savings"`
	CarbonSavings       float64   `json:"carbon_savings"`
	
	OptimizationScore   float64   `json:"optimization_score"`
}

type CoordinationEvent struct {
	CoordinationBaseModel
	Type        string                 `json:"type"`
	TaskID      *uuid.UUID             `json:"task_id,omitempty"`
	ResourceID  *uuid.UUID             `json:"resource_id,omitempty"`
	ClusterID   *uuid.UUID             `json:"cluster_id,omitempty"`
	
	Severity    string                 `json:"severity"`
	Message     string                 `json:"message"`
	Details     map[string]interface{} `gorm:"type:jsonb" json:"details"`
	
	Processed   bool                   `json:"processed"`
	ProcessedAt *time.Time             `json:"processed_at,omitempty"`
}

type RealtimeData struct {
	Timestamp         time.Time                `json:"timestamp"`
	
	TaskQueue         []ComputeTask            `json:"task_queue"`
	AvailableEnergy   []EnergyResource         `json:"available_energy"`
	StorageStatus     []StorageResource        `json:"storage_status"`
	
	CurrentPrice      float64                  `json:"current_price"`
	PriceTrend        string                   `json:"price_trend"`
	GreenRatio        float64                  `json:"green_ratio"`
	CarbonIntensity   float64                  `json:"carbon_intensity"`
	
	ClusterLoad       map[uuid.UUID]float64    `json:"cluster_load"`
	NodeLoad          map[uuid.UUID]float64    `json:"node_load"`
}

type CoordinationConfig struct {
	Enabled              bool                   `json:"enabled"`
	SchedulingInterval   int                    `json:"scheduling_interval"`
	OptimizationHorizon  int                    `json:"optimization_horizon"`
	MaxConcurrentTasks   int                    `json:"max_concurrent_tasks"`
	
	DefaultPolicy        CoordinationPolicy     `json:"default_policy"`
	
	PredictorConfig      PredictorConfig        `json:"predictor_config"`
	OptimizerConfig      OptimizationConfig     `json:"optimizer_config"`
	SchedulerConfig      SchedulerConfig        `json:"scheduler_config"`
}

type PredictorConfig struct {
	LoadForecastHorizon      int     `json:"load_forecast_horizon"`
	PriceForecastHorizon     int     `json:"price_forecast_horizon"`
	RenewableForecastHorizon int     `json:"renewable_forecast_horizon"`
	UpdateInterval           int     `json:"update_interval"`
	ConfidenceThreshold      float64 `json:"confidence_threshold"`
}

type SchedulerConfig struct {
	QueueSize            int     `json:"queue_size"`
	WorkerCount          int     `json:"worker_count"`
	RetryLimit           int     `json:"retry_limit"`
	Timeout              int     `json:"timeout"`
	PriorityLevels       int     `json:"priority_levels"`
	PreemptionEnabled    bool    `json:"preemption_enabled"`
}
