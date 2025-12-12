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

type ComplainController struct {
	DB *gorm.DB
}

// NewComplainController creates a new complain controller
func NewComplainController(db *gorm.DB) *ComplainController {
	return &ComplainController{DB: db}
}

// GetComplains godoc
// @Summary Get all complains
// @Description Get list of all complains with optional date range filtering and search.
// @Tags complains
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Param start_date query string false "Start date (YYYY-MM-DD format)"
// @Param end_date query string false "End date (YYYY-MM-DD format)"
// @Param search query string false "Search by complain code, tracking, order_ginee_id (partial match)"
// @Success 200 {object} utilities.Response{data=ComplainsListResponse}
// @Failure 400 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Router /api/complains [get]
func (cc *ComplainController) GetComplains(c *gin.Context) {
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	// Parse date range parameters
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	// Parse search parameter
	search := c.Query("search")

	var complains []models.Complain
	var total int64

	// Build query with optional search
	query := cc.DB.Model(&models.Complain{})

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
		// Search by complain code with partial match
		query = query.Where("code ILIKE ? OR tracking ILIKE ? OR order_ginee_id ILIKE ?", "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	// Get total count with search filter
	if err := query.Count(&total).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to count complains", err.Error())
		return
	}

	// ADDED: Preload relationships for complete data
	if err := query.
		Preload("ProductDetails.Product").
		Preload("UserDetails.Operator.UserRoles.Role").
		Preload("UserDetails.Operator.UserRoles.Assigner").
		Preload("Channel").
		Preload("Store").
		Preload("Creator.UserRoles.Role").
		Preload("Creator.UserRoles.Assigner").
		Order("id DESC").
		Limit(limit).
		Offset(offset).
		Find(&complains).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch complains", err.Error())
		return
	}

	// ADDED: Load order data for each complain
	for i := range complains {
		if complains[i].Tracking != "" {
			var order models.Order
			if err := cc.DB.Preload("OrderDetails").
				Preload("PickOperator.UserRoles.Role").
				Preload("PickOperator.UserRoles.Assigner").
				Where("tracking = ?", complains[i].Tracking).First(&order).Error; err == nil {
				complains[i].Order = &order
			}

			// Load return data if tracking exists in old_tracking
			var returnData models.Return
			if err := cc.DB.Preload("ReturnDetails.Product").
				Preload("Channel").
				Preload("Store").
				Preload("CreateOperator").
				Preload("UpdateOperator").
				Where("old_tracking = ?", complains[i].Tracking).First(&returnData).Error; err == nil {
				complains[i].Return = &returnData
			}
		}
	}

	// Convert to response format
	complainResponse := make([]models.ComplainResponse, len(complains))
	for i, complain := range complains {
		complainResponse[i] = complain.ToComplainResponse()
	}

	response := ComplainsListResponse{
		Complains: complainResponse,
		Pagination: utilities.PaginationResponse{
			Page:  page,
			Limit: limit,
			Total: int(total),
		},
	}

	// Build success message
	message := "Complains retrieved successfully"
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

// GetComplain godoc
// @Summary Get complain by ID
// @Description Get complain details by ID.
// @Tags complains
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Complain ID"
// @Success 200 {object} utilities.Response{data=models.ComplainResponse}
// @Failure 400 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Router /api/complains/{id} [get]
func (cc *ComplainController) GetComplain(c *gin.Context) {
	complainID := c.Param("id")

	var complain models.Complain
	if err := cc.DB.Preload("ProductDetails.Product").
		Preload("UserDetails.Operator.UserRoles.Role").
		Preload("UserDetails.Operator.UserRoles.Assigner").
		Preload("Channel").
		Preload("Store").
		Preload("Creator.UserRoles.Role").
		Preload("Creator.UserRoles.Assigner").
		First(&complain, complainID).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusNotFound, "Complain not found", err.Error())
		return
	}

	// Load order data if tracking exists
	if complain.Tracking != "" {
		var order models.Order
		if err := cc.DB.Preload("OrderDetails").
			Preload("PickOperator.UserRoles.Role").
			Preload("PickOperator.UserRoles.Assigner").
			Where("tracking = ?", complain.Tracking).First(&order).Error; err == nil {
			complain.Order = &order
		}

		// Load return data if tracking exists in old_tracking
		var returnData models.Return
		if err := cc.DB.Preload("ReturnDetails.Product").
			Preload("Channel").
			Preload("Store").
			Preload("CreateOperator").
			Preload("UpdateOperator").
			Where("old_tracking = ?", complain.Tracking).First(&returnData).Error; err == nil {
			complain.Return = &returnData
		}
	}

	utilities.SuccessResponse(c, http.StatusOK, "Complain retrieved successfully", complain.ToComplainResponse())
}

// CreateComplain godoc
// @Summary Create a new complain
// @Description Create a new complain with automatic product and user details population.
// @Tags complains
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param complain body CreateComplainRequest true "Create complain request"
// @Success 201 {object} utilities.Response{data=models.ComplainResponse}
// @Failure 400 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Router /api/complains [post]
func (cc *ComplainController) CreateComplain(c *gin.Context) {
	// Get user ID from JWT token
	userID, exists := c.Get("user_id")
	if !exists {
		utilities.ErrorResponse(c, http.StatusUnauthorized, "Unauthorized", "User not authenticated")
		return
	}

	// Get username from JWT token
	username, exists := c.Get("username")
	if !exists {
		utilities.ErrorResponse(c, http.StatusUnauthorized, "Unauthorized", "Username not found in token")
		return
	}

	var req CreateComplainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utilities.ValidationErrorResponse(c, err)
		return
	}

	// Check for duplicate tracking
	var existingComplain models.Complain
	if err := cc.DB.Where("tracking = ?", req.Tracking).First(&existingComplain).Error; err == nil {
		utilities.ErrorResponse(c, http.StatusBadRequest, "Complain tracking already exists", "A complain with this tracking already exists")
		return
	}

	// Start database transaction
	tx := cc.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Find order by tracking to get OrderGineeID and populate product details
	var order models.Order
	if err := tx.Preload("OrderDetails").Where("tracking = ?", req.Tracking).First(&order).Error; err != nil {
		tx.Rollback()
		utilities.ErrorResponse(c, http.StatusNotFound, "Order not found", "No order found with the specified tracking number")
		return
	}

	// Generate complain code with username
	complainCode := utilities.GenerateComplainCode(cc.DB, username.(string))

	complain := models.Complain{
		Code:         complainCode,
		Tracking:     req.Tracking,
		OrderGineeID: order.OrderGineeID, // ADDED: Fill OrderGineeID from order
		ChannelID:    req.ChannelID,
		StoreID:      req.StoreID,
		Description:  req.Description,
		CreatedBy:    userID.(uint),
	}

	// Create the complain
	if err := tx.Create(&complain).Error; err != nil {
		tx.Rollback()
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to create complain", err.Error())
		return
	}

	// Populate product details from order details
	for _, orderDetail := range order.OrderDetails {
		// Find product by SKU
		var product models.Product
		if err := tx.Where("sku = ?", orderDetail.Sku).First(&product).Error; err == nil {
			productDetail := models.ComplainProductDetail{
				ComplainID: complain.ID,
				ProductID:  product.ID,
				Quantity:   orderDetail.Quantity,
			}

			if err := tx.Create(&productDetail).Error; err != nil {
				tx.Rollback()
				utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to create product detail", err.Error())
				return
			}
		}
	}

	// Populate user details from workflow tables
	operatorIDs := make(map[uint]bool) // To avoid duplicate users

	// 1. Check QC-Ribbon
	var qcRibbon models.QcRibbon
	if err := tx.Where("tracking = ?", req.Tracking).First(&qcRibbon).Error; err == nil && qcRibbon.QcBy != nil {
		operatorIDs[*qcRibbon.QcBy] = true
	}

	// 2. Check QC-Online
	var qcOnline models.QcOnline
	if err := tx.Where("tracking = ?", req.Tracking).First(&qcOnline).Error; err == nil && qcOnline.QcBy != nil {
		operatorIDs[*qcOnline.QcBy] = true
	}

	// 3. Check Outbound
	var outbound models.Outbound
	if err := tx.Where("tracking = ?", req.Tracking).First(&outbound).Error; err == nil && outbound.OutboundBy != nil {
		operatorIDs[*outbound.OutboundBy] = true
	}

	// 4. Check Order
	var orderOperator models.Order
	if err := tx.Where("tracking = ?", req.Tracking).First(&orderOperator).Error; err == nil && orderOperator.PickedBy != nil {
		operatorIDs[*orderOperator.PickedBy] = true
	}

	// Create user details for each unique user found
	for operatorIDValue := range operatorIDs {
		userDetail := models.ComplainUserDetail{
			ComplainID: complain.ID,
			OperatorID: operatorIDValue,
			FeeCharge:  0, // Default fee, can be updated later
		}

		if err := tx.Create(&userDetail).Error; err != nil {
			tx.Rollback()
			utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to create user detail", err.Error())
			return
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to commit transaction", err.Error())
		return
	}

	// Load the created complain with all relationships for complete response
	cc.DB.Preload("ProductDetails.Product").
		Preload("UserDetails.Operator.UserRoles.Role").
		Preload("UserDetails.Operator.UserRoles.Assigner").
		Preload("Channel").
		Preload("Store").
		Preload("Creator.UserRoles.Role").
		Preload("Creator.UserRoles.Assigner").
		First(&complain, complain.ID)

	// Load order data (already found above)
	complain.Order = &order

	utilities.SuccessResponse(c, http.StatusCreated, "Complain created successfully", complain.ToComplainResponse())
}

// UpdateSolutionComplain godoc
// @Summary Update complain solution and user details
// @Description Update complain solution, total fee, and manage user details (logged-in users only)
// @Tags complains
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Complain ID"
// @Param request body UpdateSolutionComplainRequest true "Update Solution Complain Request"
// @Success 200 {object} utilities.Response{data=models.ComplainResponse}
// @Failure 400 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Failure 404 {object} utilities.Response
// @Router /api/complains/{id}/solution [put]
func (cc *ComplainController) UpdateSolutionComplain(c *gin.Context) {
	complainID := c.Param("id")

	var req UpdateSolutionComplainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utilities.ValidationErrorResponse(c, err)
		return
	}

	var complain models.Complain
	if err := cc.DB.First(&complain, complainID).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusNotFound, "Complain not found", err.Error())
		return
	}

	// Start database transaction
	tx := cc.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Update complain solution and total fee
	complain.Solution = req.Solution
	complain.TotalFee = req.TotalFee

	if err := tx.Save(&complain).Error; err != nil {
		tx.Rollback()
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to update complain", err.Error())
		return
	}

	// Handle user details updates
	if len(req.UserDetails) > 0 {
		// Clear existing user details
		if err := tx.Where("complain_id = ?", complain.ID).Delete(&models.ComplainUserDetail{}).Error; err != nil {
			tx.Rollback()
			utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to clear existing user details", err.Error())
			return
		}

		// Create new user details
		for _, userDetailReq := range req.UserDetails {
			// Validate user exists
			var user models.User
			if err := tx.First(&user, userDetailReq.OperatorID).Error; err != nil {
				tx.Rollback()
				utilities.ErrorResponse(c, http.StatusBadRequest, "User not found", "User with ID "+strconv.Itoa(int(userDetailReq.OperatorID))+" not found")
				return
			}

			userDetail := models.ComplainUserDetail{
				ComplainID: complain.ID,
				OperatorID: userDetailReq.OperatorID,
				FeeCharge:  userDetailReq.FeeCharge,
			}

			if err := tx.Create(&userDetail).Error; err != nil {
				tx.Rollback()
				utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to create user detail", err.Error())
				return
			}
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to commit transaction", err.Error())
		return
	}

	// Load updated complain with all relationships
	cc.DB.Preload("ProductDetails.Product").
		Preload("UserDetails.Operator.UserRoles.Role").
		Preload("UserDetails.Operator.UserRoles.Assigner").
		Preload("Channel").
		Preload("Store").
		Preload("Creator.UserRoles.Role").
		Preload("Creator.UserRoles.Assigner").
		First(&complain, complain.ID)

	// Load order data if tracking exists
	if complain.Tracking != "" {
		var order models.Order
		if err := cc.DB.Preload("OrderDetails").
			Preload("PickOperator.UserRoles.Role").
			Preload("PickOperator.UserRoles.Assigner").
			Where("tracking = ?", complain.Tracking).First(&order).Error; err == nil {
			complain.Order = &order
		}
	}

	utilities.SuccessResponse(c, http.StatusOK, "Complain solution updated successfully", complain.ToComplainResponse())
}

// UpdateCheckComplain godoc
// @Summary Update complain check status
// @Description Update complain checked status (logged-in users only)
// @Tags complains
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Complain ID"
// @Param request body UpdateCheckComplainRequest true "Update Check Complain Request"
// @Success 200 {object} utilities.Response{data=models.ComplainResponse}
// @Failure 400 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Failure 404 {object} utilities.Response
// @Router /api/complains/{id}/check [put]
func (cc *ComplainController) UpdateCheckComplain(c *gin.Context) {
	complainID := c.Param("id")

	var req UpdateCheckComplainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utilities.ValidationErrorResponse(c, err)
		return
	}

	var complain models.Complain
	if err := cc.DB.First(&complain, complainID).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusNotFound, "Complain not found", err.Error())
		return
	}

	if req.Checked == nil {
		utilities.ErrorResponse(c, http.StatusBadRequest, "Validation failed", "checked field is required")
		return
	}

	// Update complain checked status
	complain.Checked = *req.Checked

	if err := cc.DB.Save(&complain).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to update complain check status", err.Error())
		return
	}

	// Load updated complain with all relationships
	cc.DB.Preload("ProductDetails.Product").
		Preload("UserDetails.Operator.UserRoles.Role").
		Preload("UserDetails.Operator.UserRoles.Assigner").
		Preload("Channel").
		Preload("Store").
		Preload("Creator.UserRoles.Role").
		Preload("Creator.UserRoles.Assigner").
		First(&complain, complain.ID)

	// Load order data if tracking exists
	if complain.Tracking != "" {
		var order models.Order
		if err := cc.DB.Preload("OrderDetails").
			Preload("PickOperator.UserRoles.Role").
			Preload("PickOperator.UserRoles.Assigner").
			Where("tracking = ?", complain.Tracking).First(&order).Error; err == nil {
			complain.Order = &order
		}
	}

	utilities.SuccessResponse(c, http.StatusOK, "Complain check status updated successfully", complain.ToComplainResponse())
}

// Request/Response structs
type ComplainsListResponse struct {
	Complains  []models.ComplainResponse    `json:"complains"`
	Pagination utilities.PaginationResponse `json:"pagination"`
}

type CreateComplainRequest struct {
	Tracking    string `json:"tracking" binding:"required"`
	ChannelID   uint   `json:"channel_id" binding:"required"`
	StoreID     uint   `json:"store_id" binding:"required"`
	Description string `json:"description" binding:"required"`
}

type UpdateSolutionComplainRequest struct {
	Solution    string                      `json:"solution" binding:"required" example:"Replacement package sent"`
	TotalFee    uint                        `json:"total_fee" binding:"required" example:"50000"`
	UserDetails []ComplainUserDetailRequest `json:"user_details" binding:"required,dive,required"`
}

type ComplainUserDetailRequest struct {
	OperatorID uint `json:"operator_id" binding:"required" example:"1"`
	FeeCharge  uint `json:"fee_charge" binding:"required" example:"10000"`
}

type UpdateCheckComplainRequest struct {
	Checked *bool `json:"checked" binding:"required"`
}
