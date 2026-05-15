package agent

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

type mockAgentRepository struct {
	sandboxes    map[uuid.UUID]*Sandbox
	executions   map[uuid.UUID]*Execution
	agents       map[uuid.UUID]*Agent
	resources    map[uuid.UUID]*Resource
	quotas       map[uuid.UUID]*ResourceQuota
	events       []AgentEvent
}

func newMockAgentRepository() *mockAgentRepository {
	return &mockAgentRepository{
		sandboxes:  make(map[uuid.UUID]*Sandbox),
		executions: make(map[uuid.UUID]*Execution),
		agents:     make(map[uuid.UUID]*Agent),
		resources:  make(map[uuid.UUID]*Resource),
		quotas:     make(map[uuid.UUID]*ResourceQuota),
		events:     make([]AgentEvent, 0),
	}
}

func (m *mockAgentRepository) CreateSandbox(ctx context.Context, sandbox *Sandbox) error {
	m.sandboxes[sandbox.ID] = sandbox
	return nil
}

func (m *mockAgentRepository) GetSandbox(ctx context.Context, id uuid.UUID) (*Sandbox, error) {
	if sandbox, ok := m.sandboxes[id]; ok {
		return sandbox, nil
	}
	return nil, fmt.Errorf("sandbox not found")
}

func (m *mockAgentRepository) ListSandboxes(ctx context.Context, filter *SandboxFilter) ([]*Sandbox, error) {
	result := make([]*Sandbox, 0)
	for _, sandbox := range m.sandboxes {
		if filter != nil {
			if filter.Status != "" && sandbox.Status != filter.Status {
				continue
			}
		}
		result = append(result, sandbox)
	}
	return result, nil
}

func (m *mockAgentRepository) UpdateSandbox(ctx context.Context, sandbox *Sandbox) error {
	m.sandboxes[sandbox.ID] = sandbox
	return nil
}

func (m *mockAgentRepository) DeleteSandbox(ctx context.Context, id uuid.UUID) error {
	delete(m.sandboxes, id)
	return nil
}

func (m *mockAgentRepository) CreateExecution(ctx context.Context, execution *Execution) error {
	m.executions[execution.ID] = execution
	return nil
}

func (m *mockAgentRepository) GetExecution(ctx context.Context, id uuid.UUID) (*Execution, error) {
	if execution, ok := m.executions[id]; ok {
		return execution, nil
	}
	return nil, fmt.Errorf("execution not found")
}

func (m *mockAgentRepository) ListExecutions(ctx context.Context, filter *ExecutionFilter) ([]*Execution, error) {
	result := make([]*Execution, 0)
	for _, execution := range m.executions {
		if filter != nil {
			if filter.Status != "" && execution.Status != filter.Status {
				continue
			}
		}
		result = append(result, execution)
	}
	return result, nil
}

func (m *mockAgentRepository) UpdateExecution(ctx context.Context, execution *Execution) error {
	m.executions[execution.ID] = execution
	return nil
}

func (m *mockAgentRepository) CreateAgent(ctx context.Context, agent *Agent) error {
	m.agents[agent.ID] = agent
	return nil
}

func (m *mockAgentRepository) GetAgent(ctx context.Context, id uuid.UUID) (*Agent, error) {
	if agent, ok := m.agents[id]; ok {
		return agent, nil
	}
	return nil, fmt.Errorf("agent not found")
}

func (m *mockAgentRepository) ListAgents(ctx context.Context, filter *AgentFilter) ([]*Agent, error) {
	result := make([]*Agent, 0)
	for _, agent := range m.agents {
		result = append(result, agent)
	}
	return result, nil
}

func (m *mockAgentRepository) UpdateAgent(ctx context.Context, agent *Agent) error {
	m.agents[agent.ID] = agent
	return nil
}

func (m *mockAgentRepository) DeleteAgent(ctx context.Context, id uuid.UUID) error {
	delete(m.agents, id)
	return nil
}

func (m *mockAgentRepository) CreateResource(ctx context.Context, resource *Resource) error {
	m.resources[resource.ID] = resource
	return nil
}

func (m *mockAgentRepository) GetResource(ctx context.Context, id uuid.UUID) (*Resource, error) {
	if resource, ok := m.resources[id]; ok {
		return resource, nil
	}
	return nil, fmt.Errorf("resource not found")
}

func (m *mockAgentRepository) ListResources(ctx context.Context, filter *ResourceFilter) ([]*Resource, error) {
	result := make([]*Resource, 0)
	for _, resource := range m.resources {
		result = append(result, resource)
	}
	return result, nil
}

func (m *mockAgentRepository) UpdateResource(ctx context.Context, resource *Resource) error {
	m.resources[resource.ID] = resource
	return nil
}

func (m *mockAgentRepository) DeleteResource(ctx context.Context, id uuid.UUID) error {
	delete(m.resources, id)
	return nil
}

func (m *mockAgentRepository) CreateResourceQuota(ctx context.Context, quota *ResourceQuota) error {
	m.quotas[quota.ID] = quota
	return nil
}

func (m *mockAgentRepository) GetResourceQuota(ctx context.Context, id uuid.UUID) (*ResourceQuota, error) {
	if quota, ok := m.quotas[id]; ok {
		return quota, nil
	}
	return nil, fmt.Errorf("resource quota not found")
}

func (m *mockAgentRepository) UpdateResourceQuota(ctx context.Context, quota *ResourceQuota) error {
	m.quotas[quota.ID] = quota
	return nil
}

func (m *mockAgentRepository) CreateEvent(ctx context.Context, event *AgentEvent) error {
	m.events = append(m.events, *event)
	return nil
}

func (m *mockAgentRepository) ListEvents(ctx context.Context, filter *EventFilter) ([]*AgentEvent, error) {
	result := make([]*AgentEvent, 0)
	for i := range m.events {
		result = append(result, &m.events[i])
	}
	return result, nil
}

func TestNewSandboxManager(t *testing.T) {
	tests := []struct {
		name   string
		config *SandboxConfig
	}{
		{
			name:   "nil config uses defaults",
			config: nil,
		},
		{
			name: "custom config",
			config: &SandboxConfig{
				MaxSandboxes:        100,
				DefaultCPULimit:     2.0,
				DefaultMemoryLimit:  4096,
				DefaultTimeout:      30 * time.Minute,
				EnableNetwork:       true,
				EnableGPU:           false,
				AllowedImages:       []string{"python:3.9", "golang:1.19"},
				SecurityProfile:     "restricted",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockAgentRepository()
			manager := NewSandboxManager(repo, tt.config)

			if manager == nil {
				t.Error("expected non-nil SandboxManager")
			}
		})
	}
}

func TestCreateSandbox(t *testing.T) {
	tests := []struct {
		name    string
		sandbox *Sandbox
		wantErr bool
	}{
		{
			name: "valid sandbox",
			sandbox: &Sandbox{
				Name:        "test-sandbox-1",
				Image:       "python:3.9",
				CPULimit:    1.0,
				MemoryLimit: 2048,
				Timeout:     30 * time.Minute,
				TenantID:    uuid.New(),
			},
			wantErr: false,
		},
		{
			name: "sandbox with custom config",
			sandbox: &Sandbox{
				Name:        "test-sandbox-2",
				Image:       "golang:1.19",
				CPULimit:    2.0,
				MemoryLimit: 4096,
				Timeout:     60 * time.Minute,
				Env:         map[string]string{"DEBUG": "true"},
				TenantID:    uuid.New(),
			},
			wantErr: false,
		},
		{
			name: "sandbox with existing ID",
			sandbox: &Sandbox{
				ID:          uuid.New(),
				Name:        "existing-sandbox",
				Image:       "python:3.9",
				CPULimit:    1.0,
				MemoryLimit: 2048,
				TenantID:    uuid.New(),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			repo := newMockAgentRepository()
			manager := NewSandboxManager(repo, nil)

			err := manager.CreateSandbox(ctx, tt.sandbox)

			if (err != nil) != tt.wantErr {
				t.Errorf("CreateSandbox() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if tt.sandbox.ID == uuid.Nil {
					t.Error("expected ID to be set")
				}
				if tt.sandbox.CreatedAt.IsZero() {
					t.Error("expected CreatedAt to be set")
				}
				if tt.sandbox.Status == "" {
					t.Error("expected default status to be set")
				}
			}
		})
	}
}

func TestGetSandbox(t *testing.T) {
	ctx := context.Background()
	repo := newMockAgentRepository()
	manager := NewSandboxManager(repo, nil)

	sandbox := &Sandbox{
		Name:        "test-sandbox",
		Image:       "python:3.9",
		CPULimit:    1.0,
		MemoryLimit: 2048,
		TenantID:    uuid.New(),
	}

	if err := manager.CreateSandbox(ctx, sandbox); err != nil {
		t.Fatalf("failed to create sandbox: %v", err)
	}

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr bool
	}{
		{
			name:    "existing sandbox",
			id:      sandbox.ID,
			wantErr: false,
		},
		{
			name:    "non-existing sandbox",
			id:      uuid.New(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := manager.GetSandbox(ctx, tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetSandbox() error = %v, wantErr %v", err, tt.wantErr)
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

func TestUpdateSandbox(t *testing.T) {
	ctx := context.Background()
	repo := newMockAgentRepository()
	manager := NewSandboxManager(repo, nil)

	sandbox := &Sandbox{
		Name:        "test-sandbox",
		Image:       "python:3.9",
		CPULimit:    1.0,
		MemoryLimit: 2048,
		TenantID:    uuid.New(),
	}

	if err := manager.CreateSandbox(ctx, sandbox); err != nil {
		t.Fatalf("failed to create sandbox: %v", err)
	}

	tests := []struct {
		name    string
		id      uuid.UUID
		updates *SandboxUpdate
		wantErr bool
	}{
		{
			name: "update CPU and memory",
			id:   sandbox.ID,
			updates: &SandboxUpdate{
				CPULimit:    float64Ptr(2.0),
				MemoryLimit: int64Ptr(4096),
			},
			wantErr: false,
		},
		{
			name: "update status",
			id:   sandbox.ID,
			updates: &SandboxUpdate{
				Status: sandboxStatusPtr(SandboxStatusRunning),
			},
			wantErr: false,
		},
		{
			name:    "non-existing sandbox",
			id:      uuid.New(),
			updates: &SandboxUpdate{CPULimit: float64Ptr(1.5)},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.UpdateSandbox(ctx, tt.id, tt.updates)

			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateSandbox() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				updated, err := manager.GetSandbox(ctx, tt.id)
				if err != nil {
					t.Fatalf("failed to get updated sandbox: %v", err)
				}

				if tt.updates.CPULimit != nil && updated.CPULimit != *tt.updates.CPULimit {
					t.Errorf("expected CPULimit %f, got %f", *tt.updates.CPULimit, updated.CPULimit)
				}
				if tt.updates.MemoryLimit != nil && updated.MemoryLimit != *tt.updates.MemoryLimit {
					t.Errorf("expected MemoryLimit %d, got %d", *tt.updates.MemoryLimit, updated.MemoryLimit)
				}
			}
		})
	}
}

func TestDeleteSandbox(t *testing.T) {
	ctx := context.Background()
	repo := newMockAgentRepository()
	manager := NewSandboxManager(repo, nil)

	sandbox := &Sandbox{
		Name:        "test-sandbox",
		Image:       "python:3.9",
		CPULimit:    1.0,
		MemoryLimit: 2048,
		TenantID:    uuid.New(),
	}

	if err := manager.CreateSandbox(ctx, sandbox); err != nil {
		t.Fatalf("failed to create sandbox: %v", err)
	}

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr bool
	}{
		{
			name:    "delete existing sandbox",
			id:      sandbox.ID,
			wantErr: false,
		},
		{
			name:    "delete non-existing sandbox",
			id:      uuid.New(),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.DeleteSandbox(ctx, tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteSandbox() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStartSandbox(t *testing.T) {
	ctx := context.Background()
	repo := newMockAgentRepository()
	manager := NewSandboxManager(repo, nil)

	sandbox := &Sandbox{
		Name:        "test-sandbox",
		Image:       "python:3.9",
		CPULimit:    1.0,
		MemoryLimit: 2048,
		Status:      SandboxStatusCreated,
		TenantID:    uuid.New(),
	}

	if err := manager.CreateSandbox(ctx, sandbox); err != nil {
		t.Fatalf("failed to create sandbox: %v", err)
	}

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr bool
	}{
		{
			name:    "start valid sandbox",
			id:      sandbox.ID,
			wantErr: false,
		},
		{
			name:    "start non-existing sandbox",
			id:      uuid.New(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.StartSandbox(ctx, tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("StartSandbox() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStopSandbox(t *testing.T) {
	ctx := context.Background()
	repo := newMockAgentRepository()
	manager := NewSandboxManager(repo, nil)

	sandbox := &Sandbox{
		Name:        "test-sandbox",
		Image:       "python:3.9",
		CPULimit:    1.0,
		MemoryLimit: 2048,
		Status:      SandboxStatusRunning,
		TenantID:    uuid.New(),
	}

	if err := manager.CreateSandbox(ctx, sandbox); err != nil {
		t.Fatalf("failed to create sandbox: %v", err)
	}

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr bool
	}{
		{
			name:    "stop running sandbox",
			id:      sandbox.ID,
			wantErr: false,
		},
		{
			name:    "stop non-existing sandbox",
			id:      uuid.New(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.StopSandbox(ctx, tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("StopSandbox() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPauseSandbox(t *testing.T) {
	ctx := context.Background()
	repo := newMockAgentRepository()
	manager := NewSandboxManager(repo, nil)

	sandbox := &Sandbox{
		Name:        "test-sandbox",
		Image:       "python:3.9",
		CPULimit:    1.0,
		MemoryLimit: 2048,
		Status:      SandboxStatusRunning,
		TenantID:    uuid.New(),
	}

	if err := manager.CreateSandbox(ctx, sandbox); err != nil {
		t.Fatalf("failed to create sandbox: %v", err)
	}

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr bool
	}{
		{
			name:    "pause running sandbox",
			id:      sandbox.ID,
			wantErr: false,
		},
		{
			name:    "pause non-existing sandbox",
			id:      uuid.New(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.PauseSandbox(ctx, tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("PauseSandbox() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestResumeSandbox(t *testing.T) {
	ctx := context.Background()
	repo := newMockAgentRepository()
	manager := NewSandboxManager(repo, nil)

	sandbox := &Sandbox{
		Name:        "test-sandbox",
		Image:       "python:3.9",
		CPULimit:    1.0,
		MemoryLimit: 2048,
		Status:      SandboxStatusPaused,
		TenantID:    uuid.New(),
	}

	if err := manager.CreateSandbox(ctx, sandbox); err != nil {
		t.Fatalf("failed to create sandbox: %v", err)
	}

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr bool
	}{
		{
			name:    "resume paused sandbox",
			id:      sandbox.ID,
			wantErr: false,
		},
		{
			name:    "resume non-existing sandbox",
			id:      uuid.New(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.ResumeSandbox(ctx, tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("ResumeSandbox() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetSandboxStatus(t *testing.T) {
	ctx := context.Background()
	repo := newMockAgentRepository()
	manager := NewSandboxManager(repo, nil)

	sandbox := &Sandbox{
		Name:        "test-sandbox",
		Image:       "python:3.9",
		CPULimit:    1.0,
		MemoryLimit: 2048,
		Status:      SandboxStatusRunning,
		TenantID:    uuid.New(),
	}

	if err := manager.CreateSandbox(ctx, sandbox); err != nil {
		t.Fatalf("failed to create sandbox: %v", err)
	}

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr bool
	}{
		{
			name:    "existing sandbox",
			id:      sandbox.ID,
			wantErr: false,
		},
		{
			name:    "non-existing sandbox",
			id:      uuid.New(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, err := manager.GetSandboxStatus(ctx, tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetSandboxStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if status == nil {
					t.Error("expected non-nil status")
					return
				}
				if status.ID != tt.id {
					t.Errorf("expected ID %s, got %s", tt.id, status.ID)
				}
			}
		})
	}
}

func TestGetSandboxLogs(t *testing.T) {
	ctx := context.Background()
	repo := newMockAgentRepository()
	manager := NewSandboxManager(repo, nil)

	sandbox := &Sandbox{
		Name:        "test-sandbox",
		Image:       "python:3.9",
		CPULimit:    1.0,
		MemoryLimit: 2048,
		Status:      SandboxStatusRunning,
		TenantID:    uuid.New(),
	}

	if err := manager.CreateSandbox(ctx, sandbox); err != nil {
		t.Fatalf("failed to create sandbox: %v", err)
	}

	tests := []struct {
		name    string
		id      uuid.UUID
		tail    int
		wantErr bool
	}{
		{
			name:    "get logs with tail",
			id:      sandbox.ID,
			tail:    100,
			wantErr: false,
		},
		{
			name:    "get logs without tail",
			id:      sandbox.ID,
			tail:    0,
			wantErr: false,
		},
		{
			name:    "non-existing sandbox",
			id:      uuid.New(),
			tail:    100,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logs, err := manager.GetSandboxLogs(ctx, tt.id, tt.tail)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetSandboxLogs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && logs == nil {
				t.Error("expected non-nil logs")
			}
		})
	}
}

func TestExecuteInSandbox(t *testing.T) {
	ctx := context.Background()
	repo := newMockAgentRepository()
	manager := NewSandboxManager(repo, nil)

	sandbox := &Sandbox{
		Name:        "test-sandbox",
		Image:       "python:3.9",
		CPULimit:    1.0,
		MemoryLimit: 2048,
		Status:      SandboxStatusRunning,
		TenantID:    uuid.New(),
	}

	if err := manager.CreateSandbox(ctx, sandbox); err != nil {
		t.Fatalf("failed to create sandbox: %v", err)
	}

	tests := []struct {
		name    string
		id      uuid.UUID
		cmd     *ExecutionCommand
		wantErr bool
	}{
		{
			name: "valid command",
			id:   sandbox.ID,
			cmd: &ExecutionCommand{
				Command: "python",
				Args:    []string{"-c", "print('hello')"},
				Timeout: 30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "command with environment",
			id:   sandbox.ID,
			cmd: &ExecutionCommand{
				Command: "python",
				Args:    []string{"-c", "import os; print(os.environ.get('TEST'))"},
				Env:     map[string]string{"TEST": "value"},
				Timeout: 30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "non-existing sandbox",
			id:   uuid.New(),
			cmd: &ExecutionCommand{
				Command: "echo",
				Args:    []string{"test"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := manager.ExecuteInSandbox(ctx, tt.id, tt.cmd)

			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteInSandbox() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if result == nil {
					t.Error("expected non-nil result")
					return
				}
				if result.SandboxID != tt.id {
					t.Errorf("expected SandboxID %s, got %s", tt.id, result.SandboxID)
				}
			}
		})
	}
}

func TestListSandboxes(t *testing.T) {
	ctx := context.Background()
	repo := newMockAgentRepository()
	manager := NewSandboxManager(repo, nil)

	sandbox1 := &Sandbox{
		Name:        "sandbox-1",
		Image:       "python:3.9",
		CPULimit:    1.0,
		MemoryLimit: 2048,
		Status:      SandboxStatusRunning,
		TenantID:    uuid.New(),
	}
	sandbox2 := &Sandbox{
		Name:        "sandbox-2",
		Image:       "golang:1.19",
		CPULimit:    2.0,
		MemoryLimit: 4096,
		Status:      SandboxStatusCreated,
		TenantID:    uuid.New(),
	}

	manager.CreateSandbox(ctx, sandbox1)
	manager.CreateSandbox(ctx, sandbox2)

	tests := []struct {
		name   string
		filter *SandboxFilter
		count  int
	}{
		{
			name:   "list all sandboxes",
			filter: nil,
			count:  2,
		},
		{
			name:   "filter by running status",
			filter: &SandboxFilter{Status: SandboxStatusRunning},
			count:  1,
		},
		{
			name:   "filter by created status",
			filter: &SandboxFilter{Status: SandboxStatusCreated},
			count:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sandboxes, err := manager.ListSandboxes(ctx, tt.filter)

			if err != nil {
				t.Errorf("ListSandboxes() error = %v", err)
				return
			}

			if len(sandboxes) != tt.count {
				t.Errorf("expected %d sandboxes, got %d", tt.count, len(sandboxes))
			}
		})
	}
}

func TestGetSandboxMetrics(t *testing.T) {
	ctx := context.Background()
	repo := newMockAgentRepository()
	manager := NewSandboxManager(repo, nil)

	sandbox := &Sandbox{
		Name:        "test-sandbox",
		Image:       "python:3.9",
		CPULimit:    1.0,
		MemoryLimit: 2048,
		Status:      SandboxStatusRunning,
		TenantID:    uuid.New(),
	}

	if err := manager.CreateSandbox(ctx, sandbox); err != nil {
		t.Fatalf("failed to create sandbox: %v", err)
	}

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr bool
	}{
		{
			name:    "existing sandbox",
			id:      sandbox.ID,
			wantErr: false,
		},
		{
			name:    "non-existing sandbox",
			id:      uuid.New(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics, err := manager.GetSandboxMetrics(ctx, tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetSandboxMetrics() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if metrics == nil {
					t.Error("expected non-nil metrics")
					return
				}
				if metrics.SandboxID != tt.id {
					t.Errorf("expected SandboxID %s, got %s", tt.id, metrics.SandboxID)
				}
			}
		})
	}
}

func TestSandboxStatusTransitions(t *testing.T) {
	ctx := context.Background()
	repo := newMockAgentRepository()
	manager := NewSandboxManager(repo, nil)

	tests := []struct {
		name          string
		initialStatus SandboxStatus
		action        func(uuid.UUID) error
		wantErr       bool
	}{
		{
			name:          "start from created",
			initialStatus: SandboxStatusCreated,
			action: func(id uuid.UUID) error {
				return manager.StartSandbox(ctx, id)
			},
			wantErr: false,
		},
		{
			name:          "stop from running",
			initialStatus: SandboxStatusRunning,
			action: func(id uuid.UUID) error {
				return manager.StopSandbox(ctx, id)
			},
			wantErr: false,
		},
		{
			name:          "pause from running",
			initialStatus: SandboxStatusRunning,
			action: func(id uuid.UUID) error {
				return manager.PauseSandbox(ctx, id)
			},
			wantErr: false,
		},
		{
			name:          "resume from paused",
			initialStatus: SandboxStatusPaused,
			action: func(id uuid.UUID) error {
				return manager.ResumeSandbox(ctx, id)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sandbox := &Sandbox{
				Name:        "test-sandbox",
				Image:       "python:3.9",
				CPULimit:    1.0,
				MemoryLimit: 2048,
				Status:      tt.initialStatus,
				TenantID:    uuid.New(),
			}

			if err := manager.CreateSandbox(ctx, sandbox); err != nil {
				t.Fatalf("failed to create sandbox: %v", err)
			}

			err := tt.action(sandbox.ID)

			if (err != nil) != tt.wantErr {
				t.Errorf("action() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSandboxResourceLimits(t *testing.T) {
	ctx := context.Background()
	config := &SandboxConfig{
		MaxCPULimit:     8.0,
		MaxMemoryLimit:  16384,
		MaxTimeout:      2 * time.Hour,
	}
	repo := newMockAgentRepository()
	manager := NewSandboxManager(repo, config)

	tests := []struct {
		name    string
		sandbox *Sandbox
		wantErr bool
	}{
		{
			name: "within limits",
			sandbox: &Sandbox{
				Name:        "valid-sandbox",
				Image:       "python:3.9",
				CPULimit:    4.0,
				MemoryLimit: 8192,
				Timeout:     1 * time.Hour,
				TenantID:    uuid.New(),
			},
			wantErr: false,
		},
		{
			name: "CPU exceeds limit",
			sandbox: &Sandbox{
				Name:        "cpu-exceed",
				Image:       "python:3.9",
				CPULimit:    10.0,
				MemoryLimit: 2048,
				TenantID:    uuid.New(),
			},
			wantErr: true,
		},
		{
			name: "memory exceeds limit",
			sandbox: &Sandbox{
				Name:        "memory-exceed",
				Image:       "python:3.9",
				CPULimit:    1.0,
				MemoryLimit: 32768,
				TenantID:    uuid.New(),
			},
			wantErr: true,
		},
		{
			name: "timeout exceeds limit",
			sandbox: &Sandbox{
				Name:        "timeout-exceed",
				Image:       "python:3.9",
				CPULimit:    1.0,
				MemoryLimit: 2048,
				Timeout:     4 * time.Hour,
				TenantID:    uuid.New(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.CreateSandbox(ctx, tt.sandbox)

			if (err != nil) != tt.wantErr {
				t.Errorf("CreateSandbox() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func int64Ptr(i int64) *int64 {
	return &i
}

func sandboxStatusPtr(s SandboxStatus) *SandboxStatus {
	return &s
}
