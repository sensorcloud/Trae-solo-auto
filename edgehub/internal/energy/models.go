package energy

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type EnergyBaseModel struct {
	ID        uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (b *EnergyBaseModel) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}

type PowerSourceType string

const (
	PowerSourceSolar      PowerSourceType = "solar"
	PowerSourceWind       PowerSourceType = "wind"
	PowerSourceHydro      PowerSourceType = "hydro"
	PowerSourceBiomass    PowerSourceType = "biomass"
	PowerSourceGrid       PowerSourceType = "grid"
	PowerSourceStorage    PowerSourceType = "storage"
	PowerSourceGenerator  PowerSourceType = "generator"
)

type PowerSourceStatus string

const (
	PowerSourceStatusOnline    PowerSourceStatus = "online"
	PowerSourceStatusOffline   PowerSourceStatus = "offline"
	PowerSourceStatusMaintenance PowerSourceStatus = "maintenance"
	PowerSourceStatusFault     PowerSourceStatus = "fault"
)

type PowerSource struct {
	EnergyBaseModel
	Name           string           `gorm:"size:100;not null" json:"name"`
	Type           PowerSourceType  `gorm:"size:20;not null;index" json:"type"`
	Status         PowerSourceStatus `gorm:"size:20;default:'online'" json:"status"`
	Capacity       float64          `json:"capacity"`
	Unit           string           `gorm:"size:20;default:'kW'" json:"unit"`
	Location       Location         `gorm:"type:jsonb" json:"location"`
	ClusterID      *uuid.UUID       `gorm:"type:uuid;index" json:"cluster_id,omitempty"`
	NodeID         *uuid.UUID       `gorm:"type:uuid;index" json:"node_id,omitempty"`
	TenantID       uuid.UUID        `gorm:"type:uuid;index" json:"tenant_id"`
	
	RealtimeOutput float64          `json:"realtime_output"`
	TotalGenerated float64          `json:"total_generated"`
	
	Efficiency     float64          `json:"efficiency"`
	CarbonIntensity float64         `json:"carbon_intensity"`
	
	Metadata       EnergyMetadata   `gorm:"type:jsonb" json:"metadata"`
	Labels         EnergyStringMap  `gorm:"type:jsonb" json:"labels"`
}

type Location struct {
	Region    string  `json:"region"`
	Zone      string  `json:"zone"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Address   string  `json:"address"`
}

type EnergyMetadata struct {
	Manufacturer   string            `json:"manufacturer,omitempty"`
	Model          string            `json:"model,omitempty"`
	SerialNumber   string            `json:"serial_number,omitempty"`
	InstallDate    string            `json:"install_date,omitempty"`
	WarrantyExpiry string            `json:"warranty_expiry,omitempty"`
	CustomFields   map[string]string `json:"custom_fields,omitempty"`
}

type EnergyStringMap map[string]string

type StorageDeviceStatus string

const (
	StorageStatusIdle       StorageDeviceStatus = "idle"
	StorageStatusCharging   StorageDeviceStatus = "charging"
	StorageStatusDischarging StorageDeviceStatus = "discharging"
	StorageStatusMaintenance StorageDeviceStatus = "maintenance"
	StorageStatusFault      StorageDeviceStatus = "fault"
)

type StorageDevice struct {
	EnergyBaseModel
	Name              string              `gorm:"size:100;not null" json:"name"`
	Status            StorageDeviceStatus `gorm:"size:20;default:'idle'" json:"status"`
	Capacity          float64             `json:"capacity"`
	Unit              string              `gorm:"size:20;default:'kWh'" json:"unit"`
	MaxChargeRate     float64             `json:"max_charge_rate"`
	MaxDischargeRate  float64             `json:"max_discharge_rate"`
	
	SOC               float64             `json:"soc"`
	MinSOC            float64             `json:"min_soc"`
	MaxSOC            float64             `json:"max_soc"`
	
	CurrentPower      float64             `json:"current_power"`
	HealthState       float64             `json:"health_state"`
	CycleCount        int                 `json:"cycle_count"`
	MaxCycles         int                 `json:"max_cycles"`
	
	Location          Location            `gorm:"type:jsonb" json:"location"`
	ClusterID         *uuid.UUID          `gorm:"type:uuid;index" json:"cluster_id,omitempty"`
	NodeID            *uuid.UUID          `gorm:"type:uuid;index" json:"node_id,omitempty"`
	TenantID          uuid.UUID           `gorm:"type:uuid;index" json:"tenant_id"`
	
	Schedule          []StorageSchedule   `gorm:"type:jsonb" json:"schedule"`
	Strategy          StorageStrategy     `gorm:"type:jsonb" json:"strategy"`
	
	Metadata          EnergyMetadata      `gorm:"type:jsonb" json:"metadata"`
	Labels            EnergyStringMap     `gorm:"type:jsonb" json:"labels"`
}

type StorageSchedule struct {
	StartTime    string  `json:"start_time"`
	EndTime      string  `json:"end_time"`
	Mode         string  `json:"mode"`
	TargetSOC    float64 `json:"target_soc,omitempty"`
	Power        float64 `json:"power,omitempty"`
	Priority     int     `json:"priority"`
	Enabled      bool    `json:"enabled"`
}

type StorageStrategy struct {
	Type              string  `json:"type"`
	PeakPriceThreshold float64 `json:"peak_price_threshold"`
	ValleyPriceThreshold float64 `json:"valley_price_threshold"`
	MinProfitMargin   float64 `json:"min_profit_margin"`
	MaxDailyCycles    int     `json:"max_daily_cycles"`
	SafetyMargin      float64 `json:"safety_margin"`
}

type OrderType string

const (
	OrderTypeBuy  OrderType = "buy"
	OrderTypeSell OrderType = "sell"
)

type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusSubmitted OrderStatus = "submitted"
	OrderStatusFilled    OrderStatus = "filled"
	OrderStatusPartial   OrderStatus = "partial"
	OrderStatusCancelled OrderStatus = "cancelled"
	OrderStatusExpired   OrderStatus = "expired"
)

type EnergyOrderType string

const (
	EnergyOrderSpot       EnergyOrderType = "spot"
	EnergyOrderForward    EnergyOrderType = "forward"
	EnergyOrderGreenCert  EnergyOrderType = "green_cert"
)

type EnergyOrder struct {
	EnergyBaseModel
	OrderNo          string            `gorm:"size:50;uniqueIndex;not null" json:"order_no"`
	Type             OrderType         `gorm:"size:10;not null;index" json:"type"`
	EnergyType       EnergyOrderType   `gorm:"size:20;not null;index" json:"energy_type"`
	Status           OrderStatus       `gorm:"size:20;default:'pending';index" json:"status"`
	
	Quantity         float64           `json:"quantity"`
	Unit             string            `gorm:"size:20;default:'kWh'" json:"unit"`
	Price            float64           `json:"price"`
	TotalAmount      float64           `json:"total_amount"`
	Currency         string            `gorm:"size:10;default:'CNY'" json:"currency"`
	
	BuyerID          uuid.UUID         `gorm:"type:uuid;index" json:"buyer_id"`
	SellerID         uuid.UUID         `gorm:"type:uuid;index" json:"seller_id"`
	TenantID         uuid.UUID         `gorm:"type:uuid;index" json:"tenant_id"`
	
	DeliveryStart    time.Time         `json:"delivery_start"`
	DeliveryEnd      time.Time         `json:"delivery_end"`
	DeliveryRegion   string            `gorm:"size:50;index" json:"delivery_region"`
	
	IsGreen          bool              `gorm:"default:false" json:"is_green"`
	GreenCertID      *uuid.UUID        `gorm:"type:uuid" json:"green_cert_id,omitempty"`
	
	MatchedAt        *time.Time        `json:"matched_at,omitempty"`
	FilledAt         *time.Time        `json:"filled_at,omitempty"`
	CancelledAt      *time.Time        `json:"cancelled_at,omitempty"`
	CancelReason     string            `gorm:"type:text" json:"cancel_reason,omitempty"`
	
	Constraints      OrderConstraints  `gorm:"type:jsonb" json:"constraints"`
	Metadata         EnergyMetadata    `gorm:"type:jsonb" json:"metadata"`
}

type OrderConstraints struct {
	MinQuantity     float64 `json:"min_quantity,omitempty"`
	MaxPrice        float64 `json:"max_price,omitempty"`
	MinGreenRatio   float64 `json:"min_green_ratio,omitempty"`
	RequiredSource  string  `json:"required_source,omitempty"`
	FlexibleDelivery bool    `json:"flexible_delivery,omitempty"`
}

type PriceQuote struct {
	EnergyBaseModel
	Region           string          `gorm:"size:50;index;not null" json:"region"`
	EnergyType       EnergyOrderType `gorm:"size:20;index;not null" json:"energy_type"`
	
	SpotPrice        float64         `json:"spot_price"`
	ForwardPrice1M   float64         `json:"forward_price_1m"`
	ForwardPrice3M   float64         `json:"forward_price_3m"`
	ForwardPrice6M   float64         `json:"forward_price_6m"`
	ForwardPrice12M  float64         `json:"forward_price_12m"`
	
	GreenPremium     float64         `json:"green_premium"`
	GreenCertPrice   float64         `json:"green_cert_price"`
	
	PeakPrice        float64         `json:"peak_price"`
	ValleyPrice      float64         `json:"valley_price"`
	FlatPrice        float64         `json:"flat_price"`
	
	ValidFrom        time.Time       `json:"valid_from"`
	ValidUntil       time.Time       `json:"valid_until"`
	Source           string          `gorm:"size:50" json:"source"`
	
	Confidence       float64         `json:"confidence"`
	Trend            string          `gorm:"size:20" json:"trend"`
}

type VPPTypes string

const (
	VPPTypesDistributed VPPTypes = "distributed"
	VPPTypesCentralized VPPTypes = "centralized"
	VPPTypesHybrid      VPPTypes = "hybrid"
)

type VPPStatus string

const (
	VPPStatusActive      VPPStatus = "active"
	VPPStatusInactive    VPPStatus = "inactive"
	VPPStatusDispatching VPPStatus = "dispatching"
)

type VirtualPowerPlant struct {
	EnergyBaseModel
	Name              string            `gorm:"size:100;not null" json:"name"`
	Type              VPPTypes          `gorm:"size:20;not null" json:"type"`
	Status            VPPStatus         `gorm:"size:20;default:'active'" json:"status"`
	
	TotalCapacity     float64           `json:"total_capacity"`
	AvailableCapacity float64           `json:"available_capacity"`
	DispatchablePower float64           `json:"dispatchable_power"`
	
	PowerSourceIDs    []uuid.UUID       `gorm:"type:jsonb" json:"power_source_ids"`
	StorageIDs        []uuid.UUID       `gorm:"type:jsonb" json:"storage_ids"`
	LoadIDs           []uuid.UUID       `gorm:"type:jsonb" json:"load_ids"`
	
	TenantID          uuid.UUID         `gorm:"type:uuid;index" json:"tenant_id"`
	Region            string            `gorm:"size:50;index" json:"region"`
	
	AggregationLevel  int               `json:"aggregation_level"`
	ResponseTime      int               `json:"response_time"`
	MinDispatchUnit   float64           `json:"min_dispatch_unit"`
	
	ControlStrategy   VPPControlStrategy `gorm:"type:jsonb" json:"control_strategy"`
	PerformanceMetrics VPPMetrics       `gorm:"type:jsonb" json:"performance_metrics"`
	
	Labels            EnergyStringMap   `gorm:"type:jsonb" json:"labels"`
}

type VPPControlStrategy struct {
	Mode              string  `json:"mode"`
	PriorityOrder     []string `json:"priority_order"`
	PriceThreshold    float64 `json:"price_threshold"`
	LoadBalanceFactor float64 `json:"load_balance_factor"`
	ReserveRatio      float64 `json:"reserve_ratio"`
}

type VPPMetrics struct {
	TotalDispatches   int     `json:"total_dispatches"`
	SuccessRate       float64 `json:"success_rate"`
	AvgResponseTime   float64 `json:"avg_response_time"`
	TotalEnergyManaged float64 `json:"total_energy_managed"`
	Revenue           float64 `json:"revenue"`
	CarbonSaved       float64 `json:"carbon_saved"`
}

type LoadType string

const (
	LoadTypeComputing LoadType = "computing"
	LoadTypeCooling   LoadType = "cooling"
	LoadTypeLighting  LoadType = "lighting"
	LoadTypeOther     LoadType = "other"
)

type LoadProfile struct {
	EnergyBaseModel
	Name              string            `gorm:"size:100;not null" json:"name"`
	Type              LoadType          `gorm:"size:20;index" json:"type"`
	ClusterID         *uuid.UUID        `gorm:"type:uuid;index" json:"cluster_id,omitempty"`
	NodeID            *uuid.UUID        `gorm:"type:uuid;index" json:"node_id,omitempty"`
	TenantID          uuid.UUID         `gorm:"type:uuid;index" json:"tenant_id"`
	
	CurrentLoad       float64           `json:"current_load"`
	PeakLoad          float64           `json:"peak_load"`
	BaseLoad          float64           `json:"base_load"`
	
	Forecast          []LoadForecastPoint `gorm:"type:jsonb" json:"forecast"`
	AdjustableRange   AdjustableRange   `gorm:"type:jsonb" json:"adjustable_range"`
	
	Priority          int               `json:"priority"`
	IsInterruptible   bool              `json:"is_interruptible"`
	MaxInterruptTime  int               `json:"max_interrupt_time"`
	
	Labels            EnergyStringMap   `gorm:"type:jsonb" json:"labels"`
}

type LoadForecastPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	Confidence float64  `json:"confidence"`
}

type AdjustableRange struct {
	MinLoad     float64 `json:"min_load"`
	MaxLoad     float64 `json:"max_load"`
	RampUpRate  float64 `json:"ramp_up_rate"`
	RampDownRate float64 `json:"ramp_down_rate"`
}

type EnergyTransaction struct {
	EnergyBaseModel
	OrderID           uuid.UUID       `gorm:"type:uuid;index;not null" json:"order_id"`
	BuyerID           uuid.UUID       `gorm:"type:uuid;index" json:"buyer_id"`
	SellerID          uuid.UUID       `gorm:"type:uuid;index" json:"seller_id"`
	
	Quantity          float64         `json:"quantity"`
	Unit              string          `gorm:"size:20" json:"unit"`
	Price             float64         `json:"price"`
	TotalAmount       float64         `json:"total_amount"`
	Currency          string          `gorm:"size:10" json:"currency"`
	
	TransactionTime   time.Time       `json:"transaction_time"`
	SettlementTime    *time.Time      `json:"settlement_time,omitempty"`
	SettlementStatus  string          `gorm:"size:20" json:"settlement_status"`
	
	GreenCertID       *uuid.UUID      `gorm:"type:uuid" json:"green_cert_id,omitempty"`
	CarbonSaved       float64         `json:"carbon_saved"`
	
	TenantID          uuid.UUID       `gorm:"type:uuid;index" json:"tenant_id"`
}

type GreenCertificate struct {
	EnergyBaseModel
	CertNo            string        `gorm:"size:50;uniqueIndex;not null" json:"cert_no"`
	SourceType        PowerSourceType `gorm:"size:20;not null" json:"source_type"`
	
	EnergyAmount      float64       `json:"energy_amount"`
	Unit              string        `gorm:"size:20;default:'MWh'" json:"unit"`
	
	GenerationStart   time.Time     `json:"generation_start"`
	GenerationEnd     time.Time     `json:"generation_end"`
	GenerationRegion  string        `gorm:"size:50" json:"generation_region"`
	
	PowerSourceID     uuid.UUID     `gorm:"type:uuid;index" json:"power_source_id"`
	OwnerID           uuid.UUID     `gorm:"type:uuid;index" json:"owner_id"`
	TenantID          uuid.UUID     `gorm:"type:uuid;index" json:"tenant_id"`
	
	Status            string        `gorm:"size:20;default:'available'" json:"status"`
	Price             float64       `json:"price"`
	
	IssuedAt          time.Time     `json:"issued_at"`
	ExpiresAt         time.Time     `json:"expires_at"`
	TransferredAt     *time.Time    `json:"transferred_at,omitempty"`
}

type EnergyMarketConfig struct {
	Region              string  `json:"region"`
	SpotMarketEnabled   bool    `json:"spot_market_enabled"`
	ForwardMarketEnabled bool   `json:"forward_market_enabled"`
	GreenCertEnabled    bool    `json:"green_cert_enabled"`
	
	TradingHours        string  `json:"trading_hours"`
	SettlementCycle     int     `json:"settlement_cycle"`
	MinOrderQuantity    float64 `json:"min_order_quantity"`
	MaxOrderQuantity    float64 `json:"max_order_quantity"`
	
	PeakHours           []TimeRange `json:"peak_hours"`
	ValleyHours         []TimeRange `json:"valley_hours"`
	
	TransactionFee      float64 `json:"transaction_fee"`
	PlatformFee         float64 `json:"platform_fee"`
}

type TimeRange struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

type EnergyMetrics struct {
	Timestamp         time.Time `json:"timestamp"`
	
	TotalGeneration   float64   `json:"total_generation"`
	TotalConsumption  float64   `json:"total_consumption"`
	TotalStorage      float64   `json:"total_storage"`
	
	GreenRatio       float64    `json:"green_ratio"`
	CarbonIntensity  float64    `json:"carbon_intensity"`
	
	AvgPrice         float64    `json:"avg_price"`
	PeakPrice        float64    `json:"peak_price"`
	ValleyPrice      float64    `json:"valley_price"`
	
	TradingVolume    float64    `json:"trading_volume"`
	TransactionCount int        `json:"transaction_count"`
}

type StorageOptimizationResult struct {
	DeviceID          uuid.UUID          `json:"device_id"`
	RecommendedAction StorageAction      `json:"recommended_action"`
	ExpectedProfit    float64            `json:"expected_profit"`
	Confidence        float64            `json:"confidence"`
	Reason            string             `json:"reason"`
	Timestamp         time.Time          `json:"timestamp"`
}

type StorageAction struct {
	Mode      string  `json:"mode"`
	Power     float64 `json:"power"`
	Duration  int     `json:"duration"`
	StartTime string  `json:"start_time"`
	EndTime   string  `json:"end_time"`
}

type ComputeEnergyCoordination struct {
	ComputeJobID      uuid.UUID       `json:"compute_job_id"`
	EnergySourceID    uuid.UUID       `json:"energy_source_id"`
	StorageDeviceID   *uuid.UUID      `json:"storage_device_id,omitempty"`
	
	ScheduledStart    time.Time       `json:"scheduled_start"`
	ScheduledEnd      time.Time       `json:"scheduled_end"`
	
	EnergyCost        float64         `json:"energy_cost"`
	CarbonFootprint   float64         `json:"carbon_footprint"`
	GreenRatio        float64         `json:"green_ratio"`
	
	Status            string          `json:"status"`
	OptimizationScore float64         `json:"optimization_score"`
}
