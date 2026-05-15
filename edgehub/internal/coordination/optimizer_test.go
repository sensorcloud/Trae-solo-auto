package coordination

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewOptimizer(t *testing.T) {
	tests := []struct {
		name   string
		config *OptimizationConfig
	}{
		{
			name:   "nil config uses defaults",
			config: nil,
		},
		{
			name: "custom config",
			config: &OptimizationConfig{
				Objectives: []OptimizationObjective{
					ObjectiveMinCost,
					ObjectiveMaxGreen,
				},
				Weights: OptimizationWeights{
					CostWeight:        0.4,
					CarbonWeight:      0.3,
					GreenWeight:       0.2,
					ReliabilityWeight: 0.1,
				},
				Constraints: OptimizationConstraints{
					MaxCostPerTask:   100,
					MaxCarbonPerTask: 50,
					MinGreenRatio:    0.5,
					MinReliability:   0.95,
				},
				Horizon:              24,
				TimeResolution:       15,
				MaxIterations:        1000,
				ConvergenceThreshold: 0.001,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockCoordinationRepository()
			optimizer := NewOptimizer(repo, tt.config)

			if optimizer == nil {
				t.Error("expected non-nil Optimizer")
			}
		})
	}
}

func TestOptimizeTaskSchedule(t *testing.T) {
	ctx := context.Background()
	repo := newMockCoordinationRepository()
	optimizer := NewOptimizer(repo, nil)

	now := time.Now()

	tests := []struct {
		name    string
		task    *ComputeTask
		wantErr bool
	}{
		{
			name: "realtime task optimization",
			task: &ComputeTask{
				ID:              uuid.New(),
				Name:            "realtime-task",
				Type:            TaskTypeRealtime,
				Priority:        TaskPriorityHigh,
				ResourceSpec:    TaskResourceSpec{CPU: 2.0, Memory: 4096, PowerCapacity: 100},
				EstimatedPower:  100,
				EstimatedDuration: 60,
				TenantID:        uuid.New(),
			},
			wantErr: false,
		},
		{
			name: "delayable task optimization",
			task: &ComputeTask{
				ID:              uuid.New(),
				Name:            "delayable-task",
				Type:            TaskTypeDelayable,
				Priority:        TaskPriorityMedium,
				ResourceSpec:    TaskResourceSpec{CPU: 4.0, Memory: 8192, PowerCapacity: 200},
				TimeConstraint:  TimeConstraint{IsFlexible: true, MaxDelayMinutes: 120},
				EnergyPreference: EnergyPreference{PreferGreen: true},
				EstimatedPower:  200,
				EstimatedDuration: 120,
				TenantID:        uuid.New(),
			},
			wantErr: false,
		},
		{
			name: "batch task optimization",
			task: &ComputeTask{
				ID:              uuid.New(),
				Name:            "batch-task",
				Type:            TaskTypeBatch,
				Priority:        TaskPriorityLow,
				ResourceSpec:    TaskResourceSpec{CPU: 8.0, Memory: 16384, PowerCapacity: 500},
				TimeConstraint:  TimeConstraint{IsFlexible: true},
				EstimatedPower:  500,
				EstimatedDuration: 360,
				TenantID:        uuid.New(),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := optimizer.OptimizeTaskSchedule(ctx, tt.task, now, now.Add(24*time.Hour))

			if (err != nil) != tt.wantErr {
				t.Errorf("OptimizeTaskSchedule() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if result == nil {
					t.Error("expected non-nil result")
					return
				}
				if result.TaskID != tt.task.ID {
					t.Errorf("expected TaskID %s, got %s", tt.task.ID, result.TaskID)
				}
				if result.Score < 0 || result.Score > 1 {
					t.Errorf("expected Score between 0 and 1, got %f", result.Score)
				}
			}
		})
	}
}

func TestCalculateOptimizationScore(t *testing.T) {
	repo := newMockCoordinationRepository()
	optimizer := NewOptimizer(repo, nil)

	tests := []struct {
		name   string
		result *OptimizationResult
		min    float64
		max    float64
	}{
		{
			name: "high score result",
			result: &OptimizationResult{
				TotalCost:   10.0,
				TotalCarbon: 5.0,
				GreenRatio:  0.9,
				Reliability: 0.98,
			},
			min: 0.7,
			max: 1.0,
		},
		{
			name: "medium score result",
			result: &OptimizationResult{
				TotalCost:   50.0,
				TotalCarbon: 30.0,
				GreenRatio:  0.5,
				Reliability: 0.90,
			},
			min: 0.3,
			max: 0.7,
		},
		{
			name: "low score result",
			result: &OptimizationResult{
				TotalCost:   100.0,
				TotalCarbon: 80.0,
				GreenRatio:  0.2,
				Reliability: 0.80,
			},
			min: 0.0,
			max: 0.4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := optimizer.CalculateOptimizationScore(tt.result)

			if score < tt.min || score > tt.max {
				t.Errorf("expected Score between %f and %f, got %f", tt.min, tt.max, score)
			}
		})
	}
}

func TestFindOptimalTimeWindow(t *testing.T) {
	ctx := context.Background()
	repo := newMockCoordinationRepository()
	optimizer := NewOptimizer(repo, nil)

	now := time.Now()

	tests := []struct {
		name     string
		request  *OptimalTimeRequest
		wantErr  bool
	}{
		{
			name: "find window for cost optimization",
			request: &OptimalTimeRequest{
				Region:           "cn-east",
				Duration:         60,
				PowerRequirement: 100,
				Objective:        ObjectiveMinCost,
				StartTime:        now,
				EndTime:          now.Add(24 * time.Hour),
			},
			wantErr: false,
		},
		{
			name: "find window for green optimization",
			request: &OptimalTimeRequest{
				Region:           "cn-east",
				Duration:         120,
				PowerRequirement: 200,
				Objective:        ObjectiveMaxGreen,
				StartTime:        now,
				EndTime:          now.Add(48 * time.Hour),
			},
			wantErr: false,
		},
		{
			name: "find window for carbon optimization",
			request: &OptimalTimeRequest{
				Region:           "cn-east",
				Duration:         30,
				PowerRequirement: 50,
				Objective:        ObjectiveMinCarbon,
				StartTime:        now,
				EndTime:          now.Add(12 * time.Hour),
			},
			wantErr: false,
		},
		{
			name: "find window for balanced optimization",
			request: &OptimalTimeRequest{
				Region:           "cn-east",
				Duration:         90,
				PowerRequirement: 150,
				Objective:        ObjectiveBalanced,
				StartTime:        now,
				EndTime:          now.Add(24 * time.Hour),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			window, err := optimizer.FindOptimalTimeWindow(ctx, tt.request)

			if (err != nil) != tt.wantErr {
				t.Errorf("FindOptimalTimeWindow() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if window == nil {
					t.Error("expected non-nil window")
					return
				}
				if window.StartTime.Before(tt.request.StartTime) {
					t.Error("window start time before request start time")
				}
				if window.EndTime.After(tt.request.EndTime) {
					t.Error("window end time after request end time")
				}
			}
		})
	}
}

func TestEvaluateEnergyOptions(t *testing.T) {
	ctx := context.Background()
	repo := newMockCoordinationRepository()
	optimizer := NewOptimizer(repo, nil)

	resource1 := &EnergyResource{
		ID:              uuid.New(),
		Name:            "Solar Farm",
		Type:            EnergyResourceSolar,
		Status:          EnergyResourceStatusOnline,
		Capacity:        1000,
		AvailableCapacity: 800,
		CurrentOutput:   600,
		PricePerKWh:     0.3,
		CarbonIntensity: 0,
		GreenRatio:      1.0,
		Reliability:     0.85,
	}

	resource2 := &EnergyResource{
		ID:              uuid.New(),
		Name:            "Wind Farm",
		Type:            EnergyResourceWind,
		Status:          EnergyResourceStatusOnline,
		Capacity:        2000,
		AvailableCapacity: 1500,
		CurrentOutput:   1200,
		PricePerKWh:     0.25,
		CarbonIntensity: 0,
		GreenRatio:      1.0,
		Reliability:     0.90,
	}

	repo.CreateEnergyResource(ctx, resource1)
	repo.CreateEnergyResource(ctx, resource2)

	tests := []struct {
		name    string
		request *EnergyEvaluationRequest
		wantErr bool
	}{
		{
			name: "evaluate for 100kW",
			request: &EnergyEvaluationRequest{
				Region:   "cn-east",
				Power:    100,
				Duration: 60,
			},
			wantErr: false,
		},
		{
			name: "evaluate for 500kW",
			request: &EnergyEvaluationRequest{
				Region:   "cn-east",
				Power:    500,
				Duration: 120,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options, err := optimizer.EvaluateEnergyOptions(ctx, tt.request)

			if (err != nil) != tt.wantErr {
				t.Errorf("EvaluateEnergyOptions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if options == nil {
					t.Error("expected non-nil options")
				}
			}
		})
	}
}

func TestOptimizeStorageUsage(t *testing.T) {
	ctx := context.Background()
	repo := newMockCoordinationRepository()
	optimizer := NewOptimizer(repo, nil)

	storage := &StorageResource{
		ID:               uuid.New(),
		Name:             "Battery Storage",
		Status:           "idle",
		Capacity:         1000,
		AvailableEnergy:  500,
		SOC:              0.5,
		MinSOC:           0.1,
		MaxSOC:           0.95,
		MaxChargeRate:    500,
		MaxDischargeRate: 500,
		Efficiency:       0.95,
	}

	repo.CreateStorageResource(ctx, storage)

	tests := []struct {
		name    string
		request *StorageOptimizationRequest
		wantErr bool
	}{
		{
			name: "optimize for cost arbitrage",
			request: &StorageOptimizationRequest{
				StorageID: storage.ID,
				Mode:      "arbitrage",
				Horizon:   24,
			},
			wantErr: false,
		},
		{
			name: "optimize for peak shaving",
			request: &StorageOptimizationRequest{
				StorageID: storage.ID,
				Mode:      "peak_shaving",
				Horizon:   24,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schedule, err := optimizer.OptimizeStorageUsage(ctx, tt.request)

			if (err != nil) != tt.wantErr {
				t.Errorf("OptimizeStorageUsage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && schedule == nil {
				t.Error("expected non-nil schedule")
			}
		})
	}
}

func TestCalculateCarbonFootprint(t *testing.T) {
	repo := newMockCoordinationRepository()
	optimizer := NewOptimizer(repo, nil)

	tests := []struct {
		name     string
		power    float64
		duration int
		source   EnergyResourceType
		wantMin  float64
		wantMax  float64
	}{
		{
			name:     "solar energy",
			power:    100,
			duration: 60,
			source:   EnergyResourceSolar,
			wantMin:  0,
			wantMax:  10,
		},
		{
			name:     "wind energy",
			power:    200,
			duration: 120,
			source:   EnergyResourceWind,
			wantMin:  0,
			wantMax:  20,
		},
		{
			name:     "grid energy",
			power:    100,
			duration: 60,
			source:   EnergyResourceGrid,
			wantMin:  10,
			wantMax:  100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			carbon := optimizer.CalculateCarbonFootprint(tt.power, tt.duration, tt.source)

			if carbon < tt.wantMin || carbon > tt.wantMax {
				t.Errorf("expected carbon between %f and %f, got %f", tt.wantMin, tt.wantMax, carbon)
			}
		})
	}
}

func TestCalculateEnergyCost(t *testing.T) {
	repo := newMockCoordinationRepository()
	optimizer := NewOptimizer(repo, nil)

	tests := []struct {
		name     string
		power    float64
		duration int
		price    float64
		want     float64
	}{
		{
			name:     "100kW for 1 hour at 0.5/kWh",
			power:    100,
			duration: 60,
			price:    0.5,
			want:     50.0,
		},
		{
			name:     "200kW for 2 hours at 0.3/kWh",
			power:    200,
			duration: 120,
			price:    0.3,
			want:     120.0,
		},
		{
			name:     "50kW for 30 minutes at 0.4/kWh",
			power:    50,
			duration: 30,
			price:    0.4,
			want:     10.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost := optimizer.CalculateEnergyCost(tt.power, tt.duration, tt.price)

			if cost != tt.want {
				t.Errorf("expected cost %f, got %f", tt.want, cost)
			}
		})
	}
}

func TestGetOptimizationObjectives(t *testing.T) {
	repo := newMockCoordinationRepository()
	config := &OptimizationConfig{
		Objectives: []OptimizationObjective{
			ObjectiveMinCost,
			ObjectiveMaxGreen,
			ObjectiveMinCarbon,
		},
		Weights: OptimizationWeights{
			CostWeight:   0.5,
			GreenWeight:  0.3,
			CarbonWeight: 0.2,
		},
	}
	optimizer := NewOptimizer(repo, config)

	objectives := optimizer.GetOptimizationObjectives()

	if len(objectives) != 3 {
		t.Errorf("expected 3 objectives, got %d", len(objectives))
	}
}

func TestSetOptimizationWeights(t *testing.T) {
	repo := newMockCoordinationRepository()
	optimizer := NewOptimizer(repo, nil)

	tests := []struct {
		name    string
		weights OptimizationWeights
		wantErr bool
	}{
		{
			name: "valid weights",
			weights: OptimizationWeights{
				CostWeight:        0.4,
				CarbonWeight:      0.3,
				GreenWeight:       0.2,
				ReliabilityWeight: 0.1,
			},
			wantErr: false,
		},
		{
			name: "weights sum to 1",
			weights: OptimizationWeights{
				CostWeight:   0.5,
				GreenWeight:  0.5,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := optimizer.SetOptimizationWeights(tt.weights)

			if (err != nil) != tt.wantErr {
				t.Errorf("SetOptimizationWeights() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSetOptimizationConstraints(t *testing.T) {
	repo := newMockCoordinationRepository()
	optimizer := NewOptimizer(repo, nil)

	tests := []struct {
		name        string
		constraints OptimizationConstraints
		wantErr     bool
	}{
		{
			name: "valid constraints",
			constraints: OptimizationConstraints{
				MaxCostPerTask:   100,
				MaxCarbonPerTask: 50,
				MinGreenRatio:    0.5,
				MinReliability:   0.95,
			},
			wantErr: false,
		},
		{
			name: "strict constraints",
			constraints: OptimizationConstraints{
				MaxCostPerTask:   10,
				MaxCarbonPerTask: 5,
				MinGreenRatio:    0.9,
				MinReliability:   0.99,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := optimizer.SetOptimizationConstraints(tt.constraints)

			if (err != nil) != tt.wantErr {
				t.Errorf("SetOptimizationConstraints() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPredictOptimalSchedule(t *testing.T) {
	ctx := context.Background()
	repo := newMockCoordinationRepository()
	optimizer := NewOptimizer(repo, nil)

	now := time.Now()

	tests := []struct {
		name    string
		tasks   []*ComputeTask
		wantErr bool
	}{
		{
			name: "single task prediction",
			tasks: []*ComputeTask{
				{
					ID:              uuid.New(),
					Name:            "task-1",
					Type:            TaskTypeDelayable,
					Priority:        TaskPriorityMedium,
					EstimatedPower:  100,
					EstimatedDuration: 60,
					TenantID:        uuid.New(),
				},
			},
			wantErr: false,
		},
		{
			name: "multiple tasks prediction",
			tasks: []*ComputeTask{
				{
					ID:              uuid.New(),
					Name:            "task-1",
					Type:            TaskTypeDelayable,
					Priority:        TaskPriorityHigh,
					EstimatedPower:  100,
					EstimatedDuration: 60,
					TenantID:        uuid.New(),
				},
				{
					ID:              uuid.New(),
					Name:            "task-2",
					Type:            TaskTypeBatch,
					Priority:        TaskPriorityLow,
					EstimatedPower:  200,
					EstimatedDuration: 120,
					TenantID:        uuid.New(),
				},
			},
			wantErr: false,
		},
		{
			name:    "empty tasks",
			tasks:   []*ComputeTask{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schedule, err := optimizer.PredictOptimalSchedule(ctx, tt.tasks, now, now.Add(24*time.Hour))

			if (err != nil) != tt.wantErr {
				t.Errorf("PredictOptimalSchedule() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if schedule == nil {
					t.Error("expected non-nil schedule")
				}
			}
		})
	}
}

func TestCompareOptimizationResults(t *testing.T) {
	repo := newMockCoordinationRepository()
	optimizer := NewOptimizer(repo, nil)

	result1 := &OptimizationResult{
		TaskID:      uuid.New(),
		TotalCost:   50.0,
		TotalCarbon: 25.0,
		GreenRatio:  0.8,
		Reliability: 0.95,
		Score:       0.85,
	}

	result2 := &OptimizationResult{
		TaskID:      uuid.New(),
		TotalCost:   40.0,
		TotalCarbon: 30.0,
		GreenRatio:  0.6,
		Reliability: 0.92,
		Score:       0.75,
	}

	tests := []struct {
		name         string
		result1      *OptimizationResult
		result2      *OptimizationResult
		objective    OptimizationObjective
		wantBetter   int
	}{
		{
			name:       "compare by score",
			result1:    result1,
			result2:    result2,
			objective:  ObjectiveBalanced,
			wantBetter: 1,
		},
		{
			name:       "compare by cost",
			result1:    result1,
			result2:    result2,
			objective:  ObjectiveMinCost,
			wantBetter: 2,
		},
		{
			name:       "compare by green",
			result1:    result1,
			result2:    result2,
			objective:  ObjectiveMaxGreen,
			wantBetter: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			better := optimizer.CompareOptimizationResults(tt.result1, tt.result2, tt.objective)

			if better != tt.wantBetter {
				t.Errorf("expected better result %d, got %d", tt.wantBetter, better)
			}
		})
	}
}

func TestGetOptimizationHistory(t *testing.T) {
	ctx := context.Background()
	repo := newMockCoordinationRepository()
	optimizer := NewOptimizer(repo, nil)

	now := time.Now()
	start := now.Add(-24 * time.Hour)
	end := now

	tests := []struct {
		name    string
		start   time.Time
		end     time.Time
		wantErr bool
	}{
		{
			name:    "last 24 hours",
			start:   start,
			end:     end,
			wantErr: false,
		},
		{
			name:    "last 7 days",
			start:   now.Add(-7 * 24 * time.Hour),
			end:     end,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			history, err := optimizer.GetOptimizationHistory(ctx, tt.start, tt.end)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetOptimizationHistory() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && history == nil {
				t.Error("expected non-nil history")
			}
		})
	}
}

func TestOptimizationObjectives(t *testing.T) {
	repo := newMockCoordinationRepository()
	optimizer := NewOptimizer(repo, nil)

	tests := []struct {
		name      string
		objective OptimizationObjective
		wantValid bool
	}{
		{
			name:      "min cost objective",
			objective: ObjectiveMinCost,
			wantValid: true,
		},
		{
			name:      "min carbon objective",
			objective: ObjectiveMinCarbon,
			wantValid: true,
		},
		{
			name:      "max green objective",
			objective: ObjectiveMaxGreen,
			wantValid: true,
		},
		{
			name:      "max reliability objective",
			objective: ObjectiveMaxReliability,
			wantValid: true,
		},
		{
			name:      "min latency objective",
			objective: ObjectiveMinLatency,
			wantValid: true,
		},
		{
			name:      "balanced objective",
			objective: ObjectiveBalanced,
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &OptimizationConfig{
				Objectives: []OptimizationObjective{tt.objective},
			}
			optimizer := NewOptimizer(repo, config)

			objectives := optimizer.GetOptimizationObjectives()

			valid := len(objectives) == 1 && objectives[0] == tt.objective
			if valid != tt.wantValid {
				t.Errorf("expected valid %v, got %v", tt.wantValid, valid)
			}
		})
	}
}

func TestBoundaryConditions(t *testing.T) {
	ctx := context.Background()
	repo := newMockCoordinationRepository()
	optimizer := NewOptimizer(repo, nil)

	now := time.Now()

	tests := []struct {
		name    string
		task    *ComputeTask
		wantErr bool
	}{
		{
			name: "zero power task",
			task: &ComputeTask{
				ID:              uuid.New(),
				Name:            "zero-power-task",
				Type:            TaskTypeRealtime,
				EstimatedPower:  0,
				EstimatedDuration: 60,
				TenantID:        uuid.New(),
			},
			wantErr: false,
		},
		{
			name: "zero duration task",
			task: &ComputeTask{
				ID:              uuid.New(),
				Name:            "zero-duration-task",
				Type:            TaskTypeRealtime,
				EstimatedPower:  100,
				EstimatedDuration: 0,
				TenantID:        uuid.New(),
			},
			wantErr: false,
		},
		{
			name: "very high power task",
			task: &ComputeTask{
				ID:              uuid.New(),
				Name:            "high-power-task",
				Type:            TaskTypeBatch,
				EstimatedPower:  10000,
				EstimatedDuration: 360,
				TenantID:        uuid.New(),
			},
			wantErr: false,
		},
		{
			name: "very long duration task",
			task: &ComputeTask{
				ID:              uuid.New(),
				Name:            "long-duration-task",
				Type:            TaskTypeBatch,
				EstimatedPower:  100,
				EstimatedDuration: 8640,
				TenantID:        uuid.New(),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := optimizer.OptimizeTaskSchedule(ctx, tt.task, now, now.Add(24*time.Hour))

			if (err != nil) != tt.wantErr {
				t.Errorf("OptimizeTaskSchedule() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result == nil {
				t.Error("expected non-nil result")
			}
		})
	}
}
