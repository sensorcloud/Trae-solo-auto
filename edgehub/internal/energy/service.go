package energy

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type PowerSourceService interface {
	CreatePowerSource(ctx context.Context, source *PowerSource) error
	GetPowerSource(ctx context.Context, id uuid.UUID) (*PowerSource, error)
	ListPowerSources(ctx context.Context, filter *PowerSourceFilter) ([]*PowerSource, error)
	UpdatePowerSource(ctx context.Context, id uuid.UUID, updates *PowerSourceUpdate) error
	DeletePowerSource(ctx context.Context, id uuid.UUID) error
	
	UpdateRealtimeOutput(ctx context.Context, id uuid.UUID, output float64) error
	GetPowerGenerationStats(ctx context.Context, id uuid.UUID, period string) (*PowerGenerationStats, error)
}

type PowerSourceFilter struct {
	Type     PowerSourceType
	Status   PowerSourceStatus
	Region   string
	TenantID uuid.UUID
	MinCapacity float64
}

type PowerSourceUpdate struct {
	Name     *string
	Status   *PowerSourceStatus
	Capacity *float64
}

type PowerGenerationStats struct {
	TotalGenerated float64
	AvgOutput      float64
	PeakOutput     float64
	MinOutput      float64
	CarbonSaved    float64
	GreenRatio     float64
}

type StorageService interface {
	CreateStorageDevice(ctx context.Context, device *StorageDevice) error
	GetStorageDevice(ctx context.Context, id uuid.UUID) (*StorageDevice, error)
	ListStorageDevices(ctx context.Context, filter *StorageFilter) ([]*StorageDevice, error)
	UpdateStorageDevice(ctx context.Context, id uuid.UUID, updates *StorageUpdate) error
	DeleteStorageDevice(ctx context.Context, id uuid.UUID) error
	
	Charge(ctx context.Context, id uuid.UUID, power float64) error
	Discharge(ctx context.Context, id uuid.UUID, power float64) error
	StopChargeDischarge(ctx context.Context, id uuid.UUID) error
	
	UpdateSOC(ctx context.Context, id uuid.UUID, soc float64) error
	GetStorageStatus(ctx context.Context, id uuid.UUID) (*StorageStatus, error)
	
	OptimizeSchedule(ctx context.Context, id uuid.UUID, prices []*PriceQuote) (*StorageOptimizationResult, error)
	ExecuteArbitrage(ctx context.Context, id uuid.UUID) error
}

type StorageFilter struct {
	Status   StorageDeviceStatus
	Region   string
	TenantID uuid.UUID
	MinCapacity float64
}

type StorageUpdate struct {
	Name            *string
	Status          *StorageDeviceStatus
	MaxChargeRate   *float64
	MaxDischargeRate *float64
	Strategy        *StorageStrategy
}

type StorageStatus struct {
	DeviceID      uuid.UUID
	SOC           float64
	CurrentPower  float64
	Status        StorageDeviceStatus
	AvailableCapacity float64
	HealthState   float64
}

type TradingService interface {
	CreateOrder(ctx context.Context, order *EnergyOrder) error
	GetOrder(ctx context.Context, id uuid.UUID) (*EnergyOrder, error)
	GetOrderByNo(ctx context.Context, orderNo string) (*EnergyOrder, error)
	ListOrders(ctx context.Context, filter *OrderFilter) ([]*EnergyOrder, error)
	CancelOrder(ctx context.Context, id uuid.UUID, reason string) error
	
	SubmitOrder(ctx context.Context, id uuid.UUID) error
	MatchOrder(ctx context.Context, order *EnergyOrder) ([]*EnergyOrder, error)
	
	GetPriceQuote(ctx context.Context, region string, energyType EnergyOrderType) (*PriceQuote, error)
	GetPriceHistory(ctx context.Context, region string, energyType EnergyOrderType, period string) ([]*PriceQuote, error)
	
	CreateGreenCertificate(ctx context.Context, cert *GreenCertificate) error
	TransferGreenCertificate(ctx context.Context, certID, toOwnerID uuid.UUID) error
	ListGreenCertificates(ctx context.Context, filter *GreenCertFilter) ([]*GreenCertificate, error)
}

type OrderFilter struct {
	Type       OrderType
	EnergyType EnergyOrderType
	Status     OrderStatus
	BuyerID    uuid.UUID
	SellerID   uuid.UUID
	TenantID   uuid.UUID
	Region     string
	IsGreen    *bool
}

type GreenCertFilter struct {
	SourceType PowerSourceType
	OwnerID    uuid.UUID
	Status     string
	TenantID   uuid.UUID
}

type VPPService interface {
	CreateVPP(ctx context.Context, vpp *VirtualPowerPlant) error
	GetVPP(ctx context.Context, id uuid.UUID) (*VirtualPowerPlant, error)
	ListVPPs(ctx context.Context, filter *VPPFilter) ([]*VirtualPowerPlant, error)
	UpdateVPP(ctx context.Context, id uuid.UUID, updates *VPPUpdate) error
	DeleteVPP(ctx context.Context, id uuid.UUID) error
	
	AddPowerSource(ctx context.Context, vppID, sourceID uuid.UUID) error
	AddStorageDevice(ctx context.Context, vppID, storageID uuid.UUID) error
	AddLoad(ctx context.Context, vppID, loadID uuid.UUID) error
	
	Dispatch(ctx context.Context, vppID uuid.UUID, request *DispatchRequest) (*DispatchResult, error)
	GetDispatchStatus(ctx context.Context, vppID uuid.UUID) (*DispatchStatus, error)
	
	AggregateCapacity(ctx context.Context, vppID uuid.UUID) (*AggregatedCapacity, error)
}

type VPPFilter struct {
	Type     VPPTypes
	Status   VPPStatus
	Region   string
	TenantID uuid.UUID
}

type VPPUpdate struct {
	Name            *string
	Status          *VPPStatus
	ControlStrategy *VPPControlStrategy
}

type DispatchRequest struct {
	Power        float64
	Duration     int
	Priority     int
	ResponseType string
	Reason       string
}

type DispatchResult struct {
	RequestID    uuid.UUID
	VPPID        uuid.UUID
	DispatchedPower float64
	ActualPower  float64
	StartTime    string
	EndTime      string
	Status       string
}

type DispatchStatus struct {
	VPPID           uuid.UUID
	IsDispatching   bool
	CurrentPower    float64
	AvailablePower  float64
	LastDispatchAt  string
}

type AggregatedCapacity struct {
	VPPID             uuid.UUID
	TotalCapacity     float64
	AvailableCapacity float64
	DispatchablePower float64
	StorageCapacity   float64
	LoadCapacity      float64
}

type LoadService interface {
	CreateLoadProfile(ctx context.Context, profile *LoadProfile) error
	GetLoadProfile(ctx context.Context, id uuid.UUID) (*LoadProfile, error)
	ListLoadProfiles(ctx context.Context, filter *LoadFilter) ([]*LoadProfile, error)
	UpdateLoadProfile(ctx context.Context, id uuid.UUID, updates *LoadUpdate) error
	DeleteLoadProfile(ctx context.Context, id uuid.UUID) error
	
	UpdateCurrentLoad(ctx context.Context, id uuid.UUID, load float64) error
	ForecastLoad(ctx context.Context, id uuid.UUID, horizon int) ([]*LoadForecastPoint, error)
	
	AdjustLoad(ctx context.Context, id uuid.UUID, targetLoad float64) error
}

type LoadFilter struct {
	Type     LoadType
	TenantID uuid.UUID
	ClusterID uuid.UUID
}

type LoadUpdate struct {
	Name            *string
	Priority        *int
	IsInterruptible *bool
	AdjustableRange *AdjustableRange
}

type EnergyMarketService interface {
	GetMarketOverview(ctx context.Context, region string) (*MarketOverview, error)
	GetTradingVolume(ctx context.Context, region string, period string) (*TradingVolume, error)
	GetMarketDepth(ctx context.Context, region string, energyType EnergyOrderType) (*MarketDepth, error)
	
	SubscribePriceUpdates(ctx context.Context, region string, energyType EnergyOrderType) (<-chan *PriceQuote, error)
	SubscribeOrderUpdates(ctx context.Context, orderID uuid.UUID) (<-chan *EnergyOrder, error)
}

type MarketOverview struct {
	Region           string
	CurrentPrice     float64
	PriceChange      float64
	TradingVolume    float64
	GreenRatio       float64
	PeakPrice        float64
	ValleyPrice      float64
	ActiveOrders     int
	AvailablePower   float64
}

type TradingVolume struct {
	Region        string
	TotalVolume   float64
	TotalAmount   float64
	TransactionCount int
	GreenVolume   float64
	Period        string
}

type MarketDepth struct {
	Region      string
	EnergyType  EnergyOrderType
	BuyOrders   []*OrderLevel
	SellOrders  []*OrderLevel
	BestBid     float64
	BestAsk     float64
	Spread      float64
}

type OrderLevel struct {
	Price    float64
	Quantity float64
	Count    int
}

type ComputeEnergyCoordinationService interface {
	ScheduleComputeWithEnergy(ctx context.Context, request *ComputeEnergyRequest) (*ComputeEnergyCoordination, error)
	GetOptimalTimeSlot(ctx context.Context, request *OptimalTimeRequest) (*OptimalTimeSlot, error)
	GetEnergyForecast(ctx context.Context, region string, horizon int) ([]*EnergyForecastPoint, error)
}

type ComputeEnergyRequest struct {
	ComputeJobID    uuid.UUID
	EstimatedPower  float64
	Duration        int
	PreferredStart  string
	PreferredEnd    string
	MaxEnergyCost   float64
	MinGreenRatio   float64
	Region          string
	TenantID        uuid.UUID
}

type OptimalTimeRequest struct {
	EstimatedPower float64
	Duration       int
	Region         string
	OptimizationGoal string
}

type OptimalTimeSlot struct {
	StartTime       string
	EndTime         string
	ExpectedCost    float64
	ExpectedGreenRatio float64
	Confidence      float64
	Reason          string
}

type EnergyForecastPoint struct {
	Timestamp       time.Time
	ExpectedPower   float64
	ExpectedPrice   float64
	GreenRatio      float64
	Confidence      float64
}

type PriceDataProvider interface {
	GetRealtimePrice(ctx context.Context, region string) (*PriceQuote, error)
	GetHistoricalPrices(ctx context.Context, region string, start, end string) ([]*PriceQuote, error)
	SubscribePrices(ctx context.Context, region string) (<-chan *PriceQuote, error)
}

type EnergyMonitor interface {
	CollectMetrics(ctx context.Context, sourceID uuid.UUID) (*EnergyMetrics, error)
	GetRealtimeData(ctx context.Context, sourceID uuid.UUID) (map[string]float64, error)
	StartMonitoring(ctx context.Context, sourceID uuid.UUID, interval int) error
	StopMonitoring(ctx context.Context, sourceID uuid.UUID) error
}

type EnergyRepository interface {
	CreatePowerSource(ctx context.Context, source *PowerSource) error
	GetPowerSource(ctx context.Context, id uuid.UUID) (*PowerSource, error)
	ListPowerSources(ctx context.Context, filter *PowerSourceFilter) ([]*PowerSource, error)
	UpdatePowerSource(ctx context.Context, source *PowerSource) error
	DeletePowerSource(ctx context.Context, id uuid.UUID) error
	
	CreateStorageDevice(ctx context.Context, device *StorageDevice) error
	GetStorageDevice(ctx context.Context, id uuid.UUID) (*StorageDevice, error)
	ListStorageDevices(ctx context.Context, filter *StorageFilter) ([]*StorageDevice, error)
	UpdateStorageDevice(ctx context.Context, device *StorageDevice) error
	DeleteStorageDevice(ctx context.Context, id uuid.UUID) error
	
	CreateOrder(ctx context.Context, order *EnergyOrder) error
	GetOrder(ctx context.Context, id uuid.UUID) (*EnergyOrder, error)
	GetOrderByNo(ctx context.Context, orderNo string) (*EnergyOrder, error)
	ListOrders(ctx context.Context, filter *OrderFilter) ([]*EnergyOrder, error)
	UpdateOrder(ctx context.Context, order *EnergyOrder) error
	
	CreatePriceQuote(ctx context.Context, quote *PriceQuote) error
	GetLatestPriceQuote(ctx context.Context, region string, energyType EnergyOrderType) (*PriceQuote, error)
	ListPriceQuotes(ctx context.Context, region string, energyType EnergyOrderType, limit int) ([]*PriceQuote, error)
	
	CreateVPP(ctx context.Context, vpp *VirtualPowerPlant) error
	GetVPP(ctx context.Context, id uuid.UUID) (*VirtualPowerPlant, error)
	ListVPPs(ctx context.Context, filter *VPPFilter) ([]*VirtualPowerPlant, error)
	UpdateVPP(ctx context.Context, vpp *VirtualPowerPlant) error
	DeleteVPP(ctx context.Context, id uuid.UUID) error
	
	CreateTransaction(ctx context.Context, tx *EnergyTransaction) error
	GetTransaction(ctx context.Context, id uuid.UUID) (*EnergyTransaction, error)
	ListTransactions(ctx context.Context, filter *TransactionFilter) ([]*EnergyTransaction, error)
	UpdateTransaction(ctx context.Context, tx *EnergyTransaction) error
	
	CreateGreenCertificate(ctx context.Context, cert *GreenCertificate) error
	GetGreenCertificate(ctx context.Context, id uuid.UUID) (*GreenCertificate, error)
	ListGreenCertificates(ctx context.Context, filter *GreenCertFilter) ([]*GreenCertificate, error)
	UpdateGreenCertificate(ctx context.Context, cert *GreenCertificate) error
	
	CreateLoadProfile(ctx context.Context, profile *LoadProfile) error
	GetLoadProfile(ctx context.Context, id uuid.UUID) (*LoadProfile, error)
	ListLoadProfiles(ctx context.Context, filter *LoadFilter) ([]*LoadProfile, error)
	UpdateLoadProfile(ctx context.Context, profile *LoadProfile) error
	DeleteLoadProfile(ctx context.Context, id uuid.UUID) error
}

type TransactionFilter struct {
	BuyerID  uuid.UUID
	SellerID uuid.UUID
	TenantID uuid.UUID
	OrderID  uuid.UUID
}
