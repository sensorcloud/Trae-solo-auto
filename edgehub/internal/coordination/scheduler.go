package coordination

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"k8s.io/klog/v2"
)

type CoordinationScheduler interface {
	Schedule(ctx context.Context, task *ComputeTask) (*ScheduleDecision, error)
	ScheduleBatch(ctx context.Context, tasks []*ComputeTask) ([]*ScheduleDecision, error)
	Reschedule(ctx context.Context, taskID uuid.UUID, reason string) (*ScheduleDecision, error)
	Cancel(ctx context.Context, taskID uuid.UUID) error
	GetQueueStatus(ctx context.Context) (*QueueStatus, error)
}

type ComputeEnergyScheduler struct {
	config        SchedulerConfig
	optimizer     *MultiObjectiveOptimizer
	predictor     *CoordinationPredictor
	
	taskQueues    map[TaskPriority]*TaskQueue
	resources     map[uuid.UUID]*EnergyResource
	storage       map[uuid.UUID]*StorageResource
	
	scheduledTasks map[uuid.UUID]*ScheduleDecision
	runningTasks   map[uuid.UUID]*ComputeTask
	
	eventCh       chan CoordinationEvent
	mu            sync.RWMutex
	wg            sync.WaitGroup
	stopCh        chan struct{}
}

type TaskQueue struct {
	tasks    []*ComputeTask
	capacity int
	mu       sync.Mutex
}

func NewComputeEnergyScheduler(config SchedulerConfig, optimizer *MultiObjectiveOptimizer, predictor *CoordinationPredictor) *ComputeEnergyScheduler {
	return &ComputeEnergyScheduler{
		config:         config,
		optimizer:      optimizer,
		predictor:      predictor,
		taskQueues:     make(map[TaskPriority]*TaskQueue),
		resources:      make(map[uuid.UUID]*EnergyResource),
		storage:        make(map[uuid.UUID]*StorageResource),
		scheduledTasks: make(map[uuid.UUID]*ScheduleDecision),
		runningTasks:   make(map[uuid.UUID]*ComputeTask),
		eventCh:        make(chan CoordinationEvent, 1000),
		stopCh:         make(chan struct{}),
	}
}

func (s *ComputeEnergyScheduler) Start(ctx context.Context) error {
	klog.Info("启动算电协同调度器...")
	
	s.taskQueues[TaskPriorityHigh] = &TaskQueue{tasks: make([]*ComputeTask, 0), capacity: s.config.QueueSize}
	s.taskQueues[TaskPriorityMedium] = &TaskQueue{tasks: make([]*ComputeTask, 0), capacity: s.config.QueueSize}
	s.taskQueues[TaskPriorityLow] = &TaskQueue{tasks: make([]*ComputeTask, 0), capacity: s.config.QueueSize}
	
	s.wg.Add(1)
	go s.scheduleLoop(ctx)
	
	s.wg.Add(1)
	go s.monitorLoop(ctx)
	
	s.wg.Add(1)
	go s.eventLoop(ctx)
	
	klog.Info("算电协同调度器启动成功")
	return nil
}

func (s *ComputeEnergyScheduler) Stop() {
	close(s.stopCh)
	s.wg.Wait()
	close(s.eventCh)
}

func (s *ComputeEnergyScheduler) Schedule(ctx context.Context, task *ComputeTask) (*ScheduleDecision, error) {
	if task.Status == TaskStatusRunning || task.Status == TaskStatusCompleted {
		return nil, fmt.Errorf("任务状态不允许调度: %s", task.Status)
	}
	
	s.mu.RLock()
	resources := s.getResourcesList()
	storage := s.getStorageList()
	s.mu.RUnlock()
	
	forecast, err := s.predictor.GetCombinedForecast(ctx, "default", 96)
	if err != nil {
		klog.Warningf("获取综合预测失败: %v, 使用默认预测", err)
		forecast = s.getDefaultForecast()
	}
	
	optimizationResult, err := s.optimizer.Optimize(ctx, task, resources, storage, forecast)
	if err != nil {
		return nil, fmt.Errorf("优化失败: %w", err)
	}
	
	decision := &ScheduleDecision{
		TaskID:          task.ID,
		Decision:        "schedule",
		ScheduledStart:  optimizationResult.ScheduledStart,
		ScheduledEnd:    optimizationResult.ScheduledEnd,
		EnergySource:    s.determineEnergySource(optimizationResult),
		EstimatedCost:   optimizationResult.TotalCost,
		EstimatedCarbon: optimizationResult.TotalCarbon,
		Reason:          optimizationResult.Reason,
		Priority:        s.getPriorityValue(task.Priority),
	}
	
	if len(optimizationResult.AssignedResources) > 0 {
		decision.StorageAction = s.createStorageAction(optimizationResult)
	}
	
	s.mu.Lock()
	s.scheduledTasks[task.ID] = decision
	s.mu.Unlock()
	
	task.Status = TaskStatusScheduled
	task.ScheduledStart = &decision.ScheduledStart
	task.ScheduledEnd = &decision.ScheduledEnd
	task.EnergyCost = decision.EstimatedCost
	task.CarbonEmission = decision.EstimatedCarbon
	task.OptimizationScore = optimizationResult.Score
	
	s.emitEvent(CoordinationEvent{
		Type:      "task_scheduled",
		TaskID:    &task.ID,
		Severity:  "info",
		Message:   fmt.Sprintf("任务 %s 已调度到 %s", task.Name, decision.ScheduledStart.Format(time.RFC3339)),
	})
	
	klog.Infof("任务调度成功: %s, 开始时间: %s, 预估成本: %.2f, 预估碳排放: %.2f",
		task.Name, decision.ScheduledStart.Format(time.RFC3339), decision.EstimatedCost, decision.EstimatedCarbon)
	
	return decision, nil
}

func (s *ComputeEnergyScheduler) ScheduleBatch(ctx context.Context, tasks []*ComputeTask) ([]*ScheduleDecision, error) {
	decisions := make([]*ScheduleDecision, len(tasks))
	
	sortedTasks := make([]*ComputeTask, len(tasks))
	copy(sortedTasks, tasks)
	sort.Slice(sortedTasks, func(i, j int) bool {
		if sortedTasks[i].Priority != sortedTasks[j].Priority {
			return s.getPriorityValue(sortedTasks[i].Priority) > s.getPriorityValue(sortedTasks[j].Priority)
		}
		if sortedTasks[i].Type != sortedTasks[j].Type {
			return sortedTasks[i].Type == TaskTypeRealtime
		}
		return sortedTasks[i].CreatedAt.Before(sortedTasks[j].CreatedAt)
	})
	
	for i, task := range sortedTasks {
		decision, err := s.Schedule(ctx, task)
		if err != nil {
			klog.Warningf("批量调度任务 %s 失败: %v", task.Name, err)
			decisions[i] = &ScheduleDecision{
				TaskID:   task.ID,
				Decision: "failed",
				Reason:   err.Error(),
			}
			continue
		}
		decisions[i] = decision
	}
	
	klog.Infof("批量调度完成, 任务数: %d", len(tasks))
	return decisions, nil
}

func (s *ComputeEnergyScheduler) Reschedule(ctx context.Context, taskID uuid.UUID, reason string) (*ScheduleDecision, error) {
	s.mu.Lock()
	decision, exists := s.scheduledTasks[taskID]
	if !exists {
		s.mu.Unlock()
		return nil, fmt.Errorf("任务未找到: %s", taskID)
	}
	delete(s.scheduledTasks, taskID)
	s.mu.Unlock()
	
	var task *ComputeTask
	task = &ComputeTask{
		Status:            TaskStatusPending,
		Priority:          TaskPriorityMedium,
		EstimatedPower:    100,
		EstimatedDuration: 60,
	}
	task.ID = taskID
	
	newDecision, err := s.Schedule(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("重新调度失败: %w", err)
	}
	
	s.emitEvent(CoordinationEvent{
		Type:      "task_rescheduled",
		TaskID:    &taskID,
		Severity:  "warning",
		Message:   fmt.Sprintf("任务重新调度, 原因: %s", reason),
		Details:   map[string]interface{}{"reason": reason, "old_start": decision.ScheduledStart},
	})
	
	klog.Infof("任务重新调度: %s, 原因: %s", taskID, reason)
	return newDecision, nil
}

func (s *ComputeEnergyScheduler) Cancel(ctx context.Context, taskID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if decision, exists := s.scheduledTasks[taskID]; exists {
		delete(s.scheduledTasks, taskID)
		
		s.emitEvent(CoordinationEvent{
			Type:     "task_cancelled",
			TaskID:   &taskID,
			Severity: "info",
			Message:  fmt.Sprintf("任务取消, 原计划开始时间: %s", decision.ScheduledStart.Format(time.RFC3339)),
		})
		
		klog.Infof("任务取消: %s", taskID)
		return nil
	}
	
	if _, exists := s.runningTasks[taskID]; exists {
		delete(s.runningTasks, taskID)
		
		s.emitEvent(CoordinationEvent{
			Type:     "running_task_cancelled",
			TaskID:   &taskID,
			Severity: "warning",
			Message:  "正在运行的任务被取消",
		})
		
		klog.Infof("运行中任务取消: %s", taskID)
		return nil
	}
	
	return fmt.Errorf("任务未找到: %s", taskID)
}

func (s *ComputeEnergyScheduler) GetQueueStatus(ctx context.Context) (*QueueStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	status := &QueueStatus{
		Timestamp: time.Now(),
		Queues:    make(map[TaskPriority]QueueInfo),
	}
	
	for priority, queue := range s.taskQueues {
		queue.mu.Lock()
		status.Queues[priority] = QueueInfo{
			Count:    len(queue.tasks),
			Capacity: queue.capacity,
		}
		queue.mu.Unlock()
	}
	
	status.ScheduledCount = len(s.scheduledTasks)
	status.RunningCount = len(s.runningTasks)
	
	return status, nil
}

func (s *ComputeEnergyScheduler) EnqueueTask(task *ComputeTask) error {
	priority := task.Priority
	if priority == "" {
		priority = TaskPriorityMedium
	}
	
	queue, exists := s.taskQueues[priority]
	if !exists {
		return fmt.Errorf("无效的优先级: %s", priority)
	}
	
	queue.mu.Lock()
	defer queue.mu.Unlock()
	
	if len(queue.tasks) >= queue.capacity {
		return fmt.Errorf("队列已满: %s", priority)
	}
	
	queue.tasks = append(queue.tasks, task)
	
	klog.V(4).Infof("任务入队: %s, 优先级: %s", task.Name, priority)
	return nil
}

func (s *ComputeEnergyScheduler) scheduleLoop(ctx context.Context) {
	defer s.wg.Done()
	
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.processQueues(ctx)
		}
	}
}

func (s *ComputeEnergyScheduler) processQueues(ctx context.Context) {
	priorities := []TaskPriority{TaskPriorityHigh, TaskPriorityMedium, TaskPriorityLow}
	
	for _, priority := range priorities {
		queue, exists := s.taskQueues[priority]
		if !exists {
			continue
		}
		
		queue.mu.Lock()
		tasksToProcess := make([]*ComputeTask, 0)
		remaining := make([]*ComputeTask, 0)
		
		for _, task := range queue.tasks {
			if len(tasksToProcess) < s.config.WorkerCount {
				tasksToProcess = append(tasksToProcess, task)
			} else {
				remaining = append(remaining, task)
			}
		}
		queue.tasks = remaining
		queue.mu.Unlock()
		
		for _, task := range tasksToProcess {
			_, err := s.Schedule(ctx, task)
			if err != nil {
				klog.Warningf("调度任务失败: %s, 错误: %v", task.Name, err)
				
				if s.config.RetryLimit > 0 {
					task.RetryCount++
					if task.RetryCount < s.config.RetryLimit {
						s.EnqueueTask(task)
					}
				}
			}
		}
	}
}

func (s *ComputeEnergyScheduler) monitorLoop(ctx context.Context) {
	defer s.wg.Done()
	
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.checkScheduledTasks(ctx)
		}
	}
}

func (s *ComputeEnergyScheduler) checkScheduledTasks(ctx context.Context) {
	now := time.Now()
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	for taskID, decision := range s.scheduledTasks {
		if decision.ScheduledStart.Before(now) || decision.ScheduledStart.Equal(now) {
			s.startTask(taskID, decision)
			delete(s.scheduledTasks, taskID)
		}
	}
}

func (s *ComputeEnergyScheduler) startTask(taskID uuid.UUID, decision *ScheduleDecision) {
	s.emitEvent(CoordinationEvent{
		Type:     "task_started",
		TaskID:   &taskID,
		Severity: "info",
		Message:  fmt.Sprintf("任务开始执行, 能源来源: %s", decision.EnergySource),
	})
	
	klog.Infof("任务开始执行: %s", taskID)
}

func (s *ComputeEnergyScheduler) eventLoop(ctx context.Context) {
	defer s.wg.Done()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case event := <-s.eventCh:
			s.processEvent(event)
		}
	}
}

func (s *ComputeEnergyScheduler) processEvent(event CoordinationEvent) {
	klog.V(4).Infof("处理事件: %s, 消息: %s", event.Type, event.Message)
}

func (s *ComputeEnergyScheduler) emitEvent(event CoordinationEvent) {
	select {
	case s.eventCh <- event:
	default:
		klog.Warningf("事件通道已满, 丢弃事件: %s", event.Type)
	}
}

func (s *ComputeEnergyScheduler) UpdateResources(resources []EnergyResource) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	for i := range resources {
		s.resources[resources[i].ID] = &resources[i]
	}
	
	klog.V(4).Infof("更新能源资源, 数量: %d", len(resources))
}

func (s *ComputeEnergyScheduler) UpdateStorage(storage []StorageResource) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	for i := range storage {
		s.storage[storage[i].ID] = &storage[i]
	}
	
	klog.V(4).Infof("更新储能资源, 数量: %d", len(storage))
}

func (s *ComputeEnergyScheduler) getResourcesList() []EnergyResource {
	resources := make([]EnergyResource, 0, len(s.resources))
	for _, r := range s.resources {
		resources = append(resources, *r)
	}
	return resources
}

func (s *ComputeEnergyScheduler) getStorageList() []StorageResource {
	storage := make([]StorageResource, 0, len(s.storage))
	for _, s := range s.storage {
		storage = append(storage, *s)
	}
	return storage
}

func (s *ComputeEnergyScheduler) determineEnergySource(result *OptimizationResult) EnergyResourceType {
	if len(result.AssignedResources) == 0 {
		return EnergyResourceGrid
	}
	
	greenPower := 0.0
	totalPower := 0.0
	
	for _, ar := range result.AssignedResources {
		totalPower += ar.Power
		if ar.ResourceType == EnergyResourceSolar || ar.ResourceType == EnergyResourceWind ||
			ar.ResourceType == EnergyResourceHydro || ar.ResourceType == EnergyResourceStorage {
			greenPower += ar.Power
		}
	}
	
	if totalPower > 0 && greenPower/totalPower > 0.7 {
		return EnergyResourceMixed
	}
	
	return result.AssignedResources[0].ResourceType
}

func (s *ComputeEnergyScheduler) createStorageAction(result *OptimizationResult) *StorageAction {
	for _, ar := range result.AssignedResources {
		if ar.ResourceType == EnergyResourceStorage {
			return &StorageAction{
				StorageID: ar.ResourceID,
				Action:    "discharge",
				Power:     ar.Power,
				StartTime: result.ScheduledStart,
				Duration:  ar.Duration,
			}
		}
	}
	return nil
}

func (s *ComputeEnergyScheduler) getPriorityValue(priority TaskPriority) int {
	switch priority {
	case TaskPriorityHigh:
		return 100
	case TaskPriorityMedium:
		return 50
	case TaskPriorityLow:
		return 10
	default:
		return 50
	}
}

func (s *ComputeEnergyScheduler) getDefaultForecast() *CombinedForecast {
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

func (s *ComputeEnergyScheduler) GetTaskSchedule(taskID uuid.UUID) (*ScheduleDecision, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if decision, exists := s.scheduledTasks[taskID]; exists {
		return decision, nil
	}
	
	return nil, fmt.Errorf("任务调度信息未找到: %s", taskID)
}

func (s *ComputeEnergyScheduler) GetScheduledTasks() []*ScheduleDecision {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	tasks := make([]*ScheduleDecision, 0, len(s.scheduledTasks))
	for _, decision := range s.scheduledTasks {
		tasks = append(tasks, decision)
	}
	
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].ScheduledStart.Before(tasks[j].ScheduledStart)
	})
	
	return tasks
}

func (s *ComputeEnergyScheduler) ApplyPolicy(policy CoordinationPolicy) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	klog.Infof("应用协同调度策略: %s", policy.Name)
	return nil
}

type QueueStatus struct {
	Timestamp     time.Time               `json:"timestamp"`
	Queues        map[TaskPriority]QueueInfo `json:"queues"`
	ScheduledCount int                     `json:"scheduled_count"`
	RunningCount  int                     `json:"running_count"`
}

type QueueInfo struct {
	Count    int `json:"count"`
	Capacity int `json:"capacity"`
}

func (s *ComputeEnergyScheduler) GetMetrics(ctx context.Context) (*CoordinationMetrics, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	metrics := &CoordinationMetrics{
		Timestamp:      time.Now(),
		ScheduledTasks: len(s.scheduledTasks),
		RunningTasks:   len(s.runningTasks),
	}
	
	totalTasks := 0
	for _, queue := range s.taskQueues {
		queue.mu.Lock()
		totalTasks += len(queue.tasks)
		queue.mu.Unlock()
	}
	metrics.TotalTasks = totalTasks
	
	totalCost := 0.0
	totalCarbon := 0.0
	totalGreen := 0.0
	count := 0
	
	for _, decision := range s.scheduledTasks {
		totalCost += decision.EstimatedCost
		totalCarbon += decision.EstimatedCarbon
		count++
	}
	
	if count > 0 {
		metrics.AvgEnergyCost = totalCost / float64(count)
		metrics.AvgCarbonEmission = totalCarbon / float64(count)
		metrics.TotalGreenRatio = totalGreen / float64(count)
	}
	
	return metrics, nil
}
