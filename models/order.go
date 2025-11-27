package models

import (
	"time"

	"gorm.io/gorm"
)

type Order struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	OrderGineeID string         `gorm:"unique;not null" json:"order_ginee_id" example:"2509116GA36VM5"`
	Status       string         `json:"status" example:"pending"`
	Channel      string         `json:"channel" example:"Shopee"`
	Store        string         `json:"store" example:"SP deParcelRibbon"`
	Buyer        string         `json:"buyer" example:"John Doe"`
	Address      string         `json:"address" example:"Jl.  KH. Umar, Daleman RT.  07 RW.  03, Japan, KAB. MOJOKERTO, SOOKO, JAWA TIMUR, ID, 61361"`
	Courier      string         `json:"courier" example:"JNE"`
	Tracking     string         `gorm:"unique;not null" json:"tracking" example:"JNE1234567890"`
	SentBefore   *time.Time     `gorm:"default:null" json:"sent_before"`
	PickedBy     *uint          `gorm:"default:null" json:"picked_by"`
	PickedAt     *time.Time     `gorm:"default:null" json:"picked_at"`
	PendingBy    *uint          `gorm:"default:null" json:"pending_by"`
	PendingAt    *time.Time     `gorm:"default:null" json:"pending_at"`
	CancelledBy  *uint          `gorm:"default:null" json:"cancelled_by"`
	CancelledAt  *time.Time     `gorm:"default:null" json:"cancelled_at"`
	Complained   bool           `gorm:"default:false" json:"complained" example:"false"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationship
	OrderDetails    []OrderDetail `gorm:"foreignKey:OrderID" json:"order_details"`
	Picker          *User         `gorm:"foreignKey:PickedBy" json:"picker,omitempty"`
	PendingOperator *User         `gorm:"foreignKey:PendingBy" json:"pending_operator,omitempty"`
	Canceller       *User         `gorm:"foreignKey:CancelledBy" json:"canceller,omitempty"`
}

type OrderDetail struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	OrderID     uint      `json:"order_id"`
	Sku         string    `json:"sku" gorm:"index"`
	ProductName string    `json:"product_name"`
	Variant     string    `json:"variant"`
	Quantity    int       `json:"quantity"`
	Product     *Product  `json:"product,omitempty" gorm:"-"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// OrderResponse represents order data for API responses
type OrderResponse struct {
	ID           uint      `json:"id"`
	OrderGineeID string    `json:"order_id"`
	Status       string    `json:"status"`
	Channel      string    `json:"channel"`
	Store        string    `json:"store"`
	Buyer        string    `json:"buyer"`
	Address      string    `json:"address"`
	Courier      string    `json:"courier"`
	Tracking     string    `json:"tracking"`
	SentBefore   string    `json:"sent_before"`
	Complained   bool      `json:"complained"`
	PickedBy     string    `json:"picked_by"`
	PickedAt     string    `json:"picked_at"`
	PendingBy    string    `json:"pending_by"`
	PendingAt    string    `json:"pending_at"`
	CancelledBy  string    `json:"cancelled_by"`
	CancelledAt  string    `json:"cancelled_at"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	// Related data
	OrderDetails []OrderDetailResponse `json:"order_details"`
}

type OrderDetailResponse struct {
	ID          uint   `json:"id"`
	Sku         string `json:"sku"`
	ProductName string `json:"product_name"`
	Variant     string `json:"variant"`
	Quantity    int    `json:"quantity"`

	// Related data
	Product *ProductResponse `json:"product,omitempty"`
}

// ToOrderResponse converts Order model to OrderResponse
func (o *Order) ToOrderResponse() OrderResponse {
	details := make([]OrderDetailResponse, len(o.OrderDetails))
	for i, detail := range o.OrderDetails {
		detailResp := OrderDetailResponse{
			ID:          detail.ID,
			Sku:         detail.Sku,
			ProductName: detail.ProductName,
			Variant:     detail.Variant,
			Quantity:    detail.Quantity,
		}

		// Include product data if exists
		if detail.Product != nil {
			detailResp.Product = &ProductResponse{
				ID:    detail.Product.ID,
				Sku:   detail.Product.Sku,
				Name:  detail.Product.Name,
				Image: detail.Product.Image,
			}
		}

		details[i] = detailResp
	}

	return OrderResponse{
		ID:           o.ID,
		OrderGineeID: o.OrderGineeID,
		Status:       o.Status,
		Channel:      o.Channel,
		Store:        o.Store,
		Buyer:        o.Buyer,
		Address:      o.Address,
		Courier:      o.Courier,
		Tracking:     o.Tracking,
		SentBefore:   o.SentBefore.Format("2006-01-02 15:04:05"),
		Complained:   o.Complained,
		CreatedAt:    o.CreatedAt,
		UpdatedAt:    o.UpdatedAt,
		PickedBy:     o.Picker.FullName,
		PickedAt:     o.PickedAt.Format("2006-01-02 15:04:05"),
		PendingBy:    o.PendingOperator.FullName,
		PendingAt:    o.PendingAt.Format("2006-01-02 15:04:05"),
		CancelledBy:  o.Canceller.FullName,
		CancelledAt:  o.CancelledAt.Format("2006-01-02 15:04:05"),
		OrderDetails: details,
	}
}
