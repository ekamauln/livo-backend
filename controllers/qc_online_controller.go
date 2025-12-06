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

type QcOnlineController struct {
	DB *gorm.DB
}

// NewQcOnlineController creates a new qc-online controller
func NewQcOnlineController(db *gorm.DB) *QcOnlineController {
	return &QcOnlineController{DB: db}
}

// GetQcOnlines godoc
// @Summary Get all qc-onlines for logged-in user
// @Description Get list of qc-onlines for current user filtered by current date.
// @Tags onlines
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Param search query string false "Search by tracking number"
// @Success 200 {object} utilities.Response{data=QcOnlinesListResponse}
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Router /api/onlines/qc-onlines [get]
func (qoc *QcOnlineController) GetQcOnlines(c *gin.Context) {
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

	// Parse search parameter (optional)
	search := c.Query("search")

	var qcOnlines []models.QcOnline
	var total int64

	// Build query with filters
	query := qoc.DB.Model(&models.QcOnline{}).Where("user_id = ?", userID).Where("DATE(created_at) = CURRENT_DATE")

	if search != "" {
		// Search by tracking with partial match
		query = query.Where("tracking ILIKE ?", "%"+search+"%")
	}

	// Get total count with filters
	if err := query.Count(&total).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to count qc-onlines", err.Error())
		return
	}

	// Get qc-onlines with pagination, filters, and preload relationships
	if err := query.Order("id DESC").
		Preload("QcOnlineDetails.Box").
		Preload("QcOperator.UserRoles.Role").
		Preload("QcOperator.UserRoles.Assigner").
		Limit(limit).Offset(offset).Find(&qcOnlines).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve qc-onlines", err.Error())
		return
	}

	// Manually Load all orders at once using tracking field
	if len(qcOnlines) > 0 {
		trackingNumbers := make([]string, 0, len(qcOnlines))
		for _, qcOnline := range qcOnlines {
			if qcOnline.Tracking != "" {
				trackingNumbers = append(trackingNumbers, qcOnline.Tracking)
			}
		}

		var orders []models.Order
		orderMap := make(map[string]*models.Order)

		if len(trackingNumbers) > 0 {
			if err := qoc.DB.Where("tracking IN ?", trackingNumbers).
				Preload("OrderDetails").
				Preload("Picker.UserRoles.Role").
				Preload("Picker.UserRoles.Assigner").
				Find(&orders).Error; err == nil {

				for i := range orders {
					orderMap[orders[i].Tracking] = &orders[i]
				}
			}
		}

		for i := range qcOnlines {
			if order, exists := orderMap[qcOnlines[i].Tracking]; exists {
				qcOnlines[i].Order = order
			}
		}
	}

	// Convert to response format
	qcOnlineResponses := make([]models.QcOnlineResponse, len(qcOnlines))
	for i, qcOnline := range qcOnlines {
		qcOnlineResponses[i] = qcOnline.ToQcOnlineResponse()
	}

	response := QcOnlinesListResponse{
		QcOnlines: qcOnlineResponses,
		Pagination: utilities.PaginationResponse{
			Page:  page,
			Limit: limit,
			Total: int(total),
		},
	}

	// Build success message
	message := "Qc-onlines retrieved successfully"
	if search != "" {
		message += " (filtered by tracking: " + search + ")"
	}

	utilities.SuccessResponse(c, http.StatusOK, message, response)
}

// GetQcOnline godoc
// @Summary Get a specific qc-online by ID
// @Description Get  a specific qc-online by ID.
// @Tags onlines
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "QcOnline ID"
// @Success 200 {object} utilities.Response{data=models.QcOnlineResponse}
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Failure 404 {object} utilities.Response
// @Router /api/onlines/qc-onlines/{id} [get]
func (qoc *QcOnlineController) GetQcOnline(c *gin.Context) {
	qcOnlineID := c.Param("id")

	var qcOnline models.QcOnline

	if err := qoc.DB.Preload("QcOnlineDetails.Box").
		Preload("QcOperator.UserRoles.Role").
		Preload("QcOperator.UserRoles.Assigner").
		First(&qcOnline, qcOnlineID).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusNotFound, "Qc-online not found", err.Error())
		return
	}

	// Load order call from the model's LoadOrder method
	if qcOnline.Tracking != "" {
		qcOnline.LoadOrder(qoc.DB)
	}

	utilities.SuccessResponse(c, http.StatusOK, "Qc-online retrieved successfully", qcOnline.ToQcOnlineResponse())
}

// CreateQcOnline godoc
// @Summary Create a new qc-online
// @Description Create new qc-online entry with multiple box details.
// @Tags onlines
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateQcOnlineRequest true "Create qc-online request"
// @Success 201 {object} utilities.Response{data=models.QcOnlineResponse}
// @Failure 400 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Failure 404 {object} utilities.Response
// @Router /api/onlines/qc-onlines [post]
func (qoc *QcOnlineController) CreateQcOnline(c *gin.Context) {
	// Get user ID from JWT token
	userID, exists := c.Get("user_id")
	if !exists {
		utilities.ErrorResponse(c, http.StatusUnauthorized, "Unauthorized", "User not authenticated")
		return
	}

	var req CreateQcOnlineRequest
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

	// Check if tracking already exists in qc_onlines table
	var existingQcOnline models.QcOnline
	if err := qoc.DB.Where("tracking = ?", req.Tracking).First(&existingQcOnline).Error; err == nil {
		utilities.ErrorResponse(c, http.StatusBadRequest, "QC Online with this tracking already exists", "Duplicate tracking")
		return
	} else if err != gorm.ErrRecordNotFound {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to validate tracking", err.Error())
		return
	}

	// Check if tracking exists in orders table
	var order models.Order
	if err := qoc.DB.Where("tracking = ?", req.Tracking).First(&order).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utilities.ErrorResponse(c, http.StatusNotFound, "Order not found", "No order found with the specified tracking number. Please create Order first.")
			return
		}
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to validate tracking in MB Online", err.Error())
		return
	}

	// Validate all boxes exist and no duplicates
	boxIDs := make(map[uint]bool)
	for _, detail := range req.Details {
		// Check for duplicate box IDs in request
		if boxIDs[detail.BoxID] {
			utilities.ErrorResponse(c, http.StatusBadRequest, "Duplicate box ID", "Each box can only be added once per QC online")
			return
		}
		boxIDs[detail.BoxID] = true

		// Check if box exists
		var box models.Box
		if err := qoc.DB.First(&box, detail.BoxID).Error; err != nil {
			utilities.ErrorResponse(c, http.StatusBadRequest, "Box not found", "Invalid box ID: "+strconv.Itoa(int(detail.BoxID)))
			return
		}

		// Validate quantity
		if detail.Quantity <= 0 {
			utilities.ErrorResponse(c, http.StatusBadRequest, "Invalid quantity", "Quantity must be greater than 0")
			return
		}
	}

	// Start database transaction
	tx := qoc.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Create QC Online
	qcOnline := models.QcOnline{
		Tracking: req.Tracking,
		QcBy:     &userIDUint,
	}

	if err := tx.Create(&qcOnline).Error; err != nil {
		tx.Rollback()
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to create qc-online", err.Error())
		return
	}

	// Create QC Online Detail for each box
	for _, detail := range req.Details {
		qcOnlineDetail := models.QcOnlineDetail{
			QcOnlineID: qcOnline.ID,
			BoxID:      detail.BoxID,
			Quantity:   detail.Quantity,
		}

		if err := tx.Create(&qcOnlineDetail).Error; err != nil {
			tx.Rollback()
			utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to create qc-online detail", err.Error())
			return
		}
	}

	// Update order processing_status to "qc complete"
	if err := tx.Model(&models.Order{}).Where("tracking = ?", req.Tracking).Update("processing_status", "qc complete").Error; err != nil {
		tx.Rollback()
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to update order status", err.Error())
		return
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to commit transaction", err.Error())
		return
	}

	// Load the created qc-online with relationships
	qoc.DB.Preload("QcOnlineDetails.Box").
		Preload("QcOperator.UserRoles.Role").
		Preload("QcOperator.UserRoles.Assigner").
		First(&qcOnline, qcOnline.ID)

	// Load order call from the model's LoadOrder method
	if qcOnline.Tracking != "" {
		qcOnline.LoadOrder(qoc.DB)
	}

	utilities.SuccessResponse(c, http.StatusCreated, "Qc-online created successfully", qcOnline.ToQcOnlineResponse())
}

// GetChartQcOnlines godoc
// @Summary Get qc-online counts per day for current month
// @Description Get daily count of qc-onlines for current month (for chart data).
// @Tags onlines
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} utilities.Response{data=QcOnlinesDailyCountResponse}
// @Failure 400 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Router /api/onlines/qc-onlines/chart [get]
func (qoc *QcOnlineController) GetChartQcOnlines(c *gin.Context) {
	// Get current month start and end dates
	now := time.Now()
	currentYear, currentMonth, _ := now.Date()
	currentLocation := now.Location()

	// First day of current month at 00:00:00
	firstOfMonth := time.Date(currentYear, currentMonth, 1, 0, 0, 0, 0, currentLocation)

	// First day of next month at 00:00:00 (to use as upper bound)
	firstOfNextMonth := firstOfMonth.AddDate(0, 1, 0)

	// Query to get daily counts for current month
	var dailyCounts []QcOnlineDailyCount

	if err := qoc.DB.Model(&models.QcOnline{}).
		Select("DATE(created_at) as date, COUNT(*) as count").
		Where("created_at >= ?", firstOfMonth).
		Where("created_at < ?", firstOfNextMonth).
		Group("DATE(created_at)").
		Order("date ASC").
		Scan(&dailyCounts).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve qc-online counts", err.Error())
		return
	}

	// Get total count for current month
	var totalCount int64
	if err := qoc.DB.Model(&models.QcOnline{}).
		Where("created_at >= ?", firstOfMonth).
		Where("created_at < ?", firstOfNextMonth).
		Count(&totalCount).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to count qc-onlines", err.Error())
		return
	}

	response := QcOnlinesDailyCountResponse{
		Month:       currentMonth.String(),
		Year:        currentYear,
		DailyCounts: dailyCounts,
		TotalCount:  int(totalCount),
	}

	message := "Qc-online daily counts for " + currentMonth.String() + " " + strconv.Itoa(currentYear) + " retrieved successfully"

	utilities.SuccessResponse(c, http.StatusOK, message, response)
}

// Request/Response structs
type QcOnlinesListResponse struct {
	QcOnlines  []models.QcOnlineResponse    `json:"qc_onlines"`
	Pagination utilities.PaginationResponse `json:"pagination"`
}

type QcOnlineDetailRequest struct {
	BoxID    uint `json:"box_id" binding:"required" example:"1"`
	Quantity int  `json:"quantity" binding:"required,min=1" example:"5"`
}

type CreateQcOnlineRequest struct {
	Tracking string                  `json:"tracking" binding:"required" example:"TRK123456"`
	Details  []QcOnlineDetailRequest `json:"details" binding:"required,dive,required"`
}

// QcOnlineDailyCount represents the count of qc-onlines for a specific date
type QcOnlineDailyCount struct {
	Date  string `json:"date"` // Format: YYYY-MM-DD
	Count int    `json:"count"`
}

// QcOnlinesDailyCountResponse represents the response for daily qc-online counts
type QcOnlinesDailyCountResponse struct {
	Month       string               `json:"month"` // e.g., "November"
	Year        int                  `json:"year"`  // e.g., 2025
	DailyCounts []QcOnlineDailyCount `json:"daily_counts"`
	TotalCount  int                  `json:"total_count"` // Total for the month
}
