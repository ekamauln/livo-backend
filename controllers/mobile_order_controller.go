package controllers

import (
	"fmt"
	"livo-backend/models"
	"livo-backend/utilities"
	"net/http"
	"strconv"
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

// GetMyPickingOrders godoc
// @Summary Get my ongoing picking orders by mobile
// @Description Get list of orders currently being picked by the logged-in user (processing status: "picking process")
// @Tags mobile-orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} utilities.Response{data=[]models.OrderResponse}
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Router /api/mobile/orders [get]
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
	if err := moc.DB.Where("picked_by = ? AND processing_status = ?", userID, "picking process").
		Order("id ASC").
		Preload("OrderDetails").
		Preload("PickOperator").
		Preload("AssignOperator").
		Preload("PendingOperator").
		Preload("ChangeOperator").
		Preload("CancelOperator").
		Find(&orders).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve picking orders", err.Error())
		return
	}

	// Manually fetch and attach products to order details, then sort by location
	for i := range orders {
		// First, attach products to order details
		for j := range orders[i].OrderDetails {
			var product models.Product
			if err := moc.DB.Where("sku = ?", orders[i].OrderDetails[j].Sku).First(&product).Error; err == nil {
				orders[i].OrderDetails[j].Product = &product
			}
		}

		// Sort order details by product location
		// Using a simple bubble sort to keep it readable
		for j := 0; j < len(orders[i].OrderDetails)-1; j++ {
			for k := j + 1; k < len(orders[i].OrderDetails); k++ {
				locationJ := ""
				locationK := ""

				if orders[i].OrderDetails[j].Product != nil {
					locationJ = orders[i].OrderDetails[j].Product.Location
				}
				if orders[i].OrderDetails[k].Product != nil {
					locationK = orders[i].OrderDetails[k].Product.Location
				}

				// Sort alphabetically by location
				if locationJ > locationK {
					orders[i].OrderDetails[j], orders[i].OrderDetails[k] = orders[i].OrderDetails[k], orders[i].OrderDetails[j]
				}
			}
		}
	}

	// Convert to response format
	orderResponses := make([]models.OrderResponse, len(orders))
	for i, order := range orders {
		orderResponses[i] = order.ToOrderResponse()
	}

	message := fmt.Sprintf("Found %d order(s) currently being picked for you", len(orders))

	utilities.SuccessResponse(c, http.StatusOK, message, orderResponses)
}

// GetMyPickingOrder godoc
// @Summary Get my ongoing picking order by mobile
// @Description Get the order currently being picked by the logged-in user (processing status: "picking process")
// @Tags mobile-orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Order ID"
// @Success 200 {object} utilities.Response{data=models.OrderResponse}
// @Failure 400 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Failure 404 {object} utilities.Response
// @Router /api/mobile/orders/{id} [get]
func (moc *MobileOrderController) GetMyPickingOrder(c *gin.Context) {
	orderID := c.Param("id")
	var order models.Order

	if err := moc.DB.Preload("OrderDetails").
		Preload("PickOperator").
		Preload("AssignOperator").
		Preload("PendingOperator").
		Preload("ChangeOperator").
		Preload("CancelOperator").
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
		if err := moc.DB.Where("sku = ?", order.OrderDetails[i].Sku).First(&product).Error; err == nil {
			order.OrderDetails[i].Product = &product
		}
	}

	utilities.SuccessResponse(c, http.StatusOK, "Order retrieved successfully", order.ToOrderResponse())
}

// CompletePickingOrder godoc
// @Summary Complete picking process by mobile
// @Description Change order processing status from "picking process" to "picking complete" and create pick order records
// @Tags mobile-orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Order ID"
// @Success 200 {object} utilities.Response{data=models.OrderResponse}
// @Failure 400 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Failure 404 {object} utilities.Response
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
	// Find order assigned to current picker with "picking process" processing status
	if err := tx.Preload("OrderDetails").Where("id = ? AND picked_by = ? AND processing_status = ?", orderID, userID, "picking process").First(&order).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			utilities.ErrorResponse(c, http.StatusNotFound, "Order not found or not in picking process", "order not found or not in picking process")
		} else {
			utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to find order", err.Error())
		}
		return
	}

	// Update order processing status and set picked_at timestamp
	now := time.Now()
	order.ProcessingStatus = "picking complete"
	order.PickedAt = &now

	// Create PickedOrder record
	pickedOrder := models.PickedOrder{
		OrderID:  order.ID,
		PickedBy: userID,
	}

	if err := tx.Create(&pickedOrder).Error; err != nil {
		tx.Rollback()
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to create picked order record", err.Error())
		return
	}

	// Save the order changes
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
	moc.DB.Preload("OrderDetails").
		Preload("PickOperator").
		Preload("AssignOperator").
		Preload("PendingOperator").
		Preload("ChangeOperator").
		Preload("CancelOperator").
		First(&order, order.ID)

	// Manually fetch and attach products to order details
	for i := range order.OrderDetails {
		var product models.Product
		if err := moc.DB.Where("sku = ?", order.OrderDetails[i].Sku).First(&product).Error; err == nil {
			order.OrderDetails[i].Product = &product
		}
	}

	utilities.SuccessResponse(c, http.StatusOK, "Order picking completed successfully and pick order records created", order.ToOrderResponse())
}

// PendingPickOrders godoc
// @Summary Get orders pending pick assignment
// @Description Pending order that already assigned to a picker, but not picked yet. Requires coordinator username and password.
// @Tags mobile-orders
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
// @Router /api/mobile/orders/{id}/pending-pick [put]
func (moc *MobileOrderController) PendingPickOrders(c *gin.Context) {
	orderID := c.Param("id")

	var req PendingPickRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utilities.ValidationErrorResponse(c, err)
		return
	}

	// Verify coordinator credentials from request body
	var coordinator models.User
	if err := moc.DB.Preload("UserRoles.Role").Where("username = ?", req.Username).First(&coordinator).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusUnauthorized, "Invalid coordinator credentials", "coordinator user not found")
		return
	}

	// Check password
	if !utilities.CheckPasswordHash(req.Password, coordinator.Password) {
		utilities.ErrorResponse(c, http.StatusUnauthorized, "Invalid coordinator credentials", "incorrect password")
		return
	}

	// Check if user has coordinator role
	hasCoordinatorRole := false
	for _, userRole := range coordinator.UserRoles {
		if userRole.Role.Name == "coordinator" || userRole.Role.Name == "superadmin" {
			hasCoordinatorRole = true
			break
		}
	}

	if !hasCoordinatorRole {
		utilities.ErrorResponse(c, http.StatusForbidden, "Insufficient permissions", "user does not have coordinator role")
		return
	}

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
	if err := moc.DB.First(&order, orderID).Error; err != nil {
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

	if err := moc.DB.Save(&order).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to set order to pending pick", err.Error())
		return
	}

	// Reload order with all relationships
	if err := moc.DB.
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
		if err := moc.DB.Where("sku = ?", order.OrderDetails[i].Sku).First(&product).Error; err == nil {
			order.OrderDetails[i].Product = &product
		}
	}

	utilities.SuccessResponse(c, http.StatusOK, "Order set to pending pick successfully", order.ToOrderResponse())
}

// BulkAssignPicker godoc
// @Summary Bulk assign a picker to multiple orders by mobile
// @Description Assign a picker to multiple orders by scanning tracking numbers, setting assigned_by to current user, assigned_at to now, picked_by to specified picker, and processing_status to "picking process"
// @Tags mobile-orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body MobileBulkAssignPickerRequest true "Bulk assign picker request"
// @Success 200 {object} utilities.Response{data=MobileBulkAssignPickerResponse}
// @Failure 400 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Router /api/mobile/orders/bulk-assign-picker [post]
func (moc *MobileOrderController) BulkAssignPicker(c *gin.Context) {
	var req MobileBulkAssignPickerRequest
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
	if err := moc.DB.First(&picker, req.PickerID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utilities.ErrorResponse(c, http.StatusNotFound, "Picker not found", "no user found with the specified picker ID")
			return
		}
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to find picker", err.Error())
		return
	}

	var assignedOrders []models.Order
	var skippedOrders []SkippedAssignment
	var failedOrders []FailedAssignment

	now := time.Now()

	// Process each tracking number
	for i, tracking := range req.Trackings {
		var order models.Order

		// Find order by tracking number
		if err := moc.DB.Where("tracking = ?", tracking).First(&order).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				skippedOrders = append(skippedOrders, SkippedAssignment{
					Index:    i,
					Tracking: tracking,
					Reason:   "Order not found",
				})
			} else {
				failedOrders = append(failedOrders, FailedAssignment{
					Index:    i,
					Tracking: tracking,
					Error:    err.Error(),
				})
			}
			continue
		}

		// Validate order status
		if order.EventStatus != nil && *order.EventStatus == "cancelled" {
			skippedOrders = append(skippedOrders, SkippedAssignment{
				Index:    i,
				Tracking: tracking,
				Reason:   "Order is cancelled",
			})
			continue
		}

		if order.ProcessingStatus == "picking process" {
			skippedOrders = append(skippedOrders, SkippedAssignment{
				Index:    i,
				Tracking: tracking,
				Reason:   "Order already in picking process",
			})
			continue
		}

		if order.ProcessingStatus == "qc process" || order.ProcessingStatus == "completed" {
			skippedOrders = append(skippedOrders, SkippedAssignment{
				Index:    i,
				Tracking: tracking,
				Reason:   fmt.Sprintf("Cannot assign when status is '%s'", order.ProcessingStatus),
			})
			continue
		}

		// Update order with assignment details
		order.AssignedBy = &userID
		order.AssignedAt = &now
		order.PickedBy = &req.PickerID
		order.ProcessingStatus = "picking process"

		if err := moc.DB.Save(&order).Error; err != nil {
			failedOrders = append(failedOrders, FailedAssignment{
				Index:    i,
				Tracking: tracking,
				Error:    err.Error(),
			})
			continue
		}

		// Load order with relationships
		moc.DB.Preload("OrderDetails").
			Preload("PickOperator").
			Preload("AssignOperator").
			First(&order, order.ID)

		// Manually fetch and attach products
		for j := range order.OrderDetails {
			var product models.Product
			if err := moc.DB.Where("sku = ?", order.OrderDetails[j].Sku).First(&product).Error; err == nil {
				order.OrderDetails[j].Product = &product
			}
		}

		assignedOrders = append(assignedOrders, order)
	}

	// Convert assigned orders to response format
	assignedOrderResponses := make([]models.OrderResponse, len(assignedOrders))
	for i, order := range assignedOrders {
		assignedOrderResponses[i] = order.ToOrderResponse()
	}

	response := MobileBulkAssignPickerResponse{
		Summary: BulkAssignSummary{
			Total:    len(req.Trackings),
			Assigned: len(assignedOrders),
			Skipped:  len(skippedOrders),
			Failed:   len(failedOrders),
		},
		AssignedOrders: assignedOrderResponses,
		SkippedOrders:  skippedOrders,
		FailedOrders:   failedOrders,
	}

	// Determine response status and message
	statusCode := http.StatusOK
	message := "Bulk picker assignment completed"

	if len(assignedOrders) == 0 {
		if len(skippedOrders) > 0 {
			message = "All orders were skipped"
		} else {
			statusCode = http.StatusBadRequest
			message = "No orders could be assigned"
		}
	} else if len(failedOrders) > 0 || len(skippedOrders) > 0 {
		message = "Bulk picker assignment completed with some issues"
	} else {
		message = fmt.Sprintf("Successfully assigned %d order(s) to picker", len(assignedOrders))
	}

	utilities.SuccessResponse(c, statusCode, message, response)
}

// GetMobilePickedOrders godoc
// @Summary Get picked orders for coordinator by mobile
// @Description Get list of picked orders with pagination, search filter, filtered by processing status "picking process" and current date
// @Tags mobile-orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Number of items per page" default(10)
// @Param search query string false "Search term to filter by order ginee ID or tracking number"
// @Success 200 {object} utilities.Response{data=MobileOrdersListResponse}
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Router /api/mobile/orders/picked-orders [get]
func (moc *MobileOrderController) GetMobilePickedOrders(c *gin.Context) {
	// Pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	// Parse search parameter
	search := c.Query("search")

	var orders []models.Order
	var total int64

	// Build query with filters
	query := moc.DB.Model(&models.Order{}).Where("processing_status = ?", "picking process").Where("DATE(assigned_at) = ?", time.Now().Format("2006-01-02")) // Filter by current date

	// Apply search filter if provided
	if search != "" {
		query = query.Where("order_ginee_id LIKE ? OR tracking LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// Get total count with all filters
	if err := query.Count(&total).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to count picked orders", err.Error())
		return
	}

	// Get orders with pagination, filters, sorted by assigned_at descending
	if err := query.Order("assigned_at DESC").Limit(limit).Offset(offset).
		Preload("OrderDetails").
		Preload("PickOperator").
		Preload("AssignOperator").
		Preload("PendingOperator").
		Preload("ChangeOperator").
		Preload("CancelOperator").
		Find(&orders).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve picked orders", err.Error())
		return
	}

	// After loading orders, manually fetch and attach products
	for i := range orders {
		for j := range orders[i].OrderDetails {
			var product models.Product
			if err := moc.DB.Where("sku = ?", orders[i].OrderDetails[j].Sku).First(&product).Error; err == nil {
				orders[i].OrderDetails[j].Product = &product
			}
		}
	}

	// Convert to response format
	orderResponses := make([]models.OrderResponse, len(orders))
	for i, order := range orders {
		orderResponses[i] = order.ToOrderResponse()
	}

	message := fmt.Sprintf("Found %d picked order(s)", len(orders))

	utilities.SuccessResponse(c, http.StatusOK, message, orderResponses)
}

// Response struct by mobile endpoints
type MobileOrderDetailResponse struct {
	ID               uint                           `json:"id"`
	OrderGineeID     string                         `json:"order_ginee_id"`
	ProcessingStatus string                         `json:"processing_status"`
	EventStatus      *string                        `json:"event_status"`
	Channel          string                         `json:"channel"`
	Store            string                         `json:"store"`
	Buyer            string                         `json:"buyer"`
	Address          string                         `json:"address"`
	Courier          string                         `json:"courier"`
	Tracking         string                         `json:"tracking"`
	SentBefore       string                         `json:"sent_before"`
	PickedBy         string                         `json:"picked_by"`
	PickedAt         string                         `json:"picked_at"`
	PendingBy        string                         `json:"pending_by"`
	PendingAt        string                         `json:"pending_at"`
	ChangedBy        string                         `json:"changed_by"`
	ChangedAt        string                         `json:"changed_at"`
	CancelledBy      string                         `json:"cancelled_by"`
	CancelledAt      string                         `json:"cancelled_at"`
	CreatedAt        time.Time                      `json:"created_at"`
	UpdatedAt        time.Time                      `json:"updated_at"`
	OrderDetails     []MobileOrderDetailWithProduct `json:"order_details"`
}

type MobileOrderListResponse struct {
	ID               uint                           `json:"id"`
	OrderGineeID     string                         `json:"order_ginee_id"`
	ProcessingStatus string                         `json:"processing_status"`
	EventStatus      *string                        `json:"event_status"`
	Channel          string                         `json:"channel"`
	Store            string                         `json:"store"`
	Buyer            string                         `json:"buyer"`
	Address          string                         `json:"address"`
	Courier          string                         `json:"courier"`
	Tracking         string                         `json:"tracking"`
	SentBefore       string                         `json:"sent_before"`
	PickedBy         string                         `json:"picked_by"`
	PickedAt         string                         `json:"picked_at"`
	PendingBy        string                         `json:"pending_by"`
	PendingAt        string                         `json:"pending_at"`
	ChangedBy        string                         `json:"changed_by"`
	ChangedAt        string                         `json:"changed_at"`
	CancelledBy      string                         `json:"cancelled_by"`
	CancelledAt      string                         `json:"cancelled_at"`
	CreatedAt        time.Time                      `json:"created_at"`
	UpdatedAt        time.Time                      `json:"updated_at"`
	OrderDetails     []MobileOrderDetailWithProduct `json:"order_details"`
}

type MobileOrdersListResponse struct {
	Orders     []MobileOrderListResponse    `json:"orders"`
	Pagination utilities.PaginationResponse `json:"pagination"`
}

type MobileOrderDetailWithProduct struct {
	models.OrderDetailResponse
	Image    string `json:"image"`
	Location string `json:"location"`
	Barcode  string `json:"barcode"`
}

type PendingPickRequest struct {
	Username string `json:"username" binding:"required" example:"coordinator_user"`
	Password string `json:"password" binding:"required" example:"coordinator_password"`
}

type MobileBulkAssignPickerRequest struct {
	PickerID  uint     `json:"picker_id" binding:"required" example:"1"`
	Trackings []string `json:"trackings" binding:"required,min=1" example:"JNE1234567890,JNE0987654321"`
}

type MobileBulkAssignPickerResponse struct {
	Summary        BulkAssignSummary      `json:"summary"`
	AssignedOrders []models.OrderResponse `json:"assigned_orders"`
	SkippedOrders  []SkippedAssignment    `json:"skipped_orders"`
	FailedOrders   []FailedAssignment     `json:"failed_orders"`
}

type BulkAssignSummary struct {
	Total    int `json:"total"`
	Assigned int `json:"assigned"`
	Skipped  int `json:"skipped"`
	Failed   int `json:"failed"`
}

type SkippedAssignment struct {
	Index    int    `json:"index"`
	Tracking string `json:"tracking"`
	Reason   string `json:"reason"`
}

type FailedAssignment struct {
	Index    int    `json:"index"`
	Tracking string `json:"tracking"`
	Error    string `json:"error"`
}
