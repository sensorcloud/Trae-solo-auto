package coordination

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

type CoordinationEngine interface {
	Start(ctx context.Context) error
	Stop() error
	SubmitTask(ctx context.Context, task *ComputeTask) (*ScheduleDecision, error)
	SubmitBatchTasks(ctx context.Context, tasks []*ComputeTask) ([]*ScheduleDecision, error)
	GetTaskStatus(ctx context.Context, taskID uuid.UUID) (*ComputeTask, error)
	CancelTask(ctx context.Context, taskID uuid.UUID) error
	GetRealtimeData(ctx context.Context) (*RealtimeData, error)
	GetMetrics(ctx context.Context) (*CoordinationMetrics, error)
	ApplyPolicy(ctx context.Context, policy CoordinationPolicy) error
}

type ComputePowerEnergyCoordinationEngine struct {
	config     CoordinationConfig
	predictor  *CoordinationPredictor
	optimizer  *MultiObjectiveOptimizer
	scheduler  *ComputeEnergyScheduler
	
	tasks      map[uuid.UUID]*ComputeTask
	resources  map[uuid.UUID]*EnergyResource
	storage    map[uuid.UUID]*StorageResource
	policies   map[uuid.UUID]*CoordinationPolicy
	
	metrics    *CoordinationMetrics
	metricsMu  sync.RWMutex
	
	dataStream chan RealtimeData
	eventCh    chan CoordinationEvent
	
	mu         sync.RWMutex
	wg         sync.WaitGroup
	stopCh     chan struct{}
	running    bool
}

func NewComputePowerEnergyCoordinationEngine(config CoordinationConfig) *ComputePowerEnergyCoordinationEngine {
	predictor := NewCoordinationPredictor(config.PredictorConfig)
	optimizer := NewMultiObjectiveOptimizer(config.OptimizerConfig, predictor)
	scheduler := NewComputeEnergyScheduler(config.SchedulerConfig, optimizer, predictor)
	
	return &ComputePowerEnergyCoordinationEngine{
		config:     config,
		predictor:  predictor,
		optimizer:  optimizer,
		scheduler:  scheduler,
		tasks:      make(map[uuid.UUID]*ComputeTask),
		resources:  make(map[uuid.UUID]*EnergyResource),
		storage:    make(map[uuid.UUID]*StorageResource),
		policies:   make(map[uuid.UUID]*CoordinationPolicy),
		metrics:    &CoordinationMetrics{},
		dataStream: make(chan RealtimeData, 1000),
		eventCh:    make(chan CoordinationEvent, 2000),
		stopCh:     make(chan struct{}),
	}
}

func (e *ComputePowerEnergyCoordinationEngine) Start(ctx context.Context) error {
	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return fmt.Errorf("引擎已在运行")
	}
	e.running = true
	e.mu.Unlock()
	
	klog.Info("启动算电协同调度引擎...")
	
	if err := e.predictor.Start(ctx); err != nil {
		return fmt.Errorf("启动预测器失败: %w", err)
	}
	
	if err := e.scheduler.Start(ctx); err != nil {
		return fmt.Errorf("启动调度器失败: %w", err)
	}
	
	e.wg.Add(1)
	go e.dataProcessingLoop(ctx)
	
	e.wg.Add(1)
	go e.optimizationLoop(ctx)
	
	e.wg.Add(1)
	go e.metricsCollectionLoop(ctx)
	
	e.wg.Add(1)
	go e.storageCoordinationLoop(ctx)
	
	e.wg.Add(1)
	go e.policyEnforcementLoop(ctx)
	
	klog.Info("算电协同调度引擎启动成功")
	return nil
}

func (e *ComputePowerEnergyCoordinationEngine) Stop() error {
	e.mu.Lock()
	if !e.running {
		e.mu.Unlock()
		return nil
	}
	e.running = false
	e.mu.Unlock()
	
	klog.Info("停止算电协同调度引擎...")
	
	close(e.stopCh)
	e.wg.Wait()
	
	e.predictor.Stop()
	e.scheduler.Stop()
	
	close(e.dataStream)
	close(e.eventCh)
	
	klog.Info("算电协同调度引擎已停止")
	return nil
}

func (e *ComputePowerEnergyCoordinationEngine) SubmitTask(ctx context.Context, task *ComputeTask) (*ScheduleDecision, error) {
	if task.ID == uuid.Nil {
		task.ID = uuid.New()
	}
	task.CreatedAt = time.Now()
	task.UpdatedAt = time.Now()
	task.Status = TaskStatusPending
	
	switch task.Type {
	case TaskTypeRealtime:
		return e.handleRealtimeTask(ctx, task)
	case TaskTypeDelayable:
		return e.handleDelayableTask(ctx, task)
	case TaskTypeBatch:
		return e.handleBatchTask(ctx, task)
	case TaskTypeInterruptible:
		return e.handleInterruptibleTask(ctx, task)
	default:
		return e.handleDelayableTask(ctx, task)
	}
}

func (e *ComputePowerEnergyCoordinationEngine) handleRealtimeTask(ctx context.Context, task *ComputeTask) (*ScheduleDecision, error) {
	klog.V(4).Infof("处理实时任务: %s", task.Name)
	
	task.Priority = TaskPriorityHigh
	
	e.mu.RLock()
	resources := e.getAvailableResources()
	_ = e.getAvailableStorage()
	e.mu.RUnlock()
	
	localResources := e.filterLocalResources(resources)
	if len(localResources) == 0 {
		return nil, fmt.Errorf("没有可用的本地资源")
	}
	
	decision := &ScheduleDecision{
		TaskID:         task.ID,
		Decision:       "schedule",
		ScheduledStart: time.Now(),
		ScheduledEnd:   time.Now().Add(time.Duration(task.EstimatedDuration) * time.Minute),
		EnergySource:   EnergyResourceMixed,
		Priority:       100,
		Reason:         "实时任务优先调度到本地算力",
	}
	
	bestResource := e.selectBestLocalResource(localResources)
	decision.EstimatedCost = task.EstimatedPower * float64(task.EstimatedDuration) / 60 * bestResource.PricePerKWh
	decision.EstimatedCarbon = task.EstimatedPower * float64(task.EstimatedDuration) / 60 * bestResource.CarbonIntensity
	
	e.mu.Lock()
	e.tasks[task.ID] = task
	e.mu.Unlock()
	
	decision, err := e.scheduler.Schedule(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("调度实时任务失败: %w", err)
	}
	
	e.emitEvent(CoordinationEvent{
		Type:      "realtime_task_scheduled",
		TaskID:    &task.ID,
		Severity:  "info",
		Message:   fmt.Sprintf("实时任务 %s 已调度", task.Name),
	})
	
	return decision, nil
}

func (e *ComputePowerEnergyCoordinationEngine) handleDelayableTask(ctx context.Context, task *ComputeTask) (*ScheduleDecision, error) {
	klog.V(4).Infof("处理可延迟任务: %s", task.Name)
	
	if task.Priority == "" {
		task.Priority = TaskPriorityMedium
	}
	
	forecast, err := e.predictor.GetCombinedForecast(ctx, "default", 96)
	if err != nil {
		klog.Warningf("获取预测失败: %v, 使用默认预测", err)
		forecast = e.getDefaultForecast()
	}
	
	optimalSlot := e.findOptimalTimeSlot(task, forecast)
	if optimalSlot == nil {
		return nil, fmt.Errorf("未找到最优调度时段")
	}
	
	task.ScheduledStart = &optimalSlot.Start
	task.ScheduledEnd = &optimalSlot.End
	
	e.mu.Lock()
	e.tasks[task.ID] = task
	e.mu.Unlock()
	
	decision, err := e.scheduler.Schedule(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("调度可延迟任务失败: %w", err)
	}
	
	e.emitEvent(CoordinationEvent{
		Type:      "delayable_task_scheduled",
		TaskID:    &task.ID,
		Severity:  "info",
		Message:   fmt.Sprintf("可延迟任务 %s 已调度到 %s", task.Name, optimalSlot.Start.Format(time.RFC3339)),
	})
	
	return decision, nil
}

func (e *ComputePowerEnergyCoordinationEngine) handleBatchTask(ctx context.Context, task *ComputeTask) (*ScheduleDecision, error) {
	klog.V(4).Infof("处理批处理任务: %s", task.Name)
	
	if task.Priority == "" {
		task.Priority = TaskPriorityLow
	}
	
	forecast, err := e.predictor.GetCombinedForecast(ctx, "default", 96)
	if err != nil {
		klog.Warningf("获取预测失败: %v, 使用默认预测", err)
		forecast = e.getDefaultForecast()
	}
	
	valleySlots := e.findValleyTimeSlots(forecast)
	if len(valleySlots) == 0 {
		return nil, fmt.Errorf("未找到低价时段")
	}
	
	bestSlot := valleySlots[0]
	task.ScheduledStart = &bestSlot.Start
	task.ScheduledEnd = &bestSlot.End
	
	e.mu.Lock()
	e.tasks[task.ID] = task
	e.mu.Unlock()
	
	decision, err := e.scheduler.Schedule(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("调度批处理任务失败: %w", err)
	}
	
	e.emitEvent(CoordinationEvent{
		Type:      "batch_task_scheduled",
		TaskID:    &task.ID,
		Severity:  "info",
		Message:   fmt.Sprintf("批处理任务 %s 已调度到低价时段 %s", task.Name, bestSlot.Start.Format(time.RFC3339)),
	})
	
	return decision, nil
}

func (e *ComputePowerEnergyCoordinationEngine) handleInterruptibleTask(ctx context.Context, task *ComputeTask) (*ScheduleDecision, error) {
	klog.V(4).Infof("处理可中断任务: %s", task.Name)
	
	if task.Priority == "" {
		task.Priority = TaskPriorityLow
	}
	
	task.TimeConstraint.Interruptible = true
	
	decision, err := e.handleDelayableTask(ctx, task)
	if err != nil {
		return nil, err
	}
	
	decision.Reason = "可中断任务已调度，可在高电价时段暂停"
	
	return decision, nil
}

func (e *ComputePowerEnergyCoordinationEngine) SubmitBatchTasks(ctx context.Context, tasks []*ComputeTask) ([]*ScheduleDecision, error) {
	decisions := make([]*ScheduleDecision, len(tasks))
	
	for i, task := range tasks {
		decision, err := e.SubmitTask(ctx, task)
		if err != nil {
			klog.Warningf("批量提交任务失败: %s, 错误: %v", task.Name, err)
			decisions[i] = &ScheduleDecision{
				TaskID:   task.ID,
				Decision: "failed",
				Reason:   err.Error(),
			}
			continue
		}
		decisions[i] = decision
	}
	
	klog.Infof("批量提交任务完成, 总数: %d", len(tasks))
	return decisions, nil
}

func (e *ComputePowerEnergyCoordinationEngine) GetTaskStatus(ctx context.Context, taskID uuid.UUID) (*ComputeTask, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	task, exists := e.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("任务未找到: %s", taskID)
	}
	
	return task, nil
}

func (e *ComputePowerEnergyCoordinationEngine) CancelTask(ctx context.Context, taskID uuid.UUID) error {
	e.mu.Lock()
	task, exists := e.tasks[taskID]
	if !exists {
		e.mu.Unlock()
		return fmt.Errorf("任务未找到: %s", taskID)
	}
	
	task.Status = TaskStatusCancelled
	now := time.Now()
	task.ActualEnd = &now
	e.mu.Unlock()
	
	if err := e.scheduler.Cancel(ctx, taskID); err != nil {
		klog.Warningf("取消调度失败: %v", err)
	}
	
	e.emitEvent(CoordinationEvent{
		Type:     "task_cancelled",
		TaskID:   &taskID,
		Severity: "info",
		Message:  fmt.Sprintf("任务 %s 已取消", task.Name),
	})
	
	klog.Infof("任务取消: %s", taskID)
	return nil
}

func (e *ComputePowerEnergyCoordinationEngine) GetRealtimeData(ctx context.Context) (*RealtimeData, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	data := &RealtimeData{
		Timestamp: time.Now(),
	}
	
	taskQueue := make([]ComputeTask, 0)
	for _, task := range e.tasks {
		if task.Status == TaskStatusPending || task.Status == TaskStatusScheduled {
			taskQueue = append(taskQueue, *task)
		}
	}
	data.TaskQueue = taskQueue
	
	availableEnergy := make([]EnergyResource, 0)
	for _, res := range e.resources {
		if res.Status == EnergyResourceStatusOnline {
			availableEnergy = append(availableEnergy, *res)
		}
	}
	data.AvailableEnergy = availableEnergy
	
	storageStatus := make([]StorageResource, 0)
	for _, s := range e.storage {
		storageStatus = append(storageStatus, *s)
	}
	data.StorageStatus = storageStatus
	
	if priceForecast := e.predictor.GetPriceForecast("default"); priceForecast != nil && len(priceForecast.Points) > 0 {
		data.CurrentPrice = priceForecast.Points[0].Value
	}
	
	if carbonForecast := e.predictor.GetCarbonForecast("default"); carbonForecast != nil && len(carbonForecast.Points) > 0 {
		data.CarbonIntensity = carbonForecast.Points[0].Value
	}
	
	data.GreenRatio = e.calculateOverallGreenRatio()
	
	return data, nil
}

func (e *ComputePowerEnergyCoordinationEngine) GetMetrics(ctx context.Context) (*CoordinationMetrics, error) {
	e.metricsMu.RLock()
	defer e.metricsMu.RUnlock()
	
	metrics := *e.metrics
	metrics.Timestamp = time.Now()
	
	e.mu.RLock()
	metrics.TotalTasks = len(e.tasks)
	scheduled := 0
	running := 0
	completed := 0
	for _, task := range e.tasks {
		switch task.Status {
		case TaskStatusScheduled:
			scheduled++
		case TaskStatusRunning:
			running++
		case TaskStatusCompleted:
			completed++
		}
	}
	e.mu.RUnlock()
	
	metrics.ScheduledTasks = scheduled
	metrics.RunningTasks = running
	metrics.CompletedTasks = completed
	
	schedulerMetrics, err := e.scheduler.GetMetrics(ctx)
	if err == nil {
		metrics.AvgEnergyCost = schedulerMetrics.AvgEnergyCost
		metrics.AvgCarbonEmission = schedulerMetrics.AvgCarbonEmission
		metrics.TotalGreenRatio = schedulerMetrics.TotalGreenRatio
	}
	
	return &metrics, nil
}

func (e *ComputePowerEnergyCoordinationEngine) ApplyPolicy(ctx context.Context, policy CoordinationPolicy) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	if policy.ID == uuid.Nil {
		policy.ID = uuid.New()
	}
	policy.CreatedAt = time.Now()
	policy.UpdatedAt = time.Now()
	
	e.policies[policy.ID] = &policy
	
	if err := e.scheduler.ApplyPolicy(policy); err != nil {
		return fmt.Errorf("应用策略失败: %w", err)
	}
	
	klog.Infof("应用协同策略: %s", policy.Name)
	return nil
}

func (e *ComputePowerEnergyCoordinationEngine) RegisterEnergyResource(resource *EnergyResource) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	if resource.ID == uuid.Nil {
		resource.ID = uuid.New()
	}
	resource.CreatedAt = time.Now()
	resource.UpdatedAt = time.Now()
	
	e.resources[resource.ID] = resource
	
	e.scheduler.UpdateResources(e.getResourcesList())
	
	klog.Infof("注册能源资源: %s, 类型: %s, 容量: %.2f", resource.Name, resource.Type, resource.Capacity)
	return nil
}

func (e *ComputePowerEnergyCoordinationEngine) RegisterStorageResource(storage *StorageResource) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	if storage.ID == uuid.Nil {
		storage.ID = uuid.New()
	}
	storage.CreatedAt = time.Now()
	storage.UpdatedAt = time.Now()
	
	e.storage[storage.ID] = storage
	
	e.scheduler.UpdateStorage(e.getStorageList())
	
	klog.Infof("注册储能资源: %s, 容量: %.2f kWh", storage.Name, storage.Capacity)
	return nil
}

func (e *ComputePowerEnergyCoordinationEngine) UpdateResourceStatus(resourceID uuid.UUID, status EnergyResourceStatus, output float64) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	resource, exists := e.resources[resourceID]
	if !exists {
		return fmt.Errorf("资源未找到: %s", resourceID)
	}
	
	resource.Status = status
	resource.CurrentOutput = output
	resource.AvailableCapacity = math.Max(0, resource.Capacity-output)
	resource.UpdatedAt = time.Now()
	
	e.scheduler.UpdateResources(e.getResourcesList())
	
	klog.V(4).Infof("更新资源状态: %s, 状态: %s, 输出: %.2f", resourceID, status, output)
	return nil
}

func (e *ComputePowerEnergyCoordinationEngine) UpdateStorageStatus(storageID uuid.UUID, soc float64, power float64) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	s, exists := e.storage[storageID]
	if !exists {
		return fmt.Errorf("储能设备未找到: %s", storageID)
	}
	
	s.SOC = soc
	s.CurrentPower = power
	s.AvailableEnergy = soc * s.Capacity / 100
	
	if power > 0 {
		s.Status = "charging"
	} else if power < 0 {
		s.Status = "discharging"
	} else {
		s.Status = "idle"
	}
	
	e.scheduler.UpdateStorage(e.getStorageList())
	
	klog.V(4).Infof("更新储能状态: %s, SOC: %.2f%%, 功率: %.2f", storageID, soc, power)
	return nil
}

func (e *ComputePowerEnergyCoordinationEngine) dataProcessingLoop(ctx context.Context) {
	defer e.wg.Done()
	
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-e.stopCh:
			return
		case <-ticker.C:
			e.processRealtimeData(ctx)
		}
	}
}

func (e *ComputePowerEnergyCoordinationEngine) processRealtimeData(ctx context.Context) {
	data, err := e.GetRealtimeData(ctx)
	if err != nil {
		klog.Warningf("获取实时数据失败: %v", err)
		return
	}
	
	select {
	case e.dataStream <- *data:
	default:
		klog.V(4).Info("数据流通道已满，跳过本次数据")
	}
}

func (e *ComputePowerEnergyCoordinationEngine) optimizationLoop(ctx context.Context) {
	defer e.wg.Done()
	
	ticker := time.NewTicker(time.Duration(e.config.OptimizationHorizon) * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-e.stopCh:
			return
		case <-ticker.C:
			e.runPeriodicOptimization(ctx)
		}
	}
}

func (e *ComputePowerEnergyCoordinationEngine) runPeriodicOptimization(ctx context.Context) {
	klog.V(4).Info("运行周期性优化...")
	
	e.mu.RLock()
	pendingTasks := make([]*ComputeTask, 0)
	for _, task := range e.tasks {
		if task.Status == TaskStatusPending || task.Status == TaskStatusScheduled {
			pendingTasks = append(pendingTasks, task)
		}
	}
	resources := e.getResourcesList()
	storage := e.getStorageList()
	e.mu.RUnlock()
	
	if len(pendingTasks) == 0 {
		return
	}
	
	forecast, err := e.predictor.GetCombinedForecast(ctx, "default", 96)
	if err != nil {
		klog.Warningf("获取预测失败: %v", err)
		return
	}
	
	results, err := e.optimizer.OptimizeBatch(ctx, pendingTasks, resources, storage, forecast)
	if err != nil {
		klog.Warningf("批量优化失败: %v", err)
		return
	}
	
	for i, result := range results {
		if result != nil && i < len(pendingTasks) {
			task := pendingTasks[i]
			task.OptimizationScore = result.Score
			task.EnergyCost = result.TotalCost
			task.CarbonEmission = result.TotalCarbon
			task.GreenRatio = result.GreenRatio
		}
	}
	
	klog.V(4).Infof("周期性优化完成, 优化任务数: %d", len(results))
}

func (e *ComputePowerEnergyCoordinationEngine) metricsCollectionLoop(ctx context.Context) {
	defer e.wg.Done()
	
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-e.stopCh:
			return
		case <-ticker.C:
			e.collectMetrics(ctx)
		}
	}
}

func (e *ComputePowerEnergyCoordinationEngine) collectMetrics(ctx context.Context) {
	metrics, err := e.GetMetrics(ctx)
	if err != nil {
		klog.Warningf("收集指标失败: %v", err)
		return
	}
	
	e.metricsMu.Lock()
	e.metrics = metrics
	e.metricsMu.Unlock()
}

func (e *ComputePowerEnergyCoordinationEngine) storageCoordinationLoop(ctx context.Context) {
	defer e.wg.Done()
	
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-e.stopCh:
			return
		case <-ticker.C:
			e.coordinateStorage(ctx)
		}
	}
}

func (e *ComputePowerEnergyCoordinationEngine) coordinateStorage(ctx context.Context) {
	klog.V(4).Info("执行储能协同优化...")
	
	e.mu.RLock()
	storage := e.getStorageList()
	tasks := make([]*ComputeTask, 0)
	for _, task := range e.tasks {
		if task.Status == TaskStatusPending || task.Status == TaskStatusScheduled {
			tasks = append(tasks, task)
		}
	}
	e.mu.RUnlock()
	
	forecast, err := e.predictor.GetCombinedForecast(ctx, "default", 96)
	if err != nil {
		klog.Warningf("获取预测失败: %v", err)
		return
	}
	
	plan, err := e.optimizer.OptimizeStorageUsage(ctx, storage, forecast, tasks)
	if err != nil {
		klog.Warningf("储能优化失败: %v", err)
		return
	}
	
	for _, action := range plan.Actions {
		e.executeStorageAction(action)
	}
	
	e.metricsMu.Lock()
	e.metrics.StorageUtilization = e.calculateStorageUtilization()
	e.metrics.StorageCycles = e.calculateDailyStorageCycles()
	e.metrics.CostSavings += plan.ExpectedSavings
	e.metricsMu.Unlock()
	
	klog.V(4).Infof("储能协同优化完成, 动作数: %d, 预期节省: %.2f", len(plan.Actions), plan.ExpectedSavings)
}

func (e *ComputePowerEnergyCoordinationEngine) executeStorageAction(action StorageActionPlan) {
	klog.V(4).Infof("执行储能动作: %s, 设备: %s, 功率: %.2f", action.Action, action.StorageID, action.Power)
}

func (e *ComputePowerEnergyCoordinationEngine) policyEnforcementLoop(ctx context.Context) {
	defer e.wg.Done()
	
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-e.stopCh:
			return
		case <-ticker.C:
			e.enforcePolicies(ctx)
		}
	}
}

func (e *ComputePowerEnergyCoordinationEngine) enforcePolicies(ctx context.Context) {
	e.mu.RLock()
	policies := make([]*CoordinationPolicy, 0, len(e.policies))
	for _, policy := range e.policies {
		if policy.Enabled {
			policies = append(policies, policy)
		}
	}
	e.mu.RUnlock()
	
	for _, policy := range policies {
		e.enforcePolicy(ctx, policy)
	}
}

func (e *ComputePowerEnergyCoordinationEngine) enforcePolicy(ctx context.Context, policy *CoordinationPolicy) {
	if policy.PriceOptimization.Enabled {
		e.enforcePriceOptimization(ctx, &policy.PriceOptimization)
	}
	
	if policy.CarbonOptimization.Enabled {
		e.enforceCarbonOptimization(ctx, &policy.CarbonOptimization)
	}
}

func (e *ComputePowerEnergyCoordinationEngine) enforcePriceOptimization(ctx context.Context, policy *PriceOptimizationPolicy) {
	priceForecast := e.predictor.GetPriceForecast("default")
	if priceForecast == nil || len(priceForecast.Points) == 0 {
		return
	}
	
	currentPrice := priceForecast.Points[0].Value
	
	if currentPrice > policy.PriceThresholdHigh {
		e.emitEvent(CoordinationEvent{
			Type:     "high_price_alert",
			Severity: "warning",
			Message:  fmt.Sprintf("当前电价 %.2f 超过高阈值 %.2f", currentPrice, policy.PriceThresholdHigh),
		})
	}
}

func (e *ComputePowerEnergyCoordinationEngine) enforceCarbonOptimization(ctx context.Context, policy *CarbonOptimizationPolicy) {
	carbonForecast := e.predictor.GetCarbonForecast("default")
	if carbonForecast == nil || len(carbonForecast.Points) == 0 {
		return
	}
	
	currentCarbon := carbonForecast.Points[0].Value
	
	if currentCarbon > policy.MaxCarbonIntensity {
		e.emitEvent(CoordinationEvent{
			Type:     "high_carbon_alert",
			Severity: "warning",
			Message:  fmt.Sprintf("当前碳排放强度 %.2f 超过阈值 %.2f", currentCarbon, policy.MaxCarbonIntensity),
		})
	}
}

func (e *ComputePowerEnergyCoordinationEngine) emitEvent(event CoordinationEvent) {
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	event.CreatedAt = time.Now()
	
	select {
	case e.eventCh <- event:
	default:
		klog.Warningf("事件通道已满，丢弃事件: %s", event.Type)
	}
}

func (e *ComputePowerEnergyCoordinationEngine) getAvailableResources() []EnergyResource {
	resources := make([]EnergyResource, 0)
	for _, res := range e.resources {
		if res.Status == EnergyResourceStatusOnline {
			resources = append(resources, *res)
		}
	}
	return resources
}

func (e *ComputePowerEnergyCoordinationEngine) getAvailableStorage() []StorageResource {
	storage := make([]StorageResource, 0)
	for _, s := range e.storage {
		if s.Status != "fault" && s.Status != "maintenance" {
			storage = append(storage, *s)
		}
	}
	return storage
}

func (e *ComputePowerEnergyCoordinationEngine) getResourcesList() []EnergyResource {
	resources := make([]EnergyResource, 0, len(e.resources))
	for _, r := range e.resources {
		resources = append(resources, *r)
	}
	return resources
}

func (e *ComputePowerEnergyCoordinationEngine) getStorageList() []StorageResource {
	storage := make([]StorageResource, 0, len(e.storage))
	for _, s := range e.storage {
		storage = append(storage, *s)
	}
	return storage
}

func (e *ComputePowerEnergyCoordinationEngine) filterLocalResources(resources []EnergyResource) []EnergyResource {
	local := make([]EnergyResource, 0)
	for _, res := range resources {
		if res.NodeID != nil {
			local = append(local, res)
		}
	}
	return local
}

func (e *ComputePowerEnergyCoordinationEngine) selectBestLocalResource(resources []EnergyResource) *EnergyResource {
	if len(resources) == 0 {
		return nil
	}
	
	best := &resources[0]
	for i := range resources {
		if resources[i].GreenRatio > best.GreenRatio {
			best = &resources[i]
		}
	}
	return best
}

func (e *ComputePowerEnergyCoordinationEngine) findOptimalTimeSlot(task *ComputeTask, forecast *CombinedForecast) *TimeSlot {
	if forecast == nil || len(forecast.Points) == 0 {
		return nil
	}
	
	duration := task.EstimatedDuration
	if duration == 0 {
		duration = 60
	}
	
	bestSlot := &TimeSlot{
		Start:    time.Now(),
		End:      time.Now().Add(time.Duration(duration) * time.Minute),
		Duration: duration,
	}
	bestScore := math.Inf(1)
	
	for i := 0; i <= len(forecast.Points)-duration/15; i++ {
		slotStart := forecast.Points[i].Timestamp
		slotEnd := slotStart.Add(time.Duration(duration) * time.Minute)
		
		avgPrice := 0.0
		avgCarbon := 0.0
		avgGreen := 0.0
		
		for j := i; j < i+duration/15 && j < len(forecast.Points); j++ {
			avgPrice += forecast.Points[j].Price
			avgCarbon += forecast.Points[j].CarbonIntensity
		}
		
		count := float64(duration / 15)
		avgPrice /= count
		avgCarbon /= count
		
		if task.EnergyPreference.MaxPricePerKWh > 0 && avgPrice > task.EnergyPreference.MaxPricePerKWh {
			continue
		}
		
		score := avgPrice*0.4 + avgCarbon*0.3 - avgGreen*0.3
		
		if score < bestScore {
			bestScore = score
			bestSlot = &TimeSlot{
				Start:    slotStart,
				End:      slotEnd,
				Duration: duration,
				AvgPrice: avgPrice,
				AvgCarbon: avgCarbon,
			}
		}
	}
	
	return bestSlot
}

func (e *ComputePowerEnergyCoordinationEngine) findValleyTimeSlots(forecast *CombinedForecast) []TimeSlot {
	slots := make([]TimeSlot, 0)
	
	if forecast == nil || len(forecast.Points) == 0 {
		return slots
	}
	
	valleyThreshold := 0.4
	
	for i := 0; i < len(forecast.Points); i++ {
		if forecast.Points[i].Price <= valleyThreshold {
			start := forecast.Points[i].Timestamp
			end := start
			
			for i < len(forecast.Points) && forecast.Points[i].Price <= valleyThreshold {
				end = forecast.Points[i].Timestamp.Add(15 * time.Minute)
				i++
			}
			
			slots = append(slots, TimeSlot{
				Start:    start,
				End:      end,
				Duration: int(end.Sub(start).Minutes()),
				AvgPrice: valleyThreshold,
			})
		}
	}
	
	sort.Slice(slots, func(i, j int) bool {
		return slots[i].AvgPrice < slots[j].AvgPrice
	})
	
	return slots
}

func (e *ComputePowerEnergyCoordinationEngine) calculateOverallGreenRatio() float64 {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	totalPower := 0.0
	greenPower := 0.0
	
	for _, res := range e.resources {
		if res.Status == EnergyResourceStatusOnline {
			totalPower += res.CurrentOutput
			if res.Type == EnergyResourceSolar || res.Type == EnergyResourceWind ||
				res.Type == EnergyResourceHydro {
				greenPower += res.CurrentOutput
			}
		}
	}
	
	if totalPower == 0 {
		return 0
	}
	
	return greenPower / totalPower
}

func (e *ComputePowerEnergyCoordinationEngine) calculateStorageUtilization() float64 {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	totalCapacity := 0.0
	usedCapacity := 0.0
	
	for _, s := range e.storage {
		totalCapacity += s.Capacity
		usedCapacity += s.Capacity * s.SOC / 100
	}
	
	if totalCapacity == 0 {
		return 0
	}
	
	return usedCapacity / totalCapacity
}

func (e *ComputePowerEnergyCoordinationEngine) calculateDailyStorageCycles() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	totalCycles := 0
	for _, s := range e.storage {
		totalCycles += s.CycleCount
	}
	
	return totalCycles
}

func (e *ComputePowerEnergyCoordinationEngine) getDefaultForecast() *CombinedForecast {
	now := time.Now()
	forecast := &CombinedForecast{
		Region:      "default",
		GeneratedAt: now,
		Horizon:     96,
		Points:      make([]CombinedForecastPoint, 96),
	}
	
	for i := 0; i < 96; i++ {
		t := now.Add(time.Duration(i*15) * time.Minute)
		hour := t.Hour()
		
		var price, load, carbon float64
		switch {
		case hour >= 8 && hour < 12:
			price = 0.75
			load = 1500
			carbon = 0.6
		case hour >= 18 && hour < 22:
			price = 0.85
			load = 1800
			carbon = 0.7
		case hour >= 0 && hour < 6:
			price = 0.30
			load = 600
			carbon = 0.4
		default:
			price = 0.50
			load = 1000
			carbon = 0.5
		}
		
		forecast.Points[i] = CombinedForecastPoint{
			Timestamp:       t,
			Load:            load,
			Price:           price,
			CarbonIntensity: carbon,
			Confidence:      0.8,
		}
	}
	
	return forecast
}

func (e *ComputePowerEnergyCoordinationEngine) GetDataStream() <-chan RealtimeData {
	return e.dataStream
}

func (e *ComputePowerEnergyCoordinationEngine) GetEventStream() <-chan CoordinationEvent {
	return e.eventCh
}

func (e *ComputePowerEnergyCoordinationEngine) GetResource(resourceID uuid.UUID) (*EnergyResource, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	resource, exists := e.resources[resourceID]
	if !exists {
		return nil, fmt.Errorf("资源未找到: %s", resourceID)
	}
	
	return resource, nil
}

func (e *ComputePowerEnergyCoordinationEngine) GetStorage(storageID uuid.UUID) (*StorageResource, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	storage, exists := e.storage[storageID]
	if !exists {
		return nil, fmt.Errorf("储能设备未找到: %s", storageID)
	}
	
	return storage, nil
}

func (e *ComputePowerEnergyCoordinationEngine) ListTasks(filter *TaskFilter) ([]*ComputeTask, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	tasks := make([]*ComputeTask, 0)
	for _, task := range e.tasks {
		if filter == nil || e.matchesFilter(task, filter) {
			tasks = append(tasks, task)
		}
	}
	
	return tasks, nil
}

func (e *ComputePowerEnergyCoordinationEngine) matchesFilter(task *ComputeTask, filter *TaskFilter) bool {
	if filter.Status != "" && task.Status != filter.Status {
		return false
	}
	if filter.Type != "" && task.Type != filter.Type {
		return false
	}
	if filter.Priority != "" && task.Priority != filter.Priority {
		return false
	}
	if filter.ClusterID != nil && task.ClusterID != nil && *task.ClusterID != *filter.ClusterID {
		return false
	}
	return true
}

type TaskFilter struct {
	Status    TaskStatus
	Type      TaskType
	Priority  TaskPriority
	ClusterID *uuid.UUID
}
