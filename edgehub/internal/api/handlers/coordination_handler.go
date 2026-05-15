package handlers

import (
	"github.com/edgehub/edgehub/internal/energy"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type CoordinationHandler struct {
	coordSvc energy.ComputeEnergyCoordinationService
}

func NewCoordinationHandler(coordSvc energy.ComputeEnergyCoordinationService) *CoordinationHandler {
	return &CoordinationHandler{
		coordSvc: coordSvc,
	}
}

// ScheduleComputeWithEnergy 算电协同调度
// @Summary 算电协同调度
// @Description 根据能源情况调度计算任务
// @Tags 算电协同
// @Accept json
// @Produce json
// @Param request body energy.ComputeEnergyRequest true "调度请求"
// @Success 200 {object} Response
// @Router /api/v1/coordination/schedule [post]
func (h *CoordinationHandler) ScheduleComputeWithEnergy(c *gin.Context) {
	var req energy.ComputeEnergyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "无效的请求参数: "+err.Error())
		return
	}

	req.TenantID = GetTenantID(c)

	result, err := h.coordSvc.ScheduleComputeWithEnergy(c.Request.Context(), &req)
	if err != nil {
		InternalError(c, "算电协同调度失败: "+err.Error())
		return
	}

	Success(c, result)
}

// GetOptimalTimeSlot 获取最优时间槽
// @Summary 获取最优时间槽
// @Description 根据能源预测获取最优执行时间
// @Tags 算电协同
// @Accept json
// @Produce json
// @Param estimated_power query number true "预估功率(kW)"
// @Param duration query int true "执行时长(分钟)"
// @Param region query string true "区域"
// @Param optimization_goal query string false "优化目标 (cost, carbon, green)" default(cost)
// @Success 200 {object} Response
// @Router /api/v1/coordination/optimal-time [get]
func (h *CoordinationHandler) GetOptimalTimeSlot(c *gin.Context) {
	estimatedPower := parseFloatDefault(c.Query("estimated_power"), 0)
	if estimatedPower <= 0 {
		BadRequest(c, "预估功率必须大于0")
		return
	}

	duration := parseIntDefault(c.Query("duration"), 0)
	if duration <= 0 {
		BadRequest(c, "执行时长必须大于0")
		return
	}

	region := c.Query("region")
	if region == "" {
		BadRequest(c, "区域参数不能为空")
		return
	}

	optimizationGoal := c.DefaultQuery("optimization_goal", "cost")

	req := &energy.OptimalTimeRequest{
		EstimatedPower:  estimatedPower,
		Duration:        duration,
		Region:          region,
		OptimizationGoal: optimizationGoal,
	}

	slot, err := h.coordSvc.GetOptimalTimeSlot(c.Request.Context(), req)
	if err != nil {
		InternalError(c, "获取最优时间槽失败: "+err.Error())
		return
	}

	Success(c, slot)
}

// GetEnergyForecast 获取能源预测
// @Summary 获取能源预测
// @Description 获取指定区域的能源预测数据
// @Tags 算电协同
// @Accept json
// @Produce json
// @Param region query string true "区域"
// @Param horizon query int false "预测时间范围(小时)" default(24)
// @Success 200 {object} Response
// @Router /api/v1/coordination/forecast [get]
func (h *CoordinationHandler) GetEnergyForecast(c *gin.Context) {
	region := c.Query("region")
	if region == "" {
		BadRequest(c, "区域参数不能为空")
		return
	}

	horizon := parseIntDefault(c.Query("horizon"), 24)

	forecast, err := h.coordSvc.GetEnergyForecast(c.Request.Context(), region, horizon)
	if err != nil {
		InternalError(c, "获取能源预测失败: "+err.Error())
		return
	}

	Success(c, forecast)
}

// GetCarbonIntensity 获取碳排放强度
// @Summary 获取碳排放强度
// @Description 获取指定区域的碳排放强度数据
// @Tags 算电协同
// @Accept json
// @Produce json
// @Param region query string true "区域"
// @Success 200 {object} Response
// @Router /api/v1/coordination/carbon-intensity [get]
func (h *CoordinationHandler) GetCarbonIntensity(c *gin.Context) {
	region := c.Query("region")
	if region == "" {
		BadRequest(c, "区域参数不能为空")
		return
	}

	result := gin.H{
		"region":          region,
		"carbon_intensity": 0.5,
		"unit":            "kg CO2/kWh",
		"green_ratio":     0.3,
		"timestamp":       getCurrentTimestamp(),
	}

	Success(c, result)
}

// GetGreenEnergyRatio 获取绿电比例
// @Summary 获取绿电比例
// @Description 获取指定区域的绿电比例
// @Tags 算电协同
// @Accept json
// @Produce json
// @Param region query string true "区域"
// @Success 200 {object} Response
// @Router /api/v1/coordination/green-ratio [get]
func (h *CoordinationHandler) GetGreenEnergyRatio(c *gin.Context) {
	region := c.Query("region")
	if region == "" {
		BadRequest(c, "区域参数不能为空")
		return
	}

	result := gin.H{
		"region":       region,
		"green_ratio":  0.35,
		"solar_ratio":  0.15,
		"wind_ratio":   0.12,
		"hydro_ratio":  0.08,
		"timestamp":    getCurrentTimestamp(),
	}

	Success(c, result)
}

// OptimizeWorkload 优化工作负载
// @Summary 优化工作负载
// @Description 根据能源情况优化工作负载调度
// @Tags 算电协同
// @Accept json
// @Produce json
// @Param request body WorkloadOptimizationRequest true "优化请求"
// @Success 200 {object} Response
// @Router /api/v1/coordination/workloads/optimize [post]
func (h *CoordinationHandler) OptimizeWorkload(c *gin.Context) {
	var req WorkloadOptimizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "无效的请求参数: "+err.Error())
		return
	}

	result := &WorkloadOptimizationResult{
		WorkloadID:       req.WorkloadID,
		RecommendedStart: "2024-01-01T10:00:00Z",
		RecommendedEnd:   "2024-01-01T14:00:00Z",
		ExpectedCost:     100.50,
		ExpectedCarbon:   50.25,
		GreenRatio:       0.65,
		Confidence:       0.85,
		Reason:           "谷时电价期间，绿电比例较高",
	}

	Success(c, result)
}

// GetWorkloadSchedule 获取工作负载调度计划
// @Summary 获取工作负载调度计划
// @Description 获取工作负载的能源感知调度计划
// @Tags 算电协同
// @Accept json
// @Produce json
// @Param workload_id path string true "工作负载ID"
// @Success 200 {object} Response
// @Router /api/v1/coordination/workloads/{workload_id}/schedule [get]
func (h *CoordinationHandler) GetWorkloadSchedule(c *gin.Context) {
	workloadID := ParseUUID(c.Param("workload_id"))
	if workloadID == uuid.Nil {
		BadRequest(c, "无效的工作负载ID")
		return
	}

	schedule := &WorkloadSchedule{
		WorkloadID: workloadID,
		Schedules: []ScheduleItem{
			{
				StartTime:   "2024-01-01T10:00:00Z",
				EndTime:     "2024-01-01T14:00:00Z",
				Power:       100.0,
				EnergyCost:  50.25,
				CarbonSaved: 25.0,
				GreenRatio:  0.65,
			},
		},
		TotalEnergyCost:  50.25,
		TotalCarbonSaved: 25.0,
		AvgGreenRatio:    0.65,
	}

	Success(c, schedule)
}

// GetEnergyMetrics 获取能源指标
// @Summary 获取能源指标
// @Description 获取算电协同相关的能源指标
// @Tags 算电协同
// @Accept json
// @Produce json
// @Param region query string false "区域"
// @Param cluster_id query string false "集群ID"
// @Param period query string false "时间范围 (day, week, month)" default(day)
// @Success 200 {object} Response
// @Router /api/v1/coordination/metrics [get]
func (h *CoordinationHandler) GetEnergyMetrics(c *gin.Context) {
	region := c.Query("region")
	clusterID := ParseUUID(c.Query("cluster_id"))
	period := c.DefaultQuery("period", "day")

	metrics := &CoordinationMetrics{
		Region:   region,
		Period:   period,
		Metrics: EnergyMetricsData{
			TotalEnergyConsumed:    1000.0,
			TotalGreenEnergy:       350.0,
			GreenRatio:             0.35,
			AvgCarbonIntensity:     0.45,
			TotalCarbonEmission:    450.0,
			CarbonSaved:            150.0,
			TotalEnergyCost:        500.0,
			CostSaved:              75.0,
			PeakShavingEvents:      5,
			LoadShiftingEvents:     10,
			VPPDispatchCount:       3,
			OptimizationScore:      85.0,
		},
		Trend: []MetricTrendPoint{
			{Timestamp: "2024-01-01T00:00:00Z", Value: 0.32},
			{Timestamp: "2024-01-01T06:00:00Z", Value: 0.45},
			{Timestamp: "2024-01-01T12:00:00Z", Value: 0.28},
			{Timestamp: "2024-01-01T18:00:00Z", Value: 0.38},
		},
	}

	if clusterID != uuid.Nil {
		metrics.ClusterID = clusterID
	}

	Success(c, metrics)
}

// GetRealtimeStatus 获取实时状态
// @Summary 获取实时状态
// @Description 获取算电协同系统的实时状态
// @Tags 算电协同
// @Accept json
// @Produce json
// @Param region query string false "区域"
// @Success 200 {object} Response
// @Router /api/v1/coordination/status [get]
func (h *CoordinationHandler) GetRealtimeStatus(c *gin.Context) {
	region := c.Query("region")

	status := &RealtimeStatus{
		Region: region,
		CurrentPower: PowerStatus{
			TotalDemand:     500.0,
			TotalSupply:     600.0,
			GreenSupply:     210.0,
			GridSupply:      390.0,
			StorageDischarge: 0.0,
			StorageCharge:   50.0,
		},
		CurrentPrice: PriceStatus{
			SpotPrice:   0.45,
			PeakPrice:   0.65,
			ValleyPrice: 0.25,
			IsPeak:      false,
		},
		CarbonStatus: CarbonStatus{
			CurrentIntensity: 0.42,
			AvgIntensity:     0.50,
			Trend:            "decreasing",
		},
		OptimizationStatus: OptimizationStatus{
			ActiveWorkloads:   15,
			OptimizedWorkloads: 12,
			PendingOptimizations: 3,
			LastOptimization: "2024-01-01T10:30:00Z",
		},
		Timestamp: getCurrentTimestamp(),
	}

	Success(c, status)
}

// SimulateSchedule 模拟调度
// @Summary 模拟调度
// @Description 模拟不同时间段的调度效果
// @Tags 算电协同
// @Accept json
// @Produce json
// @Param request body ScheduleSimulationRequest true "模拟请求"
// @Success 200 {object} Response
// @Router /api/v1/coordination/simulate [post]
func (h *CoordinationHandler) SimulateSchedule(c *gin.Context) {
	var req ScheduleSimulationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "无效的请求参数: "+err.Error())
		return
	}

	result := &ScheduleSimulationResult{
		Scenarios: []ScheduleScenario{
			{
				Name:           "立即执行",
				StartTime:      "2024-01-01T08:00:00Z",
				EndTime:        "2024-01-01T12:00:00Z",
				EnergyCost:     120.0,
				CarbonEmission: 80.0,
				GreenRatio:     0.25,
				Score:          65.0,
			},
			{
				Name:           "优化调度",
				StartTime:      "2024-01-01T10:00:00Z",
				EndTime:        "2024-01-01T14:00:00Z",
				EnergyCost:     85.0,
				CarbonEmission: 45.0,
				GreenRatio:     0.65,
				Score:          92.0,
			},
			{
				Name:           "谷时执行",
				StartTime:      "2024-01-01T22:00:00Z",
				EndTime:        "2024-01-02T02:00:00Z",
				EnergyCost:     60.0,
				CarbonEmission: 55.0,
				GreenRatio:     0.45,
				Score:          78.0,
			},
		},
		Recommendation: "优化调度",
		Reason:         "综合考虑成本、碳排放和绿电比例，建议在10:00-14:00执行",
	}

	Success(c, result)
}

// GetPolicies 获取协同策略
// @Summary 获取协同策略
// @Description 获取算电协同策略配置
// @Tags 算电协同
// @Accept json
// @Produce json
// @Param region query string false "区域"
// @Success 200 {object} Response
// @Router /api/v1/coordination/policies [get]
func (h *CoordinationHandler) GetPolicies(c *gin.Context) {
	region := c.Query("region")

	policies := &CoordinationPolicies{
		Region: region,
		Policies: []CoordinationPolicy{
			{
				ID:          uuid.New(),
				Name:        "成本优先策略",
				Type:        "cost_optimized",
				Description: "优先考虑能源成本，在谷时电价期间执行高能耗任务",
				Enabled:     true,
				Priority:    1,
				Conditions: PolicyConditions{
					MaxPriceThreshold:    0.5,
					MinGreenRatio:        0.0,
					MaxCarbonIntensity:   1.0,
					AllowedTimeRanges:    []string{"22:00-06:00"},
				},
			},
			{
				ID:          uuid.New(),
				Name:        "低碳优先策略",
				Type:        "carbon_optimized",
				Description: "优先考虑碳排放，在绿电比例高时执行任务",
				Enabled:     true,
				Priority:    2,
				Conditions: PolicyConditions{
					MaxPriceThreshold:    1.0,
					MinGreenRatio:        0.5,
					MaxCarbonIntensity:   0.3,
					AllowedTimeRanges:    []string{"10:00-16:00"},
				},
			},
			{
				ID:          uuid.New(),
				Name:        "平衡策略",
				Type:        "balanced",
				Description: "综合考虑成本和碳排放，寻找最优平衡点",
				Enabled:     true,
				Priority:    3,
				Conditions: PolicyConditions{
					MaxPriceThreshold:    0.6,
					MinGreenRatio:        0.3,
					MaxCarbonIntensity:   0.5,
					AllowedTimeRanges:    []string{},
				},
			},
		},
	}

	Success(c, policies)
}

// UpdatePolicy 更新协同策略
// @Summary 更新协同策略
// @Description 更新算电协同策略配置
// @Tags 算电协同
// @Accept json
// @Produce json
// @Param id path string true "策略ID"
// @Param policy body CoordinationPolicy true "策略配置"
// @Success 200 {object} Response
// @Router /api/v1/coordination/policies/{id} [put]
func (h *CoordinationHandler) UpdatePolicy(c *gin.Context) {
	id := ParseUUID(c.Param("id"))
	if id == uuid.Nil {
		BadRequest(c, "无效的策略ID")
		return
	}

	var policy CoordinationPolicy
	if err := c.ShouldBindJSON(&policy); err != nil {
		BadRequest(c, "无效的请求参数: "+err.Error())
		return
	}

	policy.ID = id

	SuccessWithMessage(c, "策略已更新", policy)
}

// 辅助函数

func getCurrentTimestamp() string {
	return "2024-01-01T10:00:00Z"
}

// 请求/响应结构体

type WorkloadOptimizationRequest struct {
	WorkloadID    uuid.UUID `json:"workload_id" binding:"required"`
	EstimatedPower float64  `json:"estimated_power" binding:"required"`
	Duration      int       `json:"duration" binding:"required"`
	Region        string    `json:"region" binding:"required"`
	OptimizationGoal string  `json:"optimization_goal"`
	Constraints   OptimizationConstraints `json:"constraints"`
}

type OptimizationConstraints struct {
	MaxCost          float64 `json:"max_cost"`
	MaxCarbon        float64 `json:"max_carbon"`
	MinGreenRatio    float64 `json:"min_green_ratio"`
	EarliestStart    string  `json:"earliest_start"`
	LatestEnd        string  `json:"latest_end"`
	PreferredWindows []string `json:"preferred_windows"`
}

type WorkloadOptimizationResult struct {
	WorkloadID       uuid.UUID `json:"workload_id"`
	RecommendedStart string    `json:"recommended_start"`
	RecommendedEnd   string    `json:"recommended_end"`
	ExpectedCost     float64   `json:"expected_cost"`
	ExpectedCarbon   float64   `json:"expected_carbon"`
	GreenRatio       float64   `json:"green_ratio"`
	Confidence       float64   `json:"confidence"`
	Reason           string    `json:"reason"`
}

type WorkloadSchedule struct {
	WorkloadID        uuid.UUID       `json:"workload_id"`
	Schedules         []ScheduleItem  `json:"schedules"`
	TotalEnergyCost   float64         `json:"total_energy_cost"`
	TotalCarbonSaved  float64         `json:"total_carbon_saved"`
	AvgGreenRatio     float64         `json:"avg_green_ratio"`
}

type ScheduleItem struct {
	StartTime    string  `json:"start_time"`
	EndTime      string  `json:"end_time"`
	Power        float64 `json:"power"`
	EnergyCost   float64 `json:"energy_cost"`
	CarbonSaved  float64 `json:"carbon_saved"`
	GreenRatio   float64 `json:"green_ratio"`
}

type CoordinationMetrics struct {
	Region    string              `json:"region"`
	ClusterID uuid.UUID           `json:"cluster_id,omitempty"`
	Period    string              `json:"period"`
	Metrics   EnergyMetricsData   `json:"metrics"`
	Trend     []MetricTrendPoint  `json:"trend"`
}

type EnergyMetricsData struct {
	TotalEnergyConsumed float64 `json:"total_energy_consumed"`
	TotalGreenEnergy    float64 `json:"total_green_energy"`
	GreenRatio          float64 `json:"green_ratio"`
	AvgCarbonIntensity  float64 `json:"avg_carbon_intensity"`
	TotalCarbonEmission float64 `json:"total_carbon_emission"`
	CarbonSaved         float64 `json:"carbon_saved"`
	TotalEnergyCost     float64 `json:"total_energy_cost"`
	CostSaved           float64 `json:"cost_saved"`
	PeakShavingEvents   int     `json:"peak_shaving_events"`
	LoadShiftingEvents  int     `json:"load_shifting_events"`
	VPPDispatchCount    int     `json:"vpp_dispatch_count"`
	OptimizationScore   float64 `json:"optimization_score"`
}

type MetricTrendPoint struct {
	Timestamp string  `json:"timestamp"`
	Value     float64 `json:"value"`
}

type RealtimeStatus struct {
	Region             string              `json:"region"`
	CurrentPower       PowerStatus         `json:"current_power"`
	CurrentPrice       PriceStatus         `json:"current_price"`
	CarbonStatus       CarbonStatus        `json:"carbon_status"`
	OptimizationStatus OptimizationStatus  `json:"optimization_status"`
	Timestamp          string              `json:"timestamp"`
}

type PowerStatus struct {
	TotalDemand      float64 `json:"total_demand"`
	TotalSupply      float64 `json:"total_supply"`
	GreenSupply      float64 `json:"green_supply"`
	GridSupply       float64 `json:"grid_supply"`
	StorageDischarge float64 `json:"storage_discharge"`
	StorageCharge    float64 `json:"storage_charge"`
}

type PriceStatus struct {
	SpotPrice   float64 `json:"spot_price"`
	PeakPrice   float64 `json:"peak_price"`
	ValleyPrice float64 `json:"valley_price"`
	IsPeak      bool    `json:"is_peak"`
}

type CarbonStatus struct {
	CurrentIntensity float64 `json:"current_intensity"`
	AvgIntensity     float64 `json:"avg_intensity"`
	Trend            string  `json:"trend"`
}

type OptimizationStatus struct {
	ActiveWorkloads      int    `json:"active_workloads"`
	OptimizedWorkloads   int    `json:"optimized_workloads"`
	PendingOptimizations int    `json:"pending_optimizations"`
	LastOptimization     string `json:"last_optimization"`
}

type ScheduleSimulationRequest struct {
	EstimatedPower float64 `json:"estimated_power" binding:"required"`
	Duration       int     `json:"duration" binding:"required"`
	Region         string  `json:"region" binding:"required"`
	TimeRange      string  `json:"time_range"`
	Constraints    OptimizationConstraints `json:"constraints"`
}

type ScheduleSimulationResult struct {
	Scenarios      []ScheduleScenario `json:"scenarios"`
	Recommendation string             `json:"recommendation"`
	Reason         string             `json:"reason"`
}

type ScheduleScenario struct {
	Name           string  `json:"name"`
	StartTime      string  `json:"start_time"`
	EndTime        string  `json:"end_time"`
	EnergyCost     float64 `json:"energy_cost"`
	CarbonEmission float64 `json:"carbon_emission"`
	GreenRatio     float64 `json:"green_ratio"`
	Score          float64 `json:"score"`
}

type CoordinationPolicies struct {
	Region   string               `json:"region"`
	Policies []CoordinationPolicy `json:"policies"`
}

type CoordinationPolicy struct {
	ID          uuid.UUID          `json:"id"`
	Name        string             `json:"name"`
	Type        string             `json:"type"`
	Description string             `json:"description"`
	Enabled     bool               `json:"enabled"`
	Priority    int                `json:"priority"`
	Conditions  PolicyConditions   `json:"conditions"`
}

type PolicyConditions struct {
	MaxPriceThreshold  float64  `json:"max_price_threshold"`
	MinGreenRatio      float64  `json:"min_green_ratio"`
	MaxCarbonIntensity float64  `json:"max_carbon_intensity"`
	AllowedTimeRanges  []string `json:"allowed_time_ranges"`
}
