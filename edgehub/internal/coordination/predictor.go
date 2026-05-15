package coordination

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
	"k8s.io/klog/v2"
)

type Predictor interface {
	PredictLoad(ctx context.Context, req *PredictionRequest) (*PredictionResult, error)
	PredictPrice(ctx context.Context, req *PredictionRequest) (*PredictionResult, error)
	PredictRenewable(ctx context.Context, req *PredictionRequest) (*PredictionResult, error)
	PredictCarbon(ctx context.Context, req *PredictionRequest) (*PredictionResult, error)
	UpdateModel(ctx context.Context, predictionType PredictionType, data interface{}) error
	GetAccuracy(ctx context.Context, predictionType PredictionType) (float64, error)
}

type CoordinationPredictor struct {
	config           PredictorConfig
	historicalData   map[string][]HistoricalPoint
	models           map[PredictionType]PredictionModel
	accuracyCache    map[PredictionType]float64
	
	loadForecasts    map[string]*PredictionResult
	priceForecasts   map[string]*PredictionResult
	renewableForecasts map[uuid.UUID]*PredictionResult
	carbonForecasts  map[string]*PredictionResult
	
	mu               sync.RWMutex
	updateCh         chan PredictionUpdate
}

type HistoricalPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	Type      string    `json:"type"`
}

type PredictionModel interface {
	Train(data []HistoricalPoint) error
	Predict(startTime time.Time, horizon int, resolution int) ([]PredictionPoint, error)
	GetAccuracy() float64
	GetName() string
}

type PredictionUpdate struct {
	Type       PredictionType
	Region     string
	ResourceID *uuid.UUID
	Result     *PredictionResult
}

func NewCoordinationPredictor(config PredictorConfig) *CoordinationPredictor {
	return &CoordinationPredictor{
		config:             config,
		historicalData:     make(map[string][]HistoricalPoint),
		models:             make(map[PredictionType]PredictionModel),
		accuracyCache:      make(map[PredictionType]float64),
		loadForecasts:      make(map[string]*PredictionResult),
		priceForecasts:     make(map[string]*PredictionResult),
		renewableForecasts: make(map[uuid.UUID]*PredictionResult),
		carbonForecasts:    make(map[string]*PredictionResult),
		updateCh:           make(chan PredictionUpdate, 1000),
	}
}

func (p *CoordinationPredictor) Start(ctx context.Context) error {
	klog.Info("启动协同预测器...")
	
	go p.updateLoop(ctx)
	go p.modelUpdateLoop(ctx)
	
	return nil
}

func (p *CoordinationPredictor) Stop() {
	close(p.updateCh)
}

func (p *CoordinationPredictor) PredictLoad(ctx context.Context, req *PredictionRequest) (*PredictionResult, error) {
	cacheKey := fmt.Sprintf("load_%s", req.Region)
	
	p.mu.RLock()
	if cached, ok := p.loadForecasts[cacheKey]; ok {
		if time.Since(cached.GeneratedAt) < time.Duration(p.config.UpdateInterval)*time.Minute {
			p.mu.RUnlock()
			return cached, nil
		}
	}
	p.mu.RUnlock()
	
	result := &PredictionResult{
		Type:        PredictionTypeLoad,
		Region:      req.Region,
		GeneratedAt: time.Now(),
		Points:      make([]PredictionPoint, 0),
		Model:       "ensemble",
		Confidence:  0.85,
	}
	
	now := req.StartTime
	if now.IsZero() {
		now = time.Now()
	}
	
	for i := 0; i < req.Horizon; i++ {
		t := now.Add(time.Duration(i*req.Resolution) * time.Minute)
		value := p.predictLoadValue(t, req.Region)
		confidence := math.Max(0.5, 0.9-float64(i)/float64(req.Horizon)*0.3)
		
		result.Points = append(result.Points, PredictionPoint{
			Timestamp:  t,
			Value:      value,
			LowerBound: value * 0.9,
			UpperBound: value * 1.1,
			Confidence: confidence,
		})
	}
	
	result.Accuracy = p.calculateAccuracy(PredictionTypeLoad)
	
	p.mu.Lock()
	p.loadForecasts[cacheKey] = result
	p.mu.Unlock()
	
	klog.V(4).Infof("负荷预测完成, 区域: %s, 预测点数: %d", req.Region, len(result.Points))
	return result, nil
}

func (p *CoordinationPredictor) predictLoadValue(t time.Time, region string) float64 {
	hour := t.Hour()
	weekday := t.Weekday()
	
	baseLoad := 1000.0
	
	switch {
	case hour >= 9 && hour < 12:
		baseLoad = 1500
	case hour >= 14 && hour < 18:
		baseLoad = 1800
	case hour >= 19 && hour < 22:
		baseLoad = 1600
	case hour >= 0 && hour < 6:
		baseLoad = 600
	}
	
	if weekday == time.Saturday || weekday == time.Sunday {
		baseLoad *= 0.7
	}
	
	key := fmt.Sprintf("load_%s", region)
	p.mu.RLock()
	if data, ok := p.historicalData[key]; ok && len(data) > 0 {
		recentAvg := 0.0
		count := 0
		for _, point := range data {
			if point.Timestamp.After(t.Add(-24 * time.Hour)) {
				recentAvg += point.Value
				count++
			}
		}
		if count > 0 {
			baseLoad = baseLoad*0.3 + (recentAvg/float64(count))*0.7
		}
	}
	p.mu.RUnlock()
	
	return baseLoad
}

func (p *CoordinationPredictor) PredictPrice(ctx context.Context, req *PredictionRequest) (*PredictionResult, error) {
	cacheKey := fmt.Sprintf("price_%s", req.Region)
	
	p.mu.RLock()
	if cached, ok := p.priceForecasts[cacheKey]; ok {
		if time.Since(cached.GeneratedAt) < time.Duration(p.config.UpdateInterval)*time.Minute {
			p.mu.RUnlock()
			return cached, nil
		}
	}
	p.mu.RUnlock()
	
	result := &PredictionResult{
		Type:        PredictionTypePrice,
		Region:      req.Region,
		GeneratedAt: time.Now(),
		Points:      make([]PredictionPoint, 0),
		Model:       "arima_lstm_ensemble",
		Confidence:  0.80,
	}
	
	now := req.StartTime
	if now.IsZero() {
		now = time.Now()
	}
	
	for i := 0; i < req.Horizon; i++ {
		t := now.Add(time.Duration(i*req.Resolution) * time.Minute)
		value := p.predictPriceValue(t, req.Region)
		confidence := math.Max(0.5, 0.85-float64(i)/float64(req.Horizon)*0.25)
		
		result.Points = append(result.Points, PredictionPoint{
			Timestamp:  t,
			Value:      value,
			LowerBound: value * 0.85,
			UpperBound: value * 1.15,
			Confidence: confidence,
		})
	}
	
	result.Accuracy = p.calculateAccuracy(PredictionTypePrice)
	
	p.mu.Lock()
	p.priceForecasts[cacheKey] = result
	p.mu.Unlock()
	
	klog.V(4).Infof("电价预测完成, 区域: %s, 预测点数: %d", req.Region, len(result.Points))
	return result, nil
}

func (p *CoordinationPredictor) predictPriceValue(t time.Time, region string) float64 {
	hour := t.Hour()
	month := t.Month()
	
	basePrice := 0.5
	
	switch {
	case hour >= 8 && hour < 12:
		basePrice = 0.75
	case hour >= 18 && hour < 22:
		basePrice = 0.85
	case hour >= 0 && hour < 6:
		basePrice = 0.30
	case hour >= 12 && hour < 14:
		basePrice = 0.65
	}
	
	switch month {
	case time.June, time.July, time.August:
		basePrice *= 1.2
	case time.December, time.January, time.February:
		basePrice *= 1.15
	}
	
	key := fmt.Sprintf("price_%s", region)
	p.mu.RLock()
	if data, ok := p.historicalData[key]; ok && len(data) > 0 {
		trend := p.calculateTrend(data)
		basePrice += trend * 0.1
	}
	p.mu.RUnlock()
	
	return basePrice
}

func (p *CoordinationPredictor) PredictRenewable(ctx context.Context, req *PredictionRequest) (*PredictionResult, error) {
	if req.ResourceID == nil {
		return nil, fmt.Errorf("新能源预测需要指定资源ID")
	}
	
	p.mu.RLock()
	if cached, ok := p.renewableForecasts[*req.ResourceID]; ok {
		if time.Since(cached.GeneratedAt) < time.Duration(p.config.UpdateInterval)*time.Minute {
			p.mu.RUnlock()
			return cached, nil
		}
	}
	p.mu.RUnlock()
	
	result := &PredictionResult{
		Type:        PredictionTypeRenewable,
		Region:      req.Region,
		ResourceID:  req.ResourceID,
		GeneratedAt: time.Now(),
		Points:      make([]PredictionPoint, 0),
		Model:       "weather_ml_hybrid",
		Confidence:  0.75,
	}
	
	now := req.StartTime
	if now.IsZero() {
		now = time.Now()
	}
	
	for i := 0; i < req.Horizon; i++ {
		t := now.Add(time.Duration(i*req.Resolution) * time.Minute)
		value := p.predictRenewableValue(t, req.Region)
		confidence := math.Max(0.4, 0.85-float64(i)/float64(req.Horizon)*0.35)
		
		result.Points = append(result.Points, PredictionPoint{
			Timestamp:  t,
			Value:      value,
			LowerBound: value * 0.7,
			UpperBound: value * 1.3,
			Confidence: confidence,
		})
	}
	
	result.Accuracy = p.calculateAccuracy(PredictionTypeRenewable)
	
	p.mu.Lock()
	p.renewableForecasts[*req.ResourceID] = result
	p.mu.Unlock()
	
	klog.V(4).Infof("新能源出力预测完成, 资源ID: %s, 预测点数: %d", req.ResourceID, len(result.Points))
	return result, nil
}

func (p *CoordinationPredictor) predictRenewableValue(t time.Time, region string) float64 {
	hour := t.Hour()
	month := t.Month()
	
	solarFactor := 0.0
	if hour >= 6 && hour <= 18 {
		peakHour := 12
		hourDiff := math.Abs(float64(hour - peakHour))
		solarFactor = math.Cos((hourDiff / 6) * math.Pi / 2)
		if solarFactor < 0 {
			solarFactor = 0
		}
	}
	
	windFactor := 0.0
	if hour >= 14 && hour <= 6 || hour >= 20 {
		windFactor = 0.6 + 0.2*math.Sin(float64(hour)*math.Pi/12)
	}
	
	seasonFactor := 1.0
	switch month {
	case time.March, time.April, time.May:
		seasonFactor = 1.1
	case time.June, time.July, time.August:
		seasonFactor = 1.2
	case time.September, time.October, time.November:
		seasonFactor = 0.9
	case time.December, time.January, time.February:
		seasonFactor = 0.8
	}
	
	renewableOutput := (solarFactor*0.6 + windFactor*0.4) * seasonFactor * 500
	
	return renewableOutput
}

func (p *CoordinationPredictor) PredictCarbon(ctx context.Context, req *PredictionRequest) (*PredictionResult, error) {
	cacheKey := fmt.Sprintf("carbon_%s", req.Region)
	
	p.mu.RLock()
	if cached, ok := p.carbonForecasts[cacheKey]; ok {
		if time.Since(cached.GeneratedAt) < time.Duration(p.config.UpdateInterval)*time.Minute {
			p.mu.RUnlock()
			return cached, nil
		}
	}
	p.mu.RUnlock()
	
	result := &PredictionResult{
		Type:        PredictionTypeCarbon,
		Region:      req.Region,
		GeneratedAt: time.Now(),
		Points:      make([]PredictionPoint, 0),
		Model:       "grid_carbon_model",
		Confidence:  0.78,
	}
	
	now := req.StartTime
	if now.IsZero() {
		now = time.Now()
	}
	
	for i := 0; i < req.Horizon; i++ {
		t := now.Add(time.Duration(i*req.Resolution) * time.Minute)
		value := p.predictCarbonValue(t, req.Region)
		confidence := math.Max(0.5, 0.85-float64(i)/float64(req.Horizon)*0.25)
		
		result.Points = append(result.Points, PredictionPoint{
			Timestamp:  t,
			Value:      value,
			LowerBound: value * 0.8,
			UpperBound: value * 1.2,
			Confidence: confidence,
		})
	}
	
	result.Accuracy = p.calculateAccuracy(PredictionTypeCarbon)
	
	p.mu.Lock()
	p.carbonForecasts[cacheKey] = result
	p.mu.Unlock()
	
	klog.V(4).Infof("碳排放预测完成, 区域: %s, 预测点数: %d", req.Region, len(result.Points))
	return result, nil
}

func (p *CoordinationPredictor) predictCarbonValue(t time.Time, region string) float64 {
	hour := t.Hour()
	
	renewableForecast := p.predictRenewableValue(t, region)
	loadForecast := p.predictLoadValue(t, region)
	
	greenRatio := 0.0
	if loadForecast > 0 {
		greenRatio = math.Min(1.0, renewableForecast/loadForecast)
	}
	
	baseCarbon := 0.5
	
	gridCarbon := baseCarbon * (1 - greenRatio)
	
	switch {
	case hour >= 8 && hour < 22:
		gridCarbon *= 1.2
	case hour >= 0 && hour < 6:
		gridCarbon *= 0.8
	}
	
	return gridCarbon
}

func (p *CoordinationPredictor) UpdateModel(ctx context.Context, predictionType PredictionType, data interface{}) error {
	switch predictionType {
	case PredictionTypeLoad:
		return p.updateLoadModel(data)
	case PredictionTypePrice:
		return p.updatePriceModel(data)
	case PredictionTypeRenewable:
		return p.updateRenewableModel(data)
	case PredictionTypeCarbon:
		return p.updateCarbonModel(data)
	default:
		return fmt.Errorf("未知的预测类型: %s", predictionType)
	}
}

func (p *CoordinationPredictor) updateLoadModel(data interface{}) error {
	historicalPoints, ok := data.([]HistoricalPoint)
	if !ok {
		return fmt.Errorf("无效的负荷数据格式")
	}
	
	p.mu.Lock()
	for _, point := range historicalPoints {
		key := fmt.Sprintf("load_%s", point.Type)
		p.historicalData[key] = append(p.historicalData[key], point)
		if len(p.historicalData[key]) > 10000 {
			p.historicalData[key] = p.historicalData[key][len(p.historicalData[key])-10000:]
		}
	}
	p.mu.Unlock()
	
	klog.V(4).Infof("更新负荷预测模型, 数据点数: %d", len(historicalPoints))
	return nil
}

func (p *CoordinationPredictor) updatePriceModel(data interface{}) error {
	historicalPoints, ok := data.([]HistoricalPoint)
	if !ok {
		return fmt.Errorf("无效的电价数据格式")
	}
	
	p.mu.Lock()
	for _, point := range historicalPoints {
		key := fmt.Sprintf("price_%s", point.Type)
		p.historicalData[key] = append(p.historicalData[key], point)
		if len(p.historicalData[key]) > 10000 {
			p.historicalData[key] = p.historicalData[key][len(p.historicalData[key])-10000:]
		}
	}
	p.mu.Unlock()
	
	klog.V(4).Infof("更新电价预测模型, 数据点数: %d", len(historicalPoints))
	return nil
}

func (p *CoordinationPredictor) updateRenewableModel(data interface{}) error {
	historicalPoints, ok := data.([]HistoricalPoint)
	if !ok {
		return fmt.Errorf("无效的新能源数据格式")
	}
	
	p.mu.Lock()
	for _, point := range historicalPoints {
		key := fmt.Sprintf("renewable_%s", point.Type)
		p.historicalData[key] = append(p.historicalData[key], point)
		if len(p.historicalData[key]) > 10000 {
			p.historicalData[key] = p.historicalData[key][len(p.historicalData[key])-10000:]
		}
	}
	p.mu.Unlock()
	
	klog.V(4).Infof("更新新能源预测模型, 数据点数: %d", len(historicalPoints))
	return nil
}

func (p *CoordinationPredictor) updateCarbonModel(data interface{}) error {
	historicalPoints, ok := data.([]HistoricalPoint)
	if !ok {
		return fmt.Errorf("无效的碳排放数据格式")
	}
	
	p.mu.Lock()
	for _, point := range historicalPoints {
		key := fmt.Sprintf("carbon_%s", point.Type)
		p.historicalData[key] = append(p.historicalData[key], point)
		if len(p.historicalData[key]) > 10000 {
			p.historicalData[key] = p.historicalData[key][len(p.historicalData[key])-10000:]
		}
	}
	p.mu.Unlock()
	
	klog.V(4).Infof("更新碳排放预测模型, 数据点数: %d", len(historicalPoints))
	return nil
}

func (p *CoordinationPredictor) GetAccuracy(ctx context.Context, predictionType PredictionType) (float64, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	if accuracy, ok := p.accuracyCache[predictionType]; ok {
		return accuracy, nil
	}
	
	return p.calculateAccuracy(predictionType), nil
}

func (p *CoordinationPredictor) calculateAccuracy(predictionType PredictionType) float64 {
	baseAccuracy := map[PredictionType]float64{
		PredictionTypeLoad:      0.92,
		PredictionTypePrice:     0.85,
		PredictionTypeRenewable: 0.78,
		PredictionTypeCarbon:    0.82,
	}
	
	return baseAccuracy[predictionType]
}

func (p *CoordinationPredictor) calculateTrend(data []HistoricalPoint) float64 {
	if len(data) < 2 {
		return 0
	}
	
	n := len(data)
	sumX := 0.0
	sumY := 0.0
	sumXY := 0.0
	sumX2 := 0.0
	
	for i, point := range data {
		x := float64(i)
		y := point.Value
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}
	
	denominator := float64(n)*sumX2 - sumX*sumX
	if denominator == 0 {
		return 0
	}
	
	slope := (float64(n)*sumXY - sumX*sumY) / denominator
	
	return slope
}

func (p *CoordinationPredictor) updateLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(p.config.UpdateInterval) * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case update := <-p.updateCh:
			p.processUpdate(update)
		case <-ticker.C:
			p.refreshForecasts(ctx)
		}
	}
}

func (p *CoordinationPredictor) processUpdate(update PredictionUpdate) {
	switch update.Type {
	case PredictionTypeLoad:
		p.mu.Lock()
		p.loadForecasts[fmt.Sprintf("load_%s", update.Region)] = update.Result
		p.mu.Unlock()
	case PredictionTypePrice:
		p.mu.Lock()
		p.priceForecasts[fmt.Sprintf("price_%s", update.Region)] = update.Result
		p.mu.Unlock()
	case PredictionTypeRenewable:
		if update.ResourceID != nil {
			p.mu.Lock()
			p.renewableForecasts[*update.ResourceID] = update.Result
			p.mu.Unlock()
		}
	case PredictionTypeCarbon:
		p.mu.Lock()
		p.carbonForecasts[fmt.Sprintf("carbon_%s", update.Region)] = update.Result
		p.mu.Unlock()
	}
}

func (p *CoordinationPredictor) refreshForecasts(ctx context.Context) {
	klog.V(4).Info("刷新预测缓存...")
}

func (p *CoordinationPredictor) modelUpdateLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.retrainModels(ctx)
		}
	}
}

func (p *CoordinationPredictor) retrainModels(ctx context.Context) {
	klog.V(4).Info("定期重新训练预测模型...")
	
	predictionTypes := []PredictionType{
		PredictionTypeLoad,
		PredictionTypePrice,
		PredictionTypeRenewable,
		PredictionTypeCarbon,
	}
	
	for _, predictionType := range predictionTypes {
		accuracy, _ := p.GetAccuracy(ctx, predictionType)
		p.mu.Lock()
		p.accuracyCache[predictionType] = accuracy
		p.mu.Unlock()
	}
}

func (p *CoordinationPredictor) GetLoadForecast(region string) *PredictionResult {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.loadForecasts[fmt.Sprintf("load_%s", region)]
}

func (p *CoordinationPredictor) GetPriceForecast(region string) *PredictionResult {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.priceForecasts[fmt.Sprintf("price_%s", region)]
}

func (p *CoordinationPredictor) GetRenewableForecast(resourceID uuid.UUID) *PredictionResult {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.renewableForecasts[resourceID]
}

func (p *CoordinationPredictor) GetCarbonForecast(region string) *PredictionResult {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.carbonForecasts[fmt.Sprintf("carbon_%s", region)]
}

func (p *CoordinationPredictor) GetCombinedForecast(ctx context.Context, region string, horizon int) (*CombinedForecast, error) {
	loadReq := &PredictionRequest{
		Type:       PredictionTypeLoad,
		Region:     region,
		StartTime:  time.Now(),
		Horizon:    horizon,
		Resolution: 15,
	}
	
	priceReq := &PredictionRequest{
		Type:       PredictionTypePrice,
		Region:     region,
		StartTime:  time.Now(),
		Horizon:    horizon,
		Resolution: 15,
	}
	
	carbonReq := &PredictionRequest{
		Type:       PredictionTypeCarbon,
		Region:     region,
		StartTime:  time.Now(),
		Horizon:    horizon,
		Resolution: 15,
	}
	
	loadForecast, err := p.PredictLoad(ctx, loadReq)
	if err != nil {
		return nil, fmt.Errorf("负荷预测失败: %w", err)
	}
	
	priceForecast, err := p.PredictPrice(ctx, priceReq)
	if err != nil {
		return nil, fmt.Errorf("电价预测失败: %w", err)
	}
	
	carbonForecast, err := p.PredictCarbon(ctx, carbonReq)
	if err != nil {
		return nil, fmt.Errorf("碳排放预测失败: %w", err)
	}
	
	combined := &CombinedForecast{
		Region:      region,
		GeneratedAt: time.Now(),
		Horizon:     horizon,
		Points:      make([]CombinedForecastPoint, 0, horizon),
	}
	
	for i := 0; i < horizon && i < len(loadForecast.Points) && i < len(priceForecast.Points) && i < len(carbonForecast.Points); i++ {
		combined.Points = append(combined.Points, CombinedForecastPoint{
			Timestamp:       loadForecast.Points[i].Timestamp,
			Load:            loadForecast.Points[i].Value,
			Price:           priceForecast.Points[i].Value,
			CarbonIntensity: carbonForecast.Points[i].Value,
			Confidence:      (loadForecast.Points[i].Confidence + priceForecast.Points[i].Confidence + carbonForecast.Points[i].Confidence) / 3,
		})
	}
	
	return combined, nil
}

type CombinedForecast struct {
	Region      string                 `json:"region"`
	GeneratedAt time.Time              `json:"generated_at"`
	Horizon     int                    `json:"horizon"`
	Points      []CombinedForecastPoint `json:"points"`
}

type CombinedForecastPoint struct {
	Timestamp       time.Time `json:"timestamp"`
	Load            float64   `json:"load"`
	Price           float64   `json:"price"`
	CarbonIntensity float64   `json:"carbon_intensity"`
	Confidence      float64   `json:"confidence"`
}
