package models

import (
	"time"

	"gorm.io/gorm"
)

type Outbound struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	Tracking        string         `gorm:"unique;not null" json:"tracking" example:"SPXID056205885386"`
	OutboundBy      *uint          `gorm:"not null" json:"outbound_by" example:"1"`
	Expedition      string         `gorm:"not null" json:"expedition" example:"JNE"`
	ExpeditionColor string         `gorm:"not null" json:"expedition_color" example:"#FF5733"`
	ExpeditionSlug  string         `gorm:"not null" json:"expedition_slug" example:"jne"`
	Complained      bool           `gorm:"default:false" json:"complained" example:"false"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationship
	Order            *Order `gorm:"-" json:"order,omitempty"`
	OutboundOperator *User  `gorm:"foreignKey:OutboundBy" json:"outbound_operator,omitempty"`
}

type OutboundResponse struct {
	ID              uint      `json:"id"`
	Tracking        string    `json:"tracking"`
	OutboundBy      *uint     `json:"outbound_by"`
	Expedition      string    `json:"expedition"`
	ExpeditionColor string    `json:"expedition_color"`
	ExpeditionSlug  string    `json:"expedition_slug"`
	Complained      bool      `json:"complained"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	// Related data
	Order            *OrderResponse `json:"order,omitempty"`
	OutboundOperator *UserResponse  `json:"outbound_operator,omitempty"`
}

// ToOutboundResponse converts Outbound model to OutboundResponse
func (ob *Outbound) ToOutboundResponse() OutboundResponse {
	response := OutboundResponse{
		ID:              ob.ID,
		Tracking:        ob.Tracking,
		OutboundBy:      ob.OutboundBy,
		Expedition:      ob.Expedition,
		ExpeditionColor: ob.ExpeditionColor,
		ExpeditionSlug:  ob.ExpeditionSlug,
		Complained:      ob.Complained,
		CreatedAt:       ob.CreatedAt,
		UpdatedAt:       ob.UpdatedAt,
	}

	// Include order data if loaded
	if ob.Order != nil {
		orderResponse := ob.Order.ToOrderResponse()
		response.Order = &orderResponse
	}

	// Include outbound operator data if loaded
	if ob.OutboundOperator != nil {
		outboundOperatorResponse := ob.OutboundOperator.ToUserResponse()
		response.OutboundOperator = &outboundOperatorResponse
	}

	return response
}
