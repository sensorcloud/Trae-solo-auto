package agent

import (
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
)

var validate = validator.New()

type Agent struct {
	gorm.Model
	UserID      uint   `gorm:"not null" json:"user_id"`
	Name        string `gorm:"not null" json:"name"`
	Description string `json:"description"`
	Runtime     string `gorm:"not null" json:"runtime"`
	Status      string `gorm:"default:'stopped'" json:"status"`
	Config      string `gorm:"type:text" json:"config"`
	Code        string `gorm:"type:text" json:"code"`
	Resources   string `json:"resources"`
}

type CreateAgentRequest struct {
	Name        string `json:"name" validate:"required,max=100"`
	Description string `json:"description" validate:"max=500"`
	Runtime     string `json:"runtime" validate:"required,oneof=python node"`
	Config      string `json:"config"`
	Code        string `json:"code"`
	Resources   string `json:"resources"`
}

type UpdateAgentRequest struct {
	Name        string `json:"name" validate:"omitempty,max=100"`
	Description string `json:"description" validate:"omitempty,max=500"`
	Config      string `json:"config"`
	Code        string `json:"code"`
}

type ExecuteRequest struct {
	Input       string            `json:"input"`
	ToolCalls   []ToolCall        `json:"tool_calls"`
	Context     map[string]string `json:"context"`
}

type ToolCall struct {
	ToolName string                 `json:"tool_name"`
	Params   map[string]interface{} `json:"params"`
}

type ExecuteResponse struct {
	Output   string                 `json:"output"`
	Status   string                 `json:"status"`
	Context  map[string]string      `json:"context"`
	ToolCalls []ToolCall            `json:"tool_calls"`
}

type AgentResponse struct {
	ID          uint   `json:"id"`
	UserID      uint   `json:"user_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Runtime     string `json:"runtime"`
	Status      string `json:"status"`
	Config      string `json:"config"`
	Resources   string `json:"resources"`
	CreatedAt   int64  `json:"created_at"`
	UpdatedAt   int64  `json:"updated_at"`
}

func (a *Agent) ToResponse() AgentResponse {
	return AgentResponse{
		ID:          a.ID,
		UserID:      a.UserID,
		Name:        a.Name,
		Description: a.Description,
		Runtime:     a.Runtime,
		Status:      a.Status,
		Config:      a.Config,
		Resources:   a.Resources,
		CreatedAt:   a.CreatedAt.Unix(),
		UpdatedAt:   a.UpdatedAt.Unix(),
	}
}