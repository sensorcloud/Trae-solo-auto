package benchmark

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
	"k8s.io/klog/v2"
)

type BenchmarkStatus string
type BenchmarkType string

const (
	BenchmarkStatusPending   BenchmarkStatus = "pending"
	BenchmarkStatusRunning  BenchmarkStatus = "running"
	BenchmarkStatusCompleted BenchmarkStatus = "completed"
	BenchmarkStatusFailed   BenchmarkStatus = "failed"

	BenchmarkTypeCPU       BenchmarkType = "cpu"
	BenchmarkTypeMemory   BenchmarkType = "memory"
	BenchmarkTypeNetwork   BenchmarkType = "network"
	BenchmarkTypeGPU       BenchmarkType = "gpu"
	BenchmarkTypeStorage   BenchmarkType = "storage"
	BenchmarkTypeComposite BenchmarkType = "composite"
)

type BenchmarkResult struct {
	ID             uuid.UUID         `json:"id"`
	NodeID         uuid.UUID         `json:"node_id"`
	Type           BenchmarkType    `json:"type"`
	Status         BenchmarkStatus   `json:"status"`
	Score          float64          `json:"score"`
	Details        *BenchmarkDetail  `json:"details,omitempty"`
	Recommendations []string         `json:"recommendations,omitempty"`
	StartedAt      time.Time        `json:"started_at"`
	CompletedAt    *time.Time       `json:"completed_at,omitempty"`
	CreatedAt      time.Time        `json:"created_at"`
}

type BenchmarkDetail struct {
	CPUDetail     *CPUBenchmarkDetail    `json:"cpu_detail,omitempty"`
	MemoryDetail  *MemoryBenchmarkDetail `json:"memory_detail,omitempty"`
	NetworkDetail *NetworkBenchmarkDetail `json:"network_detail,omitempty"`
	GPUDetail     *GPUBenchmarkDetail    `json:"gpu_detail,omitempty"`
	StorageDetail *StorageBenchmarkDetail `json:"storage_detail,omitempty"`
}

type CPUBenchmarkDetail struct {
	Cores           int     `json:"cores"`
	BaseClock       float64 `json:"base_clock_mhz"`
	TurboClock      float64 `json:"turbo_clock_mhz"`
	SingleCoreScore float64 `json:"single_core_score"`
	MultiCoreScore  float64 `json:"multi_core_score"`
	LinpackScore    float64 `json:"linpack_score"`
	GeekbenchScore  float64 `json:"geekbench_score"`
}

type MemoryBenchmarkDetail struct {
	TotalGB       float64 `json:"total_gb"`
	SpeedMHz      float64 `json:"speed_mhz"`
	LatencyNS     float64 `json:"latency_ns"`
	BandwidthGBps float64 `json:"bandwidth_gbps"`
	CopyScore     float64 `json:"copy_score"`
	ReadScore     float64 `json:"read_score"`
	WriteScore    float64 `json:"write_score"`
}

type NetworkBenchmarkDetail struct {
	BandwidthGbps  float64 `json:"bandwidth_gbps"`
	LatencyUS      float64 `json:"latency_us"`
	JitterNS       float64 `json:"jitter_ns"`
	PPS            int     `json:"pps"`
	TCPThroughput  float64 `json:"tcp_throughput"`
	UDPThroughput  float64 `json:"udp_throughput"`
}

type GPUBenchmarkDetail struct {
	Model         string  `json:"model"`
	VRAMGB        float64 `json:"vram_gb"`
	BandwidthGbps float64 `json:"bandwidth_gbps"`
	FP32TFLOPS    float64 `json:"fp32_tflops"`
	FP16TFLOPS    float64 `json:"fp16_tflops"`
	TensorTFOPS   float64 `json:"tensor_tflops"`
	ComputeScore  float64 `json:"compute_score"`
	MemoryScore   float64 `json:"memory_score"`
}

type StorageBenchmarkDetail struct {
	TotalGB         float64 `json:"total_gb"`
	ReadIOPS        int     `json:"read_iops"`
	WriteIOPS       int     `json:"write_iops"`
	ReadMBps        float64 `json:"read_mbps"`
	WriteMBps       float64 `json:"write_mbps"`
	LatencyUS       float64 `json:"latency_us"`
	SequentialScore float64 `json:"sequential_score"`
	RandomScore     float64 `json:"random_score"`
}

type NodeScore struct {
	NodeID          uuid.UUID   `json:"node_id"`
	OverallScore   float64     `json:"overall_score"`
	CPU             float64    `json:"cpu_score"`
	Memory          float64    `json:"memory_score"`
	Network         float64    `json:"network_score"`
	GPU             float64    `json:"gpu_score"`
	Storage         float64    `json:"storage_score"`
	Rank            int        `json:"rank"`
	Recommendations []string    `json:"recommendations,omitempty"`
}

type BenchmarkService struct {
	results    map[uuid.UUID]*BenchmarkResult
	nodeScores map[uuid.UUID]*NodeScore
	mu         sync.RWMutex
}

func NewBenchmarkService() *BenchmarkService {
	return &BenchmarkService{
		results:    make(map[uuid.UUID]*BenchmarkResult),
		nodeScores: make(map[uuid.UUID]*NodeScore),
	}
}

func (bs *BenchmarkService) RunBenchmark(ctx context.Context, nodeID uuid.UUID, benchmarkType BenchmarkType) (*BenchmarkResult, error) {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	result := &BenchmarkResult{
		ID:        uuid.New(),
		NodeID:    nodeID,
		Type:      benchmarkType,
		Status:    BenchmarkStatusRunning,
		StartedAt: time.Now(),
		CreatedAt: time.Now(),
	}

	bs.results[result.ID] = result
	klog.Infof("Starting benchmark %s for node %s", benchmarkType, nodeID)

	go bs.executeBenchmark(result)

	return result, nil
}

func (bs *BenchmarkService) executeBenchmark(result *BenchmarkResult) {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	time.Sleep(2 * time.Second)

	result.Status = BenchmarkStatusCompleted
	result.Score = bs.calculateScore(result)

	now := time.Now()
	result.CompletedAt = &now

	bs.nodeScores[result.NodeID] = bs.calculateNodeScore(result)

	klog.Infof("Benchmark %s completed, score: %.2f", result.ID, result.Score)
}

func (bs *BenchmarkService) calculateScore(result *BenchmarkResult) float64 {
	switch result.Type {
	case BenchmarkTypeCPU:
		if result.Details != nil && result.Details.CPUDetail != nil {
			return (result.Details.CPUDetail.SingleCoreScore + result.Details.CPUDetail.MultiCoreScore) / 2
		}
	case BenchmarkTypeMemory:
		if result.Details != nil && result.Details.MemoryDetail != nil {
			return result.Details.MemoryDetail.BandwidthGBps * 100
		}
	case BenchmarkTypeNetwork:
		if result.Details != nil && result.Details.NetworkDetail != nil {
			return result.Details.NetworkDetail.BandwidthGbps * 10
		}
	case BenchmarkTypeGPU:
		if result.Details != nil && result.Details.GPUDetail != nil {
			return (result.Details.GPUDetail.ComputeScore + result.Details.GPUDetail.MemoryScore) / 2
		}
	case BenchmarkTypeStorage:
		if result.Details != nil && result.Details.StorageDetail != nil {
			return float64(result.Details.StorageDetail.ReadIOPS+result.Details.StorageDetail.WriteIOPS) / 1000
		}
	}

	return 1000
}

func (bs *BenchmarkService) calculateNodeScore(result *BenchmarkResult) *NodeScore {
	score := &NodeScore{
		NodeID: result.NodeID,
	}

	if result.Details != nil {
		if result.Details.CPUDetail != nil {
			score.CPU = result.Details.CPUDetail.MultiCoreScore / 1000
		}
		if result.Details.MemoryDetail != nil {
			score.Memory = result.Details.MemoryDetail.BandwidthGBps / 10
		}
		if result.Details.NetworkDetail != nil {
			score.Network = result.Details.NetworkDetail.BandwidthGbps / 10
		}
		if result.Details.GPUDetail != nil {
			score.GPU = result.Details.GPUDetail.ComputeScore / 1000
		}
		if result.Details.StorageDetail != nil {
			score.Storage = float64(result.Details.StorageDetail.ReadIOPS) / 10000
		}
	}

	score.OverallScore = (score.CPU + score.Memory + score.Network + score.GPU + score.Storage) / 5 * 100

	if score.OverallScore > 80 {
		score.Recommendations = append(score.Recommendations, "Excellent performance, suitable for high-demand workloads")
	} else if score.OverallScore > 60 {
		score.Recommendations = append(score.Recommendations, "Good performance, suitable for general workloads")
	} else {
		score.Recommendations = append(score.Recommendations, "Performance needs improvement, consider upgrading hardware")
	}

	return score
}

func (bs *BenchmarkService) GetResult(ctx context.Context, resultID uuid.UUID) (*BenchmarkResult, error) {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	result, exists := bs.results[resultID]
	if !exists {
		return nil, fmt.Errorf("benchmark result %s not found", resultID)
	}
	return result, nil
}

func (bs *BenchmarkService) ListResults(ctx context.Context, nodeID uuid.UUID, benchmarkType BenchmarkType) ([]*BenchmarkResult, error) {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	var result []*BenchmarkResult
	for _, r := range bs.results {
		if nodeID != uuid.Nil && r.NodeID != nodeID {
			continue
		}
		if benchmarkType != "" && r.Type != benchmarkType {
			continue
		}
		result = append(result, r)
	}
	return result, nil
}

func (bs *BenchmarkService) GetNodeScore(ctx context.Context, nodeID uuid.UUID) (*NodeScore, error) {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	score, exists := bs.nodeScores[nodeID]
	if !exists {
		return nil, fmt.Errorf("no benchmark score for node %s", nodeID)
	}
	return score, nil
}

func (bs *BenchmarkService) GetNodeRanking(ctx context.Context, limit int) ([]*NodeScore, error) {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	scores := make([]*NodeScore, 0, len(bs.nodeScores))
	for _, score := range bs.nodeScores {
		scores = append(scores, score)
	}

	for i := 0; i < len(scores)-1; i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[j].OverallScore > scores[i].OverallScore {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}

	for i, score := range scores {
		score.Rank = i + 1
	}

	if limit > 0 && limit < len(scores) {
		scores = scores[:limit]
	}

	return scores, nil
}

func (bs *BenchmarkService) GetCompositeScore(ctx context.Context, nodeID uuid.UUID) (float64, error) {
	results, err := bs.ListResults(ctx, nodeID, "")
	if err != nil {
		return 0, err
	}

	if len(results) == 0 {
		return 0, fmt.Errorf("no benchmark results for node %s", nodeID)
	}

	var totalScore float64
	var weightSum float64

	weights := map[BenchmarkType]float64{
		BenchmarkTypeCPU:     0.25,
		BenchmarkTypeMemory:  0.15,
		BenchmarkTypeNetwork: 0.15,
		BenchmarkTypeGPU:     0.30,
		BenchmarkTypeStorage: 0.15,
	}

	for _, result := range results {
		if result.Status != BenchmarkStatusCompleted {
			continue
		}
		weight := weights[result.Type]
		totalScore += result.Score * weight
		weightSum += weight
	}

	if weightSum == 0 {
		return 0, nil
	}

	return math.Round(totalScore/weightSum*100) / 100, nil
}
