package controllers

import (
	"livo-backend/models"
	"livo-backend/utilities"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type LostFoundController struct {
	DB *gorm.DB
}

// NewLostFoundController creates a new lost and found controller
func NewLostFoundController(db *gorm.DB) *LostFoundController {
	return &LostFoundController{DB: db}
}

// GetLostFounds godoc
// @Summary Get all lost and found items
// @Description Get list of all lost and found items.
// @Tags lost-founds
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Param search query string false "Search by product sku or reason (partial match)"
// @Success 200 {object} utilities.Response{data=LostFoundsListResponse}
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Router /api/lost-founds [get]
func (lfc *LostFoundController) GetLostFounds(c *gin.Context) {
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	// Parse search parameter
	search := c.Query("search")

	var lostFounds []models.LostFound
	var total int64

	// Build query with optional search
	query := lfc.DB.Model(&models.LostFound{}).
		Preload("Product").
		Preload("CreateOperator.UserRoles.Role").
		Preload("CreateOperator.UserRoles.Assigner")

	if search != "" {
		// Search by product sku or reason with partial match
		query = query.Where("product_sku ILIKE ? OR reason ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// Get total count with search filter
	if err := query.Count(&total).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to count lost and found items", err.Error())
		return
	}

	// Get lost founds with pagination, search filter, and order by id descending
	if err := query.Order("id DESC").Limit(limit).Offset(offset).Find(&lostFounds).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve lost and found items", err.Error())
		return
	}

	// Convert to response format
	lostFoundsResponse := make([]models.LostFoundResponse, len(lostFounds))
	for i, lf := range lostFounds {
		lostFoundsResponse[i] = lf.ToLostFoundResponse()
	}

	response := LostFoundsListResponse{
		LostFounds: lostFoundsResponse,
		Pagination: utilities.PaginationResponse{
			Page:  page,
			Limit: limit,
			Total: int(total),
		},
	}

	// Build success message
	message := "Lost and found items retrieved successfully"
	if search != "" {
		message = "Lost and found items retrieved successfully with search filter"
	}

	utilities.SuccessResponse(c, http.StatusOK, message, response)
}

// GetLostFound godoc
// @Summary Get a lost and found item by ID
// @Description Get lost and found item details by ID.
// @Tags lost-founds
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Lost and Found ID"
// @Success 200 {object} utilities.Response{data=models.LostFoundResponse}
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Failure 404 {object} utilities.Response
// @Router /api/lost-founds/{id} [get]
func (lfc *LostFoundController) GetLostFound(c *gin.Context) {
	lostFoundID := c.Param("id")

	var lostFound models.LostFound
	if err := lfc.DB.Preload("Product").
		Preload("CreateOperator.UserRoles.Role").
		Preload("CreateOperator.UserRoles.Assigner").
		First(&lostFound, lostFoundID).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusNotFound, "Lost and found item not found", err.Error())
		return
	}

	utilities.SuccessResponse(c, http.StatusOK, "Lost and found item retrieved successfully", lostFound.ToLostFoundResponse())
}

// UpdateLostFound godoc
// @Summary Update a lost and found item by ID
// @Description Update lost and found item details by ID.
// @Tags lost-founds
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Lost and Found ID"
// @Param lost_found body UpdateLostFoundRequest true "Lost and Found data"
// @Success 200 {object} utilities.Response{data=models.LostFoundResponse}
// @Failure 400 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Failure 404 {object} utilities.Response
// @Router /api/lost-founds/{id} [put]
func (lfc *LostFoundController) UpdateLostFound(c *gin.Context) {
	lostFoundID := c.Param("id")

	var req UpdateLostFoundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utilities.ValidationErrorResponse(c, err)
		return
	}

	var lostFound models.LostFound
	if err := lfc.DB.First(&lostFound, lostFoundID).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusNotFound, "Lost and found item not found", err.Error())
		return
	}

	// Update lost and found fields
	lostFound.ProductSKU = req.ProductSKU
	lostFound.Quantity = req.Quantity
	lostFound.Reason = req.Reason

	if err := lfc.DB.Save(&lostFound).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to update lost and found item", err.Error())
		return
	}

	// Reload with relationships
	if err := lfc.DB.Preload("Product").
		Preload("CreateOperator.UserRoles.Role").
		Preload("CreateOperator.UserRoles.Assigner").
		First(&lostFound, lostFoundID).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to reload lost and found item", err.Error())
		return
	}

	utilities.SuccessResponse(c, http.StatusOK, "Lost and found item updated successfully", lostFound.ToLostFoundResponse())
}

// RemoveLostFound godoc
// @Summary Remove a lost and found item by ID
// @Description Soft delete a lost and found item by ID.
// @Tags lost-founds
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Lost and Found ID"
// @Success 200 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Failure 404 {object} utilities.Response
// @Router /api/lost-founds/{id} [delete]
func (lfc *LostFoundController) RemoveLostFound(c *gin.Context) {
	lostFoundID := c.Param("id")

	var lostFound models.LostFound
	if err := lfc.DB.First(&lostFound, lostFoundID).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusNotFound, "Lost and found item not found", err.Error())
		return
	}

	if err := lfc.DB.Delete(&lostFound).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to remove lost and found item", err.Error())
		return
	}

	utilities.SuccessResponse(c, http.StatusOK, "Lost and found item removed successfully", nil)
}

// CreateLostFound godoc
// @Summary Create new lost and found item
// @Description Create a new lost and found item.
// @Tags lost-founds
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateLostFoundRequest true "Create lost and found request"
// @Success 201 {object} utilities.Response{data=models.LostFoundResponse}
// @Failure 400 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Router /api/lost-founds [post]
func (lfc *LostFoundController) CreateLostFound(c *gin.Context) {
	// Get user ID from JWT token
	userID, exists := c.Get("user_id")
	if !exists {
		utilities.ErrorResponse(c, http.StatusUnauthorized, "Unauthorized", "User ID not found in token")
		return
	}

	var req CreateLostFoundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utilities.ValidationErrorResponse(c, err)
		return
	}

	// Convert userID to uint
	createdBy, ok := userID.(uint)
	if !ok {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Invalid user ID", "Failed to convert user ID to uint")
		return
	}

	// Create lost and found record
	lostFound := models.LostFound{
		ProductSKU: req.ProductSKU,
		Quantity:   req.Quantity,
		Reason:     req.Reason,
		CreatedBy:  &createdBy,
	}

	if err := lfc.DB.Create(&lostFound).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to create lost and found item", err.Error())
		return
	}

	// Reload with relationships
	if err := lfc.DB.Preload("Product").
		Preload("CreateOperator.UserRoles.Role").
		Preload("CreateOperator.UserRoles.Assigner").
		First(&lostFound, lostFound.ID).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to reload lost and found item", err.Error())
		return
	}

	utilities.SuccessResponse(c, http.StatusCreated, "Lost and found item created successfully", lostFound.ToLostFoundResponse())
}

// Responses/Request structs
type LostFoundsListResponse struct {
	LostFounds []models.LostFoundResponse   `json:"lost_founds"`
	Pagination utilities.PaginationResponse `json:"pagination"`
}

type UpdateLostFoundRequest struct {
	ProductSKU string `json:"product_sku" binding:"required"`
	Quantity   int    `json:"quantity" binding:"required,min=1"`
	Reason     string `json:"reason" binding:"required"`
}

type CreateLostFoundRequest struct {
	ProductSKU string `json:"product_sku" binding:"required"`
	Quantity   int    `json:"quantity" binding:"required,min=1"`
	Reason     string `json:"reason" binding:"required"`
}
