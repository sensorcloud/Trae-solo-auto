package coordination

import (
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
)

var validate = validator.New()

type ScheduleHistory struct {
	gorm.Model
	UserID       uint    `gorm:"not null" json:"user_id"`
	Type         string  `gorm:"not null" json:"type"`
	Input        string  `json:"input"`
	Output       string  `json:"output"`
	Status       string  `gorm:"default:'completed'" json:"status"`
	Optimization string  `json:"optimization"`
	CostSaved    float64 `json:"cost_saved"`
	CarbonReduced float64 `json:"carbon_reduced"`
}

type PredictRequest struct {
	Hours     int     `json:"hours" validate:"required,min=1,max=72"`
	Region    string  `json:"region"`
	LoadLevel float64 `json:"load_level"`
}

type OptimizeRequest struct {
	TaskType      string  `json:"task_type" validate:"required"`
	Duration      int     `json:"duration" validate:"required,min=1"`
	Budget        float64 `json:"budget"`
	CarbonTarget  float64 `json:"carbon_target" validate:"min=0,max=1"`
	Preferences   string  `json:"preferences"`
}

type ExecuteRequest struct {
	ScheduleID uint `json:"schedule_id" validate:"required"`
}

type PredictResponse struct {
	Hours      int               `json:"hours"`
	Region     string            `json:"region"`
	Prediction []LoadPrediction  `json:"prediction"`
}

type LoadPrediction struct {
	Hour       int     `json:"hour"`
	Load       float64 `json:"load"`
	Price      float64 `json:"price"`
	Carbon     float64 `json:"carbon"`
}

type OptimizeResponse struct {
	BestOption    OptimizationOption `json:"best_option"`
	Alternatives  []OptimizationOption `json:"alternatives"`
	TotalCost     float64            `json:"total_cost"`
	CarbonEmission float64           `json:"carbon_emission"`
}

type OptimizationOption struct {
	ID           string  `json:"id"`
	StartTime    int     `json:"start_time"`
	Duration     int     `json:"duration"`
	Cost         float64 `json:"cost"`
	Carbon       float64 `json:"carbon"`
	Score        float64 `json:"score"`
	Description  string  `json:"description"`
}

type ScheduleHistoryResponse struct {
	ID            uint    `json:"id"`
	UserID        uint    `json:"user_id"`
	Type          string  `json:"type"`
	Status        string  `json:"status"`
	CostSaved     float64 `json:"cost_saved"`
	CarbonReduced float64 `json:"carbon_reduced"`
	CreatedAt     int64   `json:"created_at"`
}

func (s *ScheduleHistory) ToResponse() ScheduleHistoryResponse {
	return ScheduleHistoryResponse{
		ID:            s.ID,
		UserID:        s.UserID,
		Type:          s.Type,
		Status:        s.Status,
		CostSaved:     s.CostSaved,
		CarbonReduced: s.CarbonReduced,
		CreatedAt:     s.CreatedAt.Unix(),
	}
}