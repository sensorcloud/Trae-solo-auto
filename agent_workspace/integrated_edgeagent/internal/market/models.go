package market

import (
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
)

var validate = validator.New()

type Asset struct {
	gorm.Model
	UserID        uint    `gorm:"not null" json:"user_id"`
	Name          string  `gorm:"not null" json:"name"`
	Description   string  `json:"description"`
	GPUModel      string  `gorm:"not null" json:"gpu_model"`
	GPUNumber     int     `gorm:"not null" json:"gpu_number"`
	Memory        int     `json:"memory"`
	Region        string  `gorm:"not null" json:"region"`
	Price         float64 `gorm:"not null" json:"price"`
	Status        string  `gorm:"default:'available'" json:"status"`
	Performance   string  `json:"performance"`
}

type Order struct {
	gorm.Model
	UserID      uint    `gorm:"not null" json:"user_id"`
	AssetID     uint    `gorm:"not null" json:"asset_id"`
	Quantity    int     `gorm:"not null" json:"quantity"`
	TotalPrice  float64 `gorm:"not null" json:"total_price"`
	Status      string  `gorm:"default:'pending'" json:"status"`
	PaymentMethod string `json:"payment_method"`
	Duration    int     `json:"duration"`
}

type CreateAssetRequest struct {
	Name        string  `json:"name" validate:"required,max=100"`
	Description string  `json:"description" validate:"max=500"`
	GPUModel    string  `json:"gpu_model" validate:"required"`
	GPUNumber   int     `json:"gpu_number" validate:"required,min=1"`
	Memory      int     `json:"memory" validate:"min=1"`
	Region      string  `json:"region" validate:"required"`
	Price       float64 `json:"price" validate:"required,min=0"`
	Performance string  `json:"performance"`
}

type CreateOrderRequest struct {
	AssetID       uint    `json:"asset_id" validate:"required"`
	Quantity      int     `json:"quantity" validate:"required,min=1"`
	PaymentMethod string  `json:"payment_method" validate:"required,oneof=balance"`
	Duration      int     `json:"duration" validate:"required,min=1,max=72"`
}

type AssetResponse struct {
	ID          uint    `json:"id"`
	UserID      uint    `json:"user_id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	GPUModel    string  `json:"gpu_model"`
	GPUNumber   int     `json:"gpu_number"`
	Memory      int     `json:"memory"`
	Region      string  `json:"region"`
	Price       float64 `json:"price"`
	Status      string  `json:"status"`
	Performance string  `json:"performance"`
	CreatedAt   int64   `json:"created_at"`
	UpdatedAt   int64   `json:"updated_at"`
}

type OrderResponse struct {
	ID            uint    `json:"id"`
	UserID        uint    `json:"user_id"`
	AssetID       uint    `json:"asset_id"`
	Quantity      int     `json:"quantity"`
	TotalPrice    float64 `json:"total_price"`
	Status        string  `json:"status"`
	PaymentMethod string  `json:"payment_method"`
	Duration      int     `json:"duration"`
	CreatedAt     int64   `json:"created_at"`
	UpdatedAt     int64   `json:"updated_at"`
}

func (a *Asset) ToResponse() AssetResponse {
	return AssetResponse{
		ID:          a.ID,
		UserID:      a.UserID,
		Name:        a.Name,
		Description: a.Description,
		GPUModel:    a.GPUModel,
		GPUNumber:   a.GPUNumber,
		Memory:      a.Memory,
		Region:      a.Region,
		Price:       a.Price,
		Status:      a.Status,
		Performance: a.Performance,
		CreatedAt:   a.CreatedAt.Unix(),
		UpdatedAt:   a.UpdatedAt.Unix(),
	}
}

func (o *Order) ToResponse() OrderResponse {
	return OrderResponse{
		ID:            o.ID,
		UserID:        o.UserID,
		AssetID:       o.AssetID,
		Quantity:      o.Quantity,
		TotalPrice:    o.TotalPrice,
		Status:        o.Status,
		PaymentMethod: o.PaymentMethod,
		Duration:      o.Duration,
		CreatedAt:     o.CreatedAt.Unix(),
		UpdatedAt:     o.UpdatedAt.Unix(),
	}
}