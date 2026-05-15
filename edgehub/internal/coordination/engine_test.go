package coordination

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

type mockCoordinationRepository struct {
	tasks          map[uuid.UUID]*ComputeTask
	resources      map[uuid.UUID]*EnergyResource
	storage        map[uuid.UUID]*StorageResource
	policies       map[uuid.UUID]*CoordinationPolicy
	events         map[uuid.UUID]*CoordinationEvent
	predictions    map[string]*PredictionResult
	schedules      map[uuid.UUID]*ScheduleDecision
}

func newMockCoordinationRepository() *mockCoordinationRepository {
	return &mockCoordinationRepository{
		tasks:       make(map[uuid.UUID]*ComputeTask),
		resources:   make(map[uuid.UUID]*EnergyResource),
		storage:     make(map[uuid.UUID]*StorageResource),
		policies:    make(map[uuid.UUID]*CoordinationPolicy),
		events:      make(map[uuid.UUID]*CoordinationEvent),
		predictions: make(map[string]*PredictionResult),
		schedules:   make(map[uuid.UUID]*ScheduleDecision),
	}
}

func (m *mockCoordinationRepository) CreateTask(ctx context.Context, task *ComputeTask) error {
	m.tasks[task.ID] = task
	return nil
}

func (m *mockCoordinationRepository) GetTask(ctx context.Context, id uuid.UUID) (*ComputeTask, error) {
	if task, ok := m.tasks[id]; ok {
		return task, nil
	}
	return nil, fmt.Errorf("task not found")
}

func (m *mockCoordinationRepository) ListTasks(ctx context.Context, filter *TaskFilter) ([]*ComputeTask, error) {
	result := make([]*ComputeTask, 0)
	for _, task := range m.tasks {
		if filter != nil {
			if filter.Status != "" && task.Status != filter.Status {
				continue
			}
			if filter.Type != "" && task.Type != filter.Type {
				continue
			}
		}
		result = append(result, task)
	}
	return result, nil
}

func (m *mockCoordinationRepository) UpdateTask(ctx context.Context, task *ComputeTask) error {
	m.tasks[task.ID] = task
	return nil
}

func (m *mockCoordinationRepository) DeleteTask(ctx context.Context, id uuid.UUID) error {
	delete(m.tasks, id)
	return nil
}

func (m *mockCoordinationRepository) CreateEnergyResource(ctx context.Context, resource *EnergyResource) error {
	m.resources[resource.ID] = resource
	return nil
}

func (m *mockCoordinationRepository) GetEnergyResource(ctx context.Context, id uuid.UUID) (*EnergyResource, error) {
	if resource, ok := m.resources[id]; ok {
		return resource, nil
	}
	return nil, fmt.Errorf("energy resource not found")
}

func (m *mockCoordinationRepository) ListEnergyResources(ctx context.Context, filter *EnergyResourceFilter) ([]*EnergyResource, error) {
	result := make([]*EnergyResource, 0)
	for _, resource := range m.resources {
		result = append(result, resource)
	}
	return result, nil
}

func (m *mockCoordinationRepository) UpdateEnergyResource(ctx context.Context, resource *EnergyResource) error {
	m.resources[resource.ID] = resource
	return nil
}

func (m *mockCoordinationRepository) CreateStorageResource(ctx context.Context, storage *StorageResource) error {
	m.storage[storage.ID] = storage
	return nil
}

func (m *mockCoordinationRepository) GetStorageResource(ctx context.Context, id uuid.UUID) (*StorageResource, error) {
	if s, ok := m.storage[id]; ok {
		return s, nil
	}
	return nil, fmt.Errorf("storage resource not found")
}

func (m *mockCoordinationRepository) ListStorageResources(ctx context.Context, filter *StorageResourceFilter) ([]*StorageResource, error) {
	result := make([]*StorageResource, 0)
	for _, s := range m.storage {
		result = append(result, s)
	}
	return result, nil
}

func (m *mockCoordinationRepository) UpdateStorageResource(ctx context.Context, storage *StorageResource) error {
	m.storage[storage.ID] = storage
	return nil
}

func (m *mockCoordinationRepository) CreatePolicy(ctx context.Context, policy *CoordinationPolicy) error {
	m.policies[policy.ID] = policy
	return nil
}

func (m *mockCoordinationRepository) GetPolicy(ctx context.Context, id uuid.UUID) (*CoordinationPolicy, error) {
	if policy, ok := m.policies[id]; ok {
		return policy, nil
	}
	return nil, fmt.Errorf("policy not found")
}

func (m *mockCoordinationRepository) ListPolicies(ctx context.Context, filter *PolicyFilter) ([]*CoordinationPolicy, error) {
	result := make([]*CoordinationPolicy, 0)
	for _, policy := range m.policies {
		result = append(result, policy)
	}
	return result, nil
}

func (m *mockCoordinationRepository) UpdatePolicy(ctx context.Context, policy *CoordinationPolicy) error {
	m.policies[policy.ID] = policy
	return nil
}

func (m *mockCoordinationRepository) DeletePolicy(ctx context.Context, id uuid.UUID) error {
	delete(m.policies, id)
	return nil
}

func (m *mockCoordinationRepository) CreateEvent(ctx context.Context, event *CoordinationEvent) error {
	m.events[event.ID] = event
	return nil
}

func (m *mockCoordinationRepository) ListEvents(ctx context.Context, filter *EventFilter) ([]*CoordinationEvent, error) {
	result := make([]*CoordinationEvent, 0)
	for _, event := range m.events {
		result = append(result, event)
	}
	return result, nil
}

func (m *mockCoordinationRepository) GetPrediction(ctx context.Context, req *PredictionRequest) (*PredictionResult, error) {
	key := string(req.Type) + "_" + req.Region
	if result, ok := m.predictions[key]; ok {
		return result, nil
	}
	return nil, fmt.Errorf("prediction not found")
}

func (m *mockCoordinationRepository) SavePrediction(ctx context.Context, result *PredictionResult) error {
	key := string(result.Type) + "_" + result.Region
	m.predictions[key] = result
	return nil
}

func (m *mockCoordinationRepository) CreateScheduleDecision(ctx context.Context, decision *ScheduleDecision) error {
	m.schedules[decision.TaskID] = decision
	return nil
}

func (m *mockCoordinationRepository) GetScheduleDecision(ctx context.Context, taskID uuid.UUID) (*ScheduleDecision, error) {
	if decision, ok := m.schedules[taskID]; ok {
		return decision, nil
	}
	return nil, fmt.Errorf("schedule decision not found")
}

func TestNewCoordinationEngine(t *testing.T) {
	tests := []struct {
		name   string
		config *CoordinationConfig
	}{
		{
			name:   "nil config uses defaults",
			config: nil,
		},
		{
			name: "custom config",
			config: &CoordinationConfig{
				Enabled:             true,
				SchedulingInterval:  60,
				OptimizationHorizon: 24,
				MaxConcurrentTasks:  100,
				PredictorConfig: PredictorConfig{
					LoadForecastHorizon:      24,
					PriceForecastHorizon:     48,
					RenewableForecastHorizon: 24,
					UpdateInterval:           300,
					ConfidenceThreshold:      0.8,
				},
				OptimizerConfig: OptimizationConfig{
					Horizon:             24,
					TimeResolution:      15,
					MaxIterations:       1000,
					ConvergenceThreshold: 0.001,
				},
				SchedulerConfig: SchedulerConfig{
					QueueSize:         1000,
					WorkerCount:       10,
					RetryLimit:        3,
					Timeout:           300,
					PriorityLevels:    4,
					PreemptionEnabled: true,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockCoordinationRepository()
			engine := NewCoordinationEngine(repo, tt.config)

			if engine == nil {
				t.Error("expected non-nil CoordinationEngine")
			}
		})
	}
}

func TestSubmitTask(t *testing.T) {
	tests := []struct {
		name    string
		task    *ComputeTask
		wantErr bool
	}{
		{
			name: "valid realtime task",
			task: &ComputeTask{
				Name:            "realtime-task-1",
				Type:            TaskTypeRealtime,
				Priority:        TaskPriorityHigh,
				ResourceSpec:    TaskResourceSpec{CPU: 2.0, Memory: 4096},
				EstimatedPower:  100,
				EstimatedDuration: 60,
				TenantID:        uuid.New(),
			},
			wantErr: false,
		},
		{
			name: "valid delayable task",
			task: &ComputeTask{
				Name:            "delayable-task-1",
				Type:            TaskTypeDelayable,
				Priority:        TaskPriorityMedium,
				ResourceSpec:    TaskResourceSpec{CPU: 4.0, Memory: 8192},
				TimeConstraint:  TimeConstraint{IsFlexible: true, MaxDelayMinutes: 120},
				EstimatedPower:  200,
				EstimatedDuration: 120,
				TenantID:        uuid.New(),
			},
			wantErr: false,
		},
		{
			name: "valid batch task",
			task: &ComputeTask{
				Name:            "batch-task-1",
				Type:            TaskTypeBatch,
				Priority:        TaskPriorityLow,
				ResourceSpec:    TaskResourceSpec{CPU: 8.0, Memory: 16384},
				TimeConstraint:  TimeConstraint{IsFlexible: true},
				EstimatedPower:  500,
				EstimatedDuration: 360,
				TenantID:        uuid.New(),
			},
			wantErr: false,
		},
		{
			name: "task with existing ID",
			task: &ComputeTask{
				ID:              uuid.New(),
				Name:            "existing-task",
				Type:            TaskTypeRealtime,
				Priority:        TaskPriorityHigh,
				TenantID:        uuid.New(),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			repo := newMockCoordinationRepository()
			engine := NewCoordinationEngine(repo, nil)

			err := engine.SubmitTask(ctx, tt.task)

			if (err != nil) != tt.wantErr {
				t.Errorf("SubmitTask() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if tt.task.ID == uuid.Nil {
					t.Error("expected ID to be set")
				}
				if tt.task.CreatedAt.IsZero() {
					t.Error("expected CreatedAt to be set")
				}
				if tt.task.Status == "" {
					t.Error("expected default status to be set")
				}
			}
		})
	}
}

func TestGetTask(t *testing.T) {
	ctx := context.Background()
	repo := newMockCoordinationRepository()
	engine := NewCoordinationEngine(repo, nil)

	task := &ComputeTask{
		Name:            "test-task",
		Type:            TaskTypeRealtime,
		Priority:        TaskPriorityHigh,
		ResourceSpec:    TaskResourceSpec{CPU: 2.0, Memory: 4096},
		TenantID:        uuid.New(),
	}

	if err := engine.SubmitTask(ctx, task); err != nil {
		t.Fatalf("failed to submit task: %v", err)
	}

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr bool
	}{
		{
			name:    "existing task",
			id:      task.ID,
			wantErr: false,
		},
		{
			name:    "non-existing task",
			id:      uuid.New(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.GetTask(ctx, tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetTask() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if got.ID != tt.id {
					t.Errorf("expected ID %s, got %s", tt.id, got.ID)
				}
			}
		})
	}
}

func TestUpdateTask(t *testing.T) {
	ctx := context.Background()
	repo := newMockCoordinationRepository()
	engine := NewCoordinationEngine(repo, nil)

	task := &ComputeTask{
		Name:            "test-task",
		Type:            TaskTypeRealtime,
		Priority:        TaskPriorityHigh,
		ResourceSpec:    TaskResourceSpec{CPU: 2.0, Memory: 4096},
		TenantID:        uuid.New(),
	}

	if err := engine.SubmitTask(ctx, task); err != nil {
		t.Fatalf("failed to submit task: %v", err)
	}

	tests := []struct {
		name    string
		id      uuid.UUID
		updates *TaskUpdate
		wantErr bool
	}{
		{
			name: "update status",
			id:   task.ID,
			updates: &TaskUpdate{
				Status: taskStatusPtr(TaskStatusRunning),
			},
			wantErr: false,
		},
		{
			name: "update scheduled times",
			id:   task.ID,
			updates: &TaskUpdate{
				ScheduledStart: timePtr(time.Now().Add(1 * time.Hour)),
				ScheduledEnd:   timePtr(time.Now().Add(2 * time.Hour)),
			},
			wantErr: false,
		},
		{
			name:    "non-existing task",
			id:      uuid.New(),
			updates: &TaskUpdate{Status: taskStatusPtr(TaskStatusCompleted)},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := engine.UpdateTask(ctx, tt.id, tt.updates)

			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateTask() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCancelTask(t *testing.T) {
	ctx := context.Background()
	repo := newMockCoordinationRepository()
	engine := NewCoordinationEngine(repo, nil)

	pendingTask := &ComputeTask{
		Name:            "pending-task",
		Type:            TaskTypeRealtime,
		Priority:        TaskPriorityHigh,
		Status:          TaskStatusPending,
		TenantID:        uuid.New(),
	}

	runningTask := &ComputeTask{
		Name:            "running-task",
		Type:            TaskTypeRealtime,
		Priority:        TaskPriorityHigh,
		Status:          TaskStatusRunning,
		TenantID:        uuid.New(),
	}

	completedTask := &ComputeTask{
		Name:            "completed-task",
		Type:            TaskTypeRealtime,
		Priority:        TaskPriorityHigh,
		Status:          TaskStatusCompleted,
		TenantID:        uuid.New(),
	}

	engine.SubmitTask(ctx, pendingTask)
	engine.SubmitTask(ctx, runningTask)
	engine.SubmitTask(ctx, completedTask)

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr bool
	}{
		{
			name:    "cancel pending task",
			id:      pendingTask.ID,
			wantErr: false,
		},
		{
			name:    "cancel running task",
			id:      runningTask.ID,
			wantErr: false,
		},
		{
			name:    "cancel completed task",
			id:      completedTask.ID,
			wantErr: true,
		},
		{
			name:    "cancel non-existing task",
			id:      uuid.New(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := engine.CancelTask(ctx, tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("CancelTask() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestScheduleTask(t *testing.T) {
	ctx := context.Background()
	repo := newMockCoordinationRepository()
	engine := NewCoordinationEngine(repo, nil)

	task := &ComputeTask{
		Name:            "test-task",
		Type:            TaskTypeDelayable,
		Priority:        TaskPriorityMedium,
		ResourceSpec:    TaskResourceSpec{CPU: 2.0, Memory: 4096, PowerCapacity: 100},
		TimeConstraint:  TimeConstraint{IsFlexible: true, MaxDelayMinutes: 120},
		EnergyPreference: EnergyPreference{PreferGreen: true, MaxPricePerKWh: 0.5},
		EstimatedPower:  100,
		EstimatedDuration: 60,
		Status:          TaskStatusPending,
		TenantID:        uuid.New(),
	}

	if err := engine.SubmitTask(ctx, task); err != nil {
		t.Fatalf("failed to submit task: %v", err)
	}

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr bool
	}{
		{
			name:    "schedule valid task",
			id:      task.ID,
			wantErr: false,
		},
		{
			name:    "schedule non-existing task",
			id:      uuid.New(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision, err := engine.ScheduleTask(ctx, tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("ScheduleTask() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if decision == nil {
					t.Error("expected non-nil decision")
					return
				}
				if decision.TaskID != tt.id {
					t.Errorf("expected TaskID %s, got %s", tt.id, decision.TaskID)
				}
			}
		})
	}
}

func TestGetTaskSchedule(t *testing.T) {
	ctx := context.Background()
	repo := newMockCoordinationRepository()
	engine := NewCoordinationEngine(repo, nil)

	task := &ComputeTask{
		Name:            "test-task",
		Type:            TaskTypeDelayable,
		Priority:        TaskPriorityMedium,
		Status:          TaskStatusScheduled,
		TenantID:        uuid.New(),
	}

	if err := engine.SubmitTask(ctx, task); err != nil {
		t.Fatalf("failed to submit task: %v", err)
	}

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr bool
	}{
		{
			name:    "existing task",
			id:      task.ID,
			wantErr: false,
		},
		{
			name:    "non-existing task",
			id:      uuid.New(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schedule, err := engine.GetTaskSchedule(ctx, tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetTaskSchedule() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && schedule == nil {
				t.Error("expected non-nil schedule")
			}
		})
	}
}

func TestListTasks(t *testing.T) {
	ctx := context.Background()
	repo := newMockCoordinationRepository()
	engine := NewCoordinationEngine(repo, nil)

	task1 := &ComputeTask{
		Name:     "task-1",
		Type:     TaskTypeRealtime,
		Priority: TaskPriorityHigh,
		Status:   TaskStatusRunning,
		TenantID: uuid.New(),
	}
	task2 := &ComputeTask{
		Name:     "task-2",
		Type:     TaskTypeDelayable,
		Priority: TaskPriorityMedium,
		Status:   TaskStatusPending,
		TenantID: uuid.New(),
	}
	task3 := &ComputeTask{
		Name:     "task-3",
		Type:     TaskTypeBatch,
		Priority: TaskPriorityLow,
		Status:   TaskStatusCompleted,
		TenantID: uuid.New(),
	}

	engine.SubmitTask(ctx, task1)
	engine.SubmitTask(ctx, task2)
	engine.SubmitTask(ctx, task3)

	tests := []struct {
		name   string
		filter *TaskFilter
		count  int
	}{
		{
			name:   "list all tasks",
			filter: nil,
			count:  3,
		},
		{
			name:   "filter by running status",
			filter: &TaskFilter{Status: TaskStatusRunning},
			count:  1,
		},
		{
			name:   "filter by pending status",
			filter: &TaskFilter{Status: TaskStatusPending},
			count:  1,
		},
		{
			name:   "filter by realtime type",
			filter: &TaskFilter{Type: TaskTypeRealtime},
			count:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tasks, err := engine.ListTasks(ctx, tt.filter)

			if err != nil {
				t.Errorf("ListTasks() error = %v", err)
				return
			}

			if len(tasks) != tt.count {
				t.Errorf("expected %d tasks, got %d", tt.count, len(tasks))
			}
		})
	}
}

func TestGetCoordinationMetrics(t *testing.T) {
	ctx := context.Background()
	repo := newMockCoordinationRepository()
	engine := NewCoordinationEngine(repo, nil)

	task1 := &ComputeTask{
		Name:            "task-1",
		Type:            TaskTypeRealtime,
		Priority:        TaskPriorityHigh,
		Status:          TaskStatusCompleted,
		EnergyCost:      10.5,
		CarbonEmission:  5.2,
		GreenRatio:      0.8,
		TenantID:        uuid.New(),
	}
	task2 := &ComputeTask{
		Name:            "task-2",
		Type:            TaskTypeDelayable,
		Priority:        TaskPriorityMedium,
		Status:          TaskStatusRunning,
		EnergyCost:      8.3,
		CarbonEmission:  3.1,
		GreenRatio:      0.9,
		TenantID:        uuid.New(),
	}

	engine.SubmitTask(ctx, task1)
	engine.SubmitTask(ctx, task2)

	metrics, err := engine.GetCoordinationMetrics(ctx)

	if err != nil {
		t.Errorf("GetCoordinationMetrics() error = %v", err)
		return
	}

	if metrics == nil {
		t.Error("expected non-nil metrics")
		return
	}

	if metrics.TotalTasks != 2 {
		t.Errorf("expected 2 total tasks, got %d", metrics.TotalTasks)
	}
}

func TestGetRealtimeData(t *testing.T) {
	ctx := context.Background()
	repo := newMockCoordinationRepository()
	engine := NewCoordinationEngine(repo, nil)

	data, err := engine.GetRealtimeData(ctx)

	if err != nil {
		t.Errorf("GetRealtimeData() error = %v", err)
		return
	}

	if data == nil {
		t.Error("expected non-nil data")
		return
	}

	if data.Timestamp.IsZero() {
		t.Error("expected Timestamp to be set")
	}
}

func TestCreatePolicy(t *testing.T) {
	tests := []struct {
		name    string
		policy  *CoordinationPolicy
		wantErr bool
	}{
		{
			name: "valid policy",
			policy: &CoordinationPolicy{
				Name:        "default-policy",
				Description: "Default coordination policy",
				Enabled:     true,
				Priority:    1,
				TaskMatching: TaskMatchingPolicy{
					RealtimeTaskStrategy: "immediate",
					DelayableTaskStrategy: "optimized",
					BatchTaskStrategy:     "scheduled",
				},
				PriceOptimization: PriceOptimizationPolicy{
					Enabled:            true,
					PriceThresholdLow:  0.3,
					PriceThresholdHigh: 0.8,
				},
				StorageStrategy: StorageStrategyPolicy{
					Enabled:        true,
					ChargeThreshold: 0.3,
					DischargeThreshold: 0.8,
				},
				TenantID: uuid.New(),
			},
			wantErr: false,
		},
		{
			name: "policy with existing ID",
			policy: &CoordinationPolicy{
				ID:       uuid.New(),
				Name:     "existing-policy",
				Enabled:  true,
				TenantID: uuid.New(),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			repo := newMockCoordinationRepository()
			engine := NewCoordinationEngine(repo, nil)

			err := engine.CreatePolicy(ctx, tt.policy)

			if (err != nil) != tt.wantErr {
				t.Errorf("CreatePolicy() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if tt.policy.ID == uuid.Nil {
					t.Error("expected ID to be set")
				}
				if tt.policy.CreatedAt.IsZero() {
					t.Error("expected CreatedAt to be set")
				}
			}
		})
	}
}

func TestGetPolicy(t *testing.T) {
	ctx := context.Background()
	repo := newMockCoordinationRepository()
	engine := NewCoordinationEngine(repo, nil)

	policy := &CoordinationPolicy{
		Name:     "test-policy",
		Enabled:  true,
		TenantID: uuid.New(),
	}

	if err := engine.CreatePolicy(ctx, policy); err != nil {
		t.Fatalf("failed to create policy: %v", err)
	}

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr bool
	}{
		{
			name:    "existing policy",
			id:      policy.ID,
			wantErr: false,
		},
		{
			name:    "non-existing policy",
			id:      uuid.New(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.GetPolicy(ctx, tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetPolicy() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if got.ID != tt.id {
					t.Errorf("expected ID %s, got %s", tt.id, got.ID)
				}
			}
		})
	}
}

func TestTaskTypes(t *testing.T) {
	ctx := context.Background()
	repo := newMockCoordinationRepository()
	engine := NewCoordinationEngine(repo, nil)

	tests := []struct {
		name      string
		taskType  TaskType
		wantErr   bool
	}{
		{
			name:     "realtime task",
			taskType: TaskTypeRealtime,
			wantErr:  false,
		},
		{
			name:     "delayable task",
			taskType: TaskTypeDelayable,
			wantErr:  false,
		},
		{
			name:     "batch task",
			taskType: TaskTypeBatch,
			wantErr:  false,
		},
		{
			name:     "interruptible task",
			taskType: TaskTypeInterruptible,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &ComputeTask{
				Name:     string(tt.taskType) + "-task",
				Type:     tt.taskType,
				Priority: TaskPriorityMedium,
				TenantID: uuid.New(),
			}

			err := engine.SubmitTask(ctx, task)

			if (err != nil) != tt.wantErr {
				t.Errorf("SubmitTask() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTaskPriorities(t *testing.T) {
	ctx := context.Background()
	repo := newMockCoordinationRepository()
	engine := NewCoordinationEngine(repo, nil)

	tests := []struct {
		name     string
		priority TaskPriority
		wantErr  bool
	}{
		{
			name:     "high priority",
			priority: TaskPriorityHigh,
			wantErr:  false,
		},
		{
			name:     "medium priority",
			priority: TaskPriorityMedium,
			wantErr:  false,
		},
		{
			name:     "low priority",
			priority: TaskPriorityLow,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &ComputeTask{
				Name:     string(tt.priority) + "-priority-task",
				Type:     TaskTypeRealtime,
				Priority: tt.priority,
				TenantID: uuid.New(),
			}

			err := engine.SubmitTask(ctx, task)

			if (err != nil) != tt.wantErr {
				t.Errorf("SubmitTask() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTaskStatusTransitions(t *testing.T) {
	ctx := context.Background()
	repo := newMockCoordinationRepository()
	engine := NewCoordinationEngine(repo, nil)

	tests := []struct {
		name          string
		initialStatus TaskStatus
		newStatus     TaskStatus
		wantErr       bool
	}{
		{
			name:          "pending to scheduled",
			initialStatus: TaskStatusPending,
			newStatus:     TaskStatusScheduled,
			wantErr:       false,
		},
		{
			name:          "scheduled to running",
			initialStatus: TaskStatusScheduled,
			newStatus:     TaskStatusRunning,
			wantErr:       false,
		},
		{
			name:          "running to completed",
			initialStatus: TaskStatusRunning,
			newStatus:     TaskStatusCompleted,
			wantErr:       false,
		},
		{
			name:          "running to failed",
			initialStatus: TaskStatusRunning,
			newStatus:     TaskStatusFailed,
			wantErr:       false,
		},
		{
			name:          "pending to cancelled",
			initialStatus: TaskStatusPending,
			newStatus:     TaskStatusCancelled,
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &ComputeTask{
				Name:     "test-task",
				Type:     TaskTypeRealtime,
				Priority: TaskPriorityMedium,
				Status:   tt.initialStatus,
				TenantID: uuid.New(),
			}

			if err := engine.SubmitTask(ctx, task); err != nil {
				t.Fatalf("failed to submit task: %v", err)
			}

			err := engine.UpdateTask(ctx, task.ID, &TaskUpdate{
				Status: &tt.newStatus,
			})

			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateTask() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEnergyPreferences(t *testing.T) {
	ctx := context.Background()
	repo := newMockCoordinationRepository()
	engine := NewCoordinationEngine(repo, nil)

	tests := []struct {
		name            string
		energyPreference EnergyPreference
		wantErr         bool
	}{
		{
			name: "prefer green energy",
			energyPreference: EnergyPreference{
				PreferGreen:    true,
				MinGreenRatio:  0.8,
			},
			wantErr: false,
		},
		{
			name: "prefer low price",
			energyPreference: EnergyPreference{
				PreferLowPrice: true,
				MaxPricePerKWh: 0.3,
			},
			wantErr: false,
		},
		{
			name: "allow storage",
			energyPreference: EnergyPreference{
				AllowStorage: true,
			},
			wantErr: false,
		},
		{
			name: "carbon constraint",
			energyPreference: EnergyPreference{
				MaxCarbonIntensity: 100,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &ComputeTask{
				Name:             "test-task",
				Type:             TaskTypeDelayable,
				Priority:         TaskPriorityMedium,
				EnergyPreference: tt.energyPreference,
				TenantID:         uuid.New(),
			}

			err := engine.SubmitTask(ctx, task)

			if (err != nil) != tt.wantErr {
				t.Errorf("SubmitTask() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTimeConstraints(t *testing.T) {
	ctx := context.Background()
	repo := newMockCoordinationRepository()
	engine := NewCoordinationEngine(repo, nil)

	now := time.Now()

	tests := []struct {
		name           string
		timeConstraint TimeConstraint
		wantErr        bool
	}{
		{
			name: "flexible with max delay",
			timeConstraint: TimeConstraint{
				IsFlexible:      true,
				MaxDelayMinutes: 120,
			},
			wantErr: false,
		},
		{
			name: "deadline constraint",
			timeConstraint: TimeConstraint{
				Deadline: &now,
			},
			wantErr: false,
		},
		{
			name: "preferred hours",
			timeConstraint: TimeConstraint{
				PreferredHours: []int{8, 9, 10, 11, 12},
			},
			wantErr: false,
		},
		{
			name: "interruptible",
			timeConstraint: TimeConstraint{
				Interruptible:    true,
				MaxInterruptions: 3,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &ComputeTask{
				Name:           "test-task",
				Type:           TaskTypeDelayable,
				Priority:       TaskPriorityMedium,
				TimeConstraint: tt.timeConstraint,
				TenantID:       uuid.New(),
			}

			err := engine.SubmitTask(ctx, task)

			if (err != nil) != tt.wantErr {
				t.Errorf("SubmitTask() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func taskStatusPtr(s TaskStatus) *TaskStatus {
	return &s
}

func timePtr(t time.Time) *time.Time {
	return &t
}
