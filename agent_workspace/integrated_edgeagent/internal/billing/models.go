package billing

import (
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
)

var validate = validator.New()

type Bill struct {
	gorm.Model
	UserID      uint    `gorm:"not null" json:"user_id"`
	OrderID     uint    `json:"order_id"`
	Amount      float64 `gorm:"not null" json:"amount"`
	Status      string  `gorm:"default:'unpaid'" json:"status"`
	Type        string  `json:"type"`
	Description string  `json:"description"`
	DueDate     int64   `json:"due_date"`
}

type UsageRecord struct {
	gorm.Model
	UserID     uint    `gorm:"not null" json:"user_id"`
	ResourceID uint    `json:"resource_id"`
	ResourceType string `json:"resource_type"`
	Usage      float64 `gorm:"not null" json:"usage"`
	Unit       string  `json:"unit"`
	Timestamp  int64   `gorm:"not null" json:"timestamp"`
}

type PayBillRequest struct {
	BillID uint `json:"bill_id" validate:"required"`
}

type RecordUsageRequest struct {
	ResourceID   uint    `json:"resource_id" validate:"required"`
	ResourceType string  `json:"resource_type" validate:"required"`
	Usage        float64 `json:"usage" validate:"required,min=0"`
	Unit         string  `json:"unit"`
}

type BillResponse struct {
	ID          uint    `json:"id"`
	UserID      uint    `json:"user_id"`
	OrderID     uint    `json:"order_id"`
	Amount      float64 `json:"amount"`
	Status      string  `json:"status"`
	Type        string  `json:"type"`
	Description string  `json:"description"`
	DueDate     int64   `json:"due_date"`
	CreatedAt   int64   `json:"created_at"`
}

type UsageResponse struct {
	ID           uint    `json:"id"`
	UserID       uint    `json:"user_id"`
	ResourceID   uint    `json:"resource_id"`
	ResourceType string  `json:"resource_type"`
	Usage        float64 `json:"usage"`
	Unit         string  `json:"unit"`
	Timestamp    int64   `json:"timestamp"`
}

func (b *Bill) ToResponse() BillResponse {
	return BillResponse{
		ID:          b.ID,
		UserID:      b.UserID,
		OrderID:     b.OrderID,
		Amount:      b.Amount,
		Status:      b.Status,
		Type:        b.Type,
		Description: b.Description,
		DueDate:     b.DueDate,
		CreatedAt:   b.CreatedAt.Unix(),
	}
}

func (u *UsageRecord) ToResponse() UsageResponse {
	return UsageResponse{
		ID:           u.ID,
		UserID:       u.UserID,
		ResourceID:   u.ResourceID,
		ResourceType: u.ResourceType,
		Usage:        u.Usage,
		Unit:         u.Unit,
		Timestamp:    u.Timestamp,
	}
}