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
	oc.DB.Preload("OrderDetails").Preload("Picker.UserRoles.Role").Preload("Picker.UserRoles.Assigner").First(&order, order.ID)

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
		Preload("Picker.UserRoles.Role").
		Preload("Picker.UserRoles.Assigner").
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
		Preload("Picker.UserRoles.Role").
		Preload("Picker.UserRoles.Assigner").
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
		oc.DB.Preload("OrderDetails").Preload("Picker").First(&order, order.ID)
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

// GetOrderDetails godoc
// @Summary Get order details
// @Description Get order ID, tracking and all order details of a specific order by ID.
// @Tags orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Order ID"
// @Success 200 {object} utilities.Response{data=OrderDetailsOnlyResponse}
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Failure 404 {object} utilities.Response
// @Router /api/orders/{id}/details [get]
func (oc *OrderController) GetOrderDetails(c *gin.Context) {
	orderID := c.Param("id")
	var order models.Order

	if err := oc.DB.Preload("OrderDetails").First(&order, orderID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utilities.ErrorResponse(c, http.StatusNotFound, "Order not found", "no order found with the specified ID")
			return
		}
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve order", err.Error())
		return
	}

	// Convert order details to response format
	orderDetails := make([]OrderDetailResponse, len(order.OrderDetails))
	for i, detail := range order.OrderDetails {
		orderDetails[i] = OrderDetailResponse{
			ID:          detail.ID,
			Sku:         detail.Sku,
			ProductName: detail.ProductName,
			Variant:     detail.Variant,
			Quantity:    detail.Quantity,
		}
	}

	// Create custom response with only order_ginee_id, tracking, and order details
	response := OrderDetailsOnlyResponse{
		OrderGineeID: order.OrderGineeID,
		Tracking:     order.Tracking,
		OrderDetails: orderDetails,
	}

	utilities.SuccessResponse(c, http.StatusOK, "Order details retrieved successfully", response)
}

// Add these new endpoints after the GetOrderDetails function

// UpdateOrderDetail godoc
// @Summary Update order detail
// @Description Update a specific order detail by ID.
// @Tags orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Order ID"
// @Param detail_id path int true "Order Detail ID"
// @Param request body UpdateOrderDetailRequest true "Update order detail request"
// @Success 200 {object} utilities.Response{data=OrderDetailResponse}
// @Failure 400 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Failure 404 {object} utilities.Response
// @Router /api/orders/{id}/details/{detail_id} [put]
func (oc *OrderController) UpdateOrderDetail(c *gin.Context) {
	orderID := c.Param("id")
	detailID := c.Param("detail_id")

	var req UpdateOrderDetailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utilities.ValidationErrorResponse(c, err)
		return
	}

	// Verify order exists
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
	if order.ProcessingStatus == "picking process" || order.ProcessingStatus != "qc process" {
		utilities.ErrorResponse(c, http.StatusForbidden, "Order modification not allowed", fmt.Sprintf("cannot modify order details when status is '%s'. Order must be in 'ready to pick' status", order.ProcessingStatus))
		return
	}

	// Find and update the order detail
	var orderDetail models.OrderDetail
	if err := oc.DB.Where("id = ? AND order_id = ?", detailID, orderID).First(&orderDetail).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utilities.ErrorResponse(c, http.StatusNotFound, "Order detail not found", "no order detail found with the specified ID for this order")
			return
		}
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to find order detail", err.Error())
		return
	}

	// Update fields
	orderDetail.Sku = req.Sku
	orderDetail.ProductName = req.ProductName
	orderDetail.Variant = req.Variant
	orderDetail.Quantity = req.Quantity

	if err := oc.DB.Save(&orderDetail).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to update order detail", err.Error())
		return
	}

	response := OrderDetailResponse{
		ID:          orderDetail.ID,
		Sku:         orderDetail.Sku,
		ProductName: orderDetail.ProductName,
		Variant:     orderDetail.Variant,
		Quantity:    orderDetail.Quantity,
	}

	utilities.SuccessResponse(c, http.StatusOK, "Order detail updated successfully", response)
}

// AddOrderDetail godoc
// @Summary Add new order detail
// @Description Add a new order detail to an existing order.
// @Tags orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Order ID"
// @Param request body CreateOrderDetailRequest true "Add order detail request"
// @Success 201 {object} utilities.Response{data=OrderDetailResponse}
// @Failure 400 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Failure 404 {object} utilities.Response
// @Router /api/orders/{id}/details [post]
func (oc *OrderController) AddOrderDetail(c *gin.Context) {
	orderID := c.Param("id")

	var req CreateOrderDetailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utilities.ValidationErrorResponse(c, err)
		return
	}

	// Verify order exists
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
		utilities.ErrorResponse(c, http.StatusForbidden, "Order modification not allowed", fmt.Sprintf("cannot add order details when status is '%s'. Order must be in 'ready to pick' status", order.ProcessingStatus))
		return
	}

	// Convert string ID to uint
	orderIDUint, err := strconv.ParseUint(orderID, 10, 32)
	if err != nil {
		utilities.ErrorResponse(c, http.StatusBadRequest, "Invalid order ID", "order ID must be a valid number")
		return
	}

	// Create new order detail
	orderDetail := models.OrderDetail{
		OrderID:     uint(orderIDUint),
		Sku:         req.Sku,
		ProductName: req.ProductName,
		Variant:     req.Variant,
		Quantity:    req.Quantity,
	}

	if err := oc.DB.Create(&orderDetail).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to add order detail", err.Error())
		return
	}

	response := OrderDetailResponse{
		ID:          orderDetail.ID,
		Sku:         orderDetail.Sku,
		ProductName: orderDetail.ProductName,
		Variant:     orderDetail.Variant,
		Quantity:    orderDetail.Quantity,
	}

	utilities.SuccessResponse(c, http.StatusCreated, "Order detail added successfully", response)
}

// RemoveOrderDetail godoc
// @Summary Remove order detail
// @Description Remove a specific order detail from an order.
// @Tags orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Order ID"
// @Param detail_id path int true "Order Detail ID"
// @Success 200 {object} utilities.Response
// @Failure 400 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Failure 404 {object} utilities.Response
// @Router /api/orders/{id}/details/{detail_id} [delete]
func (oc *OrderController) RemoveOrderDetail(c *gin.Context) {
	orderID := c.Param("id")
	detailID := c.Param("detail_id")

	// Verify order exists
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
		utilities.ErrorResponse(c, http.StatusForbidden, "Order modification not allowed", fmt.Sprintf("cannot remove order details when status is '%s'. Order must be in 'ready to pick' status", order.ProcessingStatus))
		return
	}

	// Check if this is the last order detail
	var detailCount int64
	oc.DB.Model(&models.OrderDetail{}).Where("order_id = ?", orderID).Count(&detailCount)
	if detailCount <= 1 {
		utilities.ErrorResponse(c, http.StatusBadRequest, "Cannot remove order detail", "order must have at least one order detail")
		return
	}

	// Find and delete the order detail
	var orderDetail models.OrderDetail
	if err := oc.DB.Where("id = ? AND order_id = ?", detailID, orderID).First(&orderDetail).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utilities.ErrorResponse(c, http.StatusNotFound, "Order detail not found", "no order detail found with the specified ID for this order")
			return
		}
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to find order detail", err.Error())
		return
	}

	if err := oc.DB.Delete(&orderDetail).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to remove order detail", err.Error())
		return
	}

	utilities.SuccessResponse(c, http.StatusOK, "Order detail removed successfully", nil)
}

// Add this struct after the existing structs
type UpdateOrderDetailRequest struct {
	Sku         string `json:"sku" binding:"required" example:"PROD001"`
	ProductName string `json:"product_name" binding:"required" example:"Updated Product"`
	Variant     string `json:"variant" example:"Blue - Size L"`
	Quantity    int    `json:"quantity" binding:"required,min=1" example:"3"`
}

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

type OrderDetailResponse struct {
	ID          uint   `json:"id"`
	Sku         string `json:"sku"`
	ProductName string `json:"product_name"`
	Variant     string `json:"variant"`
	Quantity    int    `json:"quantity"`
}

type OrderDetailsOnlyResponse struct {
	OrderGineeID string                `json:"order_ginee_id"`
	Tracking     string                `json:"tracking"`
	SentBefore   time.Time             `json:"sent_before"`
	OrderDetails []OrderDetailResponse `json:"order_details"`
}
