package energy

import (
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
)

var validate = validator.New()

type PowerSource struct {
	gorm.Model
	UserID       uint    `gorm:"not null" json:"user_id"`
	Name         string  `gorm:"not null" json:"name"`
	Description  string  `json:"description"`
	Type         string  `gorm:"not null" json:"type"`
	Capacity     float64 `json:"capacity"`
	OutputPower  float64 `json:"output_power"`
	Status       string  `gorm:"default:'active'" json:"status"`
	Location     string  `json:"location"`
	PricePerKWh  float64 `json:"price_per_kwh"`
	CarbonIntensity float64 `json:"carbon_intensity"`
}

type Storage struct {
	gorm.Model
	UserID       uint    `gorm:"not null" json:"user_id"`
	Name         string  `gorm:"not null" json:"name"`
	Description  string  `json:"description"`
	Capacity     float64 `json:"capacity"`
	CurrentSOC   float64 `gorm:"default:0" json:"current_soc"`
	MaxChargeRate float64 `json:"max_charge_rate"`
	MaxDischargeRate float64 `json:"max_discharge_rate"`
	Efficiency   float64 `json:"efficiency"`
	Status       string  `gorm:"default:'active'" json:"status"`
}

type CreatePowerSourceRequest struct {
	Name         string  `json:"name" validate:"required,max=100"`
	Description  string  `json:"description" validate:"max=500"`
	Type         string  `json:"type" validate:"required,oneof=solar wind grid battery"`
	Capacity     float64 `json:"capacity" validate:"min=0"`
	OutputPower  float64 `json:"output_power" validate:"min=0"`
	Location     string  `json:"location"`
	PricePerKWh  float64 `json:"price_per_kwh" validate:"min=0"`
	CarbonIntensity float64 `json:"carbon_intensity" validate:"min=0"`
}

type CreateStorageRequest struct {
	Name             string  `json:"name" validate:"required,max=100"`
	Description      string  `json:"description" validate:"max=500"`
	Capacity         float64 `json:"capacity" validate:"min=0"`
	MaxChargeRate    float64 `json:"max_charge_rate" validate:"min=0"`
	MaxDischargeRate float64 `json:"max_discharge_rate" validate:"min=0"`
	Efficiency       float64 `json:"efficiency" validate:"min=0,max=1"`
}

type ChargeRequest struct {
	Amount float64 `json:"amount" validate:"min=0"`
}

type DischargeRequest struct {
	Amount float64 `json:"amount" validate:"min=0"`
}

type PowerSourceResponse struct {
	ID              uint    `json:"id"`
	UserID          uint    `json:"user_id"`
	Name            string  `json:"name"`
	Description     string  `json:"description"`
	Type            string  `json:"type"`
	Capacity        float64 `json:"capacity"`
	OutputPower     float64 `json:"output_power"`
	Status          string  `json:"status"`
	Location        string  `json:"location"`
	PricePerKWh     float64 `json:"price_per_kwh"`
	CarbonIntensity float64 `json:"carbon_intensity"`
	CreatedAt       int64   `json:"created_at"`
	UpdatedAt       int64   `json:"updated_at"`
}

type StorageResponse struct {
	ID               uint    `json:"id"`
	UserID           uint    `json:"user_id"`
	Name             string  `json:"name"`
	Description      string  `json:"description"`
	Capacity         float64 `json:"capacity"`
	CurrentSOC       float64 `json:"current_soc"`
	MaxChargeRate    float64 `json:"max_charge_rate"`
	MaxDischargeRate float64 `json:"max_discharge_rate"`
	Efficiency       float64 `json:"efficiency"`
	Status           string  `json:"status"`
	CreatedAt        int64   `json:"created_at"`
	UpdatedAt        int64   `json:"updated_at"`
}

func (p *PowerSource) ToResponse() PowerSourceResponse {
	return PowerSourceResponse{
		ID:              p.ID,
		UserID:          p.UserID,
		Name:            p.Name,
		Description:     p.Description,
		Type:            p.Type,
		Capacity:        p.Capacity,
		OutputPower:     p.OutputPower,
		Status:          p.Status,
		Location:        p.Location,
		PricePerKWh:     p.PricePerKWh,
		CarbonIntensity: p.CarbonIntensity,
		CreatedAt:       p.CreatedAt.Unix(),
		UpdatedAt:       p.UpdatedAt.Unix(),
	}
}

func (s *Storage) ToResponse() StorageResponse {
	return StorageResponse{
		ID:               s.ID,
		UserID:           s.UserID,
		Name:             s.Name,
		Description:      s.Description,
		Capacity:         s.Capacity,
		CurrentSOC:       s.CurrentSOC,
		MaxChargeRate:    s.MaxChargeRate,
		MaxDischargeRate: s.MaxDischargeRate,
		Efficiency:       s.Efficiency,
		Status:           s.Status,
		CreatedAt:        s.CreatedAt.Unix(),
		UpdatedAt:        s.UpdatedAt.Unix(),
	}
}