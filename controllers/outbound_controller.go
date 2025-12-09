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

type OutboundController struct {
	DB *gorm.DB
}

// NewOutboundController creates a new outbound controller
func NewOutboundController(db *gorm.DB) *OutboundController {
	return &OutboundController{DB: db}
}

// GetOutbounds godoc
// @Summary Get all outbounds
// @Description Get list of all outbounds.
// @Tags outbounds
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Param search query string false "Search by outbound tracking (partial match)"
// @Success 200 {object} utilities.Response{data=OutboundsListResponse}
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Router /api/outbounds [get]
func (oc *OutboundController) GetOutbounds(c *gin.Context) {
	// Get user ID from JWT token
	userID, exists := c.Get("user_id")
	if !exists {
		utilities.ErrorResponse(c, http.StatusUnauthorized, "Unauthorized", "User not authenticated")
		return
	}

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	// Parse search parameter
	search := c.Query("search")

	var outbounds []models.Outbound
	var total int64

	// Build query with outbound_by and current date filters
	query := oc.DB.Model(&models.Outbound{}).
		Where("outbound_by = ?", userID).
		Where("DATE(created_at) = CURRENT_DATE")

	if search != "" {
		// Search by outbound tracking with partial match
		query = query.Where("tracking ILIKE ?", "%"+search+"%")
	}

	// Get total count with search filter
	if err := query.Count(&total).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to count outbounds", err.Error())
		return
	}

	// Get outbounds with pagination, search filter, and order by ID descending
	if err := query.
		Preload("OutboundOperator.UserRoles.Role").
		Preload("OutboundOperator.UserRoles.Assigner").
		Order("id DESC").
		Limit(limit).
		Offset(offset).
		Find(&outbounds).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve outbounds", err.Error())
		return
	}

	// Manually load order data for each outbound
	for i := range outbounds {
		if outbounds[i].Tracking != "" {
			var order models.Order
			if err := oc.DB.Where("tracking = ?", outbounds[i].Tracking).
				Preload("OrderDetails").
				Preload("PickOperator.UserRoles.Role").
				Preload("PickOperator.UserRoles.Assigner").
				First(&order).Error; err == nil {
				// Manually fetch and attach products to order details
				for j := range order.OrderDetails {
					var product models.Product
					if err := oc.DB.Where("sku = ?", order.OrderDetails[j].Sku).First(&product).Error; err == nil {
						order.OrderDetails[j].Product = &product
					}
				}
				outbounds[i].Order = &order
			}
		}
	}

	// Convert to response format
	outboundResponse := make([]models.OutboundResponse, len(outbounds))
	for i, outbound := range outbounds {
		outboundResponse[i] = outbound.ToOutboundResponse()
	}

	response := OutboundsListResponse{
		Outbounds: outboundResponse,
		Pagination: utilities.PaginationResponse{
			Page:  page,
			Limit: limit,
			Total: int(total),
		},
	}

	// Build success message
	message := "Outbounds retrieved successfully"
	if search != "" {
		message += " (filtered by tracking: " + search + ")"
	}

	utilities.SuccessResponse(c, http.StatusOK, message, response)
}

// GetOutbound godoc
// @Summary Get outbound by ID
// @Description Get specific outbound by ID.
// @Tags outbounds
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Outbound ID"
// @Success 200 {object} utilities.Response{data=models.OutboundResponse}
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Failure 404 {object} utilities.Response
// @Router /api/outbounds/{id} [get]
func (oc *OutboundController) GetOutbound(c *gin.Context) {
	outboundID := c.Param("id")

	var outbound models.Outbound
	if err := oc.DB.Preload("OutboundOperator.UserRoles.Role").
		Preload("OutboundOperator.UserRoles.Assigner").
		First(&outbound, outboundID).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusNotFound, "Outbound not found", err.Error())
		return
	}

	// Manually load order data
	if outbound.Tracking != "" {
		var order models.Order
		if err := oc.DB.Where("tracking = ?", outbound.Tracking).
			Preload("OrderDetails").
			Preload("PickOperator.UserRoles.Role").
			Preload("PickOperator.UserRoles.Assigner").
			First(&order).Error; err == nil {
			// Manually fetch and attach products to order details
			for i := range order.OrderDetails {
				var product models.Product
				if err := oc.DB.Where("sku = ?", order.OrderDetails[i].Sku).First(&product).Error; err == nil {
					order.OrderDetails[i].Product = &product
				}
			}
			outbound.Order = &order
		}
	}

	utilities.SuccessResponse(c, http.StatusOK, "Outbound retrieved successfully", outbound.ToOutboundResponse())
}

// UpdateOutbound godoc
// @Summary Update outbound by ID
// @Description Update specific outbound information.
// @Tags outbounds
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Outbound ID"
// @Param outbound body UpdateOutboundRequest true "Update Outbound Request"
// @Success 200 {object} utilities.Response{data=models.OutboundResponse}
// @Failure 400 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Failure 404 {object} utilities.Response
// @Router /api/outbounds/{id} [put]
func (oc *OutboundController) UpdateOutbound(c *gin.Context) {
	outboundID := c.Param("id")

	var req UpdateOutboundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utilities.ValidationErrorResponse(c, err)
		return
	}

	var outbound models.Outbound
	if err := oc.DB.Preload("OutboundOperator.UserRoles.Role").
		Preload("OutboundOperator.UserRoles.Assigner").
		First(&outbound, outboundID).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusNotFound, "Outbound not found", err.Error())
		return
	}

	// Check if tracking starts with "TKP0"
	if len(outbound.Tracking) < 4 || outbound.Tracking[:4] != "TKP0" {
		utilities.ErrorResponse(c, http.StatusForbidden, "Update not allowed", "Only outbounds with tracking starting with 'TKP0' can be updated")
		return
	}

	// Update outbound fields
	outbound.Expedition = req.Expedition
	outbound.ExpeditionColor = req.ExpeditionColor
	outbound.ExpeditionSlug = req.ExpeditionSlug

	if err := oc.DB.Save(&outbound).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to update outbound", err.Error())
		return
	}

	// Load order data after update
	if outbound.Tracking != "" {
		var order models.Order
		if err := oc.DB.Where("tracking = ?", outbound.Tracking).
			Preload("OrderDetails").
			Preload("PickOperator.UserRoles.Role").
			Preload("PickOperator.UserRoles.Assigner").
			First(&order).Error; err == nil {
			// Manually fetch and attach products to order details
			for i := range order.OrderDetails {
				var product models.Product
				if err := oc.DB.Where("sku = ?", order.OrderDetails[i].Sku).First(&product).Error; err == nil {
					order.OrderDetails[i].Product = &product
				}
			}
			outbound.Order = &order
		}
	}

	utilities.SuccessResponse(c, http.StatusOK, "Outbound updated successfully", outbound.ToOutboundResponse())
}

// CreateOutbound godoc
// @Summary Create new outbound
// @Description Create a new outbound with automatic expedition detection.
// @Tags outbounds
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param outbound body CreateOutboundRequest true "Create Outbound Request"
// @Success 201 {object} utilities.Response{data=models.OutboundResponse}
// @Failure 400 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Failure 500 {object} utilities.Response
// @Router /api/outbounds [post]
func (oc *OutboundController) CreateOutbound(c *gin.Context) {
	// Get user ID from JWT token
	userID, exists := c.Get("user_id")
	if !exists {
		utilities.ErrorResponse(c, http.StatusUnauthorized, "Unauthorized", "User not authenticated")
		return
	}

	var req CreateOutboundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utilities.ValidationErrorResponse(c, err)
		return
	}

	// Convert tracking to uppercase
	req.Tracking = strings.ToUpper(strings.TrimSpace(req.Tracking))

	// Convert userID to uint
	userIDUint, ok := userID.(uint)
	if !ok {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Invalid user ID", "Failed to convert user ID")
		return
	}

	// Check if tracking exists in orders table first
	var order models.Order
	if err := oc.DB.Where("tracking = ?", req.Tracking).First(&order).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utilities.ErrorResponse(c, http.StatusNotFound, "Order not found", "No order found with the specified tracking number")
			return
		}
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to check order", err.Error())
		return
	}

	// Check if tracking exists in QC-Ribbon OR QC-Online (Quality Control process)
	var qcRibbon models.QcRibbon
	var qcOnline models.QcOnline

	qcRibbonExists := oc.DB.Where("tracking = ?", req.Tracking).First(&qcRibbon).Error == nil
	qcOnlineExists := oc.DB.Where("tracking = ?", req.Tracking).First(&qcOnline).Error == nil

	// Tracking must exist in either QC-Ribbon OR QC-Online
	if !qcRibbonExists && !qcOnlineExists {
		utilities.ErrorResponse(c, http.StatusBadRequest, "QC process required", "Tracking must go through Quality Control (QC-Ribbon or QC-Online) before outbound")
		return
	}

	// Check for duplicate tracking
	var existing models.Outbound
	if err := oc.DB.Where("tracking = ?", req.Tracking).First(&existing).Error; err == nil {
		utilities.ErrorResponse(c, http.StatusBadRequest, "Tracking already exists", "An outbound with this tracking number already exists")
		return
	}

	var expedition string
	var expeditionColor string
	var expeditionSlug string

	// Special case: If tracking starts with "TKP0", use request body values
	if len(req.Tracking) >= 4 && req.Tracking[:4] == "TKP0" {
		expedition = req.Expedition
		expeditionColor = req.ExpeditionColor
		expeditionSlug = req.ExpeditionSlug
	} else {
		// Auto-detect expedition based on tracking prefix
		var expeditions []models.Expedition
		if err := oc.DB.Find(&expeditions).Error; err != nil {
			utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve expeditions", err.Error())
			return
		}

		expeditionFound := false

		// Check each expedition code to see if tracking starts with it
		for _, exp := range expeditions {
			if len(req.Tracking) >= len(exp.Code) &&
				req.Tracking[:len(exp.Code)] == exp.Code {
				expedition = exp.Name
				expeditionColor = exp.Color
				expeditionSlug = exp.Slug
				expeditionFound = true
				break
			}
		}

		// If no expedition found based on prefix, return error
		if !expeditionFound {
			utilities.ErrorResponse(c, http.StatusBadRequest, "Invalid tracking code", "Tracking number does not match any known expedition prefix")
			return
		}
	}

	outbound := models.Outbound{
		Tracking:        req.Tracking,
		OutboundBy:      &userIDUint,
		Expedition:      expedition,
		ExpeditionColor: expeditionColor,
		ExpeditionSlug:  expeditionSlug,
	}

	// Create outbound
	if err := oc.DB.Create(&outbound).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to create outbound", err.Error())
		return
	}

	// Load the created outbound with user relationship
	oc.DB.Preload("OutboundOperator.UserRoles.Role").
		Preload("OutboundOperator.UserRoles.Assigner").
		First(&outbound, outbound.ID)

	// Load order data if exists
	if outbound.Tracking != "" {
		var order models.Order
		if err := oc.DB.Where("tracking = ?", outbound.Tracking).
			Preload("OrderDetails").
			Preload("PickOperator.UserRoles.Role").
			Preload("PickOperator.UserRoles.Assigner").
			First(&order).Error; err == nil {
			// Manually fetch and attach products to order details
			for i := range order.OrderDetails {
				var product models.Product
				if err := oc.DB.Where("sku = ?", order.OrderDetails[i].Sku).First(&product).Error; err == nil {
					order.OrderDetails[i].Product = &product
				}
			}
			outbound.Order = &order
		}
	}

	utilities.SuccessResponse(c, http.StatusCreated, "Outbound created successfully", outbound.ToOutboundResponse())
}

// GetChartOutbounds godoc
// @Summary Get outbound counts per day for current month
// @Description Get daily count of outbounds for current month (for chart data).
// @Tags outbounds
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} utilities.Response{data=OutboundsDailyCountResponse}
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Router /api/outbounds/chart [get]
func (oc *OutboundController) GetChartOutbounds(c *gin.Context) {
	// Get current month start and end dates
	now := time.Now()
	currentYear, currentMonth, _ := now.Date()
	currentLocation := now.Location()

	// First day of current month
	firstOfMonth := time.Date(currentYear, currentMonth, 1, 0, 0, 0, 0, currentLocation)

	// First day of next month at 00:00:00 (to use as upper bound)
	firstOfNextMonth := firstOfMonth.AddDate(0, 1, 0)

	// Query to get daily counts for current month
	var dailyCounts []OutboundsDailyCount

	if err := oc.DB.Model(&models.Outbound{}).
		Select("DATE(created_at) as date, COUNT(*) as count").
		Where("created_at >= ?", firstOfMonth).
		Where("created_at < ?", firstOfNextMonth).
		Group("DATE(created_at)").
		Order("date ASC").
		Scan(&dailyCounts).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve outbound counts", err.Error())
		return
	}

	// Get total count for current month
	var totalCount int64
	if err := oc.DB.Model(&models.Outbound{}).
		Where("created_at >= ?", firstOfMonth).
		Where("created_at < ?", firstOfNextMonth).
		Count(&totalCount).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to count outbounds", err.Error())
		return
	}

	response := OutboundsDailyCountResponse{
		Month:       currentMonth.String(),
		Year:        currentYear,
		DailyCounts: dailyCounts,
		TotalCount:  int(totalCount),
	}

	message := "Outbound daily counts for " + currentMonth.String() + " " + strconv.Itoa(currentYear) + " retrieved successfully"

	utilities.SuccessResponse(c, http.StatusOK, message, response)
}

// Request/Response structs
type OutboundsListResponse struct {
	Outbounds  []models.OutboundResponse    `json:"outbounds"`
	Pagination utilities.PaginationResponse `json:"pagination"`
}

type UpdateOutboundRequest struct {
	Expedition      string `json:"expedition" binding:"required"`
	ExpeditionColor string `json:"expedition_color" binding:"required"`
	ExpeditionSlug  string `json:"expedition_slug" binding:"required"`
}

type CreateOutboundRequest struct {
	Tracking        string `json:"tracking" binding:"required"`
	Expedition      string `json:"expedition"`
	ExpeditionColor string `json:"expedition_color"`
	ExpeditionSlug  string `json:"expedition_slug"`
}

// OutboundsDailyCount represents the count of outbounds for a specific date
type OutboundsDailyCount struct {
	Date  time.Time `json:"date"`
	Count int64     `json:"count"`
}

// OutboundsDailyCountResponse represents the response for daily outbound counts
type OutboundsDailyCountResponse struct {
	Month       string                `json:"month"` // e.g., "November"
	Year        int                   `json:"year"`  // e.g., 2023
	DailyCounts []OutboundsDailyCount `json:"daily_counts"`
	TotalCount  int                   `json:"total_count"` // Total for the month
}
