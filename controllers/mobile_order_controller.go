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

type MobileOrderController struct {
	DB *gorm.DB
}

// NewMobileOrderController creates a new mobile order controller
func NewMobileOrderController(db *gorm.DB) *MobileOrderController {
	return &MobileOrderController{DB: db}
}

// GetMobileOrders godoc
// @Summary Get all orders for pickers with search capability
// @Description Get list of all orders with "ready to pick" status, Optional search by order ID or tracking number.
// @Tags orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Param search query string false "Search by order Ginee ID or tracking number"
// @Success 200 {object} utilities.Response{data=MobileOrdersListResponse}
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Router /api/mobile/orders [get]
func (moc *MobileOrderController) GetMobileOrders(c *gin.Context) {
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	// Parse search parameter
	search := strings.TrimSpace(c.Query("search"))

	var orders []models.Order
	var total int64

	// Build base query for "ready to pick" orders
	query := moc.DB.Model(&models.Order{}).Where("status IN = ?", []string{"ready to pick", "pending picking"})

	// Add search conditions if search parameter is provided
	if search != "" {
		searchCondition := "order_ginee_id ILIKE ? OR tracking ILIKE ?"
		searchPattern := "%" + search + "%"
		query = query.Where(searchCondition, searchPattern, searchPattern)
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to count orders", err.Error())
		return
	}

	// Get orders with pagination, sorted by ID ascending
	if err := query.Order("id ASC").
		Limit(limit).Offset(offset).
		Preload("OrderDetails").
		Find(&orders).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve orders", err.Error())
		return
	}

	// Convert to response format with product location and barcode
	orderResponses := make([]MobileOrderListResponse, len(orders))
	for i, order := range orders {
		// Get product details for each order detail
		var orderDetailsWithProduct []MobileOrderDetailWithProduct
		for _, detail := range order.OrderDetails {
			var product models.Product

			// Find product by SKU
			if err := moc.DB.Where("sku = ?", detail.Sku).First(&product).Error; err != nil {
				// If product not found, use placeholder values
				orderDetailsWithProduct = append(orderDetailsWithProduct, MobileOrderDetailWithProduct{
					OrderDetailResponse: models.OrderDetailResponse{
						ID:          detail.ID,
						Sku:         detail.Sku,
						ProductName: detail.ProductName,
						Variant:     detail.Variant,
						Quantity:    detail.Quantity,
					},
					Location: "Location not found",
					Barcode:  "Barcode not found",
				})
			} else {
				// Product found, include location and barcode
				orderDetailsWithProduct = append(orderDetailsWithProduct, MobileOrderDetailWithProduct{
					OrderDetailResponse: models.OrderDetailResponse{
						ID:          detail.ID,
						Sku:         detail.Sku,
						ProductName: detail.ProductName,
						Variant:     detail.Variant,
						Quantity:    detail.Quantity,
					},
					Location: product.Location,
					Barcode:  product.Barcode,
				})
			}
		}

		orderResponses[i] = MobileOrderListResponse{
			ID:           order.ID,
			OrderGineeID: order.OrderGineeID,
			Status:       order.Status,
			Channel:      order.Channel,
			Store:        order.Store,
			Buyer:        order.Buyer,
			Address:      order.Address,
			Courier:      order.Courier,
			Tracking:     order.Tracking,
			SentBefore:   order.SentBefore.Format("2006-01-02 15:04:05"),
			PickedBy:     order.Picker.FullName,
			PickedAt:     order.PickedAt.Format("2006-01-02 15:04:05"),
			PendingBy:    order.PendingOperator.FullName,
			PendingAt:    order.PendingAt.Format("2006-01-02 15:04:05"),
			CancelledBy:  order.Canceller.FullName,
			CancelledAt:  order.CancelledAt.Format("2006-01-02 15:04:05"),
			CreatedAt:    order.CreatedAt,
			UpdatedAt:    order.UpdatedAt,
			OrderDetails: orderDetailsWithProduct,
		}
	}

	response := MobileOrdersListResponse{
		Orders: orderResponses,
		Pagination: utilities.PaginationResponse{
			Page:  page,
			Limit: limit,
			Total: int(total),
		},
	}

	utilities.SuccessResponse(c, http.StatusOK, "Orders retrieved successfully", response)
}

// GetMyPickingOrders godoc
// @Summary Get my ongoing picking orders
// @Description Get list of orders currently being picked by the logged-in user (status: "picking process")
// @Tags orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} utilities.Response{data=[]models.OrderResponse}
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Router /api/mobile/orders/my-picking [get]
func (moc *MobileOrderController) GetMyPickingOrders(c *gin.Context) {
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

	var orders []models.Order

	// Get orders currently being picked by this user
	if err := moc.DB.Where("picker_id = ? AND status = ?", userID, "picking process").
		Order("id ASC").
		Preload("OrderDetails").
		Preload("Picker").
		Find(&orders).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve picking orders", err.Error())
		return
	}

	// Convert to response format
	orderResponses := make([]models.OrderResponse, len(orders))
	for i, order := range orders {
		orderResponses[i] = order.ToOrderResponse()
	}

	message := fmt.Sprintf("Found %d order(s) currently being picked by you", len(orders))
	utilities.SuccessResponse(c, http.StatusOK, message, orderResponses)
}

// PickingOrder godoc
// @Summary Pick an order for processing
// @Description Change order status from "ready to pick" to "picking process" and assign to current picker
// @Tags orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Order ID"
// @Success 200 {object} utilities.Response{data=models.OrderResponse}
// @Failure 400 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Failure 404 {object} utilities.Response
// @Router /api/mobile/orders/{id}/pick [put]
func (moc *MobileOrderController) PickingOrder(c *gin.Context) {
	// Get order ID from URL parameter
	orderID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utilities.ErrorResponse(c, http.StatusBadRequest, "Invalid order ID", err.Error())
		return
	}

	// Get current user ID from context (set by auth middleware)
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

	var order models.Order
	// Find order and check if it's available to pick
	if err := moc.DB.Where("id = ? AND status IN = ?", orderID, []string{"ready to pick", "pending picking"}).First(&order).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utilities.ErrorResponse(c, http.StatusNotFound, "Order not found or not available for picking", "order not found or already picked")
		} else {
			utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to find order", err.Error())
		}
		return
	}

	// Update order status and assign picker
	now := time.Now()
	order.Status = "picking process"
	order.PickedBy = &userID
	order.PickedAt = &now

	// Save the changes
	if err := moc.DB.Save(&order).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to update order", err.Error())
		return
	}

	// Load order with details and picker for response
	moc.DB.Preload("OrderDetails").Preload("Picker").First(&order, order.ID)

	utilities.SuccessResponse(c, http.StatusOK, "Order picked successfully", order.ToOrderResponse())
}

// GetMobileOrder godoc
// @Summary Get order details with product information
// @Description Get specific order details with product location and barcode joined by SKU.
// @Tags mobile-orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Order ID"
// @Success 200 {object} utils.Response{data=MobileOrderDetailResponse}
// @Failure 401 {object} utils.Response
// @Failure 403 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Router /api/mobile/orders/{id} [get]
func (moc *MobileOrderController) GetMobileOrder(c *gin.Context) {
	// Get order ID from URL parameter
	orderID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utilities.ErrorResponse(c, http.StatusBadRequest, "Invalid order ID", err.Error())
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

	var order models.Order
	// Find order assigned to current picker
	if err := moc.DB.Where("id = ? AND picker_id = ?", orderID, userID).
		Preload("OrderDetails").
		Preload("Picker").
		First(&order).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utilities.ErrorResponse(c, http.StatusNotFound, "Order not found", "order not found or not assigned to you")
		} else {
			utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to find order", err.Error())
		}
		return
	}

	// Get product details for each order detail
	var orderDetailsWithProduct []MobileOrderDetailWithProduct
	for _, detail := range order.OrderDetails {
		var product models.Product

		// Find product by SKU
		if err := moc.DB.Where("sku = ?", detail.Sku).First(&product).Error; err != nil {
			// If product not found, use empty location and barcode
			orderDetailsWithProduct = append(orderDetailsWithProduct, MobileOrderDetailWithProduct{
				OrderDetailResponse: models.OrderDetailResponse{
					ID:          detail.ID,
					Sku:         detail.Sku,
					ProductName: detail.ProductName,
					Variant:     detail.Variant,
					Quantity:    detail.Quantity,
				},
				Location: "Location not found",
				Barcode:  "Barcode not found",
			})
		} else {
			// Product found, include location and barcode
			orderDetailsWithProduct = append(orderDetailsWithProduct, MobileOrderDetailWithProduct{
				OrderDetailResponse: models.OrderDetailResponse{
					ID:          detail.ID,
					Sku:         detail.Sku,
					ProductName: detail.ProductName,
					Variant:     detail.Variant,
					Quantity:    detail.Quantity,
				},
				Location: product.Location,
				Barcode:  product.Barcode,
			})
		}
	}

	response := MobileOrderDetailResponse{
		ID:           order.ID,
		OrderGineeID: order.OrderGineeID,
		Status:       order.Status,
		Channel:      order.Channel,
		Store:        order.Store,
		Buyer:        order.Buyer,
		Address:      order.Address,
		Courier:      order.Courier,
		Tracking:     order.Tracking,
		SentBefore:   order.SentBefore.Format("2006-01-02 15:04:05"),
		PickedBy:     order.Picker.FullName,
		PickedAt:     order.PickedAt.Format("2006-01-02 15:04:05"),
		PendingBy:    order.PendingOperator.FullName,
		PendingAt:    order.PendingAt.Format("2006-01-02 15:04:05"),
		CancelledBy:  order.Canceller.FullName,
		CancelledAt:  order.CancelledAt.Format("2006-01-02 15:04:05"),
		CreatedAt:    order.CreatedAt,
		UpdatedAt:    order.UpdatedAt,
		OrderDetails: orderDetailsWithProduct,
	}

	utilities.SuccessResponse(c, http.StatusOK, "Order details retrieved successfully", response)
}

// CompletePickingOrder godoc
// @Summary Complete picking process
// @Description Change order status from "picking process" to "picking complete" and create pick order records
// @Tags mobile-orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Order ID"
// @Success 200 {object} utils.Response{data=models.OrderResponse}
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 403 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Router /api/mobile/orders/{id}/complete [put]
func (moc *MobileOrderController) CompletePickingOrder(c *gin.Context) {
	// Get order ID from URL parameter
	orderID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utilities.ErrorResponse(c, http.StatusBadRequest, "Invalid order ID", err.Error())
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

	// Start database transaction
	tx := moc.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var order models.Order
	// Find order assigned to current picker with "picking process" status
	if err := tx.Preload("OrderDetails").Where("id = ? AND picker_id = ? AND status = ?", orderID, userID, "picking process").First(&order).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			utilities.ErrorResponse(c, http.StatusNotFound, "Order not found or not in picking process", "order not found or not in picking process")
		} else {
			utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to find order", err.Error())
		}
		return
	}

	// Create PickOrder record
	pickOrder := models.PickedOrder{
		OrderGineeID: order.ID,
		PickedBy:     userID,
	}

	if err := tx.Create(&pickOrder).Error; err != nil {
		tx.Rollback()
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to create pick order", err.Error())
		return
	}

	// Create PickOrderDetail records from OrderDetails
	for _, orderDetail := range order.OrderDetails {
		pickOrderDetail := models.PickedOrderDetail{
			PickedOrderID: pickOrder.ID,
			Sku:           orderDetail.Sku,
			ProductName:   orderDetail.ProductName,
			Variant:       orderDetail.Variant,
			Quantity:      orderDetail.Quantity,
		}

		if err := tx.Create(&pickOrderDetail).Error; err != nil {
			tx.Rollback()
			utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to create pick order detail", err.Error())
			return
		}
	}

	// Update order status to complete
	order.Status = "picking complete"

	// Save the changes
	if err := tx.Save(&order).Error; err != nil {
		tx.Rollback()
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to complete order", err.Error())
		return
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to commit transaction", err.Error())
		return
	}

	// Load order with details and picker for response
	moc.DB.Preload("OrderDetails").Preload("Picker").First(&order, order.ID)

	utilities.SuccessResponse(c, http.StatusOK, "Order picking completed successfully and pick order records created", order.ToOrderResponse())
}

// PendingPickingOrder godoc
// @Summary Pending picking process
// @Description Change order status from "picking process" to "pending picking" and unassign picker
// @Tags orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Order ID"
// @Success 200 {object} utilities.Response{data=models.OrderResponse}
// @Failure 400 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Failure 404 {object} utilities.Response
// @Router /api/mobile/orders/{id}/pending [put]
func (moc *MobileOrderController) PendingPickingOrder(c *gin.Context) {
	// Get order ID from URL parameter
	orderID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utilities.ErrorResponse(c, http.StatusBadRequest, "Invalid order ID", err.Error())
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

	var order models.Order
	// Find order assigned to current picker with "picking process" status
	if err := moc.DB.Preload("OrderDetails").Where("id = ? AND picker_id = ? AND status = ?", orderID, userID, "picking process").First(&order).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utilities.ErrorResponse(c, http.StatusNotFound, "Order not found or not in picking process", "order not found or not in picking process")
		} else {
			utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to find order", err.Error())
		}
		return
	}

	// Update order status to "pending picking" and unassign picker
	if err := moc.DB.Model(&order).Preload("OrderDetails").Select("status", "picker_id", "picked_at").Updates(models.Order{
		Status:   "pending picking",
		PickedBy: nil,
		PickedAt: nil,
	}).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to pending picking order", err.Error())
		return
	}

	utilities.SuccessResponse(c, http.StatusOK, "Pending picking order successfully", order.ToOrderResponse())
}

// Response struct for mobile endpoints
type MobileOrderDetailResponse struct {
	ID           uint                           `json:"id"`
	OrderGineeID string                         `json:"order_ginee_id"`
	Status       string                         `json:"status"`
	Channel      string                         `json:"channel"`
	Store        string                         `json:"store"`
	Buyer        string                         `json:"buyer"`
	Address      string                         `json:"address"`
	Courier      string                         `json:"courier"`
	Tracking     string                         `json:"tracking"`
	SentBefore   string                         `json:"sent_before"`
	PickedBy     string                         `json:"picked_by"`
	PickedAt     string                         `json:"picked_at"`
	PendingBy    string                         `json:"pending_by"`
	PendingAt    string                         `json:"pending_at"`
	CancelledBy  string                         `json:"cancelled_by"`
	CancelledAt  string                         `json:"cancelled_at"`
	CreatedAt    time.Time                      `json:"created_at"`
	UpdatedAt    time.Time                      `json:"updated_at"`
	OrderDetails []MobileOrderDetailWithProduct `json:"order_details"`
}

type MobileOrderListResponse struct {
	ID           uint                           `json:"id"`
	OrderGineeID string                         `json:"order_ginee_id"`
	Status       string                         `json:"status"`
	Channel      string                         `json:"channel"`
	Store        string                         `json:"store"`
	Buyer        string                         `json:"buyer"`
	Address      string                         `json:"address"`
	Courier      string                         `json:"courier"`
	Tracking     string                         `json:"tracking"`
	SentBefore   string                         `json:"sent_before"`
	PickedBy     string                         `json:"picked_by"`
	PickedAt     string                         `json:"picked_at"`
	PendingBy    string                         `json:"pending_by"`
	PendingAt    string                         `json:"pending_at"`
	CancelledBy  string                         `json:"cancelled_by"`
	CancelledAt  string                         `json:"cancelled_at"`
	CreatedAt    time.Time                      `json:"created_at"`
	UpdatedAt    time.Time                      `json:"updated_at"`
	OrderDetails []MobileOrderDetailWithProduct `json:"order_details"`
}

type MobileOrdersListResponse struct {
	Orders     []MobileOrderListResponse    `json:"orders"`
	Pagination utilities.PaginationResponse `json:"pagination"`
}

type MobileOrderDetailWithProduct struct {
	models.OrderDetailResponse
	Location string `json:"location"`
	Barcode  string `json:"barcode"`
}
