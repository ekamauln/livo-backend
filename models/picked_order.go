package models

import (
	"time"

	"gorm.io/gorm"
)

type PickedOrder struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	OrderGineeID uint           `gorm:"not null;index" json:"order_ginee_id"`
	PickedBy     uint           `gorm:"not null;index" json:"picked_by"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationship
	Order              *Order              `gorm:"foreignKey:OrderGineeID" json:"order,omitempty"`
	Picker             *User               `gorm:"foreignKey:PickedBy" json:"picker,omitempty"`
	PickedOrderDetails []PickedOrderDetail `gorm:"foreignKey:PickedOrderID" json:"picked_order_details"`
}

type PickedOrderDetail struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	PickedOrderID uint      `gorm:"not null;index" json:"picked_order_id"`
	Sku           string    `gorm:"not null;index" json:"sku"`
	ProductName   string    `gorm:"not null" json:"product_name"`
	Variant       string    `json:"variant"`
	Quantity      int       `gorm:"not null" json:"quantity"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`

	// Relationship
	Product *Product `json:"product,omitempty" gorm:"-"`
}

type PickedOrderResponse struct {
	ID                 uint                        `json:"id"`
	OrderGineeID       uint                        `json:"order_ginee_id"`
	PickedBy           uint                        `json:"picked_by"`
	CreatedAt          time.Time                   `json:"created_at"`
	UpdatedAt          time.Time                   `json:"updated_at"`
	Order              *OrderResponse              `json:"order,omitempty"`
	Picker             *UserResponse               `json:"picker,omitempty"`
	PickedOrderDetails []PickedOrderDetailResponse `json:"picked_order_details"`
}

type PickedOrderDetailResponse struct {
	ID            uint             `json:"id"`
	PickedOrderID uint             `json:"picked_order_id"`
	Sku           string           `json:"sku"`
	ProductName   string           `json:"product_name"`
	Variant       string           `json:"variant"`
	Quantity      int              `json:"quantity"`
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
	Product       *ProductResponse `json:"product,omitempty"`
}

// ToPickedOrderResponse converts PickedOrder model to PickedOrderResponse
func (po *PickedOrder) ToPickedOrderResponse() PickedOrderResponse {
	details := make([]PickedOrderDetailResponse, len(po.PickedOrderDetails))
	for i, detail := range po.PickedOrderDetails {
		detailResp := PickedOrderDetailResponse{
			ID:            detail.ID,
			PickedOrderID: detail.PickedOrderID,
			Sku:           detail.Sku,
			ProductName:   detail.ProductName,
			Variant:       detail.Variant,
			Quantity:      detail.Quantity,
			CreatedAt:     detail.CreatedAt,
			UpdatedAt:     detail.UpdatedAt,
		}

		// Include product data if exists
		if detail.Product != nil {
			productResp := detail.Product.ToProductResponse()
			detailResp.Product = &productResp
		}

		details[i] = detailResp
	}

	response := PickedOrderResponse{
		ID:                 po.ID,
		OrderGineeID:       po.OrderGineeID,
		PickedBy:           po.PickedBy,
		CreatedAt:          po.CreatedAt,
		UpdatedAt:          po.UpdatedAt,
		PickedOrderDetails: details,
	}

	// Include order data if exists
	if po.Order != nil {
		orderResp := po.Order.ToOrderResponse()
		response.Order = &orderResp
	}

	// Include picker data if exists
	if po.Picker != nil {
		pickerResp := po.Picker.ToUserResponse()
		response.Picker = &pickerResp
	}

	return response
}

// LoadProducts manually loads products for all pick order details by SKU
func (po *PickedOrder) LoadProducts(db *gorm.DB) error {
	for i := range po.PickedOrderDetails {
		var product Product
		if err := db.Where("sku = ?", po.PickedOrderDetails[i].Sku).First(&product).Error; err == nil {
			po.PickedOrderDetails[i].Product = &product
		}
		// Silently skip if product not found
	}
	return nil
}
