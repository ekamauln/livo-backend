package controllers

import (
	"livo-backend/models"
	"livo-backend/utilities"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type MobileChannelController struct {
	DB *gorm.DB
}

// NewMobileChannelController creates a new mobile channel controller
func NewMobileChannelController(db *gorm.DB) *MobileChannelController {
	return &MobileChannelController{DB: db}
}

// GetMobileChannels godoc
// @Summary Get all channels for mobile
// @Description Get list of all channels.
// @Tags channels
// @Accept json
// @Produce json
// @Param search query string false "Search by channel code or name (partial match)"
// @Success 200 {object} utils.Response{data=MobileChannelsListResponse}
// @Router /api/mobile/channels [get]
func (mcc *MobileChannelController) GetMobileChannels(c *gin.Context) {
	// Parse search parameter
	search := c.Query("search")

	var channels []models.Channel
	var total int64

	// Build query with optional search
	query := mcc.DB.Model(&models.Channel{})

	if search != "" {
		// Search by channel code or name with partial match
		query = query.Where("code ILIKE ? OR name ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// Get total count with search filter
	if err := query.Count(&total).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to count channels", err.Error())
		return
	}

	// Execute query to get all channels (no pagination)
	if err := query.Order("id ASC").Find(&channels).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve channels", err.Error())
		return
	}

	// Convert to response format
	channelResponses := make([]models.ChannelResponse, len(channels))
	for i, channel := range channels {
		channelResponses[i] = channel.ToChannelResponse()
	}

	response := MobileChannelsListResponse{
		Channels: channelResponses,
		Total:    int(total),
	}

	// Build success message
	message := "Channels retrieved successfully"
	if search != "" {
		message += " (filtered by code or name: " + search + ")"
	}

	utilities.SuccessResponse(c, http.StatusOK, message, response)
}

// Request/Response structs
type MobileChannelsListResponse struct {
	Channels []models.ChannelResponse `json:"channels"`
	Total    int                      `json:"total"`
}
