package controllers

import (
	"livo-backend/models"
	"livo-backend/utilities"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type MobileReturnController struct {
	DB *gorm.DB
}

// NewMobileReturnController creates a new mobile return controller
func NewMobileReturnController(db *gorm.DB) *MobileReturnController {
	return &MobileReturnController{DB: db}
}

// GetMobileReturns godoc
// @Summary Get all returns by mobile
// @Description Get list of mobile returns from the last 7 days (public access, no login required)
// @Tags returns
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Param search query string false "Search by return mobile tracking (partial match)"
// @Success 200 {object} utilities.Response{data=MobileReturnsListResponse}
// @Router /api/mobile/returns [get]
func (mrc *MobileReturnController) GetMobileReturns(c *gin.Context) {
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	// Parse search parameter
	search := c.Query("search")

	var mobileReturns []models.Return
	var total int64

	// Calculate date 7 days ago from now
	oneWeekAgo := time.Now().AddDate(0, 0, -7)

	// Build query with optional search and date filter
	query := mrc.DB.Model(&models.Return{}).Where("created_at >= ?", oneWeekAgo)

	if search != "" {
		// Search by return mobile tracking with partial match
		query = query.Where("new_tracking ILIKE ?", "%"+search+"%")
	}

	// Get total count with search filter and date filter
	if err := query.Count(&total).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to count return", err.Error())
		return
	}

	// Get returns with pagination, search filter, date filter, preload relationships, and order by ID descending
	if err := query.Preload("Channel").Preload("Store").Order("id DESC").Limit(limit).Offset(offset).Find(&mobileReturns).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch return mobiles", err.Error())
		return
	}

	// Convert to response format
	mobileReturnResponses := make([]models.MobileReturnResponse, len(mobileReturns))
	for i, mobileReturn := range mobileReturns {
		mobileReturnResponses[i] = mobileReturn.ToMobileReturnResponse()
	}

	response := MobileReturnsListResponse{
		MobileReturns: mobileReturnResponses,
		Pagination: utilities.PaginationResponse{
			Page:  page,
			Limit: limit,
			Total: int(total),
		},
	}

	// Build success message
	message := "Return mobiles from the last 7 days retrieved successfully"
	if search != "" {
		message += " (filtered by tracking: " + search + ")"
	}

	utilities.SuccessResponse(c, http.StatusOK, message, response)
}

// GetMobileReturn godoc
// @Summary Get a return mobile by ID
// @Description Get a return mobile by ID (public access, no login required).
// @Tags returns
// @Accept json
// @Produce json
// @Param id path int true "Return ID"
// @Success 200 {object} utilities.Response{data=models.MobileReturnResponse}
// @Failure 400 {object} utilities.Response
// @Failure 404 {object} utilities.Response
// @Router /api/mobile/returns/{id} [get]
func (mrc *MobileReturnController) GetMobileReturn(c *gin.Context) {
	mobileReturnID := c.Param("id")

	var mobileReturn models.Return
	if err := mrc.DB.Preload("Channel").Preload("Store").First(&mobileReturn, mobileReturnID).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusNotFound, "Return not found", err.Error())
		return
	}

	utilities.SuccessResponse(c, http.StatusOK, "Return mobile retrieved successfully", mobileReturn.ToMobileReturnResponse())
}

// CreateMobileReturn godoc
// @Summary Create a new return mobile
// @Description Create a new return mobile (public access, no login required)
// @Tags returns
// @Accept json
// @Produce json
// @Param mobile_return body CreateMobileReturnRequest true "Create return mobile request"
// @Success 201 {object} utilities.Response{data=models.MobileReturnResponse}
// @Failure 400 {object} utilities.Response
// @Router /api/mobile/returns [post]
func (mrc *MobileReturnController) CreateMobileReturn(c *gin.Context) {
	var req CreateMobileReturnRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utilities.ValidationErrorResponse(c, err)
		return
	}

	// Convert tracking to uppercase and trim spaces
	req.Tracking = strings.ToUpper(strings.TrimSpace(req.Tracking))

	mobileReturn := models.Return{
		NewTracking: req.Tracking,
		ChannelID:   req.ChannelID,
		StoreID:     req.StoreID,
	}

	// Check for duplicate tracking
	var existingMobileReturn models.Return
	if err := mrc.DB.Where("new_tracking = ?", req.Tracking).First(&existingMobileReturn).Error; err == nil {
		utilities.ErrorResponse(c, http.StatusBadRequest, "Return mobile tracking already exists", "A return mobile with this tracking already exists")
		return
	}

	// Create a new return mobile and return the response
	if err := mrc.DB.Create(&mobileReturn).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to create return mobile", err.Error())
		return
	}

	utilities.SuccessResponse(c, http.StatusCreated, "Return mobile created successfully", mobileReturn.ToMobileReturnResponse())
}

// Request/Response structs
type MobileReturnsListResponse struct {
	MobileReturns []models.MobileReturnResponse `json:"mobile_returns"`
	Pagination    utilities.PaginationResponse  `json:"pagination"`
}

type CreateMobileReturnRequest struct {
	Tracking  string `json:"tracking" binding:"required"`
	ChannelID uint   `json:"channel_id" binding:"required"`
	StoreID   uint   `json:"store_id" binding:"required"`
}
