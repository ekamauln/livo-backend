package models

import (
	"time"

	"gorm.io/gorm"
)

type Expedition struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Code      string         `gorm:"unique;not null" json:"code" example:"JNE"`
	Name      string         `gorm:"not null" json:"name" example:"J&T Express"`
	Slug      string         `gorm:"not null" json:"slug" example:"j&t-express"`
	Color     string         `json:"color" example:"#FF5733"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type ExpeditionResponse struct {
	ID      uint      `json:"id"`
	Code    string    `json:"code"`
	Name    string    `json:"name"`
	Slug    string    `json:"slug"`
	Color   string    `json:"color"`
	Created time.Time `json:"created_at"`
	Updated time.Time `json:"updated_at"`
}

// ToExpeditionResponse converts Expedition model to ExpeditionResponse
func (e *Expedition) ToExpeditionResponse() ExpeditionResponse {
	return ExpeditionResponse{
		ID:      e.ID,
		Code:    e.Code,
		Name:    e.Name,
		Slug:    e.Slug,
		Color:   e.Color,
		Created: e.CreatedAt,
		Updated: e.UpdatedAt,
	}
}
