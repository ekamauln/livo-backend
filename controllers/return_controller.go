package controllers

import (
	"fmt"
	"livo-backend/models"
	"livo-backend/utilities"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ReturnController struct {
	DB *gorm.DB
}

// NewReturnController creates a new return controller
func NewReturnController(db *gorm.DB) *ReturnController {
	return &ReturnController{DB: db}
}

// GetReturns godoc
// @Summary Get all returns
// @Description Get a list of all returns with optional date range filtering and search.
// @Tags returns
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param pageSize query int false "Page size" default(10)
// @Param start_date query string false "Start date (YYYY-MM-DD format)"
// @Param end_date query string false "End date (YYYY-MM-DD format)"
// @Param search query string false "Search by return new tracking (partial match)"
// @Success 200 {object} utilities.Response{data=ReturnsListResponse}
// @Failure 400 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Router /api/returns [get]
func (rc *ReturnController) GetReturns(c *gin.Context) {
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	offset := (page - 1) * limit

	// Parse date range parameters
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	// Parse search parameter
	search := c.Query("search")

	var rets []models.Return
	var total int64

	// Build query with optional search
	query := rc.DB.Model(&models.Return{})

	// Apply date range filters if provided
	if startDate != "" {
		// Parse start date and set time to beginning of day
		if parsedStartDate, err := time.Parse("2006-01-02", startDate); err != nil {
			utilities.ErrorResponse(c, http.StatusBadRequest, "Invalid start_date format", "start_date must be in YYYY-MM-DD format")
			return
		} else {
			startOfDay := parsedStartDate.Format("2006-01-02 00:00:00")
			query = query.Where("created_at >= ?", startOfDay)
		}
	}

	if endDate != "" {
		// Parse end date and set time to end of day
		if parsedEndDate, err := time.Parse("2006-01-02", endDate); err != nil {
			utilities.ErrorResponse(c, http.StatusBadRequest, "Invalid end_date format", "end_date must be in YYYY-MM-DD format")
			return
		} else {
			// Add 24 hours to get the start of next day, then use < instead of <=
			nextDay := parsedEndDate.AddDate(0, 0, 1).Format("2006-01-02 00:00:00")
			query = query.Where("created_at < ?", nextDay)
		}
	}

	if search != "" {
		// Search by return new tracking or order ID with partial match
		query = query.Where("new_tracking ILIKE ? OR old_tracking ILIKE ? OR order_ginee_id ILIKE ?", "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	// Get total count with search filter
	if err := query.Count(&total).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to count returns", err.Error())
		return
	}

	// Get returns with pagination, search filter, and order by ID desc
	if err := query.Preload("ReturnDetails.Product").
		Preload("Channel").
		Preload("Store").
		Preload("CreateOperator").
		Preload("UpdateOperator").Order("id DESC").Limit(limit).Offset(offset).Find(&rets).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve returns", err.Error())
		return
	}

	// Load order data for each return
	for i := range rets {
		if rets[i].OldTracking != "" {
			var order models.Order
			if err := rc.DB.Preload("OrderDetails").
				Preload("PickOperator.UserRoles.Role").
				Preload("PickOperator.UserRoles.Assigner").
				Where("tracking = ?", rets[i].OldTracking).First(&order).Error; err == nil {
				rets[i].Order = &order
			}
		}
	}

	// Convert to response format
	returnResponse := make([]models.ReturnResponse, len(rets))
	for i, ret := range rets {
		returnResponse[i] = ret.ToReturnResponse()
	}

	response := ReturnsListResponse{
		Returns: returnResponse,
		Pagination: utilities.PaginationResponse{
			Page:  page,
			Limit: limit,
			Total: int(total),
		},
	}

	// Build success message
	message := "Returns retrieved successfully"
	var filters []string

	if startDate != "" || endDate != "" {
		var dateRange []string
		if startDate != "" {
			dateRange = append(dateRange, "from: "+startDate)
		}
		if endDate != "" {
			dateRange = append(dateRange, "to: "+endDate)
		}
		filters = append(filters, "date: "+strings.Join(dateRange, ", "))
	}

	if search != "" {
		filters = append(filters, "search: "+search)
	}

	if len(filters) > 0 {
		message += fmt.Sprintf(" (filtered by %s)", strings.Join(filters, " | "))
	}

	utilities.SuccessResponse(c, http.StatusOK, message, response)
}

// GetReturn godoc
// @Summary Get return by ID
// @Description Get return details by ID.
// @Tags returns
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Return ID"
// @Success 200 {object} utilities.Response{data=models.ReturnResponse}
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Failure 404 {object} utilities.Response
// @Router /api/returns/{id} [get]
func (rc *ReturnController) GetReturn(c *gin.Context) {
	returnID := c.Param("id")

	var ret models.Return
	if err := rc.DB.Preload("ReturnDetails.Product").
		Preload("Channel").
		Preload("Store").
		Preload("CreateOperator").
		Preload("UpdateOperator").
		First(&ret, returnID).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusNotFound, "Return not found", err.Error())
		return
	}

	// Load order data if old_tracking exists
	if ret.OldTracking != "" {
		var order models.Order
		if err := rc.DB.Preload("OrderDetails").
			Preload("PickOperator.UserRoles.Role").
			Preload("PickOperator.UserRoles.Assigner").
			Where("tracking = ?", ret.OldTracking).First(&order).Error; err == nil {
			ret.Order = &order
		}
	}

	utilities.SuccessResponse(c, http.StatusOK, "Return retrieved successfully", ret.ToReturnResponse())
}

// CreateReturn godoc
// @Summary Create a new return
// @Description Create a new return.
// @Tags returns
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateReturnRequest true "Create Return Request"
// @Success 201 {object} utilities.Response{data=models.ReturnResponse}
// @Failure 400 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Router /api/returns [post]
func (rc *ReturnController) CreateReturn(c *gin.Context) {
	// Get user ID from JWT token
	userID, exists := c.Get("user_id")
	if !exists {
		utilities.ErrorResponse(c, http.StatusUnauthorized, "Unauthorized", "User not authenticated")
		return
	}

	var req CreateReturnRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utilities.ValidationErrorResponse(c, err)
		return
	}

	// Convert userID to uint
	userIDUint, ok := userID.(uint)
	if !ok {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Invalid user ID", "Failed to convert user ID")
		return
	}

	// Convert new tracking to uppercase and trim spaces
	req.NewTracking = strings.ToUpper(strings.TrimSpace(req.NewTracking))

	// Check for duplicate new tracking
	var existingReturn models.Return
	if err := rc.DB.Where("new_tracking = ?", req.NewTracking).First(&existingReturn).Error; err == nil {
		utilities.ErrorResponse(c, http.StatusBadRequest, "Tracking already exists", "Return with this new tracking already exists")
		return
	}

	// Convert old tracking to uppercase and trim spaces
	req.OldTracking = strings.ToUpper(strings.TrimSpace(req.OldTracking))

	// Find order by old_tracking to get order_ginee_id and details (before transaction)
	var order models.Order
	if err := rc.DB.Preload("OrderDetails").Where("tracking = ?", req.OldTracking).First(&order).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusNotFound, "Order not found", "No order found with the specified old tracking number")
		return
	}

	// Check if order has details
	if len(order.OrderDetails) == 0 {
		utilities.ErrorResponse(c, http.StatusBadRequest, "Order has no details", "The order has no order details to copy")
		return
	}

	// Start database transaction
	tx := rc.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	ret := models.Return{
		NewTracking:  req.NewTracking,
		OldTracking:  req.OldTracking,
		ReturnType:   req.ReturnType,
		ChannelID:    req.ChannelID,
		StoreID:      req.StoreID,
		ReturnReason: req.ReturnReason,
		CreatedBy:    userIDUint,
		OrderGineeID: order.OrderGineeID,
	}

	// Create return within transaction
	if err := tx.Create(&ret).Error; err != nil {
		tx.Rollback()
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to create return", err.Error())
		return
	}

	// Track products not found and created count
	var productsNotFound []string
	var createdCount int

	// Create return details based on order details
	for _, orderDetail := range order.OrderDetails {
		// Find product by SKU from order detail
		var product models.Product
		if err := tx.Where("sku = ?", orderDetail.Sku).First(&product).Error; err != nil {
			// Track products not found
			productsNotFound = append(productsNotFound, orderDetail.Sku)
			continue
		}

		returnDetail := models.ReturnDetail{
			ReturnID:  ret.ID,
			ProductID: product.ID,
			Quantity:  orderDetail.Quantity,
		}

		if err := tx.Create(&returnDetail).Error; err != nil {
			tx.Rollback()
			utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to create return detail", err.Error())
			return
		}
		createdCount++
	}

	// If no return details were created, return an error
	if createdCount == 0 {
		tx.Rollback()
		errorMsg := fmt.Sprintf("No return details created. Products not found in database: %s", strings.Join(productsNotFound, ", "))
		utilities.ErrorResponse(c, http.StatusBadRequest, "Failed to create return details", errorMsg)
		return
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to commit transaction", err.Error())
		return
	}

	// Reload return with relationships
	rc.DB.Preload("ReturnDetails.Product").
		Preload("Channel").
		Preload("Store").
		Preload("CreateOperator").
		Preload("UpdateOperator").
		First(&ret, ret.ID)

	// Load order data
	if ret.OldTracking != "" {
		var order models.Order
		if err := rc.DB.Preload("OrderDetails").
			Preload("PickOperator.UserRoles.Role").
			Preload("PickOperator.UserRoles.Assigner").
			Where("tracking = ?", ret.OldTracking).First(&order).Error; err == nil {
			ret.Order = &order
		}
	}

	// Build success message with warning if some products weren't found
	message := fmt.Sprintf("Return created successfully (%d of %d products synced)", createdCount, len(order.OrderDetails))
	if len(productsNotFound) > 0 {
		message += fmt.Sprintf(". Warning: %d product(s) not found - SKU: %s", len(productsNotFound), strings.Join(productsNotFound, ", "))
	}

	utilities.SuccessResponse(c, http.StatusOK, message, ret.ToReturnResponse())
}

// UpdateDataReturn godoc
// @Summary Update return data
// @Description Update return data.
// @Tags returns
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Return ID"
// @Param request body UpdateReturnRequest true "Update Return Request"
// @Success 200 {object} utilities.Response{data=models.ReturnResponse}
// @Failure 400 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Failure 404 {object} utilities.Response
// @Router /api/returns/{id} [put]
func (rc *ReturnController) UpdateDataReturn(c *gin.Context) {
	// Get user ID from JWT token
	userID, exists := c.Get("user_id")
	if !exists {
		utilities.ErrorResponse(c, http.StatusUnauthorized, "Unauthorized", "User not authenticated")
		return
	}

	returnID := c.Param("id")

	var req UpdateReturnRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utilities.ValidationErrorResponse(c, err)
		return
	}

	var ret models.Return
	if err := rc.DB.First(&ret, returnID).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusNotFound, "Return not found", err.Error())
		return
	}

	// Convert userID to uint
	userIDUint, ok := userID.(uint)
	if !ok {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Invalid user ID", "Failed to convert user ID")
		return
	}

	// Update return data fields
	ret.ReturnNumber = req.ReturnNumber
	ret.ScrapNumber = req.ScrapNumber
	ret.UpdatedBy = &userIDUint

	// Save the updated return
	if err := rc.DB.Save(&ret).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to update return", err.Error())
		return
	}

	// Load updated return with relationships
	rc.DB.Preload("ReturnDetails.Product").
		Preload("Channel").
		Preload("Store").
		Preload("CreateOperator").
		Preload("UpdateOperator").
		First(&ret, ret.ID)

	// Load order data if old_tracking matches
	if ret.OldTracking != "" {
		var order models.Order
		if err := rc.DB.Preload("OrderDetails").
			Preload("PickOperator.UserRoles.Role").
			Preload("PickOperator.UserRoles.Assigner").
			Where("tracking = ?", ret.OldTracking).First(&order).Error; err == nil {
			ret.Order = &order
		}
	}

	utilities.SuccessResponse(c, http.StatusOK, "Return data updated successfully", ret.ToReturnResponse())
}

// Request/Response structs
type ReturnsListResponse struct {
	Returns    []models.ReturnResponse      `json:"returns"`
	Pagination utilities.PaginationResponse `json:"pagination"`
}

type CreateReturnRequest struct {
	NewTracking  string  `json:"new_tracking" binding:"required"`
	OldTracking  string  `json:"old_tracking" binding:"required"`
	ReturnType   string  `json:"return_type" binding:"required"`
	ReturnReason string  `json:"return_reason" binding:"required"`
	ReturnNumber *string `json:"return_number"`
	ScrapNumber  *string `json:"scrap_number"`
	ChannelID    uint    `json:"channel_id" binding:"required"`
	StoreID      uint    `json:"store_id" binding:"required"`
}

type UpdateReturnRequest struct {
	ReturnNumber string `json:"return_number" binding:"required"`
	ScrapNumber  string `json:"scrap_number" binding:"required"`
}
