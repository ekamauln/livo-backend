package models

import (
	"time"

	"gorm.io/gorm"
)

type Box struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Code      string         `gorm:"unique;not null" json:"code" example:"PB"`
	Name      string         `gorm:"not null" json:"name" example:"Panjang Besar"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type BoxResponse struct {
	ID      uint      `json:"id"`
	Code    string    `json:"code"`
	Name    string    `json:"name"`
	Created time.Time `json:"created_at"`
	Updated time.Time `json:"updated_at"`
}

// ToBoxResponse converts Box model to BoxResponse
func (b *Box) ToBoxResponse() BoxResponse {
	return BoxResponse{
		ID:      b.ID,
		Code:    b.Code,
		Name:    b.Name,
		Created: b.CreatedAt,
		Updated: b.UpdatedAt,
	}
}
