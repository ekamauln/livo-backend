package models

import (
	"time"

	"gorm.io/gorm"
)

type Return struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	NewTracking  string         `gorm:"unique" json:"new_tracking" example:"JNE0987654321"`
	OldTracking  string         `gorm:"unique" json:"old_tracking" example:"JNE1234567890"`
	OrderGineeID string         `gorm:"unique" json:"order_ginee_id" example:"2509116GA36VM5"`
	ChannelID    uint           `gorm:"not null" json:"channel_id"`
	StoreID      uint           `gorm:"not null" json:"store_id"`
	ReturnType   string         `json:"return_type" example:"Cancelled"`
	ReturnReason string         `json:"return_reason" example:"Customer cancelled the order"`
	ReturnNumber string         `json:"return_number" example:"RMA123456"`
	ScrapNumber  string         `json:"scrap_number" example:"SCRAP123456"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationship
	ReturnDetails []ReturnDetail `gorm:"foreignKey:ReturnID" json:"return_details"`
	Order         *Order         `gorm:"-" json:"order,omitempty"`
	Channel       *Channel       `gorm:"foreignKey:ChannelID" json:"channel,omitempty"`
	Store         *Store         `gorm:"foreignKey:StoreID" json:"store,omitempty"`
}

type ReturnDetail struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	ReturnID  uint           `gorm:"not null" json:"return_id"`
	ProductID uint           `gorm:"not null" json:"product_id"`
	Quantity  int            `gorm:"not null" json:"quantity" example:"2"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationship
	Return  Return  `gorm:"foreignKey:ReturnID" json:"-"` // Back reference (excluded from JSON)
	Product Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
}

// Response structures
type ReturnDetailResponse struct {
	ID        uint            `json:"id"`
	ReturnID  uint            `json:"return_id"`
	ProductID uint            `json:"product_id"`
	Quantity  int             `json:"quantity"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
	Product   ProductResponse `json:"product"`
}

type ReturnResponse struct {
	ID            uint                   `json:"id"`
	NewTracking   string                 `json:"new_tracking"`
	OldTracking   string                 `json:"old_tracking"`
	OrderGineeID  string                 `json:"order_ginee_id"`
	ChannelID     uint                   `json:"channel_id"`
	StoreID       uint                   `json:"store_id"`
	ReturnType    string                 `json:"return_type"`
	ReturnReason  string                 `json:"return_reason"`
	ReturnNumber  string                 `json:"return_number"`
	ScrapNumber   string                 `json:"scrap_number"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
	ReturnDetails []ReturnDetailResponse `json:"return_details"`

	// Related data
	Order   *OrderResponse   `json:"order,omitempty"`
	Channel *ChannelResponse `json:"channel,omitempty"`
	Store   *StoreResponse   `json:"store,omitempty"`
}

// ReturnMobileResponse is a simplified response for mobile use
type ReturnMobileResponse struct {
	ID        uint      `json:"id"`
	Tracking  string    `json:"tracking"`
	ChannelID uint      `json:"channel_id"`
	StoreID   uint      `json:"store_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Related data
	Channel *ChannelResponse `json:"channel,omitempty"`
	Store   *StoreResponse   `json:"store,omitempty"`
}

// ToReturnResponse converts Return model to ReturnResponse
func (r *Return) ToReturnResponse() ReturnResponse {
	// Convert return details to response format
	detailResponses := make([]ReturnDetailResponse, len(r.ReturnDetails))
	for i, detail := range r.ReturnDetails {
		detailResponse := ReturnDetailResponse{
			ID:        detail.ID,
			ReturnID:  detail.ReturnID,
			ProductID: detail.ProductID,
			Quantity:  detail.Quantity,
			CreatedAt: detail.CreatedAt,
			UpdatedAt: detail.UpdatedAt,
		}

		// Include product data if loaded
		if detail.Product.ID != 0 {
			detailResponse.Product = detail.Product.ToProductResponse()
		}

		detailResponses[i] = detailResponse
	}

	response := ReturnResponse{
		ID:            r.ID,
		NewTracking:   r.NewTracking,
		OldTracking:   r.OldTracking,
		OrderGineeID:  r.OrderGineeID,
		ChannelID:     r.ChannelID,
		StoreID:       r.StoreID,
		ReturnType:    r.ReturnType,
		ReturnReason:  r.ReturnReason,
		ReturnNumber:  r.ReturnNumber,
		ScrapNumber:   r.ScrapNumber,
		CreatedAt:     r.CreatedAt,
		UpdatedAt:     r.UpdatedAt,
		ReturnDetails: detailResponses,
	}

	// Include order data if loaded (this will include OrderGineeID)
	if r.Order != nil {
		orderResponse := r.Order.ToOrderResponse()
		response.Order = &orderResponse
	}

	// Include channel data if loaded
	if r.Channel != nil {
		channelResponse := r.Channel.ToChannelResponse()
		response.Channel = &channelResponse
	}

	// Include store data if loaded
	if r.Store != nil {
		storeResponse := r.Store.ToStoreResponse()
		response.Store = &storeResponse
	}

	return response
}

// Helper method to convert multiple Returns to responses
func ToReturnResponses(returns []Return) []ReturnResponse {
	responses := make([]ReturnResponse, len(returns))
	for i, ret := range returns {
		responses[i] = ret.ToReturnResponse()
	}
	return responses
}

// ToReturnMobileResponse converts Return model to ReturnMobileResponse
func (r *Return) ToReturnMobileResponse() ReturnMobileResponse {
	response := ReturnMobileResponse{
		ID:        r.ID,
		Tracking:  r.NewTracking,
		ChannelID: r.ChannelID,
		StoreID:   r.StoreID,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}

	// Include channel data if loaded
	if r.Channel != nil {
		channelResponse := r.Channel.ToChannelResponse()
		response.Channel = &channelResponse
	}

	// Include store data if loaded
	if r.Store != nil {
		storeResponse := r.Store.ToStoreResponse()
		response.Store = &storeResponse
	}

	return response
}
