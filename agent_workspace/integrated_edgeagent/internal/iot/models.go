package iot

import (
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
)

var validate = validator.New()

type Device struct {
	gorm.Model
	UserID      uint    `gorm:"not null" json:"user_id"`
	Name        string  `gorm:"not null" json:"name"`
	Description string  `json:"description"`
	Protocol    string  `gorm:"not null" json:"protocol"`
	Address     string  `json:"address"`
	Port        int     `json:"port"`
	Status      string  `gorm:"default:'offline'" json:"status"`
	LastSeen    int64   `json:"last_seen"`
	Metadata    string  `json:"metadata"`
}

type Telemetry struct {
	gorm.Model
	DeviceID   uint    `gorm:"not null" json:"device_id"`
	Timestamp  int64   `gorm:"not null" json:"timestamp"`
	Metric     string  `gorm:"not null" json:"metric"`
	Value      float64 `gorm:"not null" json:"value"`
	Unit       string  `json:"unit"`
}

type CreateDeviceRequest struct {
	Name        string `json:"name" validate:"required,max=100"`
	Description string `json:"description" validate:"max=500"`
	Protocol    string `json:"protocol" validate:"required,oneof=mqtt modbus opcua http"`
	Address     string `json:"address" validate:"required"`
	Port        int    `json:"port"`
	Metadata    string `json:"metadata"`
}

type SubmitTelemetryRequest struct {
	Metric    string  `json:"metric" validate:"required"`
	Value     float64 `json:"value" validate:"required"`
	Unit      string  `json:"unit"`
	Timestamp int64   `json:"timestamp"`
}

type DeviceResponse struct {
	ID          uint    `json:"id"`
	UserID      uint    `json:"user_id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Protocol    string  `json:"protocol"`
	Address     string  `json:"address"`
	Port        int     `json:"port"`
	Status      string  `json:"status"`
	LastSeen    int64   `json:"last_seen"`
	Metadata    string  `json:"metadata"`
	CreatedAt   int64   `json:"created_at"`
	UpdatedAt   int64   `json:"updated_at"`
}

type TelemetryResponse struct {
	ID         uint    `json:"id"`
	DeviceID   uint    `json:"device_id"`
	Timestamp  int64   `json:"timestamp"`
	Metric     string  `json:"metric"`
	Value      float64 `json:"value"`
	Unit       string  `json:"unit"`
}

func (d *Device) ToResponse() DeviceResponse {
	return DeviceResponse{
		ID:          d.ID,
		UserID:      d.UserID,
		Name:        d.Name,
		Description: d.Description,
		Protocol:    d.Protocol,
		Address:     d.Address,
		Port:        d.Port,
		Status:      d.Status,
		LastSeen:    d.LastSeen,
		Metadata:    d.Metadata,
		CreatedAt:   d.CreatedAt.Unix(),
		UpdatedAt:   d.UpdatedAt.Unix(),
	}
}

func (t *Telemetry) ToResponse() TelemetryResponse {
	return TelemetryResponse{
		ID:        t.ID,
		DeviceID:  t.DeviceID,
		Timestamp: t.Timestamp,
		Metric:    t.Metric,
		Value:     t.Value,
		Unit:      t.Unit,
	}
}