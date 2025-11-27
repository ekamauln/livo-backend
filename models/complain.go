package models

import (
	"time"

	"gorm.io/gorm"
)

type Complain struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	Code         string         `gorm:"unique;not null" json:"code" example:"CMP123456"`
	Tracking     string         `gorm:"unique;not null" json:"tracking" example:"JNE1234567890"`
	OrderGineeID string         `gorm:"unique;not null" json:"order_ginee_id" example:"2509116GA36VM5"`
	ChannelID    uint           `gorm:"not null" json:"channel_id"`
	StoreID      uint           `gorm:"not null" json:"store_id"`
	CreatedBy    uint           `gorm:"not null" json:"created_by"`
	Description  string         `json:"description" example:"Item damaged during shipping"`
	Solution     string         `json:"solution" example:"Refund issued"`
	TotalFee     uint           `json:"total_fee" example:"15000"`
	Checked      bool           `gorm:"default:false" json:"checked" example:"false"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationship
	ProductDetails []ComplainProductDetail `gorm:"foreignKey:ComplainID" json:"product_details"`
	UserDetails    []ComplainUserDetail    `gorm:"foreignKey:ComplainID" json:"user_details"`
	Order          *Order                  `gorm:"-" json:"order,omitempty"`
	Channel        *Channel                `gorm:"foreignKey:ChannelID" json:"channel,omitempty"`
	Store          *Store                  `gorm:"foreignKey:StoreID" json:"store,omitempty"`
	Creator        *User                   `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
}

type ComplainProductDetail struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	ComplainID uint           `gorm:"not null" json:"complain_id"`
	ProductID  uint           `gorm:"not null" json:"product_id"`
	Quantity   int            `gorm:"not null" json:"quantity" example:"1"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationship
	Complain Complain `gorm:"foreignKey:ComplainID" json:"-"` // Back reference (excluded from JSON)
	Product  *Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
}

type ComplainUserDetail struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	ComplainID uint           `gorm:"not null" json:"complain_id"`
	OperatorID uint           `gorm:"not null" json:"operator_id"` // User being complained about
	FeeCharge  uint           `json:"fee_charge" example:"5000"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`

	// Relations
	Complain Complain `gorm:"foreignKey:ComplainID" json:"-"`                  // Back reference (excluded from JSON)
	Operator *User    `gorm:"foreignKey:OperatorID" json:"operator,omitempty"` // User being complained about
}

// Response structures
type ComplainProductDetailResponse struct {
	ID         uint      `json:"id"`
	ComplainID uint      `json:"complain_id"`
	ProductID  uint      `json:"product_id"`
	Quantity   int       `json:"quantity"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`

	// Related data
	Product ProductResponse `json:"product"`
}

type ComplainUserDetailResponse struct {
	ID         uint      `json:"id"`
	ComplainID uint      `json:"complain_id"`
	OperatorID uint      `json:"operator_id"`
	FeeCharge  uint      `json:"fee_charge"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`

	// Related data
	Operator UserResponse `json:"operator"`
}

type ComplainResponse struct {
	ID           uint      `json:"id"`
	Code         string    `json:"code"`
	Tracking     string    `json:"tracking"`
	OrderGineeID string    `json:"order_ginee_id"`
	ChannelID    uint      `json:"channel_id"`
	StoreID      uint      `json:"store_id"`
	CreatedBy    uint      `json:"created_by"`
	Description  string    `json:"description"`
	Solution     string    `json:"solution"`
	TotalFee     uint      `json:"total_fee"`
	Checked      bool      `json:"checked"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	// Related data
	ProductDetails []ComplainProductDetailResponse `json:"product_details"`
	UserDetails    []ComplainUserDetailResponse    `json:"user_details"`
	Order          *OrderResponse                  `json:"order,omitempty"`   // Order info (includes OrderGineeID)
	Channel        *ChannelResponse                `json:"channel,omitempty"` // Channel info
	Store          *StoreResponse                  `json:"store,omitempty"`   // Store info
	Creator        *UserResponse                   `json:"creator,omitempty"` // User who created the complain
}

// ToComplainResponse converts Complain model to ComplainResponse
func (c *Complain) ToComplainResponse() ComplainResponse {
	// Convert product details to response format
	productDetailResponses := make([]ComplainProductDetailResponse, len(c.ProductDetails))
	for i, pd := range c.ProductDetails {
		productDetailResponse := ComplainProductDetailResponse{
			ID:         pd.ID,
			ComplainID: pd.ComplainID,
			ProductID:  pd.ProductID,
			Quantity:   pd.Quantity,
			CreatedAt:  pd.CreatedAt,
			UpdatedAt:  pd.UpdatedAt,
		}

		// Include product data if loaded
		if pd.Product != nil && pd.Product.ID != 0 {
			productDetailResponse.Product = pd.Product.ToProductResponse()
		}

		productDetailResponses[i] = productDetailResponse
	}

	// Convert user details to response format
	userDetailResponses := make([]ComplainUserDetailResponse, len(c.UserDetails))
	for i, ud := range c.UserDetails {
		userDetailResponse := ComplainUserDetailResponse{
			ID:         ud.ID,
			ComplainID: ud.ComplainID,
			OperatorID: ud.OperatorID,
			FeeCharge:  ud.FeeCharge,
			CreatedAt:  ud.CreatedAt,
			UpdatedAt:  ud.UpdatedAt,
		}

		// Include user data if loaded (user being complained about)
		if ud.Operator != nil && ud.Operator.ID != 0 {
			userDetailResponse.Operator = ud.Operator.ToUserResponse()
		}

		userDetailResponses[i] = userDetailResponse
	}

	response := ComplainResponse{
		ID:             c.ID,
		Code:           c.Code,
		Tracking:       c.Tracking,
		OrderGineeID:   c.OrderGineeID,
		ChannelID:      c.ChannelID,
		StoreID:        c.StoreID,
		CreatedBy:      c.CreatedBy,
		Description:    c.Description,
		Solution:       c.Solution,
		TotalFee:       c.TotalFee,
		Checked:        c.Checked,
		CreatedAt:      c.CreatedAt,
		UpdatedAt:      c.UpdatedAt,
		ProductDetails: productDetailResponses,
		UserDetails:    userDetailResponses,
	}

	// Include order data if loaded (this will include OrderGineeID)
	if c.Order != nil {
		orderResponse := c.Order.ToOrderResponse()
		response.Order = &orderResponse
	}

	// Include channel data if loaded
	if c.Channel != nil {
		channelResponse := c.Channel.ToChannelResponse()
		response.Channel = &channelResponse
	}

	// Include store data if loaded
	if c.Store != nil {
		storeResponse := c.Store.ToStoreResponse()
		response.Store = &storeResponse
	}

	// Include creator data if loaded (user who created the complain)
	if c.Creator != nil {
		creatorResponse := c.Creator.ToUserResponse()
		response.Creator = &creatorResponse
	}

	return response
}

// Helper method to convert multiple Complains to responses
func ToComplainResponses(complains []Complain) []ComplainResponse {
	responses := make([]ComplainResponse, len(complains))
	for i, complain := range complains {
		responses[i] = complain.ToComplainResponse()
	}

	return responses
}
