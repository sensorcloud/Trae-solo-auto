package agent

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewExecutionEngine(t *testing.T) {
	tests := []struct {
		name   string
		config *ExecutionConfig
	}{
		{
			name:   "nil config uses defaults",
			config: nil,
		},
		{
			name: "custom config",
			config: &ExecutionConfig{
				MaxConcurrent:      100,
				DefaultTimeout:     30 * time.Minute,
				RetryCount:         3,
				RetryDelay:         5 * time.Second,
				QueueSize:          1000,
				WorkerCount:        10,
				EnablePriority:     true,
				EnablePreemption:   true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockAgentRepository()
			engine := NewExecutionEngine(repo, tt.config)

			if engine == nil {
				t.Error("expected non-nil ExecutionEngine")
			}
		})
	}
}

func TestCreateExecution(t *testing.T) {
	tests := []struct {
		name      string
		execution *Execution
		wantErr   bool
	}{
		{
			name: "valid execution",
			execution: &Execution{
				Name:        "test-execution-1",
				SandboxID:   uuid.New(),
				Command:     "python",
				Args:        []string{"script.py"},
				Timeout:     30 * time.Minute,
				Priority:    ExecutionPriorityNormal,
				TenantID:    uuid.New(),
			},
			wantErr: false,
		},
		{
			name: "execution with environment",
			execution: &Execution{
				Name:        "test-execution-2",
				SandboxID:   uuid.New(),
				Command:     "go",
				Args:        []string{"run", "main.go"},
				Env:         map[string]string{"GOPATH": "/go"},
				Timeout:     60 * time.Minute,
				Priority:    ExecutionPriorityHigh,
				TenantID:    uuid.New(),
			},
			wantErr: false,
		},
		{
			name: "execution with existing ID",
			execution: &Execution{
				ID:          uuid.New(),
				Name:        "existing-execution",
				SandboxID:   uuid.New(),
				Command:     "echo",
				Args:        []string{"hello"},
				TenantID:    uuid.New(),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			repo := newMockAgentRepository()
			engine := NewExecutionEngine(repo, nil)

			err := engine.CreateExecution(ctx, tt.execution)

			if (err != nil) != tt.wantErr {
				t.Errorf("CreateExecution() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if tt.execution.ID == uuid.Nil {
					t.Error("expected ID to be set")
				}
				if tt.execution.CreatedAt.IsZero() {
					t.Error("expected CreatedAt to be set")
				}
				if tt.execution.Status == "" {
					t.Error("expected default status to be set")
				}
			}
		})
	}
}

func TestGetExecution(t *testing.T) {
	ctx := context.Background()
	repo := newMockAgentRepository()
	engine := NewExecutionEngine(repo, nil)

	execution := &Execution{
		Name:      "test-execution",
		SandboxID: uuid.New(),
		Command:   "python",
		Args:      []string{"script.py"},
		TenantID:  uuid.New(),
	}

	if err := engine.CreateExecution(ctx, execution); err != nil {
		t.Fatalf("failed to create execution: %v", err)
	}

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr bool
	}{
		{
			name:    "existing execution",
			id:      execution.ID,
			wantErr: false,
		},
		{
			name:    "non-existing execution",
			id:      uuid.New(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.GetExecution(ctx, tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetExecution() error = %v, wantErr %v", err, tt.wantErr)
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

func TestUpdateExecution(t *testing.T) {
	ctx := context.Background()
	repo := newMockAgentRepository()
	engine := NewExecutionEngine(repo, nil)

	execution := &Execution{
		Name:      "test-execution",
		SandboxID: uuid.New(),
		Command:   "python",
		Args:      []string{"script.py"},
		TenantID:  uuid.New(),
	}

	if err := engine.CreateExecution(ctx, execution); err != nil {
		t.Fatalf("failed to create execution: %v", err)
	}

	tests := []struct {
		name    string
		id      uuid.UUID
		updates *ExecutionUpdate
		wantErr bool
	}{
		{
			name: "update status",
			id:   execution.ID,
			updates: &ExecutionUpdate{
				Status: executionStatusPtr(ExecutionStatusRunning),
			},
			wantErr: false,
		},
		{
			name: "update output",
			id:   execution.ID,
			updates: &ExecutionUpdate{
				Stdout: strPtr("output"),
				Stderr: strPtr(""),
			},
			wantErr: false,
		},
		{
			name:    "non-existing execution",
			id:      uuid.New(),
			updates: &ExecutionUpdate{Status: executionStatusPtr(ExecutionStatusCompleted)},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := engine.UpdateExecution(ctx, tt.id, tt.updates)

			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateExecution() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				updated, err := engine.GetExecution(ctx, tt.id)
				if err != nil {
					t.Fatalf("failed to get updated execution: %v", err)
				}

				if tt.updates.Status != nil && updated.Status != *tt.updates.Status {
					t.Errorf("expected Status %s, got %s", *tt.updates.Status, updated.Status)
				}
			}
		})
	}
}

func TestStartExecution(t *testing.T) {
	ctx := context.Background()
	repo := newMockAgentRepository()
	engine := NewExecutionEngine(repo, nil)

	execution := &Execution{
		Name:      "test-execution",
		SandboxID: uuid.New(),
		Command:   "python",
		Args:      []string{"script.py"},
		Status:    ExecutionStatusPending,
		TenantID:  uuid.New(),
	}

	if err := engine.CreateExecution(ctx, execution); err != nil {
		t.Fatalf("failed to create execution: %v", err)
	}

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr bool
	}{
		{
			name:    "start pending execution",
			id:      execution.ID,
			wantErr: false,
		},
		{
			name:    "start non-existing execution",
			id:      uuid.New(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := engine.StartExecution(ctx, tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("StartExecution() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStopExecution(t *testing.T) {
	ctx := context.Background()
	repo := newMockAgentRepository()
	engine := NewExecutionEngine(repo, nil)

	execution := &Execution{
		Name:      "test-execution",
		SandboxID: uuid.New(),
		Command:   "python",
		Args:      []string{"script.py"},
		Status:    ExecutionStatusRunning,
		TenantID:  uuid.New(),
	}

	if err := engine.CreateExecution(ctx, execution); err != nil {
		t.Fatalf("failed to create execution: %v", err)
	}

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr bool
	}{
		{
			name:    "stop running execution",
			id:      execution.ID,
			wantErr: false,
		},
		{
			name:    "stop non-existing execution",
			id:      uuid.New(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := engine.StopExecution(ctx, tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("StopExecution() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCancelExecution(t *testing.T) {
	ctx := context.Background()
	repo := newMockAgentRepository()
	engine := NewExecutionEngine(repo, nil)

	pendingExecution := &Execution{
		Name:      "pending-execution",
		SandboxID: uuid.New(),
		Command:   "python",
		Args:      []string{"script.py"},
		Status:    ExecutionStatusPending,
		TenantID:  uuid.New(),
	}

	runningExecution := &Execution{
		Name:      "running-execution",
		SandboxID: uuid.New(),
		Command:   "python",
		Args:      []string{"script.py"},
		Status:    ExecutionStatusRunning,
		TenantID:  uuid.New(),
	}

	completedExecution := &Execution{
		Name:      "completed-execution",
		SandboxID: uuid.New(),
		Command:   "python",
		Args:      []string{"script.py"},
		Status:    ExecutionStatusCompleted,
		TenantID:  uuid.New(),
	}

	engine.CreateExecution(ctx, pendingExecution)
	engine.CreateExecution(ctx, runningExecution)
	engine.CreateExecution(ctx, completedExecution)

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr bool
	}{
		{
			name:    "cancel pending execution",
			id:      pendingExecution.ID,
			wantErr: false,
		},
		{
			name:    "cancel running execution",
			id:      runningExecution.ID,
			wantErr: false,
		},
		{
			name:    "cancel completed execution",
			id:      completedExecution.ID,
			wantErr: true,
		},
		{
			name:    "cancel non-existing execution",
			id:      uuid.New(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := engine.CancelExecution(ctx, tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("CancelExecution() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRetryExecution(t *testing.T) {
	ctx := context.Background()
	repo := newMockAgentRepository()
	engine := NewExecutionEngine(repo, nil)

	failedExecution := &Execution{
		Name:       "failed-execution",
		SandboxID:  uuid.New(),
		Command:    "python",
		Args:       []string{"script.py"},
		Status:     ExecutionStatusFailed,
		RetryCount: 0,
		MaxRetries: 3,
		TenantID:   uuid.New(),
	}

	maxRetriesExecution := &Execution{
		Name:       "max-retries-execution",
		SandboxID:  uuid.New(),
		Command:    "python",
		Args:       []string{"script.py"},
		Status:     ExecutionStatusFailed,
		RetryCount: 3,
		MaxRetries: 3,
		TenantID:   uuid.New(),
	}

	runningExecution := &Execution{
		Name:      "running-execution",
		SandboxID: uuid.New(),
		Command:   "python",
		Args:      []string{"script.py"},
		Status:    ExecutionStatusRunning,
		TenantID:  uuid.New(),
	}

	engine.CreateExecution(ctx, failedExecution)
	engine.CreateExecution(ctx, maxRetriesExecution)
	engine.CreateExecution(ctx, runningExecution)

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr bool
	}{
		{
			name:    "retry failed execution",
			id:      failedExecution.ID,
			wantErr: false,
		},
		{
			name:    "retry max retries reached",
			id:      maxRetriesExecution.ID,
			wantErr: true,
		},
		{
			name:    "retry running execution",
			id:      runningExecution.ID,
			wantErr: true,
		},
		{
			name:    "retry non-existing execution",
			id:      uuid.New(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := engine.RetryExecution(ctx, tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("RetryExecution() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetExecutionLogs(t *testing.T) {
	ctx := context.Background()
	repo := newMockAgentRepository()
	engine := NewExecutionEngine(repo, nil)

	execution := &Execution{
		Name:      "test-execution",
		SandboxID: uuid.New(),
		Command:   "python",
		Args:      []string{"script.py"},
		Status:    ExecutionStatusRunning,
		TenantID:  uuid.New(),
	}

	if err := engine.CreateExecution(ctx, execution); err != nil {
		t.Fatalf("failed to create execution: %v", err)
	}

	tests := []struct {
		name    string
		id      uuid.UUID
		tail    int
		wantErr bool
	}{
		{
			name:    "get logs with tail",
			id:      execution.ID,
			tail:    100,
			wantErr: false,
		},
		{
			name:    "get logs without tail",
			id:      execution.ID,
			tail:    0,
			wantErr: false,
		},
		{
			name:    "non-existing execution",
			id:      uuid.New(),
			tail:    100,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logs, err := engine.GetExecutionLogs(ctx, tt.id, tt.tail)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetExecutionLogs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && logs == nil {
				t.Error("expected non-nil logs")
			}
		})
	}
}

func TestListExecutions(t *testing.T) {
	ctx := context.Background()
	repo := newMockAgentRepository()
	engine := NewExecutionEngine(repo, nil)

	execution1 := &Execution{
		Name:      "execution-1",
		SandboxID: uuid.New(),
		Command:   "python",
		Args:      []string{"script1.py"},
		Status:    ExecutionStatusRunning,
		TenantID:  uuid.New(),
	}
	execution2 := &Execution{
		Name:      "execution-2",
		SandboxID: uuid.New(),
		Command:   "python",
		Args:      []string{"script2.py"},
		Status:    ExecutionStatusPending,
		TenantID:  uuid.New(),
	}
	execution3 := &Execution{
		Name:      "execution-3",
		SandboxID: uuid.New(),
		Command:   "python",
		Args:      []string{"script3.py"},
		Status:    ExecutionStatusCompleted,
		TenantID:  uuid.New(),
	}

	engine.CreateExecution(ctx, execution1)
	engine.CreateExecution(ctx, execution2)
	engine.CreateExecution(ctx, execution3)

	tests := []struct {
		name   string
		filter *ExecutionFilter
		count  int
	}{
		{
			name:   "list all executions",
			filter: nil,
			count:  3,
		},
		{
			name:   "filter by running status",
			filter: &ExecutionFilter{Status: ExecutionStatusRunning},
			count:  1,
		},
		{
			name:   "filter by pending status",
			filter: &ExecutionFilter{Status: ExecutionStatusPending},
			count:  1,
		},
		{
			name:   "filter by completed status",
			filter: &ExecutionFilter{Status: ExecutionStatusCompleted},
			count:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executions, err := engine.ListExecutions(ctx, tt.filter)

			if err != nil {
				t.Errorf("ListExecutions() error = %v", err)
				return
			}

			if len(executions) != tt.count {
				t.Errorf("expected %d executions, got %d", tt.count, len(executions))
			}
		})
	}
}

func TestGetExecutionMetrics(t *testing.T) {
	ctx := context.Background()
	repo := newMockAgentRepository()
	engine := NewExecutionEngine(repo, nil)

	execution := &Execution{
		Name:      "test-execution",
		SandboxID: uuid.New(),
		Command:   "python",
		Args:      []string{"script.py"},
		Status:    ExecutionStatusRunning,
		TenantID:  uuid.New(),
	}

	if err := engine.CreateExecution(ctx, execution); err != nil {
		t.Fatalf("failed to create execution: %v", err)
	}

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr bool
	}{
		{
			name:    "existing execution",
			id:      execution.ID,
			wantErr: false,
		},
		{
			name:    "non-existing execution",
			id:      uuid.New(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics, err := engine.GetExecutionMetrics(ctx, tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetExecutionMetrics() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if metrics == nil {
					t.Error("expected non-nil metrics")
					return
				}
				if metrics.ExecutionID != tt.id {
					t.Errorf("expected ExecutionID %s, got %s", tt.id, metrics.ExecutionID)
				}
			}
		})
	}
}

func TestGetQueueStatus(t *testing.T) {
	ctx := context.Background()
	repo := newMockAgentRepository()
	engine := NewExecutionEngine(repo, nil)

	status, err := engine.GetQueueStatus(ctx)

	if err != nil {
		t.Errorf("GetQueueStatus() error = %v", err)
		return
	}

	if status == nil {
		t.Error("expected non-nil status")
		return
	}

	if status.QueueSize < 0 {
		t.Error("expected non-negative queue size")
	}
	if status.ActiveWorkers < 0 {
		t.Error("expected non-negative active workers")
	}
}

func TestExecutionPriority(t *testing.T) {
	ctx := context.Background()
	repo := newMockAgentRepository()
	engine := NewExecutionEngine(repo, nil)

	tests := []struct {
		name     string
		priority ExecutionPriority
		wantErr  bool
	}{
		{
			name:     "low priority",
			priority: ExecutionPriorityLow,
			wantErr:  false,
		},
		{
			name:     "normal priority",
			priority: ExecutionPriorityNormal,
			wantErr:  false,
		},
		{
			name:     "high priority",
			priority: ExecutionPriorityHigh,
			wantErr:  false,
		},
		{
			name:     "critical priority",
			priority: ExecutionPriorityCritical,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			execution := &Execution{
				Name:      "test-execution",
				SandboxID: uuid.New(),
				Command:   "python",
				Args:      []string{"script.py"},
				Priority:  tt.priority,
				TenantID:  uuid.New(),
			}

			err := engine.CreateExecution(ctx, execution)

			if (err != nil) != tt.wantErr {
				t.Errorf("CreateExecution() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestExecutionTimeout(t *testing.T) {
	ctx := context.Background()
	repo := newMockAgentRepository()
	engine := NewExecutionEngine(repo, nil)

	tests := []struct {
		name    string
		timeout time.Duration
		wantErr bool
	}{
		{
			name:    "short timeout",
			timeout: 1 * time.Minute,
			wantErr: false,
		},
		{
			name:    "medium timeout",
			timeout: 30 * time.Minute,
			wantErr: false,
		},
		{
			name:    "long timeout",
			timeout: 2 * time.Hour,
			wantErr: false,
		},
		{
			name:    "zero timeout",
			timeout: 0,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			execution := &Execution{
				Name:      "test-execution",
				SandboxID: uuid.New(),
				Command:   "python",
				Args:      []string{"script.py"},
				Timeout:   tt.timeout,
				TenantID:  uuid.New(),
			}

			err := engine.CreateExecution(ctx, execution)

			if (err != nil) != tt.wantErr {
				t.Errorf("CreateExecution() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestExecutionStatusTransitions(t *testing.T) {
	ctx := context.Background()
	repo := newMockAgentRepository()
	engine := NewExecutionEngine(repo, nil)

	tests := []struct {
		name          string
		initialStatus ExecutionStatus
		action        func(uuid.UUID) error
		wantErr       bool
	}{
		{
			name:          "start from pending",
			initialStatus: ExecutionStatusPending,
			action: func(id uuid.UUID) error {
				return engine.StartExecution(ctx, id)
			},
			wantErr: false,
		},
		{
			name:          "stop from running",
			initialStatus: ExecutionStatusRunning,
			action: func(id uuid.UUID) error {
				return engine.StopExecution(ctx, id)
			},
			wantErr: false,
		},
		{
			name:          "cancel from pending",
			initialStatus: ExecutionStatusPending,
			action: func(id uuid.UUID) error {
				return engine.CancelExecution(ctx, id)
			},
			wantErr: false,
		},
		{
			name:          "cancel from running",
			initialStatus: ExecutionStatusRunning,
			action: func(id uuid.UUID) error {
				return engine.CancelExecution(ctx, id)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			execution := &Execution{
				Name:      "test-execution",
				SandboxID: uuid.New(),
				Command:   "python",
				Args:      []string{"script.py"},
				Status:    tt.initialStatus,
				TenantID:  uuid.New(),
			}

			if err := engine.CreateExecution(ctx, execution); err != nil {
				t.Fatalf("failed to create execution: %v", err)
			}

			err := tt.action(execution.ID)

			if (err != nil) != tt.wantErr {
				t.Errorf("action() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBatchExecution(t *testing.T) {
	ctx := context.Background()
	repo := newMockAgentRepository()
	engine := NewExecutionEngine(repo, nil)

	sandboxID := uuid.New()

	executions := []*Execution{
		{
			Name:      "batch-1",
			SandboxID: sandboxID,
			Command:   "python",
			Args:      []string{"script1.py"},
			TenantID:  uuid.New(),
		},
		{
			Name:      "batch-2",
			SandboxID: sandboxID,
			Command:   "python",
			Args:      []string{"script2.py"},
			TenantID:  uuid.New(),
		},
		{
			Name:      "batch-3",
			SandboxID: sandboxID,
			Command:   "python",
			Args:      []string{"script3.py"},
			TenantID:  uuid.New(),
		},
	}

	tests := []struct {
		name       string
		executions []*Execution
		wantErr    bool
	}{
		{
			name:       "valid batch",
			executions: executions,
			wantErr:    false,
		},
		{
			name:       "empty batch",
			executions: []*Execution{},
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ids, err := engine.CreateBatchExecutions(ctx, tt.executions)

			if (err != nil) != tt.wantErr {
				t.Errorf("CreateBatchExecutions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(ids) != len(tt.executions) {
					t.Errorf("expected %d IDs, got %d", len(tt.executions), len(ids))
				}
			}
		})
	}
}

func TestGetExecutionStatistics(t *testing.T) {
	ctx := context.Background()
	repo := newMockAgentRepository()
	engine := NewExecutionEngine(repo, nil)

	execution1 := &Execution{
		Name:      "execution-1",
		SandboxID: uuid.New(),
		Command:   "python",
		Args:      []string{"script1.py"},
		Status:    ExecutionStatusCompleted,
		TenantID:  uuid.New(),
	}
	execution2 := &Execution{
		Name:      "execution-2",
		SandboxID: uuid.New(),
		Command:   "python",
		Args:      []string{"script2.py"},
		Status:    ExecutionStatusFailed,
		TenantID:  uuid.New(),
	}
	execution3 := &Execution{
		Name:      "execution-3",
		SandboxID: uuid.New(),
		Command:   "python",
		Args:      []string{"script3.py"},
		Status:    ExecutionStatusRunning,
		TenantID:  uuid.New(),
	}

	engine.CreateExecution(ctx, execution1)
	engine.CreateExecution(ctx, execution2)
	engine.CreateExecution(ctx, execution3)

	stats, err := engine.GetExecutionStatistics(ctx)

	if err != nil {
		t.Errorf("GetExecutionStatistics() error = %v", err)
		return
	}

	if stats == nil {
		t.Error("expected non-nil statistics")
		return
	}

	if stats.TotalExecutions != 3 {
		t.Errorf("expected 3 total executions, got %d", stats.TotalExecutions)
	}
}

func executionStatusPtr(s ExecutionStatus) *ExecutionStatus {
	return &s
}
