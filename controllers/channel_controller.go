package controllers

import (
	"livo-backend/models"
	"livo-backend/utilities"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ChannelController struct {
	DB *gorm.DB
}

// NewChannelController creates a new channel controller
func NewChannelController(db *gorm.DB) *ChannelController {
	return &ChannelController{DB: db}
}

// GetChannels godoc
// @Summary Get all channels
// @Description Get list of all channels.
// @Tags channels
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Param search query string false "Search by Code or Name (partial match)"
// @Success 200 {object} utilities.Response{data=ChannelsListResponse}
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Router /api/channels [get]
func (cc *ChannelController) GetChannels(c *gin.Context) {
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	// Parse search parameter
	search := c.Query("search")

	var channels []models.Channel
	var total int64

	// Build query with optional search
	query := cc.DB.Model(&models.Channel{})

	if search != "" {
		// Search by Code or Name with partial match
		query = query.Where("code ILIKE ? OR name ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// Get total count with search filter
	if err := query.Count(&total).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to count channels", err.Error())
		return
	}

	// Get channels with pagination, search filter, and order by ID ascending
	if err := query.Order("id ASC").Limit(limit).Offset(offset).Find(&channels).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve channels", err.Error())
		return
	}

	// Convert to response format
	channelResponses := make([]models.ChannelResponse, len(channels))
	for i, channel := range channels {
		channelResponses[i] = channel.ToChannelResponse()
	}

	response := ChannelsListResponse{
		Channels: channelResponses,
		Pagination: utilities.PaginationResponse{
			Page:  page,
			Limit: limit,
			Total: int(total),
		},
	}

	// Build success message
	message := "Channels retrieved successfully"
	if search != "" {
		message += " (filtered by code or name: " + search + ")"
	}

	utilities.SuccessResponse(c, http.StatusOK, message, response)
}

// GetChannel godoc
// @Summary Get channel by ID
// @Description Get specific channel by ID.
// @Tags channels
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Channel ID"
// @Success 200 {object} utilities.Response{data=models.ChannelResponse}
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Failure 404 {object} utilities.Response
// @Router /api/channels/{id} [get]
func (cc *ChannelController) GetChannel(c *gin.Context) {
	channelID := c.Param("id")

	var channel models.Channel
	if err := cc.DB.First(&channel, channelID).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusNotFound, "Channel not found", err.Error())
		return
	}

	utilities.SuccessResponse(c, http.StatusOK, "Channel retrieved successfully", channel.ToChannelResponse())
}

// UpdateChannel godoc
// @Summary Update channel
// @Description Update specific channel information.
// @Tags channels
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Channel ID"
// @Param channel body UpdateChannelRequest true "Update channel request"
// @Success 200 {object} utilities.Response{data=models.ChannelResponse}
// @Failure 400 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Failure 404 {object} utilities.Response
// @Router /api/channels/{id} [put]
func (cc *ChannelController) UpdateChannel(c *gin.Context) {
	channelID := c.Param("id")

	var req UpdateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utilities.ValidationErrorResponse(c, err)
		return
	}

	var channel models.Channel
	if err := cc.DB.First(&channel, channelID).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusNotFound, "Channel not found", err.Error())
		return
	}

	// Check for duplicate channel code (excluding current channel)
	var existingChannel models.Channel
	if err := cc.DB.Where("code = ? AND id <> ?", req.Code, channelID).First(&existingChannel).Error; err == nil {
		utilities.ErrorResponse(c, http.StatusBadRequest, "Channel code already exists", "A channel with this code already exists")
		return
	}

	// Update channel fields
	channel.Code = req.Code
	channel.Name = req.Name

	if err := cc.DB.Save(&channel).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to update channel", err.Error())
		return
	}

	utilities.SuccessResponse(c, http.StatusOK, "Channel updated successfully", channel.ToChannelResponse())
}

// RemoveChannel godoc
// @Summary Remove channel
// @Description Soft delete a channel.
// @Tags channels
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Channel ID"
// @Success 200 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Failure 404 {object} utilities.Response
// @Router /api/channels/{id} [delete]
func (cc *ChannelController) RemoveChannel(c *gin.Context) {
	channelID := c.Param("id")

	var channel models.Channel
	if err := cc.DB.First(&channel, channelID).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusNotFound, "Channel not found", err.Error())
		return
	}

	if err := cc.DB.Delete(&channel).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to remove channel", err.Error())
		return
	}

	utilities.SuccessResponse(c, http.StatusOK, "Channel removed successfully", nil)
}

// CreateChannel godoc
// @Summary Create new channel
// @Description Create a new channel.
// @Tags channels
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param channel body CreateChannelRequest true "Create channel request"
// @Success 201 {object} utilities.Response{data=models.ChannelResponse}
// @Failure 400 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Router /api/channels [post]
func (cc *ChannelController) CreateChannel(c *gin.Context) {
	var req CreateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utilities.ValidationErrorResponse(c, err)
		return
	}

	// convert code to uppercase and trim spaces
	req.Code = strings.ToUpper(strings.TrimSpace(req.Code))

	channel := models.Channel{
		Code: req.Code,
		Name: req.Name,
	}

	// Check for duplicate channel code
	var existingChannel models.Channel
	if err := cc.DB.Where("code = ?", req.Code).First(&existingChannel).Error; err == nil {
		utilities.ErrorResponse(c, http.StatusBadRequest, "Channel code already exists", "A channel with this code already exists")
		return
	}

	// Create a new channel and return the response
	if err := cc.DB.Create(&channel).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to create channel", err.Error())
		return
	}

	utilities.SuccessResponse(c, http.StatusCreated, "Channel created successfully", channel.ToChannelResponse())
}

// Request/Response structs
type ChannelsListResponse struct {
	Channels   []models.ChannelResponse     `json:"channels"`
	Pagination utilities.PaginationResponse `json:"pagination"`
}

type UpdateChannelRequest struct {
	Code string `json:"code" binding:"required"`
	Name string `json:"name" binding:"required"`
}

type CreateChannelRequest struct {
	Code string `json:"code" binding:"required"`
	Name string `json:"name" binding:"required"`
}
