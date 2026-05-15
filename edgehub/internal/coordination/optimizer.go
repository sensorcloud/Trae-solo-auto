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

type Optimizer interface {
	Optimize(ctx context.Context, task *ComputeTask, resources []EnergyResource, storage []StorageResource, forecast *CombinedForecast) (*OptimizationResult, error)
	OptimizeBatch(ctx context.Context, tasks []*ComputeTask, resources []EnergyResource, storage []StorageResource, forecast *CombinedForecast) ([]*OptimizationResult, error)
	SetWeights(weights OptimizationWeights)
	SetConstraints(constraints OptimizationConstraints)
}

type MultiObjectiveOptimizer struct {
	config      OptimizationConfig
	weights     OptimizationWeights
	constraints OptimizationConstraints
	
	predictor   *CoordinationPredictor
	
	solutionCache map[string]*OptimizationResult
	mu           sync.RWMutex
}

func NewMultiObjectiveOptimizer(config OptimizationConfig, predictor *CoordinationPredictor) *MultiObjectiveOptimizer {
	weights := config.Weights
	if weights.CostWeight == 0 && weights.CarbonWeight == 0 && weights.GreenWeight == 0 {
		weights = OptimizationWeights{
			CostWeight:        0.4,
			CarbonWeight:      0.25,
			GreenWeight:       0.2,
			ReliabilityWeight: 0.1,
			LatencyWeight:     0.05,
		}
	}
	
	return &MultiObjectiveOptimizer{
		config:        config,
		weights:       weights,
		constraints:   config.Constraints,
		predictor:     predictor,
		solutionCache: make(map[string]*OptimizationResult),
	}
}

func (o *MultiObjectiveOptimizer) Optimize(ctx context.Context, task *ComputeTask, resources []EnergyResource, storage []StorageResource, forecast *CombinedForecast) (*OptimizationResult, error) {
	cacheKey := fmt.Sprintf("%s_%d", task.ID.String(), task.UpdatedAt.Unix())
	
	o.mu.RLock()
	if cached, ok := o.solutionCache[cacheKey]; ok {
		o.mu.RUnlock()
		return cached, nil
	}
	o.mu.RUnlock()
	
	availableSlots := o.findAvailableSlots(task, forecast)
	if len(availableSlots) == 0 {
		return nil, fmt.Errorf("没有找到可用的调度时段")
	}
	
	candidates := o.generateCandidates(task, resources, storage, availableSlots)
	if len(candidates) == 0 {
		return nil, fmt.Errorf("无法生成有效的调度候选方案")
	}
	
	scoredCandidates := o.scoreCandidates(candidates, task)
	
	sort.Slice(scoredCandidates, func(i, j int) bool {
		return scoredCandidates[i].Score > scoredCandidates[j].Score
	})
	
	best := scoredCandidates[0]
	
	alternatives := make([]AlternativeResult, 0)
	for i := 1; i < len(scoredCandidates) && i < 5; i++ {
		alternatives = append(alternatives, AlternativeResult{
			ScheduledStart: scoredCandidates[i].ScheduledStart,
			ScheduledEnd:   scoredCandidates[i].ScheduledEnd,
			TotalCost:      scoredCandidates[i].TotalCost,
			TotalCarbon:    scoredCandidates[i].TotalCarbon,
			GreenRatio:     scoredCandidates[i].GreenRatio,
			Score:          scoredCandidates[i].Score,
		})
	}
	best.Alternatives = alternatives
	
	o.mu.Lock()
	o.solutionCache[cacheKey] = best
	o.mu.Unlock()
	
	klog.V(4).Infof("优化完成, 任务: %s, 最优得分: %.4f, 成本: %.2f, 碳排放: %.2f", 
		task.Name, best.Score, best.TotalCost, best.TotalCarbon)
	
	return best, nil
}

func (o *MultiObjectiveOptimizer) findAvailableSlots(task *ComputeTask, forecast *CombinedForecast) []TimeSlot {
	slots := make([]TimeSlot, 0)
	
	if forecast == nil || len(forecast.Points) == 0 {
		return slots
	}
	
	now := time.Now()
	earliestStart := now
	latestEnd := now.Add(24 * time.Hour)
	
	if task.TimeConstraint.EarliestStart != nil {
		earliestStart = *task.TimeConstraint.EarliestStart
	}
	if task.TimeConstraint.Deadline != nil {
		latestEnd = *task.TimeConstraint.Deadline
	}
	if task.TimeConstraint.MaxDelayMinutes > 0 {
		maxEnd := now.Add(time.Duration(task.TimeConstraint.MaxDelayMinutes) * time.Minute)
		if maxEnd.Before(latestEnd) {
			latestEnd = maxEnd
		}
	}
	
	duration := task.EstimatedDuration
	if duration == 0 {
		duration = 60
	}
	
	for i := 0; i <= len(forecast.Points)-duration/15; i++ {
		slotStart := forecast.Points[i].Timestamp
		
		if slotStart.Before(earliestStart) {
			continue
		}
		
		slotEnd := slotStart.Add(time.Duration(duration) * time.Minute)
		if slotEnd.After(latestEnd) {
			continue
		}
		
		if !o.isPreferredTime(slotStart, task.TimeConstraint) {
			continue
		}
		
		slot := TimeSlot{
			Start:    slotStart,
			End:      slotEnd,
			Duration: duration,
		}
		
		for j := i; j < i+duration/15 && j < len(forecast.Points); j++ {
			slot.AvgPrice += forecast.Points[j].Price
			slot.AvgLoad += forecast.Points[j].Load
			slot.AvgCarbon += forecast.Points[j].CarbonIntensity
			slot.Confidence += forecast.Points[j].Confidence
		}
		
		count := float64(duration / 15)
		slot.AvgPrice /= count
		slot.AvgLoad /= count
		slot.AvgCarbon /= count
		slot.Confidence /= count
		
		slots = append(slots, slot)
	}
	
	return slots
}

func (o *MultiObjectiveOptimizer) isPreferredTime(t time.Time, constraint TimeConstraint) bool {
	hour := t.Hour()
	
	for _, avoidHour := range constraint.AvoidHours {
		if hour == avoidHour {
			return false
		}
	}
	
	if len(constraint.PreferredHours) > 0 {
		preferred := false
		for _, ph := range constraint.PreferredHours {
			if hour == ph {
				preferred = true
				break
			}
		}
		if !preferred {
			return false
		}
	}
	
	return true
}

func (o *MultiObjectiveOptimizer) generateCandidates(task *ComputeTask, resources []EnergyResource, storage []StorageResource, slots []TimeSlot) []*OptimizationResult {
	candidates := make([]*OptimizationResult, 0)
	
	for _, slot := range slots {
		assignedResources := o.assignResources(task, resources, storage, slot)
		if len(assignedResources) == 0 {
			continue
		}
		
		totalCost := 0.0
		totalCarbon := 0.0
		totalGreen := 0.0
		totalReliability := 0.0
		
		for _, ar := range assignedResources {
			totalCost += ar.Cost
			totalCarbon += ar.Carbon
			totalGreen += ar.GreenRatio * ar.Power * float64(ar.Duration) / 60
			totalReliability += ar.GreenRatio
		}
		
		totalEnergy := task.EstimatedPower * float64(slot.Duration) / 60
		if totalEnergy > 0 {
			totalGreen = totalGreen / totalEnergy
		}
		
		candidate := &OptimizationResult{
			TaskID:            task.ID,
			ScheduledStart:    slot.Start,
			ScheduledEnd:      slot.End,
			AssignedResources: assignedResources,
			TotalCost:         totalCost,
			TotalCarbon:       totalCarbon,
			GreenRatio:        totalGreen,
			Reliability:       totalReliability / float64(len(assignedResources)),
			Confidence:        slot.Confidence,
			ObjectiveValues:   make(map[string]float64),
		}
		
		candidate.ObjectiveValues["cost"] = totalCost
		candidate.ObjectiveValues["carbon"] = totalCarbon
		candidate.ObjectiveValues["green"] = totalGreen
		candidate.ObjectiveValues["reliability"] = candidate.Reliability
		
		candidates = append(candidates, candidate)
	}
	
	return candidates
}

func (o *MultiObjectiveOptimizer) assignResources(task *ComputeTask, resources []EnergyResource, storage []StorageResource, slot TimeSlot) []AssignedResource {
	assigned := make([]AssignedResource, 0)
	
	requiredPower := task.EstimatedPower
	if requiredPower == 0 {
		requiredPower = 100
	}
	
	sort.Slice(resources, func(i, j int) bool {
		scoreI := o.calculateResourceScore(&resources[i], slot)
		scoreJ := o.calculateResourceScore(&resources[j], slot)
		return scoreI > scoreJ
	})
	
	for i := range resources {
		if requiredPower <= 0 {
			break
		}
		
		res := &resources[i]
		if res.Status != EnergyResourceStatusOnline {
			continue
		}
		
		availablePower := math.Min(res.AvailableCapacity, requiredPower)
		if availablePower <= 0 {
			continue
		}
		
		duration := slot.Duration
		energy := availablePower * float64(duration) / 60
		
		ar := AssignedResource{
			ResourceID:   res.ID,
			ResourceType: res.Type,
			Power:        availablePower,
			Duration:     duration,
			Cost:         energy * res.PricePerKWh,
			Carbon:       energy * res.CarbonIntensity,
			GreenRatio:   res.GreenRatio,
		}
		
		assigned = append(assigned, ar)
		requiredPower -= availablePower
	}
	
	if requiredPower > 0 && len(storage) > 0 && task.EnergyPreference.AllowStorage {
		for i := range storage {
			if requiredPower <= 0 {
				break
			}
			
			s := &storage[i]
			if s.Status != "idle" && s.Status != "discharging" {
				continue
			}
			
			availableEnergy := (s.SOC - s.MinSOC) * s.Capacity / 100
			if availableEnergy <= 0 {
				continue
			}
			
			availablePower := math.Min(s.MaxDischargeRate, requiredPower)
			availablePower = math.Min(availablePower, availableEnergy*60/float64(slot.Duration))
			
			if availablePower <= 0 {
				continue
			}
			
			duration := slot.Duration
			energy := availablePower * float64(duration) / 60
			
			ar := AssignedResource{
				ResourceID:   s.ID,
				ResourceType: EnergyResourceStorage,
				Power:        availablePower,
				Duration:     duration,
				Cost:         energy * 0.3,
				Carbon:       energy * 0.1,
				GreenRatio:   0.9,
			}
			
			assigned = append(assigned, ar)
			requiredPower -= availablePower
		}
	}
	
	return assigned
}

func (o *MultiObjectiveOptimizer) calculateResourceScore(res *EnergyResource, slot TimeSlot) float64 {
	score := 0.0
	
	costScore := 1.0 - (res.PricePerKWh / 1.0)
	if costScore < 0 {
		costScore = 0
	}
	
	greenScore := res.GreenRatio
	
	carbonScore := 1.0 - (res.CarbonIntensity / 0.8)
	if carbonScore < 0 {
		carbonScore = 0
	}
	
	reliabilityScore := res.Reliability
	
	score = costScore*o.weights.CostWeight +
		greenScore*o.weights.GreenWeight +
		carbonScore*o.weights.CarbonWeight +
		reliabilityScore*o.weights.ReliabilityWeight
	
	return score
}

func (o *MultiObjectiveOptimizer) scoreCandidates(candidates []*OptimizationResult, task *ComputeTask) []*OptimizationResult {
	for _, candidate := range candidates {
		candidate.Score = o.calculateOverallScore(candidate, task)
	}
	return candidates
}

func (o *MultiObjectiveOptimizer) calculateOverallScore(result *OptimizationResult, task *ComputeTask) float64 {
	normalizedCost := 1.0 - (result.TotalCost / (task.EstimatedPower * float64(result.ScheduledEnd.Sub(result.ScheduledStart).Minutes()) / 60 * 1.0))
	if normalizedCost < 0 {
		normalizedCost = 0
	}
	
	normalizedCarbon := 1.0 - (result.TotalCarbon / (task.EstimatedPower * float64(result.ScheduledEnd.Sub(result.ScheduledStart).Minutes()) / 60 * 0.8))
	if normalizedCarbon < 0 {
		normalizedCarbon = 0
	}
	
	normalizedGreen := result.GreenRatio
	
	normalizedReliability := result.Reliability
	
	latency := time.Since(result.ScheduledStart).Minutes()
	normalizedLatency := 1.0 - (latency / 1440)
	if normalizedLatency < 0 {
		normalizedLatency = 0
	}
	
	score := normalizedCost*o.weights.CostWeight +
		normalizedCarbon*o.weights.CarbonWeight +
		normalizedGreen*o.weights.GreenWeight +
		normalizedReliability*o.weights.ReliabilityWeight +
		normalizedLatency*o.weights.LatencyWeight
	
	return score
}

func (o *MultiObjectiveOptimizer) OptimizeBatch(ctx context.Context, tasks []*ComputeTask, resources []EnergyResource, storage []StorageResource, forecast *CombinedForecast) ([]*OptimizationResult, error) {
	results := make([]*OptimizationResult, len(tasks))
	
	sort.Slice(tasks, func(i, j int) bool {
		if tasks[i].Priority != tasks[j].Priority {
			return tasks[i].Priority > tasks[j].Priority
		}
		return tasks[i].CreatedAt.Before(tasks[j].CreatedAt)
	})
	
	usedResources := make(map[uuid.UUID]float64)
	usedStorage := make(map[uuid.UUID]float64)
	
	for i, task := range tasks {
		adjustedResources := o.adjustAvailableResources(resources, usedResources)
		adjustedStorage := o.adjustAvailableStorage(storage, usedStorage)
		
		result, err := o.Optimize(ctx, task, adjustedResources, adjustedStorage, forecast)
		if err != nil {
			klog.Warningf("批量优化任务 %s 失败: %v", task.Name, err)
			continue
		}
		
		results[i] = result
		
		for _, ar := range result.AssignedResources {
			if ar.ResourceType == EnergyResourceStorage {
				usedStorage[ar.ResourceID] += ar.Power * float64(ar.Duration) / 60
			} else {
				usedResources[ar.ResourceID] += ar.Power
			}
		}
	}
	
	klog.Infof("批量优化完成, 任务数: %d, 成功数: %d", len(tasks), len(filterNilResults(results)))
	return results, nil
}

func filterNilResults(results []*OptimizationResult) []*OptimizationResult {
	filtered := make([]*OptimizationResult, 0)
	for _, r := range results {
		if r != nil {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

func (o *MultiObjectiveOptimizer) adjustAvailableResources(resources []EnergyResource, used map[uuid.UUID]float64) []EnergyResource {
	adjusted := make([]EnergyResource, len(resources))
	copy(adjusted, resources)
	
	for i := range adjusted {
		if usedPower, ok := used[adjusted[i].ID]; ok {
			adjusted[i].AvailableCapacity = math.Max(0, adjusted[i].AvailableCapacity-usedPower)
		}
	}
	
	return adjusted
}

func (o *MultiObjectiveOptimizer) adjustAvailableStorage(storage []StorageResource, used map[uuid.UUID]float64) []StorageResource {
	adjusted := make([]StorageResource, len(storage))
	copy(adjusted, storage)
	
	for i := range adjusted {
		if usedEnergy, ok := used[adjusted[i].ID]; ok {
			adjusted[i].AvailableEnergy = math.Max(0, adjusted[i].AvailableEnergy-usedEnergy)
		}
	}
	
	return adjusted
}

func (o *MultiObjectiveOptimizer) SetWeights(weights OptimizationWeights) {
	o.mu.Lock()
	defer o.mu.Unlock()
	
	total := weights.CostWeight + weights.CarbonWeight + weights.GreenWeight + weights.ReliabilityWeight + weights.LatencyWeight
	if total > 0 {
		o.weights = OptimizationWeights{
			CostWeight:        weights.CostWeight / total,
			CarbonWeight:      weights.CarbonWeight / total,
			GreenWeight:       weights.GreenWeight / total,
			ReliabilityWeight: weights.ReliabilityWeight / total,
			LatencyWeight:     weights.LatencyWeight / total,
		}
	}
}

func (o *MultiObjectiveOptimizer) SetConstraints(constraints OptimizationConstraints) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.constraints = constraints
}

func (o *MultiObjectiveOptimizer) OptimizeStorageUsage(ctx context.Context, storage []StorageResource, forecast *CombinedForecast, tasks []*ComputeTask) (*StorageOptimizationPlan, error) {
	plan := &StorageOptimizationPlan{
		GeneratedAt: time.Now(),
		Actions:     make([]StorageActionPlan, 0),
	}
	
	for _, s := range storage {
		action := o.determineStorageAction(s, forecast)
		if action != nil {
			plan.Actions = append(plan.Actions, *action)
		}
	}
	
	for _, task := range tasks {
		if task.EnergyPreference.AllowStorage {
			storageAction := o.planStorageForTask(task, storage, forecast)
			if storageAction != nil {
				plan.Actions = append(plan.Actions, *storageAction)
			}
		}
	}
	
	plan.ExpectedSavings = o.calculateStorageSavings(plan.Actions, forecast)
	
	klog.V(4).Infof("储能优化计划生成完成, 动作数: %d, 预期节省: %.2f", len(plan.Actions), plan.ExpectedSavings)
	return plan, nil
}

func (o *MultiObjectiveOptimizer) determineStorageAction(storage StorageResource, forecast *CombinedForecast) *StorageActionPlan {
	if len(forecast.Points) == 0 {
		return nil
	}
	
	currentPrice := forecast.Points[0].Price
	
	valleyPrice := math.MaxFloat64
	peakPrice := 0.0
	
	for _, point := range forecast.Points {
		if point.Price < valleyPrice {
			valleyPrice = point.Price
		}
		if point.Price > peakPrice {
			peakPrice = point.Price
		}
	}
	
	if currentPrice <= valleyPrice*1.1 && storage.SOC < storage.MaxSOC {
		chargePower := math.Min(storage.MaxChargeRate, (storage.MaxSOC-storage.SOC)*storage.Capacity/100)
		return &StorageActionPlan{
			StorageID:   storage.ID,
			Action:      "charge",
			Power:       chargePower,
			StartTime:   time.Now(),
			Duration:    int((storage.MaxSOC - storage.SOC) * storage.Capacity / chargePower),
			Reason:      "低价时段充电",
			ExpectedCost: chargePower * float64(int((storage.MaxSOC-storage.SOC)*storage.Capacity/chargePower)) / 60 * currentPrice,
		}
	}
	
	if currentPrice >= peakPrice*0.9 && storage.SOC > storage.MinSOC {
		dischargePower := math.Min(storage.MaxDischargeRate, (storage.SOC-storage.MinSOC)*storage.Capacity/100)
		return &StorageActionPlan{
			StorageID:   storage.ID,
			Action:      "discharge",
			Power:       dischargePower,
			StartTime:   time.Now(),
			Duration:    int((storage.SOC - storage.MinSOC) * storage.Capacity / dischargePower),
			Reason:      "高价时段放电获利",
			ExpectedCost: -dischargePower * float64(int((storage.SOC-storage.MinSOC)*storage.Capacity/dischargePower)) / 60 * currentPrice,
		}
	}
	
	return nil
}

func (o *MultiObjectiveOptimizer) planStorageForTask(task *ComputeTask, storage []StorageResource, forecast *CombinedForecast) *StorageActionPlan {
	if len(storage) == 0 || len(forecast.Points) == 0 {
		return nil
	}
	
	for _, s := range storage {
		if s.SOC > s.MinSOC && s.MaxDischargeRate >= task.EstimatedPower {
			return &StorageActionPlan{
				StorageID:   s.ID,
				Action:      "discharge",
				Power:       task.EstimatedPower,
				StartTime:   time.Now(),
				Duration:    task.EstimatedDuration,
				Reason:      fmt.Sprintf("为任务 %s 提供电量", task.Name),
				TaskID:      task.ID,
			}
		}
	}
	
	return nil
}

func (o *MultiObjectiveOptimizer) calculateStorageSavings(actions []StorageActionPlan, forecast *CombinedForecast) float64 {
	savings := 0.0
	
	for _, action := range actions {
		if action.Action == "discharge" {
			savings += action.Power * float64(action.Duration) / 60 * forecast.Points[0].Price
		}
	}
	
	return savings
}

type TimeSlot struct {
	Start      time.Time `json:"start"`
	End        time.Time `json:"end"`
	Duration   int       `json:"duration"`
	AvgPrice   float64   `json:"avg_price"`
	AvgLoad    float64   `json:"avg_load"`
	AvgCarbon  float64   `json:"avg_carbon"`
	Confidence float64   `json:"confidence"`
}

type StorageOptimizationPlan struct {
	GeneratedAt     time.Time            `json:"generated_at"`
	Actions         []StorageActionPlan  `json:"actions"`
	ExpectedSavings float64              `json:"expected_savings"`
}

type StorageActionPlan struct {
	StorageID    uuid.UUID  `json:"storage_id"`
	Action       string     `json:"action"`
	Power        float64    `json:"power"`
	StartTime    time.Time  `json:"start_time"`
	Duration     int        `json:"duration"`
	Reason       string     `json:"reason"`
	TaskID       uuid.UUID  `json:"task_id,omitempty"`
	ExpectedCost float64    `json:"expected_cost"`
}

func (o *MultiObjectiveOptimizer) SolveLinearProgram(ctx context.Context, problem *LPProblem) (*LPSolution, error) {
	solution := &LPSolution{
		Status:    "optimal",
		Variables: make(map[string]float64),
	}
	
	n := len(problem.Variables)
	if n == 0 {
		return solution, nil
	}
	
	x := make([]float64, n)
	for i := range x {
		x[i] = 0.5
	}
	
	learningRate := 0.1
	for iter := 0; iter < o.config.MaxIterations; iter++ {
		grad := o.computeGradient(problem, x)
		
		for i := range x {
			x[i] -= learningRate * grad[i]
			x[i] = math.Max(0, math.Min(1, x[i]))
		}
		
		if o.checkConstraints(problem, x) {
			break
		}
	}
	
	for i, v := range problem.Variables {
		solution.Variables[v.Name] = x[i]
	}
	
	solution.ObjectiveValue = o.computeObjective(problem, x)
	
	return solution, nil
}

func (o *MultiObjectiveOptimizer) computeGradient(problem *LPProblem, x []float64) []float64 {
	n := len(x)
	grad := make([]float64, n)
	
	for i := range grad {
		grad[i] = problem.Objective[i]
	}
	
	return grad
}

func (o *MultiObjectiveOptimizer) checkConstraints(problem *LPProblem, x []float64) bool {
	for _, constraint := range problem.Constraints {
		sum := 0.0
		for i, coef := range constraint.Coefficients {
			sum += coef * x[i]
		}
		
		switch constraint.Operator {
		case "<=":
			if sum > constraint.RHS {
				return false
			}
		case ">=":
			if sum < constraint.RHS {
				return false
			}
		case "==":
			if math.Abs(sum-constraint.RHS) > 0.01 {
				return false
			}
		}
	}
	
	return true
}

func (o *MultiObjectiveOptimizer) computeObjective(problem *LPProblem, x []float64) float64 {
	obj := 0.0
	for i, coef := range problem.Objective {
		obj += coef * x[i]
	}
	return obj
}

type LPProblem struct {
	Variables   []LPVariable     `json:"variables"`
	Objective   []float64        `json:"objective"`
	Constraints []LPConstraint   `json:"constraints"`
	Maximize    bool             `json:"maximize"`
}

type LPVariable struct {
	Name    string  `json:"name"`
	Lower   float64 `json:"lower"`
	Upper   float64 `json:"upper"`
}

type LPConstraint struct {
	Coefficients []float64 `json:"coefficients"`
	Operator     string    `json:"operator"`
	RHS          float64   `json:"rhs"`
}

type LPSolution struct {
	Status          string             `json:"status"`
	Variables       map[string]float64 `json:"variables"`
	ObjectiveValue  float64            `json:"objective_value"`
	Iterations      int                `json:"iterations"`
}
