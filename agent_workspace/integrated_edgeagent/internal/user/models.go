package user

import (
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
)

var validate = validator.New()

type User struct {
	gorm.Model
	Email        string `gorm:"unique;not null" json:"email"`
	Password     string `gorm:"not null" json:"-"`
	Phone        string `json:"phone"`
	Nickname     string `json:"nickname"`
	Role         string `gorm:"default:'user'" json:"role"`
	Status       string `gorm:"default:'active'" json:"status"`
	Balance      float64 `gorm:"default:0" json:"balance"`
	EmailVerified bool   `gorm:"default:false" json:"email_verified"`
}

type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	Phone    string `json:"phone" validate:"omitempty,e164"`
	Nickname string `json:"nickname" validate:"max=50"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type UpdateUserRequest struct {
	Nickname string `json:"nickname" validate:"max=50"`
	Phone    string `json:"phone" validate:"omitempty,e164"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpireAt     int64  `json:"expire_at"`
}

type UserResponse struct {
	ID            uint    `json:"id"`
	Email         string  `json:"email"`
	Phone         string  `json:"phone"`
	Nickname      string  `json:"nickname"`
	Role          string  `json:"role"`
	Status        string  `json:"status"`
	Balance       float64 `json:"balance"`
	EmailVerified bool    `json:"email_verified"`
	CreatedAt     int64   `json:"created_at"`
	UpdatedAt     int64   `json:"updated_at"`
}

func (u *User) ToResponse() UserResponse {
	return UserResponse{
		ID:            u.ID,
		Email:         u.Email,
		Phone:         u.Phone,
		Nickname:      u.Nickname,
		Role:          u.Role,
		Status:        u.Status,
		Balance:       u.Balance,
		EmailVerified: u.EmailVerified,
		CreatedAt:     u.CreatedAt.Unix(),
		UpdatedAt:     u.UpdatedAt.Unix(),
	}
}