package monitor

import (
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
)

var validate = validator.New()

type Alert struct {
	gorm.Model
	UserID        uint    `gorm:"not null" json:"user_id"`
	Name          string  `gorm:"not null" json:"name"`
	Description   string  `json:"description"`
	Metric        string  `gorm:"not null" json:"metric"`
	Operator      string  `gorm:"not null" json:"operator"`
	Threshold     float64 `gorm:"not null" json:"threshold"`
	Severity      string  `gorm:"default:'warning'" json:"severity"`
	Status        string  `gorm:"default:'active'" json:"status"`
	Notification  string  `json:"notification"`
}

type CreateAlertRequest struct {
	Name         string  `json:"name" validate:"required,max=100"`
	Description  string  `json:"description" validate:"max=500"`
	Metric       string  `json:"metric" validate:"required"`
	Operator     string  `json:"operator" validate:"required,oneof=> < >= <= == !="`
	Threshold    float64 `json:"threshold" validate:"required"`
	Severity     string  `json:"severity" validate:"oneof=info warning critical"`
	Notification string  `json:"notification"`
}

type UpdateAlertRequest struct {
	Name         string  `json:"name" validate:"omitempty,max=100"`
	Description  string  `json:"description" validate:"omitempty,max=500"`
	Metric       string  `json:"metric"`
	Operator     string  `json:"operator" validate:"omitempty,oneof=> < >= <= == !="`
	Threshold    float64 `json:"threshold"`
	Severity     string  `json:"severity" validate:"omitempty,oneof=info warning critical"`
	Notification string  `json:"notification"`
}

type AlertResponse struct {
	ID           uint    `json:"id"`
	UserID       uint    `json:"user_id"`
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	Metric       string  `json:"metric"`
	Operator     string  `json:"operator"`
	Threshold    float64 `json:"threshold"`
	Severity     string  `json:"severity"`
	Status       string  `json:"status"`
	Notification string  `json:"notification"`
	CreatedAt    int64   `json:"created_at"`
}

type HealthResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Uptime  string `json:"uptime"`
}

type SystemMetricsResponse struct {
	CPU        float64 `json:"cpu"`
	Memory     float64 `json:"memory"`
	Disk       float64 `json:"disk"`
	Network    float64 `json:"network"`
	ActiveAgents int   `json:"active_agents"`
	ActiveOrders int   `json:"active_orders"`
}

func (a *Alert) ToResponse() AlertResponse {
	return AlertResponse{
		ID:           a.ID,
		UserID:       a.UserID,
		Name:         a.Name,
		Description:  a.Description,
		Metric:       a.Metric,
		Operator:     a.Operator,
		Threshold:    a.Threshold,
		Severity:     a.Severity,
		Status:       a.Status,
		Notification: a.Notification,
		CreatedAt:    a.CreatedAt.Unix(),
	}
}