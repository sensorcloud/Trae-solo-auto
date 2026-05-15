package energy

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
	"k8s.io/klog/v2"
)

type EnergyMarketCore struct {
	repo          EnergyRepository
	priceProvider PriceDataProvider
	storageSvc    StorageService
	tradingSvc    TradingService
	vppSvc        VPPService
	loadSvc       LoadService
	
	powerSources  map[uuid.UUID]*PowerSource
	vpps          map[uuid.UUID]*VirtualPowerPlant
	priceCache    map[string]*PriceQuote
	
	mu            sync.RWMutex
	config        *EnergyMarketConfig
	
	priceSubscribers   map[string][]chan *PriceQuote
	orderSubscribers   map[uuid.UUID][]chan *EnergyOrder
	subscriberMu       sync.RWMutex
}

func NewEnergyMarketCore(repo EnergyRepository, config *EnergyMarketConfig) *EnergyMarketCore {
	if config == nil {
		config = &EnergyMarketConfig{
			Region:              "default",
			SpotMarketEnabled:   true,
			ForwardMarketEnabled: true,
			GreenCertEnabled:    true,
			TradingHours:        "00:00-23:59",
			SettlementCycle:     24,
			MinOrderQuantity:    1,
			MaxOrderQuantity:    1000000,
			TransactionFee:      0.001,
			PlatformFee:         0.0005,
		}
	}
	
	return &EnergyMarketCore{
		repo:             repo,
		config:           config,
		powerSources:     make(map[uuid.UUID]*PowerSource),
		vpps:             make(map[uuid.UUID]*VirtualPowerPlant),
		priceCache:       make(map[string]*PriceQuote),
		priceSubscribers: make(map[string][]chan *PriceQuote),
		orderSubscribers: make(map[uuid.UUID][]chan *EnergyOrder),
	}
}

func (e *EnergyMarketCore) SetStorageService(svc StorageService) {
	e.storageSvc = svc
}

func (e *EnergyMarketCore) SetTradingService(svc TradingService) {
	e.tradingSvc = svc
}

func (e *EnergyMarketCore) SetVPPService(svc VPPService) {
	e.vppSvc = svc
}

func (e *EnergyMarketCore) SetLoadService(svc LoadService) {
	e.loadSvc = svc
}

func (e *EnergyMarketCore) SetPriceProvider(provider PriceDataProvider) {
	e.priceProvider = provider
}

func (e *EnergyMarketCore) CreatePowerSource(ctx context.Context, source *PowerSource) error {
	if source.ID == uuid.Nil {
		source.ID = uuid.New()
	}
	source.CreatedAt = time.Now()
	source.UpdatedAt = time.Now()
	
	if source.Status == "" {
		source.Status = PowerSourceStatusOnline
	}
	
	if err := e.repo.CreatePowerSource(ctx, source); err != nil {
		return fmt.Errorf("创建电源失败: %w", err)
	}
	
	e.mu.Lock()
	e.powerSources[source.ID] = source
	e.mu.Unlock()
	
	klog.Infof("创建电源成功: %s, 类型: %s, 容量: %.2f %s", source.Name, source.Type, source.Capacity, source.Unit)
	return nil
}

func (e *EnergyMarketCore) GetPowerSource(ctx context.Context, id uuid.UUID) (*PowerSource, error) {
	e.mu.RLock()
	if source, ok := e.powerSources[id]; ok {
		e.mu.RUnlock()
		return source, nil
	}
	e.mu.RUnlock()
	
	source, err := e.repo.GetPowerSource(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("获取电源失败: %w", err)
	}
	
	e.mu.Lock()
	e.powerSources[id] = source
	e.mu.Unlock()
	
	return source, nil
}

func (e *EnergyMarketCore) ListPowerSources(ctx context.Context, filter *PowerSourceFilter) ([]*PowerSource, error) {
	sources, err := e.repo.ListPowerSources(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("列出电源失败: %w", err)
	}
	return sources, nil
}

func (e *EnergyMarketCore) UpdatePowerSource(ctx context.Context, id uuid.UUID, updates *PowerSourceUpdate) error {
	source, err := e.GetPowerSource(ctx, id)
	if err != nil {
		return err
	}
	
	if updates.Name != nil {
		source.Name = *updates.Name
	}
	if updates.Status != nil {
		source.Status = *updates.Status
	}
	if updates.Capacity != nil {
		source.Capacity = *updates.Capacity
	}
	source.UpdatedAt = time.Now()
	
	if err := e.repo.UpdatePowerSource(ctx, source); err != nil {
		return fmt.Errorf("更新电源失败: %w", err)
	}
	
	e.mu.Lock()
	e.powerSources[id] = source
	e.mu.Unlock()
	
	klog.Infof("更新电源成功: %s", id)
	return nil
}

func (e *EnergyMarketCore) DeletePowerSource(ctx context.Context, id uuid.UUID) error {
	if err := e.repo.DeletePowerSource(ctx, id); err != nil {
		return fmt.Errorf("删除电源失败: %w", err)
	}
	
	e.mu.Lock()
	delete(e.powerSources, id)
	e.mu.Unlock()
	
	klog.Infof("删除电源成功: %s", id)
	return nil
}

func (e *EnergyMarketCore) UpdateRealtimeOutput(ctx context.Context, id uuid.UUID, output float64) error {
	source, err := e.GetPowerSource(ctx, id)
	if err != nil {
		return err
	}
	
	source.RealtimeOutput = output
	source.TotalGenerated += output / 60
	
	e.mu.Lock()
	e.powerSources[id] = source
	e.mu.Unlock()
	
	return nil
}

func (e *EnergyMarketCore) GetPowerGenerationStats(ctx context.Context, id uuid.UUID, period string) (*PowerGenerationStats, error) {
	source, err := e.GetPowerSource(ctx, id)
	if err != nil {
		return nil, err
	}
	
	stats := &PowerGenerationStats{
		TotalGenerated: source.TotalGenerated,
		AvgOutput:      source.RealtimeOutput,
		PeakOutput:     source.Capacity,
		MinOutput:      0,
	}
	
	if source.Type == PowerSourceSolar || source.Type == PowerSourceWind || source.Type == PowerSourceHydro {
		stats.CarbonSaved = source.TotalGenerated * getCarbonFactor(source.Type)
		stats.GreenRatio = 1.0
	}
	
	return stats, nil
}

func getCarbonFactor(sourceType PowerSourceType) float64 {
	switch sourceType {
	case PowerSourceSolar:
		return 0.85
	case PowerSourceWind:
		return 0.82
	case PowerSourceHydro:
		return 0.90
	case PowerSourceBiomass:
		return 0.45
	default:
		return 0.0
	}
}

func (e *EnergyMarketCore) CreateVPP(ctx context.Context, vpp *VirtualPowerPlant) error {
	if vpp.ID == uuid.Nil {
		vpp.ID = uuid.New()
	}
	vpp.CreatedAt = time.Now()
	vpp.UpdatedAt = time.Now()
	
	if vpp.Status == "" {
		vpp.Status = VPPStatusActive
	}
	
	if err := e.repo.CreateVPP(ctx, vpp); err != nil {
		return fmt.Errorf("创建虚拟电厂失败: %w", err)
	}
	
	e.mu.Lock()
	e.vpps[vpp.ID] = vpp
	e.mu.Unlock()
	
	klog.Infof("创建虚拟电厂成功: %s, 类型: %s, 总容量: %.2f", vpp.Name, vpp.Type, vpp.TotalCapacity)
	return nil
}

func (e *EnergyMarketCore) GetVPP(ctx context.Context, id uuid.UUID) (*VirtualPowerPlant, error) {
	e.mu.RLock()
	if vpp, ok := e.vpps[id]; ok {
		e.mu.RUnlock()
		return vpp, nil
	}
	e.mu.RUnlock()
	
	vpp, err := e.repo.GetVPP(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("获取虚拟电厂失败: %w", err)
	}
	
	e.mu.Lock()
	e.vpps[id] = vpp
	e.mu.Unlock()
	
	return vpp, nil
}

func (e *EnergyMarketCore) ListVPPs(ctx context.Context, filter *VPPFilter) ([]*VirtualPowerPlant, error) {
	return e.repo.ListVPPs(ctx, filter)
}

func (e *EnergyMarketCore) UpdateVPP(ctx context.Context, id uuid.UUID, updates *VPPUpdate) error {
	vpp, err := e.GetVPP(ctx, id)
	if err != nil {
		return err
	}
	
	if updates.Name != nil {
		vpp.Name = *updates.Name
	}
	if updates.Status != nil {
		vpp.Status = *updates.Status
	}
	if updates.ControlStrategy != nil {
		vpp.ControlStrategy = *updates.ControlStrategy
	}
	vpp.UpdatedAt = time.Now()
	
	if err := e.repo.UpdateVPP(ctx, vpp); err != nil {
		return fmt.Errorf("更新虚拟电厂失败: %w", err)
	}
	
	e.mu.Lock()
	e.vpps[id] = vpp
	e.mu.Unlock()
	
	return nil
}

func (e *EnergyMarketCore) DeleteVPP(ctx context.Context, id uuid.UUID) error {
	if err := e.repo.DeleteVPP(ctx, id); err != nil {
		return fmt.Errorf("删除虚拟电厂失败: %w", err)
	}
	
	e.mu.Lock()
	delete(e.vpps, id)
	e.mu.Unlock()
	
	klog.Infof("删除虚拟电厂成功: %s", id)
	return nil
}

func (e *EnergyMarketCore) AddPowerSourceToVPP(ctx context.Context, vppID, sourceID uuid.UUID) error {
	vpp, err := e.GetVPP(ctx, vppID)
	if err != nil {
		return err
	}
	
	source, err := e.GetPowerSource(ctx, sourceID)
	if err != nil {
		return fmt.Errorf("获取电源失败: %w", err)
	}
	
	vpp.PowerSourceIDs = append(vpp.PowerSourceIDs, sourceID)
	vpp.TotalCapacity += source.Capacity
	vpp.AvailableCapacity += source.Capacity
	
	if err := e.repo.UpdateVPP(ctx, vpp); err != nil {
		return fmt.Errorf("更新虚拟电厂失败: %w", err)
	}
	
	e.mu.Lock()
	e.vpps[vppID] = vpp
	e.mu.Unlock()
	
	klog.Infof("将电源 %s 添加到虚拟电厂 %s", sourceID, vppID)
	return nil
}

func (e *EnergyMarketCore) AddStorageToVPP(ctx context.Context, vppID, storageID uuid.UUID) error {
	vpp, err := e.GetVPP(ctx, vppID)
	if err != nil {
		return err
	}
	
	vpp.StorageIDs = append(vpp.StorageIDs, storageID)
	
	if err := e.repo.UpdateVPP(ctx, vpp); err != nil {
		return fmt.Errorf("更新虚拟电厂失败: %w", err)
	}
	
	e.mu.Lock()
	e.vpps[vppID] = vpp
	e.mu.Unlock()
	
	klog.Infof("将储能设备 %s 添加到虚拟电厂 %s", storageID, vppID)
	return nil
}

func (e *EnergyMarketCore) DispatchVPP(ctx context.Context, vppID uuid.UUID, request *DispatchRequest) (*DispatchResult, error) {
	vpp, err := e.GetVPP(ctx, vppID)
	if err != nil {
		return nil, err
	}
	
	if vpp.Status != VPPStatusActive {
		return nil, fmt.Errorf("虚拟电厂状态不允许调度: %s", vpp.Status)
	}
	
	if request.Power > vpp.AvailableCapacity {
		return nil, fmt.Errorf("请求功率 %.2f 超过可用容量 %.2f", request.Power, vpp.AvailableCapacity)
	}
	
	dispatchedPower := 0.0
	var dispatchDetails []string
	
	for _, sourceID := range vpp.PowerSourceIDs {
		if dispatchedPower >= request.Power {
			break
		}
		
		source, err := e.GetPowerSource(ctx, sourceID)
		if err != nil {
			continue
		}
		
		if source.Status != PowerSourceStatusOnline {
			continue
		}
		
		availableFromSource := math.Min(source.RealtimeOutput, request.Power-dispatchedPower)
		dispatchedPower += availableFromSource
		dispatchDetails = append(dispatchDetails, fmt.Sprintf("电源 %s: %.2f", source.Name, availableFromSource))
	}
	
	if dispatchedPower < request.Power && len(vpp.StorageIDs) > 0 {
		neededFromStorage := request.Power - dispatchedPower
		for _, storageID := range vpp.StorageIDs {
			if dispatchedPower >= request.Power {
				break
			}
			
			storage, err := e.storageSvc.GetStorageDevice(ctx, storageID)
			if err != nil {
				continue
			}
			
			if storage.Status != StorageStatusIdle && storage.Status != StorageStatusDischarging {
				continue
			}
			
			availableFromStorage := math.Min(storage.MaxDischargeRate, neededFromStorage)
			if err := e.storageSvc.Discharge(ctx, storageID, availableFromStorage); err != nil {
				klog.Warningf("储能设备 %s 放电失败: %v", storageID, err)
				continue
			}
			
			dispatchedPower += availableFromStorage
			dispatchDetails = append(dispatchDetails, fmt.Sprintf("储能 %s: %.2f", storage.Name, availableFromStorage))
		}
	}
	
	vpp.Status = VPPStatusDispatching
	e.mu.Lock()
	e.vpps[vppID] = vpp
	e.mu.Unlock()
	
	result := &DispatchResult{
		RequestID:       uuid.New(),
		VPPID:           vppID,
		DispatchedPower: dispatchedPower,
		ActualPower:     dispatchedPower,
		StartTime:       time.Now().Format(time.RFC3339),
		EndTime:         time.Now().Add(time.Duration(request.Duration) * time.Minute).Format(time.RFC3339),
		Status:          "dispatched",
	}
	
	klog.Infof("虚拟电厂 %s 调度成功, 总功率: %.2f, 详情: %v", vppID, dispatchedPower, dispatchDetails)
	return result, nil
}

func (e *EnergyMarketCore) AggregateVPPCapacity(ctx context.Context, vppID uuid.UUID) (*AggregatedCapacity, error) {
	vpp, err := e.GetVPP(ctx, vppID)
	if err != nil {
		return nil, err
	}
	
	aggregated := &AggregatedCapacity{
		VPPID:             vppID,
		TotalCapacity:     vpp.TotalCapacity,
		AvailableCapacity: vpp.AvailableCapacity,
		DispatchablePower: 0,
		StorageCapacity:   0,
		LoadCapacity:      0,
	}
	
	for _, sourceID := range vpp.PowerSourceIDs {
		source, err := e.GetPowerSource(ctx, sourceID)
		if err != nil {
			continue
		}
		if source.Status == PowerSourceStatusOnline {
			aggregated.DispatchablePower += source.RealtimeOutput
		}
	}
	
	for _, storageID := range vpp.StorageIDs {
		storage, err := e.storageSvc.GetStorageDevice(ctx, storageID)
		if err != nil {
			continue
		}
		availableEnergy := (storage.SOC - storage.MinSOC) * storage.Capacity / 100
		aggregated.StorageCapacity += availableEnergy
		aggregated.DispatchablePower += storage.MaxDischargeRate
	}
	
	return aggregated, nil
}

func (e *EnergyMarketCore) GetMarketOverview(ctx context.Context, region string) (*MarketOverview, error) {
	quote, err := e.GetLatestPriceQuote(ctx, region, EnergyOrderSpot)
	if err != nil {
		klog.Warningf("获取最新价格失败: %v", err)
	}
	
	overview := &MarketOverview{
		Region:         region,
		CurrentPrice:   0,
		PriceChange:    0,
		TradingVolume:  0,
		GreenRatio:     0,
		PeakPrice:      0,
		ValleyPrice:    0,
		ActiveOrders:   0,
		AvailablePower: 0,
	}
	
	if quote != nil {
		overview.CurrentPrice = quote.SpotPrice
		overview.PeakPrice = quote.PeakPrice
		overview.ValleyPrice = quote.ValleyPrice
	}
	
	e.mu.RLock()
	for _, source := range e.powerSources {
		if source.Location.Region == region || region == "" {
			overview.AvailablePower += source.RealtimeOutput
			if source.Type == PowerSourceSolar || source.Type == PowerSourceWind || source.Type == PowerSourceHydro {
				overview.GreenRatio += source.RealtimeOutput
			}
		}
	}
	
	if overview.AvailablePower > 0 {
		overview.GreenRatio = overview.GreenRatio / overview.AvailablePower * 100
	}
	e.mu.RUnlock()
	
	return overview, nil
}

func (e *EnergyMarketCore) GetTradingVolume(ctx context.Context, region string, period string) (*TradingVolume, error) {
	filter := &OrderFilter{
		Status: OrderStatusFilled,
	}
	
	orders, err := e.repo.ListOrders(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("获取订单失败: %w", err)
	}
	
	volume := &TradingVolume{
		Region:          region,
		TotalVolume:     0,
		TotalAmount:     0,
		TransactionCount: 0,
		GreenVolume:     0,
		Period:          period,
	}
	
	now := time.Now()
	var startTime time.Time
	switch period {
	case "day":
		startTime = now.Add(-24 * time.Hour)
	case "week":
		startTime = now.Add(-7 * 24 * time.Hour)
	case "month":
		startTime = now.Add(-30 * 24 * time.Hour)
	default:
		startTime = time.Time{}
	}
	
	for _, order := range orders {
		if !startTime.IsZero() && order.CreatedAt.Before(startTime) {
			continue
		}
		if order.DeliveryRegion != region && region != "" {
			continue
		}
		
		volume.TotalVolume += order.Quantity
		volume.TotalAmount += order.TotalAmount
		volume.TransactionCount++
		
		if order.IsGreen {
			volume.GreenVolume += order.Quantity
		}
	}
	
	return volume, nil
}

func (e *EnergyMarketCore) GetMarketDepth(ctx context.Context, region string, energyType EnergyOrderType) (*MarketDepth, error) {
	buyFilter := &OrderFilter{
		Type:       OrderTypeBuy,
		EnergyType: energyType,
		Status:     OrderStatusPending,
	}
	sellFilter := &OrderFilter{
		Type:       OrderTypeSell,
		EnergyType: energyType,
		Status:     OrderStatusPending,
	}
	
	buyOrders, err := e.repo.ListOrders(ctx, buyFilter)
	if err != nil {
		return nil, fmt.Errorf("获取买单失败: %w", err)
	}
	
	sellOrders, err := e.repo.ListOrders(ctx, sellFilter)
	if err != nil {
		return nil, fmt.Errorf("获取卖单失败: %w", err)
	}
	
	depth := &MarketDepth{
		Region:     region,
		EnergyType: energyType,
		BuyOrders:  aggregateOrderLevels(buyOrders),
		SellOrders: aggregateOrderLevels(sellOrders),
	}
	
	if len(depth.BuyOrders) > 0 {
		depth.BestBid = depth.BuyOrders[0].Price
	}
	if len(depth.SellOrders) > 0 {
		depth.BestAsk = depth.SellOrders[0].Price
	}
	if depth.BestBid > 0 && depth.BestAsk > 0 {
		depth.Spread = depth.BestAsk - depth.BestBid
	}
	
	return depth, nil
}

func aggregateOrderLevels(orders []*EnergyOrder) []*OrderLevel {
	priceMap := make(map[float64]*OrderLevel)
	
	for _, order := range orders {
		if level, exists := priceMap[order.Price]; exists {
			level.Quantity += order.Quantity
			level.Count++
		} else {
			priceMap[order.Price] = &OrderLevel{
				Price:    order.Price,
				Quantity: order.Quantity,
				Count:    1,
			}
		}
	}
	
	levels := make([]*OrderLevel, 0, len(priceMap))
	for _, level := range priceMap {
		levels = append(levels, level)
	}
	
	return levels
}

func (e *EnergyMarketCore) GetLatestPriceQuote(ctx context.Context, region string, energyType EnergyOrderType) (*PriceQuote, error) {
	cacheKey := fmt.Sprintf("%s_%s", region, energyType)
	
	e.mu.RLock()
	if quote, ok := e.priceCache[cacheKey]; ok {
		if time.Since(quote.UpdatedAt) < 5*time.Minute {
			e.mu.RUnlock()
			return quote, nil
		}
	}
	e.mu.RUnlock()
	
	quote, err := e.repo.GetLatestPriceQuote(ctx, region, energyType)
	if err != nil {
		if e.priceProvider != nil {
			quote, err = e.priceProvider.GetRealtimePrice(ctx, region)
			if err != nil {
				return nil, fmt.Errorf("获取实时价格失败: %w", err)
			}
		} else {
			return nil, fmt.Errorf("获取价格报价失败: %w", err)
		}
	}
	
	e.mu.Lock()
	e.priceCache[cacheKey] = quote
	e.mu.Unlock()
	
	return quote, nil
}

func (e *EnergyMarketCore) SubscribePriceUpdates(ctx context.Context, region string, energyType EnergyOrderType) (<-chan *PriceQuote, error) {
	key := fmt.Sprintf("%s_%s", region, energyType)
	ch := make(chan *PriceQuote, 100)
	
	e.subscriberMu.Lock()
	e.priceSubscribers[key] = append(e.priceSubscribers[key], ch)
	e.subscriberMu.Unlock()
	
	go func() {
		<-ctx.Done()
		e.subscriberMu.Lock()
		subs := e.priceSubscribers[key]
		for i, sub := range subs {
			if sub == ch {
				e.priceSubscribers[key] = append(subs[:i], subs[i+1:]...)
				break
			}
		}
		e.subscriberMu.Unlock()
		close(ch)
	}()
	
	return ch, nil
}

func (e *EnergyMarketCore) SubscribeOrderUpdates(ctx context.Context, orderID uuid.UUID) (<-chan *EnergyOrder, error) {
	ch := make(chan *EnergyOrder, 10)
	
	e.subscriberMu.Lock()
	e.orderSubscribers[orderID] = append(e.orderSubscribers[orderID], ch)
	e.subscriberMu.Unlock()
	
	go func() {
		<-ctx.Done()
		e.subscriberMu.Lock()
		subs := e.orderSubscribers[orderID]
		for i, sub := range subs {
			if sub == ch {
				e.orderSubscribers[orderID] = append(subs[:i], subs[i+1:]...)
				break
			}
		}
		e.subscriberMu.Unlock()
		close(ch)
	}()
	
	return ch, nil
}

func (e *EnergyMarketCore) notifyPriceUpdate(key string, quote *PriceQuote) {
	e.subscriberMu.RLock()
	subs := e.priceSubscribers[key]
	e.subscriberMu.RUnlock()
	
	for _, ch := range subs {
		select {
		case ch <- quote:
		default:
			klog.Warningf("价格更新通知通道已满: %s", key)
		}
	}
}

func (e *EnergyMarketCore) notifyOrderUpdate(orderID uuid.UUID, order *EnergyOrder) {
	e.subscriberMu.RLock()
	subs := e.orderSubscribers[orderID]
	e.subscriberMu.RUnlock()
	
	for _, ch := range subs {
		select {
		case ch <- order:
		default:
			klog.Warningf("订单更新通知通道已满: %s", orderID)
		}
	}
}

func (e *EnergyMarketCore) ScheduleComputeWithEnergy(ctx context.Context, request *ComputeEnergyRequest) (*ComputeEnergyCoordination, error) {
	optimalSlot, err := e.GetOptimalTimeSlot(ctx, &OptimalTimeRequest{
		EstimatedPower:   request.EstimatedPower,
		Duration:         request.Duration,
		Region:           request.Region,
		OptimizationGoal: "cost",
	})
	if err != nil {
		return nil, fmt.Errorf("获取最优时段失败: %w", err)
	}
	
	startTime, _ := time.Parse(time.RFC3339, optimalSlot.StartTime)
	endTime, _ := time.Parse(time.RFC3339, optimalSlot.EndTime)
	
	coordination := &ComputeEnergyCoordination{
		ComputeJobID:      request.ComputeJobID,
		ScheduledStart:    startTime,
		ScheduledEnd:      endTime,
		EnergyCost:        optimalSlot.ExpectedCost,
		GreenRatio:        optimalSlot.ExpectedGreenRatio,
		CarbonFootprint:   request.EstimatedPower * float64(request.Duration) / 60 * (1 - optimalSlot.ExpectedGreenRatio) * 0.5,
		Status:            "scheduled",
		OptimizationScore: optimalSlot.Confidence,
	}
	
	klog.Infof("计算任务 %s 能源协同调度成功, 开始时间: %s, 预期成本: %.2f, 绿电比例: %.2f%%",
		request.ComputeJobID, optimalSlot.StartTime, optimalSlot.ExpectedCost, optimalSlot.ExpectedGreenRatio*100)
	
	return coordination, nil
}

func (e *EnergyMarketCore) GetOptimalTimeSlot(ctx context.Context, request *OptimalTimeRequest) (*OptimalTimeSlot, error) {
	forecast, err := e.GetEnergyForecast(ctx, request.Region, request.Duration)
	if err != nil {
		return nil, fmt.Errorf("获取能源预测失败: %w", err)
	}
	
	if len(forecast) == 0 {
		return nil, fmt.Errorf("无可用预测数据")
	}
	
	var bestSlot *OptimalTimeSlot
	bestScore := math.Inf(1)
	
	for i := 0; i <= len(forecast)-request.Duration; i++ {
		slot := forecast[i]
		endSlot := forecast[i+request.Duration-1]
		
		var score float64
		switch request.OptimizationGoal {
		case "cost":
			score = slot.ExpectedPrice
		case "green":
			score = -slot.GreenRatio
		case "balanced":
			score = slot.ExpectedPrice * (1 - slot.GreenRatio)
		default:
			score = slot.ExpectedPrice
		}
		
		if score < bestScore {
			bestScore = score
			bestSlot = &OptimalTimeSlot{
				StartTime:        slot.Timestamp.Format(time.RFC3339),
				EndTime:          endSlot.Timestamp.Format(time.RFC3339),
				ExpectedCost:     request.EstimatedPower * slot.ExpectedPrice * float64(request.Duration) / 60,
				ExpectedGreenRatio: slot.GreenRatio,
				Confidence:       slot.Confidence,
				Reason:           fmt.Sprintf("优化目标: %s, 评分: %.4f", request.OptimizationGoal, score),
			}
		}
	}
	
	if bestSlot == nil {
		bestSlot = &OptimalTimeSlot{
			StartTime:        time.Now().Format(time.RFC3339),
			EndTime:          time.Now().Add(time.Duration(request.Duration) * time.Minute).Format(time.RFC3339),
			ExpectedCost:     request.EstimatedPower * 0.5 * float64(request.Duration) / 60,
			ExpectedGreenRatio: 0.5,
			Confidence:       0.3,
			Reason:           "使用默认时段",
		}
	}
	
	return bestSlot, nil
}

func (e *EnergyMarketCore) GetEnergyForecast(ctx context.Context, region string, horizon int) ([]*EnergyForecastPoint, error) {
	forecast := make([]*EnergyForecastPoint, 0, horizon)
	
	now := time.Now()
	for i := 0; i < horizon; i++ {
		t := now.Add(time.Duration(i) * time.Minute)
		hour := t.Hour()
		
		var basePrice float64
		var greenRatio float64
		
		if hour >= 8 && hour < 12 || hour >= 18 && hour < 22 {
			basePrice = 0.8
		} else if hour >= 0 && hour < 6 {
			basePrice = 0.3
		} else {
			basePrice = 0.5
		}
		
		if hour >= 8 && hour < 18 {
			greenRatio = 0.7
		} else {
			greenRatio = 0.3
		}
		
		confidence := math.Max(0.5, 0.9-float64(i)/float64(horizon)*0.4)
		
		forecast = append(forecast, &EnergyForecastPoint{
			Timestamp:     t,
			ExpectedPower: 1000 + float64(i%60)*10,
			ExpectedPrice: basePrice + float64(i%30)*0.01,
			GreenRatio:    greenRatio,
			Confidence:    confidence,
		})
	}
	
	return forecast, nil
}

func (e *EnergyMarketCore) StartPriceMonitoring(ctx context.Context, interval int) error {
	if e.priceProvider == nil {
		return fmt.Errorf("未配置价格数据提供者")
	}
	
	go func() {
		ticker := time.NewTicker(time.Duration(interval) * time.Second)
		defer ticker.Stop()
		
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				quote, err := e.priceProvider.GetRealtimePrice(ctx, e.config.Region)
				if err != nil {
					klog.Warningf("获取实时价格失败: %v", err)
					continue
				}
				
				cacheKey := fmt.Sprintf("%s_%s", e.config.Region, EnergyOrderSpot)
				e.mu.Lock()
				e.priceCache[cacheKey] = quote
				e.mu.Unlock()
				
				e.notifyPriceUpdate(cacheKey, quote)
			}
		}
	}()
	
	klog.Infof("启动价格监控, 区域: %s, 间隔: %d秒", e.config.Region, interval)
	return nil
}

func (e *EnergyMarketCore) GetConfig() *EnergyMarketConfig {
	return e.config
}

func (e *EnergyMarketCore) IsPeakHour(t time.Time) bool {
	hour := t.Hour()
	for _, tr := range e.config.PeakHours {
		startHour := parseHour(tr.Start)
		endHour := parseHour(tr.End)
		if hour >= startHour && hour < endHour {
			return true
		}
	}
	return false
}

func (e *EnergyMarketCore) IsValleyHour(t time.Time) bool {
	hour := t.Hour()
	for _, tr := range e.config.ValleyHours {
		startHour := parseHour(tr.Start)
		endHour := parseHour(tr.End)
		if hour >= startHour && hour < endHour {
			return true
		}
	}
	return false
}

func parseHour(timeStr string) int {
	var h, m int
	fmt.Sscanf(timeStr, "%d:%d", &h, &m)
	return h
}
