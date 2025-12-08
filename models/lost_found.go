package models

import (
	"time"

	"gorm.io/gorm"
)

type LostFound struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	ProductSKU string         `json:"product_sku" example:"SKU12345"`
	Quantity   int            `gorm:"not null" json:"quantity" example:"10"`
	Reason     string         `gorm:"not null" json:"reason" example:"Damaged during transit"`
	CreatedBy  *uint          `gorm:"not null" json:"created_by" example:"john.doe"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationship
	CreateOperator *User    `gorm:"foreignKey:CreatedBy" json:"create_operator,omitempty"`
	Product        *Product `gorm:"foreignKey:ProductSKU;references:Sku" json:"product,omitempty"`
}

// LostFoundResponse represents lost and found data for API responses
type LostFoundResponse struct {
	ID         uint      `json:"id"`
	ProductSKU string    `json:"product_sku"`
	Quantity   int       `json:"quantity"`
	Reason     string    `json:"reason"`
	CreatedBy  *uint     `json:"created_by"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`

	// Related data
	CreateOperator *UserResponse    `json:"create_operator,omitempty"`
	Product        *ProductResponse `json:"product,omitempty"`
}

// ToLostFoundResponse converts LostFound model to LostFoundResponse
func (lf *LostFound) ToLostFoundResponse() LostFoundResponse {
	response := LostFoundResponse{
		ID:         lf.ID,
		ProductSKU: lf.ProductSKU,
		Quantity:   lf.Quantity,
		Reason:     lf.Reason,
		CreatedBy:  lf.CreatedBy,
		CreatedAt:  lf.CreatedAt,
		UpdatedAt:  lf.UpdatedAt,
	}

	// Include product details if loaded
	if lf.Product != nil {
		productResponse := lf.Product.ToProductResponse()
		response.Product = &productResponse
	}

	// Include create operator details if loaded
	if lf.CreateOperator != nil {
		userResponse := lf.CreateOperator.ToUserResponse()
		response.CreateOperator = &userResponse
	}

	return response
}
