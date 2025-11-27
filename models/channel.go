package models

import (
	"time"

	"gorm.io/gorm"
)

type Channel struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Code      string         `gorm:"unique;not null" json:"code" example:"SP"`
	Name      string         `gorm:"not null;unique" json:"name" example:"Shopee"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type ChannelResponse struct {
	ID      uint      `json:"id"`
	Code    string    `json:"code" example:"SP"`
	Name    string    `json:"name"`
	Created time.Time `json:"created_at"`
	Updated time.Time `json:"updated_at"`
}

// ToChannelResponse converts Channel model to ChannelResponse
func (c *Channel) ToChannelResponse() ChannelResponse {
	return ChannelResponse{
		ID:      c.ID,
		Code:    c.Code,
		Name:    c.Name,
		Created: c.CreatedAt,
		Updated: c.UpdatedAt,
	}
}

// ToChannelMobileResponse converts Channel model to ChannelResponse for mobile use
func (c *Channel) ToChannelMobileResponse() ChannelResponse {
	return ChannelResponse{
		ID:      c.ID,
		Code:    c.Code,
		Name:    c.Name,
		Created: c.CreatedAt,
		Updated: c.UpdatedAt,
	}
}
