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

type PickedOrderController struct {
	DB *gorm.DB
}

// NewPickedOrderController creates a new PickedOrderController
func NewPickedOrderController(db *gorm.DB) *PickedOrderController {
	return &PickedOrderController{DB: db}
}

// GetPickedOrders godoc
// @Summary Get all Picked Orders
// @Description Get a list of all picked orders with their details and search.
// @Tags Picked-Orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(10)
// @Param start_date query string false "Start date (YYYY-MM-DD format)"
// @Param end_date query string false "End date (YYYY-MM-DD format)"
// @Param search query string false "Search by Picker name, Order Ginee ID, or Tracking (partial match)"
// @Success 200 {object} utilities.Response{data=PickedOrdersListResponse}
// @Failure 400 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Router /api/picked-orders [get]
func (poc *PickedOrderController) GetPickedOrders(c *gin.Context) {
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	offset := (page - 1) * limit

	// Parse date range parameters
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	// Parse search parameter
	search := c.Query("search")

	var pickedOrders []models.PickedOrder
	var total int64

	// Build query with optional search
	query := poc.DB.Model(&models.PickedOrder{})

	// Apply date range filters if provided
	if startDate != "" {
		// Parse start date and set time to beginning of day
		parsedStartDate, err := time.Parse("2006-01-02", startDate)
		if err != nil {
			utilities.ErrorResponse(c, http.StatusBadRequest, "Invalid start_date format", "start_date must be in YYYY-MM-DD format")
			return
		}
		startOfDay := parsedStartDate.Format("2006-01-02 00:00:00")
		query = query.Where("picked_orders.created_at >= ?", startOfDay)
	}

	if endDate != "" {
		// Parse end date and set time to end of day
		parsedEndDate, err := time.Parse("2006-01-02", endDate)
		if err != nil {
			utilities.ErrorResponse(c, http.StatusBadRequest, "Invalid end_date format", "end_date must be in YYYY-MM-DD format")
			return
		}
		// Add 24 hours to get the start of next day, then use < instead of <=
		nextDay := parsedEndDate.AddDate(0, 0, 1).Format("2006-01-02 00:00:00")
		query = query.Where("picked_orders.created_at < ?", nextDay)
	}

	if search != "" {
		// Search by picker name, order ginee ID, or tracking with partial match
		query = query.Joins("LEFT JOIN users ON users.id = picked_orders.picker_id AND users.deleted_at IS NULL").
			Joins("LEFT JOIN orders ON orders.id = picked_orders.order_id AND orders.deleted_at IS NULL").
			Where("users.full_name ILIKE ? OR orders.order_ginee_id ILIKE ? OR orders.tracking ILIKE ?",
				"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	// Get total count with search filter
	if err := query.Count(&total).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to count pick orders", err.Error())
		return
	}

	// Get pick orders with pagination, search filter, and order by ID desc
	if err := query.Preload("PickedOrderDetails").
		Preload("Picker.UserRoles.Role").
		Preload("Picker.UserRoles.Assigner").
		Preload("Order.OrderDetails").
		Preload("Order.Picker.UserRoles.Role").
		Preload("Order.Picker.UserRoles.Assigner").
		Order("picked_orders.id DESC").
		Limit(limit).
		Offset(offset).
		Find(&pickedOrders).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve pick orders", err.Error())
		return
	}

	// Load products for each pick order
	for i := range pickedOrders {
		if err := pickedOrders[i].LoadProducts(poc.DB); err != nil {
			utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to load products for pick order details", err.Error())
			return
		}
	}

	// Convert to response format
	pickedOrderResponses := make([]models.PickedOrderResponse, len(pickedOrders))
	for i, pickedOrder := range pickedOrders {
		pickedOrderResponses[i] = pickedOrder.ToPickedOrderResponse()
	}

	response := PickedOrdersListResponse{
		PickedOrders: pickedOrderResponses,
		Pagination: utilities.PaginationResponse{
			Page:  page,
			Limit: limit,
			Total: int(total),
		},
	}

	// Build success message
	message := "Picked orders retrieved successfully"
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

// GetPickedOrder godoc
// @Summary Get a picked order by ID
// @Description Get a picked order by ID.
// @Tags Picked-Orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Pick order ID"
// @Success 200 {object} utilities.Response{data=models.PickedOrderResponse}
// @Failure 400 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Failure 404 {object} utilities.Response
// @Router /api/picked-orders/{id} [get]
func (poc *PickedOrderController) GetPickedOrder(c *gin.Context) {
	pickedOrderId := c.Param("id")

	var pickedOrder models.PickedOrder
	if err := poc.DB.Preload("PickedOrderDetails").
		Preload("Picker.UserRoles.Role").
		Preload("Picker.UserRoles.Assigner").
		Preload("Order.OrderDetails").
		Preload("Order.Picker.UserRoles.Role").
		Preload("Order.Picker.UserRoles.Assigner").
		First(&pickedOrder, pickedOrderId).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utilities.ErrorResponse(c, http.StatusNotFound, "Picked order not found", err.Error())
			return
		}
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve picked order", err.Error())
		return
	}

	// Load products for pick order details
	if err := pickedOrder.LoadProducts(poc.DB); err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to load products for picked order details", err.Error())
		return
	}

	utilities.SuccessResponse(c, http.StatusOK, "Picked order retrieved successfully", pickedOrder.ToPickedOrderResponse())
}

// Request/Response structs
type PickedOrdersListResponse struct {
	PickedOrders []models.PickedOrderResponse `json:"picked_orders"`
	Pagination   utilities.PaginationResponse `json:"pagination"`
}
