package benchmark

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestBenchmarkService_RunBenchmark(t *testing.T) {
	service := NewBenchmarkService()
	ctx := context.Background()
	nodeID := uuid.New()

	result, err := service.RunBenchmark(ctx, nodeID, BenchmarkTypeCPU)
	if err != nil {
		t.Fatalf("Failed to run benchmark: %v", err)
	}

	if result.NodeID != nodeID {
		t.Errorf("Expected node ID %s, got %s", nodeID, result.NodeID)
	}

	if result.Type != BenchmarkTypeCPU {
		t.Errorf("Expected type %s, got %s", BenchmarkTypeCPU, result.Type)
	}

	if result.Status != BenchmarkStatusRunning {
		t.Errorf("Expected status %s, got %s", BenchmarkStatusRunning, result.Status)
	}
}

func TestBenchmarkService_GetResult(t *testing.T) {
	service := NewBenchmarkService()
	ctx := context.Background()
	nodeID := uuid.New()

	result, _ := service.RunBenchmark(ctx, nodeID, BenchmarkTypeCPU)

	time.Sleep(3 * time.Second)

	retrieved, err := service.GetResult(ctx, result.ID)
	if err != nil {
		t.Fatalf("Failed to get result: %v", err)
	}

	if retrieved.ID != result.ID {
		t.Errorf("Expected ID %s, got %s", result.ID, retrieved.ID)
	}
}

func TestBenchmarkService_ListResults(t *testing.T) {
	service := NewBenchmarkService()
	ctx := context.Background()
	nodeID := uuid.New()

	service.RunBenchmark(ctx, nodeID, BenchmarkTypeCPU)
	service.RunBenchmark(ctx, nodeID, BenchmarkTypeMemory)
	service.RunBenchmark(ctx, uuid.New(), BenchmarkTypeGPU)

	results, err := service.ListResults(ctx, nodeID, "")
	if err != nil {
		t.Fatalf("Failed to list results: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results for node, got %d", len(results))
	}
}

func TestBenchmarkService_GetNodeScore(t *testing.T) {
	service := NewBenchmarkService()
	ctx := context.Background()
	nodeID := uuid.New()

	result := &BenchmarkResult{
		ID:     uuid.New(),
		NodeID: nodeID,
		Type:   BenchmarkTypeCPU,
		Status: BenchmarkStatusCompleted,
		Details: &BenchmarkDetail{
			CPUDetail: &CPUBenchmarkDetail{
				MultiCoreScore: 5000,
			},
		},
		Score:      5000,
		StartedAt:  time.Now(),
		CompletedAt: func() *time.Time { t := time.Now(); return &t }(),
		CreatedAt:  time.Now(),
	}

	service.results[result.ID] = result
	service.nodeScores[nodeID] = service.calculateNodeScore(result)

	score, err := service.GetNodeScore(ctx, nodeID)
	if err != nil {
		t.Fatalf("Failed to get node score: %v", err)
	}

	if score.NodeID != nodeID {
		t.Errorf("Expected node ID %s, got %s", nodeID, score.NodeID)
	}
}

func TestBenchmarkService_GetNodeRanking(t *testing.T) {
	service := NewBenchmarkService()
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		nodeID := uuid.New()
		result := &BenchmarkResult{
			ID:     uuid.New(),
			NodeID: nodeID,
			Type:   BenchmarkTypeCPU,
			Status: BenchmarkStatusCompleted,
			Score:  float64(1000 + i*500),
		}
		service.results[result.ID] = result
		service.nodeScores[nodeID] = &NodeScore{
			NodeID:       nodeID,
			OverallScore: result.Score,
		}
	}

	ranking, err := service.GetNodeRanking(ctx, 10)
	if err != nil {
		t.Fatalf("Failed to get node ranking: %v", err)
	}

	if len(ranking) != 3 {
		t.Errorf("Expected 3 nodes in ranking, got %d", len(ranking))
	}

	if ranking[0].OverallScore < ranking[1].OverallScore {
		t.Error("Ranking should be in descending order")
	}
}

func TestBenchmarkService_CalculateScore(t *testing.T) {
	service := NewBenchmarkService()

	result := &BenchmarkResult{
		Type: BenchmarkTypeCPU,
		Details: &BenchmarkDetail{
			CPUDetail: &CPUBenchmarkDetail{
				SingleCoreScore: 2000,
				MultiCoreScore:  8000,
			},
		},
	}

	score := service.calculateScore(result)
	expected := 5000.0

	if score != expected {
		t.Errorf("Expected score %.2f, got %.2f", expected, score)
	}
}

func TestBenchmarkService_GetCompositeScore(t *testing.T) {
	service := NewBenchmarkService()
	ctx := context.Background()
	nodeID := uuid.New()

	for i := 0; i < 3; i++ {
		result := &BenchmarkResult{
			ID:     uuid.New(),
			NodeID: nodeID,
			Type:   BenchmarkTypeCPU,
			Status: BenchmarkStatusCompleted,
			Score:  1000,
		}
		service.results[result.ID] = result
	}

	score, err := service.GetCompositeScore(ctx, nodeID)
	if err != nil {
		t.Fatalf("Failed to get composite score: %v", err)
	}

	if score <= 0 {
		t.Error("Composite score should be greater than 0")
	}
}
