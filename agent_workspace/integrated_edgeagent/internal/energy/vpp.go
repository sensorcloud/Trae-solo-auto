package energy

type VirtualPowerPlant struct {
	ID              uint    `gorm:"primaryKey" json:"id"`
	UserID          uint    `gorm:"not null" json:"user_id"`
	Name            string  `gorm:"not null" json:"name"`
	Description     string  `json:"description"`
	Type            string  `gorm:"not null" json:"type"`
	Status          string  `gorm:"default:'active'" json:"status"`
	ControlStrategy string  `json:"control_strategy"`
	TotalCapacity   float64 `json:"total_capacity"`
}

type VPPResponse struct {
	ID              uint    `json:"id"`
	UserID          uint    `json:"user_id"`
	Name            string  `json:"name"`
	Description     string  `json:"description"`
	Type            string  `json:"type"`
	Status          string  `json:"status"`
	ControlStrategy string  `json:"control_strategy"`
	TotalCapacity   float64 `json:"total_capacity"`
}

func (v *VirtualPowerPlant) ToResponse() VPPResponse {
	return VPPResponse{
		ID:              v.ID,
		UserID:          v.UserID,
		Name:            v.Name,
		Description:     v.Description,
		Type:            v.Type,
		Status:          v.Status,
		ControlStrategy: v.ControlStrategy,
		TotalCapacity:   v.TotalCapacity,
	}
}
