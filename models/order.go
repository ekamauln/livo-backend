package models

import (
	"time"

	"gorm.io/gorm"
)

type Order struct {
	ID               uint           `gorm:"primaryKey" json:"id"`
	OrderGineeID     string         `gorm:"unique;not null" json:"order_ginee_id" example:"2509116GA36VM5"`
	ProcessingStatus string         `json:"processing_status" example:"ready to pick"`
	EventStatus      *string        `gorm:"default:null" json:"event_status" example:"pending"`
	Channel          string         `json:"channel" example:"Shopee"`
	Store            string         `json:"store" example:"SP deParcelRibbon"`
	Buyer            string         `json:"buyer" example:"John Doe"`
	Address          string         `json:"address" example:"123 Main St, Cityville, Country"`
	Courier          string         `json:"courier" example:"JNE"`
	Tracking         string         `gorm:"unique;not null" json:"tracking" example:"JNE1234567890"`
	SentBefore       time.Time      `json:"sent_before"`
	AssignedBy       *uint          `gorm:"default:null" json:"assigned_by"`
	AssignedAt       *time.Time     `gorm:"default:null" json:"assigned_at"`
	PickedBy         *uint          `gorm:"default:null" json:"picked_by"`
	PickedAt         *time.Time     `gorm:"default:null" json:"picked_at"`
	PendingBy        *uint          `gorm:"default:null" json:"pending_by"`
	PendingAt        *time.Time     `gorm:"default:null" json:"pending_at"`
	ChangedBy        *uint          `gorm:"default:null" json:"changed_by"`
	ChangedAt        *time.Time     `gorm:"default:null" json:"changed_at"`
	CancelledBy      *uint          `gorm:"default:null" json:"cancelled_by"`
	CancelledAt      *time.Time     `gorm:"default:null" json:"cancelled_at"`
	Complained       bool           `gorm:"default:false" json:"complained" example:"false"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationship
	OrderDetails    []OrderDetail `gorm:"foreignKey:OrderID" json:"order_details"`
	PickOperator    *User         `gorm:"foreignKey:PickedBy" json:"picker,omitempty"`
	PendingOperator *User         `gorm:"foreignKey:PendingBy" json:"pending_operator,omitempty"`
	CancelOperator  *User         `gorm:"foreignKey:CancelledBy" json:"canceller,omitempty"`
	ChangeOperator  *User         `gorm:"foreignKey:ChangedBy" json:"changer,omitempty"`
	AssignOperator  *User         `gorm:"foreignKey:AssignedBy" json:"assigner,omitempty"`
}

type OrderDetail struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	OrderID     uint      `json:"order_id"`
	Sku         string    `json:"sku" gorm:"index"`
	ProductName string    `json:"product_name"`
	Variant     string    `json:"variant"`
	Quantity    int       `json:"quantity"`
	Price       int       `json:"price"`
	Product     *Product  `json:"product,omitempty" gorm:"-"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// OrderResponse represents order data for API responses
type OrderResponse struct {
	ID               uint      `json:"id"`
	OrderGineeID     string    `json:"order_ginee_id"`
	ProcessingStatus string    `json:"processing_status"`
	EventStatus      *string   `json:"event_status"`
	Channel          string    `json:"channel"`
	Store            string    `json:"store"`
	Buyer            string    `json:"buyer"`
	Address          string    `json:"address"`
	Courier          string    `json:"courier"`
	Tracking         string    `json:"tracking"`
	SentBefore       string    `json:"sent_before"`
	Complained       bool      `json:"complained"`
	AssignedBy       string    `json:"assigned_by"`
	AssignedAt       string    `json:"assigned_at"`
	PickedBy         string    `json:"picked_by"`
	PickedAt         string    `json:"picked_at"`
	PendingBy        string    `json:"pending_by"`
	PendingAt        string    `json:"pending_at"`
	ChangedBy        string    `json:"changed_by"`
	ChangedAt        string    `json:"changed_at"`
	CancelledBy      string    `json:"cancelled_by"`
	CancelledAt      string    `json:"cancelled_at"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`

	// Related data
	OrderDetails []OrderDetailResponse `json:"order_details"`
}

type OrderDetailResponse struct {
	ID          uint   `json:"id"`
	Sku         string `json:"sku"`
	ProductName string `json:"product_name"`
	Variant     string `json:"variant"`
	Quantity    int    `json:"quantity"`
	Price       int    `json:"price"`

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
			Price:       detail.Price,
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

	// Null visual handler
	var pickedBy string
	if o.PickOperator != nil {
		pickedBy = o.PickOperator.FullName
	} else {
		pickedBy = "-"
	}

	var pickedAt string
	if o.PickedAt != nil {
		pickedAt = o.PickedAt.Format("2006-01-02 15:04:05")
	} else {
		pickedAt = "-"
	}

	var changedBy string
	if o.ChangeOperator != nil {
		changedBy = o.ChangeOperator.FullName
	} else {
		changedBy = "-"
	}

	var changedAt string
	if o.ChangedAt != nil {
		changedAt = o.ChangedAt.Format("2006-01-02 15:04:05")
	} else {
		changedAt = "-"
	}

	var pendingBy string
	if o.PendingOperator != nil {
		pendingBy = o.PendingOperator.FullName
	} else {
		pendingBy = "-"
	}

	var pendingAt string
	if o.PendingAt != nil {
		pendingAt = o.PendingAt.Format("2006-01-02 15:04:05")
	} else {
		pendingAt = "-"
	}

	var cancelledBy string
	if o.CancelOperator != nil {
		cancelledBy = o.CancelOperator.FullName
	} else {
		cancelledBy = "-"
	}

	var cancelledAt string
	if o.CancelledAt != nil {
		cancelledAt = o.CancelledAt.Format("2006-01-02 15:04:05")
	} else {
		cancelledAt = "-"
	}

	var assignedBy string
	if o.AssignOperator != nil {
		assignedBy = o.AssignOperator.FullName
	} else {
		assignedBy = "-"
	}

	var assignedAt string
	if o.AssignedAt != nil {
		assignedAt = o.AssignedAt.Format("2006-01-02 15:04:05")
	} else {
		assignedAt = "-"
	}

	return OrderResponse{
		ID:               o.ID,
		OrderGineeID:     o.OrderGineeID,
		ProcessingStatus: o.ProcessingStatus,
		EventStatus:      o.EventStatus,
		Channel:          o.Channel,
		Store:            o.Store,
		Buyer:            o.Buyer,
		Address:          o.Address,
		Courier:          o.Courier,
		Tracking:         o.Tracking,
		SentBefore:       o.SentBefore.Format("2006-01-02 15:04:05"),
		Complained:       o.Complained,
		CreatedAt:        o.CreatedAt,
		UpdatedAt:        o.UpdatedAt,
		AssignedBy:       assignedBy,
		AssignedAt:       assignedAt,
		PickedBy:         pickedBy,
		PickedAt:         pickedAt,
		ChangedBy:        changedBy,
		ChangedAt:        changedAt,
		PendingBy:        pendingBy,
		PendingAt:        pendingAt,
		CancelledBy:      cancelledBy,
		CancelledAt:      cancelledAt,
		OrderDetails:     details,
	}
}
