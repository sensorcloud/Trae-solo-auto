package market

import (
	"math"
	"time"
)

type PriceAggregator struct {
	records []*PriceRecord
}

type PriceInfo struct {
	AvgPrice    float64            `json:"avg_price"`
	MinPrice    float64            `json:"min_price"`
	MaxPrice    float64            `json:"max_price"`
	Last24Hours *PriceInfoSnapshot `json:"last_24h,omitempty"`
	Last7Days  *PriceInfoSnapshot `json:"last_7d,omitempty"`
	History    []*PriceRecord    `json:"history,omitempty"`
}

type PriceInfoSnapshot struct {
	AvgPrice float64 `json:"avg_price"`
	MinPrice float64 `json:"min_price"`
	MaxPrice float64 `json:"max_price"`
	Count    int     `json:"count"`
}

type PriceRecommendation struct {
	RecommendedPrice float64                `json:"recommended_price"`
	Confidence       float64               `json:"confidence"`
	Trend            string                `json:"trend"`
	Factors          []RecommendationFactor `json:"factors"`
}

type RecommendationFactor struct {
	Name   string  `json:"name"`
	Weight float64 `json:"weight"`
	Impact string  `json:"impact"`
}

func NewPriceAggregator() *PriceAggregator {
	return &PriceAggregator{
		records: make([]*PriceRecord, 0),
	}
}

func (pa *PriceAggregator) AddRecord(record *PriceRecord) {
	pa.records = append(pa.records, record)
	if len(pa.records) > 10000 {
		pa.records = pa.records[len(pa.records)-10000:]
	}
}

func (pa *PriceAggregator) GetPriceInfo(spec *ResourceSpec) *PriceInfo {
	info := &PriceInfo{
		AvgPrice: 0,
		MinPrice: math.MaxFloat64,
		MaxPrice: 0,
	}

	var totalPrice float64
	var count int

	now := time.Now()
	h24Ago := now.Add(-24 * time.Hour)
	d7Ago := now.Add(-7 * 24 * time.Hour)

	var h24Total, h24Min, h24Max float64
	var h24Count int
	var d7Total, d7Min, d7Max float64
	var d7Count int

	for _, record := range pa.records {
		if spec != nil && !pa.matchSpec(&record.ResourceSpec, spec) {
			continue
		}

		totalPrice += record.AvgPrice
		count++

		if record.AvgPrice < info.MinPrice {
			info.MinPrice = record.AvgPrice
		}
		if record.AvgPrice > info.MaxPrice {
			info.MaxPrice = record.AvgPrice
		}

		if record.Timestamp.After(h24Ago) {
			h24Total += record.AvgPrice
			h24Count++
			if h24Min == 0 || record.AvgPrice < h24Min {
				h24Min = record.AvgPrice
			}
			if record.AvgPrice > h24Max {
				h24Max = record.AvgPrice
			}
		}

		if record.Timestamp.After(d7Ago) {
			d7Total += record.AvgPrice
			d7Count++
			if d7Min == 0 || record.AvgPrice < d7Min {
				d7Min = record.AvgPrice
			}
			if record.AvgPrice > d7Max {
				d7Max = record.AvgPrice
			}
		}
	}

	if count > 0 {
		info.AvgPrice = totalPrice / float64(count)
	}
	if info.MinPrice == math.MaxFloat64 {
		info.MinPrice = 0
	}

	if h24Count > 0 {
		info.Last24Hours = &PriceInfoSnapshot{
			AvgPrice: h24Total / float64(h24Count),
			MinPrice: h24Min,
			MaxPrice: h24Max,
			Count:    h24Count,
		}
	}

	if d7Count > 0 {
		info.Last7Days = &PriceInfoSnapshot{
			AvgPrice: d7Total / float64(d7Count),
			MinPrice: d7Min,
			MaxPrice: d7Max,
			Count:    d7Count,
		}
	}

	if len(pa.records) > 100 {
		info.History = pa.records[len(pa.records)-100:]
	} else {
		info.History = pa.records
	}

	return info
}

func (pa *PriceAggregator) matchSpec(record, filter *ResourceSpec) bool {
	if filter == nil {
		return true
	}
	if filter.CPU > 0 && record.CPU < float64(int(filter.CPU*0.8)) {
		return false
	}
	if filter.GPU > 0 && record.GPU < filter.GPU/2 {
		return false
	}
	if filter.Memory > 0 && record.Memory < filter.Memory/2 {
		return false
	}
	return true
}

func (pa *PriceAggregator) GetRecommendation(spec *ResourceSpec) *PriceRecommendation {
	info := pa.GetPriceInfo(spec)

	trend := "stable"
	confidence := 0.5

	var factors []RecommendationFactor

	if info.Last24Hours != nil && info.Last7Days != nil {
		h24Avg := info.Last24Hours.AvgPrice
		d7Avg := info.Last7Days.AvgPrice

		if d7Avg > 0 {
			change := (h24Avg - d7Avg) / d7Avg

			if change > 0.1 {
				trend = "rising"
				factors = append(factors, RecommendationFactor{
					Name:   "market_trend",
					Weight: 0.3,
					Impact: "positive",
				})
			} else if change < -0.1 {
				trend = "falling"
				factors = append(factors, RecommendationFactor{
					Name:   "market_trend",
					Weight: 0.3,
					Impact: "negative",
				})
			}

			confidence = math.Min(0.9, 0.5+float64(info.Last7Days.Count)/1000)
		}
	}

	supplyFactor := 1.0
	if info.AvgPrice > 0 {
		if info.MinPrice < info.AvgPrice*0.7 {
			supplyFactor = 0.9
			factors = append(factors, RecommendationFactor{
				Name:   "supply_level",
				Weight: 0.2,
				Impact: "negative",
			})
		} else if info.MaxPrice > info.AvgPrice*1.3 {
			supplyFactor = 1.1
			factors = append(factors, RecommendationFactor{
				Name:   "demand_level",
				Weight: 0.2,
				Impact: "positive",
			})
		}
	}

	basePrice := info.AvgPrice
	if basePrice == 0 {
		basePrice = 100
	}

	recommendedPrice := basePrice * supplyFactor

	if spec != nil {
		if spec.GPU > 0 {
			gpuMultiplier := 1.0 + float64(spec.GPU)*0.5
			recommendedPrice *= gpuMultiplier
			factors = append(factors, RecommendationFactor{
				Name:   "gpu_requirement",
				Weight: 0.2,
				Impact: "positive",
			})
		}
		if spec.Duration > 0 && spec.Duration >= 60 {
			recommendedPrice *= 0.85
			factors = append(factors, RecommendationFactor{
				Name:   "long_duration",
				Weight: 0.1,
				Impact: "positive",
			})
		}
	}

	return &PriceRecommendation{
		RecommendedPrice: math.Round(recommendedPrice*100) / 100,
		Confidence:       confidence,
		Trend:            trend,
		Factors:          factors,
	}
}

type MarketStats struct {
	TotalOffers     int     `json:"total_offers"`
	ActiveOffers    int     `json:"active_offers"`
	TotalOrders     int     `json:"total_orders"`
	FulfilledOrders int     `json:"fulfilled_orders"`
	TotalVolume     float64 `json:"total_volume"`
}
