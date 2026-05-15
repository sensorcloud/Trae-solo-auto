package energy

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"k8s.io/klog/v2"
)

type StorageManager struct {
	repo        EnergyRepository
	priceSvc    PriceDataProvider
	energyCore  *EnergyMarketCore
	
	devices     map[uuid.UUID]*StorageDevice
	statusCache map[uuid.UUID]*StorageStatus
	
	mu          sync.RWMutex
	
	arbitrageConfig *ArbitrageConfig
}

type ArbitrageConfig struct {
	MinPriceSpread      float64 `json:"min_price_spread"`
	MinProfitMargin     float64 `json:"min_profit_margin"`
	MaxDailyCycles      int     `json:"max_daily_cycles"`
	ChargeEfficiency    float64 `json:"charge_efficiency"`
	DischargeEfficiency float64 `json:"discharge_efficiency"`
	SafetyMarginSOC     float64 `json:"safety_margin_soc"`
}

func DefaultArbitrageConfig() *ArbitrageConfig {
	return &ArbitrageConfig{
		MinPriceSpread:      0.1,
		MinProfitMargin:     0.05,
		MaxDailyCycles:      2,
		ChargeEfficiency:    0.95,
		DischargeEfficiency: 0.95,
		SafetyMarginSOC:     0.1,
	}
}

func NewStorageManager(repo EnergyRepository, energyCore *EnergyMarketCore) *StorageManager {
	return &StorageManager{
		repo:            repo,
		energyCore:      energyCore,
		devices:         make(map[uuid.UUID]*StorageDevice),
		statusCache:     make(map[uuid.UUID]*StorageStatus),
		arbitrageConfig: DefaultArbitrageConfig(),
	}
}

func (sm *StorageManager) SetPriceProvider(provider PriceDataProvider) {
	sm.priceSvc = provider
}

func (sm *StorageManager) SetArbitrageConfig(config *ArbitrageConfig) {
	sm.arbitrageConfig = config
}

func (sm *StorageManager) CreateStorageDevice(ctx context.Context, device *StorageDevice) error {
	if device.ID == uuid.Nil {
		device.ID = uuid.New()
	}
	device.CreatedAt = time.Now()
	device.UpdatedAt = time.Now()
	
	if device.Status == "" {
		device.Status = StorageStatusIdle
	}
	if device.MinSOC == 0 {
		device.MinSOC = 10
	}
	if device.MaxSOC == 0 {
		device.MaxSOC = 100
	}
	if device.SOC < device.MinSOC {
		device.SOC = device.MinSOC
	}
	if device.SOC > device.MaxSOC {
		device.SOC = device.MaxSOC
	}
	
	if err := sm.repo.CreateStorageDevice(ctx, device); err != nil {
		return fmt.Errorf("创建储能设备失败: %w", err)
	}
	
	sm.mu.Lock()
	sm.devices[device.ID] = device
	sm.statusCache[device.ID] = &StorageStatus{
		DeviceID:          device.ID,
		SOC:               device.SOC,
		CurrentPower:      device.CurrentPower,
		Status:            device.Status,
		AvailableCapacity: device.Capacity * (device.SOC - device.MinSOC) / 100,
		HealthState:       device.HealthState,
	}
	sm.mu.Unlock()
	
	klog.Infof("创建储能设备成功: %s, 容量: %.2f %s, SOC: %.1f%%", device.Name, device.Capacity, device.Unit, device.SOC)
	return nil
}

func (sm *StorageManager) GetStorageDevice(ctx context.Context, id uuid.UUID) (*StorageDevice, error) {
	sm.mu.RLock()
	if device, ok := sm.devices[id]; ok {
		sm.mu.RUnlock()
		return device, nil
	}
	sm.mu.RUnlock()
	
	device, err := sm.repo.GetStorageDevice(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("获取储能设备失败: %w", err)
	}
	
	sm.mu.Lock()
	sm.devices[id] = device
	sm.mu.Unlock()
	
	return device, nil
}

func (sm *StorageManager) ListStorageDevices(ctx context.Context, filter *StorageFilter) ([]*StorageDevice, error) {
	return sm.repo.ListStorageDevices(ctx, filter)
}

func (sm *StorageManager) UpdateStorageDevice(ctx context.Context, id uuid.UUID, updates *StorageUpdate) error {
	device, err := sm.GetStorageDevice(ctx, id)
	if err != nil {
		return err
	}
	
	if updates.Name != nil {
		device.Name = *updates.Name
	}
	if updates.Status != nil {
		device.Status = *updates.Status
	}
	if updates.MaxChargeRate != nil {
		device.MaxChargeRate = *updates.MaxChargeRate
	}
	if updates.MaxDischargeRate != nil {
		device.MaxDischargeRate = *updates.MaxDischargeRate
	}
	if updates.Strategy != nil {
		device.Strategy = *updates.Strategy
	}
	device.UpdatedAt = time.Now()
	
	if err := sm.repo.UpdateStorageDevice(ctx, device); err != nil {
		return fmt.Errorf("更新储能设备失败: %w", err)
	}
	
	sm.mu.Lock()
	sm.devices[id] = device
	sm.mu.Unlock()
	
	klog.Infof("更新储能设备成功: %s", id)
	return nil
}

func (sm *StorageManager) DeleteStorageDevice(ctx context.Context, id uuid.UUID) error {
	if err := sm.repo.DeleteStorageDevice(ctx, id); err != nil {
		return fmt.Errorf("删除储能设备失败: %w", err)
	}
	
	sm.mu.Lock()
	delete(sm.devices, id)
	delete(sm.statusCache, id)
	sm.mu.Unlock()
	
	klog.Infof("删除储能设备成功: %s", id)
	return nil
}

func (sm *StorageManager) Charge(ctx context.Context, id uuid.UUID, power float64) error {
	device, err := sm.GetStorageDevice(ctx, id)
	if err != nil {
		return err
	}
	
	if device.Status == StorageStatusFault || device.Status == StorageStatusMaintenance {
		return fmt.Errorf("储能设备状态不允许充电: %s", device.Status)
	}
	
	if device.Status == StorageStatusDischarging {
		return fmt.Errorf("储能设备正在放电，无法同时充电")
	}
	
	if power > device.MaxChargeRate {
		power = device.MaxChargeRate
	}
	
	if device.SOC >= device.MaxSOC {
		return fmt.Errorf("储能设备已满，SOC: %.1f%%", device.SOC)
	}
	
	device.Status = StorageStatusCharging
	device.CurrentPower = power
	device.UpdatedAt = time.Now()
	
	sm.mu.Lock()
	sm.devices[id] = device
	if status, ok := sm.statusCache[id]; ok {
		status.Status = device.Status
		status.CurrentPower = power
	}
	sm.mu.Unlock()
	
	klog.Infof("储能设备 %s 开始充电, 功率: %.2f kW, 当前SOC: %.1f%%", device.Name, power, device.SOC)
	return nil
}

func (sm *StorageManager) Discharge(ctx context.Context, id uuid.UUID, power float64) error {
	device, err := sm.GetStorageDevice(ctx, id)
	if err != nil {
		return err
	}
	
	if device.Status == StorageStatusFault || device.Status == StorageStatusMaintenance {
		return fmt.Errorf("储能设备状态不允许放电: %s", device.Status)
	}
	
	if device.Status == StorageStatusCharging {
		return fmt.Errorf("储能设备正在充电，无法同时放电")
	}
	
	if power > device.MaxDischargeRate {
		power = device.MaxDischargeRate
	}
	
	if device.SOC <= device.MinSOC {
		return fmt.Errorf("储能设备电量不足，SOC: %.1f%%, 最低SOC: %.1f%%", device.SOC, device.MinSOC)
	}
	
	device.Status = StorageStatusDischarging
	device.CurrentPower = power
	device.UpdatedAt = time.Now()
	
	sm.mu.Lock()
	sm.devices[id] = device
	if status, ok := sm.statusCache[id]; ok {
		status.Status = device.Status
		status.CurrentPower = power
	}
	sm.mu.Unlock()
	
	klog.Infof("储能设备 %s 开始放电, 功率: %.2f kW, 当前SOC: %.1f%%", device.Name, power, device.SOC)
	return nil
}

func (sm *StorageManager) StopChargeDischarge(ctx context.Context, id uuid.UUID) error {
	device, err := sm.GetStorageDevice(ctx, id)
	if err != nil {
		return err
	}
	
	device.Status = StorageStatusIdle
	device.CurrentPower = 0
	device.UpdatedAt = time.Now()
	
	sm.mu.Lock()
	sm.devices[id] = device
	if status, ok := sm.statusCache[id]; ok {
		status.Status = device.Status
		status.CurrentPower = 0
	}
	sm.mu.Unlock()
	
	klog.Infof("储能设备 %s 停止充放电", device.Name)
	return nil
}

func (sm *StorageManager) UpdateSOC(ctx context.Context, id uuid.UUID, soc float64) error {
	device, err := sm.GetStorageDevice(ctx, id)
	if err != nil {
		return err
	}
	
	if soc < 0 {
		soc = 0
	}
	if soc > 100 {
		soc = 100
	}
	
	oldSOC := device.SOC
	device.SOC = soc
	device.UpdatedAt = time.Now()
	
	if soc < device.MinSOC && device.Status == StorageStatusDischarging {
		device.Status = StorageStatusIdle
		device.CurrentPower = 0
	}
	
	if soc >= device.MaxSOC && device.Status == StorageStatusCharging {
		device.Status = StorageStatusIdle
		device.CurrentPower = 0
	}
	
	if oldSOC < soc {
		device.CycleCount += int((soc - oldSOC) / 100)
	}
	
	sm.mu.Lock()
	sm.devices[id] = device
	if status, ok := sm.statusCache[id]; ok {
		status.SOC = soc
		status.Status = device.Status
		status.AvailableCapacity = device.Capacity * (soc - device.MinSOC) / 100
	}
	sm.mu.Unlock()
	
	klog.V(4).Infof("储能设备 %s SOC更新: %.1f%% -> %.1f%%", device.Name, oldSOC, soc)
	return nil
}

func (sm *StorageManager) GetStorageStatus(ctx context.Context, id uuid.UUID) (*StorageStatus, error) {
	sm.mu.RLock()
	if status, ok := sm.statusCache[id]; ok {
		sm.mu.RUnlock()
		return status, nil
	}
	sm.mu.RUnlock()
	
	device, err := sm.GetStorageDevice(ctx, id)
	if err != nil {
		return nil, err
	}
	
	status := &StorageStatus{
		DeviceID:          device.ID,
		SOC:               device.SOC,
		CurrentPower:      device.CurrentPower,
		Status:            device.Status,
		AvailableCapacity: device.Capacity * (device.SOC - device.MinSOC) / 100,
		HealthState:       device.HealthState,
	}
	
	sm.mu.Lock()
	sm.statusCache[id] = status
	sm.mu.Unlock()
	
	return status, nil
}

func (sm *StorageManager) OptimizeSchedule(ctx context.Context, id uuid.UUID, prices []*PriceQuote) (*StorageOptimizationResult, error) {
	device, err := sm.GetStorageDevice(ctx, id)
	if err != nil {
		return nil, err
	}
	
	if len(prices) == 0 {
		return nil, fmt.Errorf("无价格数据用于优化")
	}
	
	result := &StorageOptimizationResult{
		DeviceID:  id,
		Timestamp: time.Now(),
	}
	
	pricePoints := sm.analyzePricePattern(prices)
	
	bestChargeWindow, bestDischargeWindow := sm.findOptimalWindows(pricePoints)
	
	if bestChargeWindow != nil && bestDischargeWindow != nil {
		priceSpread := bestDischargeWindow.avgPrice - bestChargeWindow.avgPrice
		adjustedSpread := priceSpread * sm.arbitrageConfig.ChargeEfficiency * sm.arbitrageConfig.DischargeEfficiency
		
		if adjustedSpread > sm.arbitrageConfig.MinPriceSpread {
			chargeEnergy := device.Capacity * (device.MaxSOC - device.SOC) / 100
			dischargeEnergy := device.Capacity * (device.SOC - device.MinSOC) / 100
			energy := math.Min(chargeEnergy, dischargeEnergy)
			
			profit := energy * adjustedSpread
			profitMargin := profit / (energy * bestChargeWindow.avgPrice)
			
			if profitMargin >= sm.arbitrageConfig.MinProfitMargin {
				result.RecommendedAction = StorageAction{
					Mode:      "arbitrage",
					Power:     device.MaxChargeRate,
					Duration:  int(energy / device.MaxChargeRate * 60),
					StartTime: bestChargeWindow.startTime,
					EndTime:   bestDischargeWindow.endTime,
				}
				result.ExpectedProfit = profit
				result.Confidence = sm.calculateConfidence(pricePoints, bestChargeWindow, bestDischargeWindow)
				result.Reason = fmt.Sprintf("峰谷价差套利: 充电时段 %s (均价 %.3f), 放电时段 %s (均价 %.3f), 预期收益 %.2f",
					bestChargeWindow.startTime, bestChargeWindow.avgPrice,
					bestDischargeWindow.startTime, bestDischargeWindow.avgPrice, profit)
			}
		}
	}
	
	if result.RecommendedAction.Mode == "" {
		if device.SOC < device.MinSOC+10 {
			result.RecommendedAction = StorageAction{
				Mode:  "charge",
				Power: device.MaxChargeRate * 0.5,
			}
			result.Reason = "SOC偏低，建议充电"
		} else if device.SOC > device.MaxSOC-10 {
			result.RecommendedAction = StorageAction{
				Mode:  "discharge",
				Power: device.MaxDischargeRate * 0.5,
			}
			result.Reason = "SOC偏高，建议放电"
		} else {
			result.RecommendedAction = StorageAction{
				Mode:  "idle",
				Power: 0,
			}
			result.Reason = "当前无需操作"
		}
		result.ExpectedProfit = 0
		result.Confidence = 0.5
	}
	
	klog.Infof("储能设备 %s 调度优化: %s", device.Name, result.Reason)
	return result, nil
}

type priceWindow struct {
	startTime string
	endTime   string
	avgPrice  float64
	minPrice  float64
	maxPrice  float64
	hours     int
}

type pricePoint struct {
	hour  int
	price float64
}

func (sm *StorageManager) analyzePricePattern(prices []*PriceQuote) []*pricePoint {
	points := make([]*pricePoint, 24)
	
	for i := 0; i < 24; i++ {
		points[i] = &pricePoint{
			hour:  i,
			price: 0.5,
		}
	}
	
	if len(prices) > 0 {
		latestPrice := prices[len(prices)-1]
		points[8].price = latestPrice.PeakPrice
		points[9].price = latestPrice.PeakPrice
		points[10].price = latestPrice.PeakPrice
		points[11].price = latestPrice.PeakPrice
		points[18].price = latestPrice.PeakPrice
		points[19].price = latestPrice.PeakPrice
		points[20].price = latestPrice.PeakPrice
		points[21].price = latestPrice.PeakPrice
		
		points[0].price = latestPrice.ValleyPrice
		points[1].price = latestPrice.ValleyPrice
		points[2].price = latestPrice.ValleyPrice
		points[3].price = latestPrice.ValleyPrice
		points[4].price = latestPrice.ValleyPrice
		points[5].price = latestPrice.ValleyPrice
		
		points[6].price = latestPrice.FlatPrice
		points[7].price = latestPrice.FlatPrice
		points[12].price = latestPrice.FlatPrice
		points[13].price = latestPrice.FlatPrice
		points[14].price = latestPrice.FlatPrice
		points[15].price = latestPrice.FlatPrice
		points[16].price = latestPrice.FlatPrice
		points[17].price = latestPrice.FlatPrice
		points[22].price = latestPrice.FlatPrice
		points[23].price = latestPrice.FlatPrice
	}
	
	return points
}

func (sm *StorageManager) findOptimalWindows(points []*pricePoint) (*priceWindow, *priceWindow) {
	sort.Slice(points, func(i, j int) bool {
		return points[i].price < points[j].price
	})
	
	valleyHours := make([]int, 0, 6)
	for _, p := range points[:6] {
		valleyHours = append(valleyHours, p.hour)
	}
	sort.Ints(valleyHours)
	
	peakHours := make([]int, 0, 6)
	for _, p := range points[18:] {
		peakHours = append(peakHours, p.hour)
	}
	sort.Ints(peakHours)
	
	chargeWindow := sm.consolidateHours(valleyHours, points)
	dischargeWindow := sm.consolidateHours(peakHours, points)
	
	return chargeWindow, dischargeWindow
}

func (sm *StorageManager) consolidateHours(hours []int, points []*pricePoint) *priceWindow {
	if len(hours) == 0 {
		return nil
	}
	
	var avgPrice, minPrice, maxPrice float64
	minPrice = math.MaxFloat64
	maxPrice = 0
	
	for _, h := range hours {
		price := points[h].price
		avgPrice += price
		if price < minPrice {
			minPrice = price
		}
		if price > maxPrice {
			maxPrice = price
		}
	}
	avgPrice /= float64(len(hours))
	
	startHour := hours[0]
	endHour := hours[len(hours)-1] + 1
	
	return &priceWindow{
		startTime: fmt.Sprintf("%02d:00", startHour),
		endTime:   fmt.Sprintf("%02d:00", endHour),
		avgPrice:  avgPrice,
		minPrice:  minPrice,
		maxPrice:  maxPrice,
		hours:     len(hours),
	}
}

func (sm *StorageManager) calculateConfidence(points []*pricePoint, chargeWindow, dischargeWindow *priceWindow) float64 {
	if chargeWindow == nil || dischargeWindow == nil {
		return 0
	}
	
	spread := dischargeWindow.avgPrice - chargeWindow.avgPrice
	avgPrice := (chargeWindow.avgPrice + dischargeWindow.avgPrice) / 2
	
	spreadRatio := spread / avgPrice
	
	confidence := math.Min(0.95, 0.5+spreadRatio*2)
	
	return confidence
}

func (sm *StorageManager) ExecuteArbitrage(ctx context.Context, id uuid.UUID) error {
	device, err := sm.GetStorageDevice(ctx, id)
	if err != nil {
		return err
	}
	
	if device.Strategy.Type == "" {
		device.Strategy = StorageStrategy{
			Type:               "peak_valley",
			PeakPriceThreshold: 0.6,
			ValleyPriceThreshold: 0.3,
			MinProfitMargin:    sm.arbitrageConfig.MinProfitMargin,
			MaxDailyCycles:     sm.arbitrageConfig.MaxDailyCycles,
			SafetyMargin:       sm.arbitrageConfig.SafetyMarginSOC,
		}
	}
	
	region := device.Location.Region
	if region == "" {
		region = "default"
	}
	
	quote, err := sm.energyCore.GetLatestPriceQuote(ctx, region, EnergyOrderSpot)
	if err != nil {
		return fmt.Errorf("获取价格失败: %w", err)
	}
	
	currentPrice := quote.SpotPrice
	now := time.Now()
	hour := now.Hour()
	
	isValley := currentPrice <= device.Strategy.ValleyPriceThreshold || (hour >= 0 && hour < 6)
	isPeak := currentPrice >= device.Strategy.PeakPriceThreshold || (hour >= 8 && hour < 12) || (hour >= 18 && hour < 22)
	
	switch device.Status {
	case StorageStatusIdle:
		if isValley && device.SOC < device.MaxSOC-10 {
			if err := sm.Charge(ctx, id, device.MaxChargeRate); err != nil {
				return err
			}
			klog.Infof("储能设备 %s 执行谷时充电, 当前价格: %.3f, SOC: %.1f%%", device.Name, currentPrice, device.SOC)
		} else if isPeak && device.SOC > device.MinSOC+10 {
			if err := sm.Discharge(ctx, id, device.MaxDischargeRate); err != nil {
				return err
			}
			klog.Infof("储能设备 %s 执行峰时放电, 当前价格: %.3f, SOC: %.1f%%", device.Name, currentPrice, device.SOC)
		}
		
	case StorageStatusCharging:
		if !isValley || device.SOC >= device.MaxSOC {
			if err := sm.StopChargeDischarge(ctx, id); err != nil {
				return err
			}
			klog.Infof("储能设备 %s 停止充电, 当前价格: %.3f, SOC: %.1f%%", device.Name, currentPrice, device.SOC)
		}
		
	case StorageStatusDischarging:
		if !isPeak || device.SOC <= device.MinSOC {
			if err := sm.StopChargeDischarge(ctx, id); err != nil {
				return err
			}
			klog.Infof("储能设备 %s 停止放电, 当前价格: %.3f, SOC: %.1f%%", device.Name, currentPrice, device.SOC)
		}
	}
	
	return nil
}

func (sm *StorageManager) CalculateArbitrageProfit(ctx context.Context, id uuid.UUID, period string) (*ArbitrageProfitResult, error) {
	device, err := sm.GetStorageDevice(ctx, id)
	if err != nil {
		return nil, err
	}
	
	result := &ArbitrageProfitResult{
		DeviceID:    id,
		Period:      period,
		TotalProfit: 0,
		ChargeEnergy: 0,
		DischargeEnergy: 0,
		AvgChargePrice: 0,
		AvgDischargePrice: 0,
	}
	
	cycles := device.CycleCount
	if cycles > 0 {
		avgCycleProfit := device.Capacity * 0.3
		result.TotalProfit = float64(cycles) * avgCycleProfit
		result.ChargeEnergy = float64(cycles) * device.Capacity
		result.DischargeEnergy = float64(cycles) * device.Capacity * sm.arbitrageConfig.DischargeEfficiency
	}
	
	return result, nil
}

type ArbitrageProfitResult struct {
	DeviceID          uuid.UUID `json:"device_id"`
	Period            string    `json:"period"`
	TotalProfit       float64   `json:"total_profit"`
	ChargeEnergy      float64   `json:"charge_energy"`
	DischargeEnergy   float64   `json:"discharge_energy"`
	AvgChargePrice    float64   `json:"avg_charge_price"`
	AvgDischargePrice float64   `json:"avg_discharge_price"`
	Cycles            int       `json:"cycles"`
}

func (sm *StorageManager) GetAvailableCapacity(ctx context.Context, region string) (float64, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	var totalCapacity float64
	for _, device := range sm.devices {
		if region == "" || device.Location.Region == region {
			if device.Status != StorageStatusFault && device.Status != StorageStatusMaintenance {
				available := device.Capacity * (device.SOC - device.MinSOC) / 100
				if available > 0 {
					totalCapacity += available
				}
			}
		}
	}
	
	return totalCapacity, nil
}

func (sm *StorageManager) GetTotalDispatchablePower(ctx context.Context, region string) (float64, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	var totalPower float64
	for _, device := range sm.devices {
		if region == "" || device.Location.Region == region {
			if device.Status == StorageStatusIdle || device.Status == StorageStatusDischarging {
				if device.SOC > device.MinSOC {
					totalPower += device.MaxDischargeRate
				}
			}
		}
	}
	
	return totalPower, nil
}

func (sm *StorageManager) StartAutoArbitrage(ctx context.Context, id uuid.UUID, intervalSeconds int) error {
	device, err := sm.GetStorageDevice(ctx, id)
	if err != nil {
		return err
	}
	
	go func() {
		ticker := time.NewTicker(time.Duration(intervalSeconds) * time.Second)
		defer ticker.Stop()
		
		for {
			select {
			case <-ctx.Done():
				klog.Infof("储能设备 %s 自动套利停止", device.Name)
				return
			case <-ticker.C:
				if err := sm.ExecuteArbitrage(ctx, id); err != nil {
					klog.Warningf("储能设备 %s 自动套利执行失败: %v", device.Name, err)
				}
			}
		}
	}()
	
	klog.Infof("储能设备 %s 启动自动套利, 间隔: %d秒", device.Name, intervalSeconds)
	return nil
}

func (sm *StorageManager) SetSchedule(ctx context.Context, id uuid.UUID, schedules []StorageSchedule) error {
	device, err := sm.GetStorageDevice(ctx, id)
	if err != nil {
		return err
	}
	
	device.Schedule = schedules
	device.UpdatedAt = time.Now()
	
	if err := sm.repo.UpdateStorageDevice(ctx, device); err != nil {
		return fmt.Errorf("设置调度计划失败: %w", err)
	}
	
	sm.mu.Lock()
	sm.devices[id] = device
	sm.mu.Unlock()
	
	klog.Infof("储能设备 %s 设置调度计划成功, 共 %d 条", device.Name, len(schedules))
	return nil
}

func (sm *StorageManager) ExecuteSchedule(ctx context.Context, id uuid.UUID) error {
	device, err := sm.GetStorageDevice(ctx, id)
	if err != nil {
		return err
	}
	
	if len(device.Schedule) == 0 {
		return nil
	}
	
	now := time.Now()
	currentTime := now.Format("15:04")
	
	for _, schedule := range schedulesByPriority(device.Schedule) {
		if !schedule.Enabled {
			continue
		}
		
		if currentTime >= schedule.StartTime && currentTime < schedule.EndTime {
			switch schedule.Mode {
			case "charge":
				if device.Status != StorageStatusCharging {
					power := schedule.Power
					if power == 0 {
						power = device.MaxChargeRate
					}
					if err := sm.Charge(ctx, id, power); err != nil {
						klog.Warningf("执行充电计划失败: %v", err)
					}
				}
				return nil
				
			case "discharge":
				if device.Status != StorageStatusDischarging {
					power := schedule.Power
					if power == 0 {
						power = device.MaxDischargeRate
					}
					if err := sm.Discharge(ctx, id, power); err != nil {
						klog.Warningf("执行放电计划失败: %v", err)
					}
				}
				return nil
				
			case "idle":
				if device.Status != StorageStatusIdle {
					if err := sm.StopChargeDischarge(ctx, id); err != nil {
						klog.Warningf("执行空闲计划失败: %v", err)
					}
				}
				return nil
			}
		}
	}
	
	return nil
}

func schedulesByPriority(schedules []StorageSchedule) []StorageSchedule {
	result := make([]StorageSchedule, len(schedules))
	copy(result, schedules)
	sort.Slice(result, func(i, j int) bool {
		return result[i].Priority > result[j].Priority
	})
	return result
}

func (sm *StorageManager) GetStorageHealthReport(ctx context.Context, id uuid.UUID) (*StorageHealthReport, error) {
	device, err := sm.GetStorageDevice(ctx, id)
	if err != nil {
		return nil, err
	}
	
	report := &StorageHealthReport{
		DeviceID:       id,
		DeviceName:     device.Name,
		HealthState:    device.HealthState,
		SOC:            device.SOC,
		CycleCount:     device.CycleCount,
		MaxCycles:      device.MaxCycles,
		CycleUsage:     float64(device.CycleCount) / float64(device.MaxCycles) * 100,
		RemainingLife:  (1 - float64(device.CycleCount)/float64(device.MaxCycles)) * 100,
		Status:         string(device.Status),
		LastUpdated:    device.UpdatedAt,
	}
	
	if device.HealthState < 0.7 {
		report.Recommendations = append(report.Recommendations, "储能健康状态较低，建议检修")
	}
	if device.SOC < device.MinSOC+5 {
		report.Recommendations = append(report.Recommendations, "SOC接近最低限制，建议充电")
	}
	if report.CycleUsage > 80 {
		report.Recommendations = append(report.Recommendations, "循环次数接近上限，建议规划更换")
	}
	
	return report, nil
}

type StorageHealthReport struct {
	DeviceID       uuid.UUID `json:"device_id"`
	DeviceName     string    `json:"device_name"`
	HealthState    float64   `json:"health_state"`
	SOC            float64   `json:"soc"`
	CycleCount     int       `json:"cycle_count"`
	MaxCycles      int       `json:"max_cycles"`
	CycleUsage     float64   `json:"cycle_usage"`
	RemainingLife  float64   `json:"remaining_life"`
	Status         string    `json:"status"`
	LastUpdated    time.Time `json:"last_updated"`
	Recommendations []string `json:"recommendations"`
}
