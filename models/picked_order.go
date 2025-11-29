package models

import (
	"time"

	"gorm.io/gorm"
)

type PickedOrder struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	OrderID   uint           `gorm:"not null;index" json:"order_id"`
	PickedBy  uint           `gorm:"not null;index" json:"picked_by"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationship
	Order        *Order `gorm:"foreignKey:OrderID" json:"order,omitempty"`
	PickOperator *User  `gorm:"foreignKey:PickedBy" json:"picker,omitempty"`
}

type PickedOrderResponse struct {
	ID        uint           `json:"id"`
	OrderID   uint           `json:"order_id"`
	PickedBy  uint           `json:"picked_by"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`

	// Related data
	Order        *OrderResponse `json:"order,omitempty"`
	PickOperator *UserResponse  `json:"picker,omitempty"`
}

// ToPickedOrderResponse converts PickedOrder model to PickedOrderResponse
func (po *PickedOrder) ToPickedOrderResponse() PickedOrderResponse {
	response := PickedOrderResponse{
		ID:        po.ID,
		OrderID:   po.OrderID,
		PickedBy:  po.PickedBy,
		CreatedAt: po.CreatedAt,
		UpdatedAt: po.UpdatedAt,
	}

	// Include order data if loaded
	if po.Order != nil {
		orderResponse := po.Order.ToOrderResponse()
		response.Order = &orderResponse
	}

	// Include picker data if loaded
	if po.PickOperator != nil {
		pickerResponse := po.PickOperator.ToUserResponse()
		response.PickOperator = &pickerResponse
	}

	return response
}
