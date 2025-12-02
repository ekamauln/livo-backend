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

type OrderController struct {
	DB *gorm.DB
}

// NewOrderController creates a new order controller
func NewOrderController(db *gorm.DB) *OrderController {
	return &OrderController{DB: db}
}

// UpdateOrderComplainedStatus godoc
// @Summary Update order complained status
// @Description Update the complained status of an order.
// @Tags orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Order ID"
// @Param request body UpdateComplainedStatusRequest true "Update complained status request"
// @Success 200 {object} utilities.Response{data=models.OrderResponse}
// @Failure 400 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Failure 404 {object} utilities.Response
// @Router /api/orders/{id}/complained [put]
func (oc *OrderController) UpdateOrderComplainedStatus(c *gin.Context) {
	orderID := c.Param("id")

	var req UpdateComplainedStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utilities.ValidationErrorResponse(c, err)
		return
	}

	// Find the order
	var order models.Order
	if err := oc.DB.First(&order, orderID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utilities.ErrorResponse(c, http.StatusNotFound, "Order not found", "no order found with the specified ID")
			return
		}
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to find order", err.Error())
		return
	}

	// Update complained status
	order.Complained = req.Complained

	if err := oc.DB.Save(&order).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to update order complained status", err.Error())
		return
	}

	// Load order with details for response
	oc.DB.Preload("OrderDetails").Preload("PickOperator.UserRoles.Role").Preload("PickOperator.UserRoles.Assigner").First(&order, order.ID)

	message := "Order complained status updated successfully"
	if req.Complained {
		message = "Order marked as complained"
	} else {
		message = "Order unmarked as complained"
	}

	utilities.SuccessResponse(c, http.StatusOK, message, order.ToOrderResponse())
}

// Add this struct with the other request structs
type UpdateComplainedStatusRequest struct {
	Complained bool `json:"complained" binding:"required" example:"true"`
}

// GetOrders godoc
// @Summary Get all orders
// @Description Get list of all orders with optional date range filtering and search.
// @Tags orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Param start_date query string false "Start date (YYYY-MM-DD format)"
// @Param end_date query string false "End date (YYYY-MM-DD format)"
// @Param search query string false "Search by Order Ginee ID or Tracking number"
// @Success 200 {object} utilities.Response{data=OrdersListResponse}
// @Failure 400 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Router /api/orders [get]
func (oc *OrderController) GetOrders(c *gin.Context) {
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	// Parse date range parameters
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	// Parse search parameter
	search := c.Query("search")

	var orders []models.Order
	var total int64

	// Build the query
	query := oc.DB.Model(&models.Order{})

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

	// Apply search filter if provided
	if search != "" {
		// Search in both order_ginee_id and tracking fields
		query = query.Where("order_ginee_id ILIKE ? OR tracking ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// Get total count with all filters
	if err := query.Count(&total).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to count orders", err.Error())
		return
	}

	// Get orders with pagination, filters, sorted by ID descending
	if err := query.Order("id DESC").Limit(limit).Offset(offset).
		Preload("PickOperator.UserRoles.Role").
		Preload("PickOperator.UserRoles.Assigner").
		Preload("OrderDetails").
		Find(&orders).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve orders", err.Error())
		return
	}

	// After loading orders, manually fetch and attach products
	for i := range orders {
		for j := range orders[i].OrderDetails {
			var product models.Product
			if err := oc.DB.Where("sku = ?", orders[i].OrderDetails[j].Sku).First(&product).Error; err == nil {
				orders[i].OrderDetails[j].Product = &product
			}
		}
	}

	// Convert to response format
	orderResponses := make([]models.OrderResponse, len(orders))
	for i, order := range orders {
		orderResponses[i] = order.ToOrderResponse()
	}

	response := OrdersListResponse{
		Orders: orderResponses,
		Pagination: utilities.PaginationResponse{
			Page:  page,
			Limit: limit,
			Total: int(total),
		},
	}

	// Build success message
	message := "Orders retrieved successfully"
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

// GetOrder godoc
// @Summary Get order by ID
// @Description Get specific order information with complete details.
// @Tags orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Order ID"
// @Success 200 {object} utilities.Response{data=models.OrderResponse}
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Failure 404 {object} utilities.Response
// @Router /api/orders/{id} [get]
func (oc *OrderController) GetOrder(c *gin.Context) {
	orderID := c.Param("id")
	var order models.Order

	if err := oc.DB.
		Preload("OrderDetails").
		Preload("PickOperator.UserRoles.Role").
		Preload("PickOperator.UserRoles.Assigner").
		First(&order, orderID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utilities.ErrorResponse(c, http.StatusNotFound, "Order not found", "no order found with the specified ID")
			return
		}
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve order", err.Error())
		return
	}

	// Manually fetch and attach products
	for i := range order.OrderDetails {
		var product models.Product
		if err := oc.DB.Where("sku = ?", order.OrderDetails[i].Sku).First(&product).Error; err == nil {
			order.OrderDetails[i].Product = &product
		}
	}

	utilities.SuccessResponse(c, http.StatusOK, "Order retrieved successfully", order.ToOrderResponse())
}

// BulkCreateOrders godoc
// @Summary Bulk create orders
// @Description Create multiple orders at once, skipping duplicates.
// @Tags orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body BulkCreateOrderRequest true "Bulk create order request"
// @Success 201 {object} utilities.Response{data=BulkCreateOrderResponse}
// @Failure 400 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Router /api/orders/bulk [post]
func (oc *OrderController) BulkCreateOrders(c *gin.Context) {
	var req BulkCreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utilities.ValidationErrorResponse(c, err)
		return
	}

	var createdOrders []models.Order
	var skippedOrders []SkippedOrder
	var failedOrders []FailedOrder

	for i, orderReq := range req.Orders {
		// Check if order with same OrderGineeID already exists
		var existingOrder models.Order
		if err := oc.DB.Where("order_ginee_id = ?", orderReq.OrderGineeID).First(&existingOrder).Error; err == nil {
			// Order exists, skip it
			skippedOrders = append(skippedOrders, SkippedOrder{
				Index:        i,
				OrderGineeID: orderReq.OrderGineeID,
				Reason:       "Order already exists",
			})
			continue
		}

		// Create order
		order := models.Order{
			OrderGineeID:     orderReq.OrderGineeID,
			ProcessingStatus: "ready to pick", // Always set to "ready to pick"
			Channel:          orderReq.Channel,
			Store:            orderReq.Store,
			Buyer:            orderReq.Buyer,
			Address:          orderReq.Address,
			Courier:          orderReq.Courier,
			Tracking:         orderReq.Tracking,
		}

		if orderReq.SentBefore != "" {
			if parsedTime, err := time.Parse("2006-01-02 15:04:00", orderReq.SentBefore); err == nil {
				order.SentBefore = parsedTime
			} else {
				if parsedTime, err := time.Parse("2006-01-02 15:04", orderReq.SentBefore); err == nil {
					order.SentBefore = parsedTime
				}
			}
		}

		// Create order details
		for _, detailReq := range orderReq.OrderDetails {
			orderDetail := models.OrderDetail{
				Sku:         detailReq.Sku,
				ProductName: detailReq.ProductName,
				Variant:     detailReq.Variant,
				Quantity:    detailReq.Quantity,
			}
			order.OrderDetails = append(order.OrderDetails, orderDetail)
		}

		// Try to create the order
		if err := oc.DB.Create(&order).Error; err != nil {
			// Failed to create order
			failedOrders = append(failedOrders, FailedOrder{
				Index:        i,
				OrderGineeID: orderReq.OrderGineeID,
				Error:        err.Error(),
			})
			continue
		}

		// Load order with details for response
		oc.DB.Preload("OrderDetails").Preload("PickOperator").First(&order, order.ID)
		createdOrders = append(createdOrders, order)
	}

	// Convert created orders to response format
	createdOrderResponses := make([]models.OrderResponse, len(createdOrders))
	for i, order := range createdOrders {
		createdOrderResponses[i] = order.ToOrderResponse()
	}

	response := BulkCreateOrderResponse{
		Summary: BulkCreateSummary{
			Total:   len(req.Orders),
			Created: len(createdOrders),
			Skipped: len(skippedOrders),
			Failed:  len(failedOrders),
		},
		CreatedOrders: createdOrderResponses,
		SkippedOrders: skippedOrders,
		FailedOrders:  failedOrders,
	}

	// Determine response status
	statusCode := http.StatusCreated
	message := "Bulk order creation completed"

	if len(createdOrders) == 0 {
		if len(skippedOrders) > 0 {
			statusCode = http.StatusOK
			message = "All orders were skipped (already exist)"
		} else {
			statusCode = http.StatusBadRequest
			message = "No orders could be created"
		}
	} else if len(failedOrders) > 0 || len(skippedOrders) > 0 {
		message = "Bulk order creation completed with some issues"
	}

	utilities.SuccessResponse(c, statusCode, message, response)
}

// UpdateOrder godoc
// @Summary Update order and order details
// @Description Update order information and manage order details (add, update, remove products)
// @Tags orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Order ID"
// @Param request body UpdateOrderRequest true "Update order request"
// @Success 200 {object} utilities.Response{data=models.OrderResponse}
// @Failure 400 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Failure 404 {object} utilities.Response
// @Router /api/orders/{id} [put]
func (oc *OrderController) UpdateOrder(c *gin.Context) {
	orderID := c.Param("id")

	var req UpdateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utilities.ValidationErrorResponse(c, err)
		return
	}

	// Get current user ID from context
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		utilities.ErrorResponse(c, http.StatusUnauthorized, "User not found", "user ID not found in context")
		return
	}

	userID, ok := userIDInterface.(uint)
	if !ok {
		utilities.ErrorResponse(c, http.StatusUnauthorized, "Invalid user ID", "user ID has invalid type")
		return
	}

	// Find the order
	var order models.Order
	if err := oc.DB.Preload("OrderDetails").First(&order, orderID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utilities.ErrorResponse(c, http.StatusNotFound, "Order not found", "no order found with the specified ID")
			return
		}
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to find order", err.Error())
		return
	}

	// Check if order status allows modification
	if order.ProcessingStatus == "picking process" || order.ProcessingStatus == "qc process" {
		utilities.ErrorResponse(c, http.StatusForbidden, "Order modification not allowed", fmt.Sprintf("cannot modify order when processing status is '%s'.", order.ProcessingStatus))
		return
	}

	// Check if order is cancelled
	if order.EventStatus != nil && *order.EventStatus == "cancelled" {
		utilities.ErrorResponse(c, http.StatusBadRequest, "Order already cancelled", "this order has already been cancelled")
		return
	}

	// Update basic order fields
	eventStatus := "changed"
	order.EventStatus = &eventStatus
	order.Channel = req.Channel
	order.Store = req.Store
	order.Buyer = req.Buyer
	order.Address = req.Address
	order.Courier = req.Courier
	order.Tracking = req.Tracking

	if req.SentBefore != "" {
		if parsedTime, err := time.Parse("2006-01-02 15:04:05", req.SentBefore); err == nil {
			order.SentBefore = parsedTime
		}
	}

	// Set changed_by and changed_at
	now := time.Now()
	order.ChangedBy = &userID
	order.ChangedAt = &now

	// Begin transaction
	tx := oc.DB.Begin()

	// Save order changes
	if err := tx.Save(&order).Error; err != nil {
		tx.Rollback()
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to update order", err.Error())
		return
	}

	// Process order details updates
	// Create a map of existing order details by ID for quick lookup
	existingDetailsMap := make(map[uint]models.OrderDetail)
	for _, detail := range order.OrderDetails {
		existingDetailsMap[detail.ID] = detail
	}

	// Track which existing details are still in the update
	updatedDetailIDs := make(map[uint]bool)

	// Process each detail in the request
	for _, detailReq := range req.OrderDetails {
		if detailReq.ID == 0 {
			// New product - create new order detail
			newDetail := models.OrderDetail{
				OrderID:     order.ID,
				Sku:         detailReq.Sku,
				ProductName: detailReq.ProductName,
				Variant:     detailReq.Variant,
				Quantity:    detailReq.Quantity,
			}
			if err := tx.Create(&newDetail).Error; err != nil {
				tx.Rollback()
				utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to add new order detail", err.Error())
				return
			}
		} else {
			// Update existing product
			if existingDetail, exists := existingDetailsMap[detailReq.ID]; exists {
				existingDetail.Sku = detailReq.Sku
				existingDetail.ProductName = detailReq.ProductName
				existingDetail.Variant = detailReq.Variant
				existingDetail.Quantity = detailReq.Quantity

				if err := tx.Save(&existingDetail).Error; err != nil {
					tx.Rollback()
					utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to update order detail", err.Error())
					return
				}
				updatedDetailIDs[detailReq.ID] = true
			} else {
				tx.Rollback()
				utilities.ErrorResponse(c, http.StatusNotFound, "Order detail not found", fmt.Sprintf("order detail with ID %d not found for this order", detailReq.ID))
				return
			}
		}
	}

	// Remove products that are not in the update request
	for detailID := range existingDetailsMap {
		if !updatedDetailIDs[detailID] {
			// This detail was not in the update request, so delete it
			if err := tx.Delete(&models.OrderDetail{}, detailID).Error; err != nil {
				tx.Rollback()
				utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to remove order detail", err.Error())
				return
			}
		}
	}

	// Verify at least one order detail remains
	var detailCount int64
	if err := tx.Model(&models.OrderDetail{}).Where("order_id = ?", order.ID).Count(&detailCount).Error; err != nil {
		tx.Rollback()
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to count order details", err.Error())
		return
	}

	if detailCount == 0 {
		tx.Rollback()
		utilities.ErrorResponse(c, http.StatusBadRequest, "Invalid update", "order must have at least one order detail")
		return
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to commit transaction", err.Error())
		return
	}

	// Reload order with all relationships
	if err := oc.DB.
		Preload("OrderDetails").
		Preload("PickOperator.UserRoles.Role").
		Preload("PickOperator.UserRoles.Assigner").
		Preload("ChangeOperator").
		First(&order, order.ID).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to reload order", err.Error())
		return
	}

	// Manually fetch and attach products to order details
	for i := range order.OrderDetails {
		var product models.Product
		if err := oc.DB.Where("sku = ?", order.OrderDetails[i].Sku).First(&product).Error; err == nil {
			order.OrderDetails[i].Product = &product
		}
	}

	utilities.SuccessResponse(c, http.StatusOK, "Order updated successfully", order.ToOrderResponse())
}

// DuplicateOrder godoc
// @Summary Duplicate an order
// @Description Duplicate an existing order with all its details. The new order will have "X-" prefix added to tracking and the original order will have "-X2" suffix added to order_ginee_id
// @Tags orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Order ID to duplicate"
// @Success 201 {object} utilities.Response{data=DuplicateOrderResponse}
// @Failure 400 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Failure 404 {object} utilities.Response
// @Router /api/orders/{id}/duplicate [post]
func (oc *OrderController) DuplicateOrder(c *gin.Context) {
	orderID := c.Param("id")

	// Get current user ID from context
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		utilities.ErrorResponse(c, http.StatusUnauthorized, "User not found", "user ID not found in context")
		return
	}

	userID, ok := userIDInterface.(uint)
	if !ok {
		utilities.ErrorResponse(c, http.StatusUnauthorized, "Invalid user ID", "user ID has invalid type")
		return
	}

	// Find the original order
	var originalOrder models.Order
	if err := oc.DB.Preload("OrderDetails").First(&originalOrder, orderID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utilities.ErrorResponse(c, http.StatusNotFound, "Order not found", "no order found with the specified ID")
			return
		}
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to find order", err.Error())
		return
	}

	// Check if order status allows modification
	if originalOrder.ProcessingStatus == "picking process" || originalOrder.ProcessingStatus == "qc process" {
		utilities.ErrorResponse(c, http.StatusForbidden, "Order modification not allowed", fmt.Sprintf("cannot modify order when processing status is '%s'.", originalOrder.ProcessingStatus))
		return
	}

	// Check if order is cancelled
	if originalOrder.EventStatus != nil && *originalOrder.EventStatus == "cancelled" {
		utilities.ErrorResponse(c, http.StatusBadRequest, "Order already cancelled", "this order has already been cancelled")
		return
	}

	// Begin transaction
	tx := oc.DB.Begin()

	// Store the original tracking before modification
	originalTracking := originalOrder.Tracking

	// Update original order's order_ginee_id by adding "-X2" suffix and tracking with "X-" prefix
	// eventStatus := "duplicated"
	// originalOrder.EventStatus = &eventStatus
	originalOrder.OrderGineeID = originalOrder.OrderGineeID + "-X2"
	originalOrder.Tracking = "X-" + originalOrder.Tracking
	if err := tx.Save(&originalOrder).Error; err != nil {
		tx.Rollback()
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to update original order", err.Error())
		return
	}

	// Create new duplicated order
	now := time.Now()
	duplicatedEventStatus := "duplicated"
	duplicatedOrder := models.Order{
		OrderGineeID:     originalOrder.OrderGineeID[:len(originalOrder.OrderGineeID)-3], // Remove the "-X2" suffix for the new order
		ProcessingStatus: originalOrder.ProcessingStatus,
		EventStatus:      &duplicatedEventStatus,
		Channel:          originalOrder.Channel,
		Store:            originalOrder.Store,
		Buyer:            originalOrder.Buyer,
		Address:          originalOrder.Address,
		Courier:          originalOrder.Courier,
		Tracking:         originalTracking, // Use original tracking without "X-" prefix
		SentBefore:       originalOrder.SentBefore,
		Complained:       false,
		ChangedBy:        &userID,
		ChangedAt:        &now,
	}

	// Create the duplicated order
	if err := tx.Create(&duplicatedOrder).Error; err != nil {
		tx.Rollback()
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to create duplicated order", err.Error())
		return
	}

	// Duplicate order details
	for _, detail := range originalOrder.OrderDetails {
		duplicatedDetail := models.OrderDetail{
			OrderID:     duplicatedOrder.ID,
			Sku:         detail.Sku,
			ProductName: detail.ProductName,
			Variant:     detail.Variant,
			Quantity:    detail.Quantity,
		}
		if err := tx.Create(&duplicatedDetail).Error; err != nil {
			tx.Rollback()
			utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to duplicate order details", err.Error())
			return
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to commit transaction", err.Error())
		return
	}

	// Reload both orders with all relationships
	if err := oc.DB.
		Preload("OrderDetails").
		Preload("PickOperator.UserRoles.Role").
		Preload("PickOperator.UserRoles.Assigner").
		Preload("ChangeOperator").
		First(&originalOrder, originalOrder.ID).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to reload original order", err.Error())
		return
	}

	if err := oc.DB.
		Preload("OrderDetails").
		Preload("PickOperator.UserRoles.Role").
		Preload("PickOperator.UserRoles.Assigner").
		Preload("ChangeOperator").
		First(&duplicatedOrder, duplicatedOrder.ID).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to reload duplicated order", err.Error())
		return
	}

	// Manually fetch and attach products to order details for both orders
	for i := range originalOrder.OrderDetails {
		var product models.Product
		if err := oc.DB.Where("sku = ?", originalOrder.OrderDetails[i].Sku).First(&product).Error; err == nil {
			originalOrder.OrderDetails[i].Product = &product
		}
	}

	for i := range duplicatedOrder.OrderDetails {
		var product models.Product
		if err := oc.DB.Where("sku = ?", duplicatedOrder.OrderDetails[i].Sku).First(&product).Error; err == nil {
			duplicatedOrder.OrderDetails[i].Product = &product
		}
	}

	response := DuplicateOrderResponse{
		OriginalOrder:   originalOrder.ToOrderResponse(),
		DuplicatedOrder: duplicatedOrder.ToOrderResponse(),
	}

	utilities.SuccessResponse(c, http.StatusCreated, "Order duplicated successfully", response)
}

// CancelOrder godoc
// @Summary Cancel an order
// @Description Cancel an order by setting event_status to "cancelled" and recording who cancelled it and when
// @Tags orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Order ID to cancel"
// @Success 200 {object} utilities.Response{data=models.OrderResponse}
// @Failure 400 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Failure 404 {object} utilities.Response
// @Router /api/orders/{id}/cancel [put]
func (oc *OrderController) CancelOrder(c *gin.Context) {
	orderID := c.Param("id")

	// Get current user ID from context
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		utilities.ErrorResponse(c, http.StatusUnauthorized, "User not found", "user ID not found in context")
		return
	}

	userID, ok := userIDInterface.(uint)
	if !ok {
		utilities.ErrorResponse(c, http.StatusUnauthorized, "Invalid user ID", "user ID has invalid type")
		return
	}

	// Find the order
	var order models.Order
	if err := oc.DB.First(&order, orderID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utilities.ErrorResponse(c, http.StatusNotFound, "Order not found", "no order found with the specified ID")
			return
		}
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to find order", err.Error())
		return
	}

	// Check if order status allows modification
	if order.ProcessingStatus == "picking process" || order.ProcessingStatus == "qc process" {
		utilities.ErrorResponse(c, http.StatusForbidden, "Order modification not allowed", fmt.Sprintf("cannot modify order when processing status is '%s'.", order.ProcessingStatus))
		return
	}

	// Check if order is already cancelled
	if order.EventStatus != nil && *order.EventStatus == "cancelled" {
		utilities.ErrorResponse(c, http.StatusBadRequest, "Order already cancelled", "this order has already been cancelled")
		return
	}

	// Update order with cancellation details
	eventStatus := "cancelled"
	now := time.Now()
	order.EventStatus = &eventStatus
	order.CancelledBy = &userID
	order.CancelledAt = &now

	if err := oc.DB.Save(&order).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to cancel order", err.Error())
		return
	}

	// Reload order with all relationships
	if err := oc.DB.
		Preload("OrderDetails").
		Preload("PickOperator.UserRoles.Role").
		Preload("PickOperator.UserRoles.Assigner").
		Preload("CancelOperator").
		First(&order, order.ID).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to reload order", err.Error())
		return
	}

	// Manually fetch and attach products to order details
	for i := range order.OrderDetails {
		var product models.Product
		if err := oc.DB.Where("sku = ?", order.OrderDetails[i].Sku).First(&product).Error; err == nil {
			order.OrderDetails[i].Product = &product
		}
	}

	utilities.SuccessResponse(c, http.StatusOK, "Order cancelled successfully", order.ToOrderResponse())
}

// AssignPicker godoc
// @Summary Assign a picker to an order
// @Description Assign a picker to an order, setting assigned_by to current user, assigned_at to now, picked_by to specified picker, and processing_status to "picking process"
// @Tags orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Order ID to assign picker"
// @Param request body AssignPickerRequest true "Assign picker request"
// @Success 200 {object} utilities.Response{data=models.OrderResponse}
// @Failure 400 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Failure 404 {object} utilities.Response
// @Router /api/orders/{id}/assign-picker [put]
func (oc *OrderController) AssignPicker(c *gin.Context) {
	orderID := c.Param("id")

	var req AssignPickerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utilities.ValidationErrorResponse(c, err)
		return
	}

	// Get current user ID from context (assigner)
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		utilities.ErrorResponse(c, http.StatusUnauthorized, "User not found", "user ID not found in context")
		return
	}

	userID, ok := userIDInterface.(uint)
	if !ok {
		utilities.ErrorResponse(c, http.StatusUnauthorized, "Invalid user ID", "user ID has invalid type")
		return
	}

	// Verify the picker exists
	var picker models.User
	if err := oc.DB.First(&picker, req.PickerID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utilities.ErrorResponse(c, http.StatusNotFound, "Picker not found", "no user found with the specified picker ID")
			return
		}
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to find picker", err.Error())
		return
	}

	// Find the order
	var order models.Order
	if err := oc.DB.First(&order, orderID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utilities.ErrorResponse(c, http.StatusNotFound, "Order not found", "no order found with the specified ID")
			return
		}
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to find order", err.Error())
		return
	}

	// Check if order is cancelled
	if order.EventStatus != nil && *order.EventStatus == "cancelled" {
		utilities.ErrorResponse(c, http.StatusBadRequest, "Order already cancelled", "cannot assign picker to a cancelled order")
		return
	}

	// Check if order is already in picking process or completed
	if order.ProcessingStatus == "picking process" {
		utilities.ErrorResponse(c, http.StatusBadRequest, "Order already being picked", "this order is already in picking process")
		return
	}

	if order.ProcessingStatus == "qc process" || order.ProcessingStatus == "completed" {
		utilities.ErrorResponse(c, http.StatusBadRequest, "Order cannot be assigned", fmt.Sprintf("cannot assign picker when processing status is '%s'", order.ProcessingStatus))
		return
	}

	// Update order with assignment details
	now := time.Now()
	order.AssignedBy = &userID
	order.AssignedAt = &now
	order.PickedBy = &req.PickerID
	order.ProcessingStatus = "picking process"

	if err := oc.DB.Save(&order).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to assign picker", err.Error())
		return
	}

	// Reload order with all relationships
	if err := oc.DB.
		Preload("OrderDetails").
		Preload("PickOperator.UserRoles.Role").
		Preload("PickOperator.UserRoles.Assigner").
		Preload("AssignOperator").
		First(&order, order.ID).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to reload order", err.Error())
		return
	}

	// Manually fetch and attach products to order details
	for i := range order.OrderDetails {
		var product models.Product
		if err := oc.DB.Where("sku = ?", order.OrderDetails[i].Sku).First(&product).Error; err == nil {
			order.OrderDetails[i].Product = &product
		}
	}

	utilities.SuccessResponse(c, http.StatusOK, "Picker assigned successfully", order.ToOrderResponse())
}

// PendingPickOrders godoc
// @Summary Get orders pending pick assignment
// @Description Pending order that already assigned to a picker, but not picked yet.
// @Tags orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Order ID to pending picking process"
// @Param request body PendingPickRequest true "Pending pick request with coordinator credentials"
// @Success 200 {object} utilities.Response{data=models.OrderResponse}
// @Failure 400 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Failure 404 {object} utilities.Response
// @Router /api/orders/{id}/pending-pick [put]
func (oc *OrderController) PendingPickOrders(c *gin.Context) {
	orderID := c.Param("id")

	// Get current user ID from context (pending operator)
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		utilities.ErrorResponse(c, http.StatusUnauthorized, "User not found", "user ID not found in context")
		return
	}

	userID, ok := userIDInterface.(uint)
	if !ok {
		utilities.ErrorResponse(c, http.StatusUnauthorized, "Invalid user ID", "user ID has invalid type")
		return
	}

	// Find the order
	var order models.Order
	if err := oc.DB.First(&order, orderID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utilities.ErrorResponse(c, http.StatusNotFound, "Order not found", "no order found with the specified ID")
			return
		}
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to find order", err.Error())
		return
	}

	// Check if status order is "picking process"
	if order.ProcessingStatus != "picking process" {
		utilities.ErrorResponse(c, http.StatusBadRequest, "Order not in picking process", "only orders in 'picking process' status can be set to pending pick")
		return
	}

	// Update order with pending pick details
	now := time.Now()
	order.ProcessingStatus = "pending picking"
	order.PendingBy = &userID // Set pending operator
	order.PendingAt = &now
	order.PickedBy = nil   // Clear picked_by since it's pending
	order.AssignedBy = nil // Clear assigned_by since it's pending
	order.AssignedAt = nil // Clear assigned_at since it's pending

	if err := oc.DB.Save(&order).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to set order to pending pick", err.Error())
		return
	}

	// Reload order with all relationships
	if err := oc.DB.
		Preload("OrderDetails").
		Preload("PickOperator.UserRoles.Role").
		Preload("PickOperator.UserRoles.Assigner").
		Preload("PendingOperator").
		First(&order, order.ID).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to reload order", err.Error())
		return
	}

	// Manually fetch and attach products to order details
	for i := range order.OrderDetails {
		var product models.Product
		if err := oc.DB.Where("sku = ?", order.OrderDetails[i].Sku).First(&product).Error; err == nil {
			order.OrderDetails[i].Product = &product
		}
	}

	utilities.SuccessResponse(c, http.StatusOK, "Order set to pending pick successfully", order.ToOrderResponse())
}

// Request and Response Structs
type OrdersListResponse struct {
	Orders     []models.OrderResponse       `json:"orders"`
	Pagination utilities.PaginationResponse `json:"pagination"`
}

type CreateOrderRequest struct {
	OrderGineeID string                     `json:"order_ginee_id" binding:"required" example:"2509116GA36VM5"`
	Status       string                     `json:"status" example:"ready to pick"`
	Channel      string                     `json:"channel" binding:"required" example:"Shopee"`
	Store        string                     `json:"store" binding:"required" example:"SP deParcelRibbon"`
	Buyer        string                     `json:"buyer" binding:"required" example:"John Doe"`
	Address      string                     `json:"address" binding:"required" example:"123 Main St, City, Country"`
	Courier      string                     `json:"courier" example:"JNE"`
	Tracking     string                     `json:"tracking" example:"JNE1234567890"`
	SentBefore   string                     `json:"sent_before" example:"2023-01-01 12:00"`
	OrderDetails []CreateOrderDetailRequest `json:"order_details" binding:"required,min=1"`
}

type CreateOrderDetailRequest struct {
	Sku         string `json:"sku" binding:"required" example:"PROD001"`
	ProductName string `json:"product_name" binding:"required" example:"Sample Product"`
	Variant     string `json:"variant" example:"Red - Size M"`
	Quantity    int    `json:"quantity" binding:"required,min=1" example:"2"`
}

type BulkCreateOrderRequest struct {
	Orders []CreateOrderRequest `json:"orders" binding:"required,min=1"`
}

type BulkCreateOrderResponse struct {
	Summary       BulkCreateSummary      `json:"summary"`
	CreatedOrders []models.OrderResponse `json:"created_orders"`
	SkippedOrders []SkippedOrder         `json:"skipped_orders"`
	FailedOrders  []FailedOrder          `json:"failed_orders"`
}

type BulkCreateSummary struct {
	Total   int `json:"total"`
	Created int `json:"created"`
	Skipped int `json:"skipped"`
	Failed  int `json:"failed"`
}

type SkippedOrder struct {
	Index        int    `json:"index"`
	OrderGineeID string `json:"order_ginee_id"`
	Reason       string `json:"reason"`
}

type FailedOrder struct {
	Index        int    `json:"index"`
	OrderGineeID string `json:"order_ginee_id"`
	Error        string `json:"error"`
}

type UpdateOrderRequest struct {
	EventStatus  string                     `json:"event_status" example:"data changed"`
	Channel      string                     `json:"channel" binding:"required" example:"Shopee"`
	Store        string                     `json:"store" binding:"required" example:"SP deParcelRibbon"`
	Buyer        string                     `json:"buyer" binding:"required" example:"John Doe"`
	Address      string                     `json:"address" binding:"required" example:"123 Main St, City, Country"`
	Courier      string                     `json:"courier" binding:"required" example:"JNE"`
	Tracking     string                     `json:"tracking" binding:"required" example:"JNE1234567890"`
	SentBefore   string                     `json:"sent_before" example:"2023-01-01 12:00:00"`
	OrderDetails []UpdateOrderDetailRequest `json:"order_details" binding:"required,min=1"`
}

type UpdateOrderDetailRequest struct {
	ID          uint   `json:"id" example:"1"` // 0 for new product, existing ID for update
	Sku         string `json:"sku" binding:"required" example:"PROD001"`
	ProductName string `json:"product_name" binding:"required" example:"Sample Product"`
	Variant     string `json:"variant" example:"Red - Size M"`
	Quantity    int    `json:"quantity" binding:"required,min=1" example:"2"`
}

type DuplicateOrderResponse struct {
	OriginalOrder   models.OrderResponse `json:"original_order"`
	DuplicatedOrder models.OrderResponse `json:"duplicated_order"`
}

type AssignPickerRequest struct {
	PickerID uint `json:"picker_id" binding:"required" example:"1"`
}
