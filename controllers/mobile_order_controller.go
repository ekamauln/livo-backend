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
