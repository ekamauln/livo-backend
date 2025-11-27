package controllers

import (
	"livo-backend/models"
	"livo-backend/utilities"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type BoxController struct {
	DB *gorm.DB
}

// NewBoxController creates a new box controller
func NewBoxController(db *gorm.DB) *BoxController {
	return &BoxController{DB: db}
}

// GetBoxes godoc
// @Summary Get all boxes
// @Description Get list of all boxes (logged-in users only)
// @Tags boxes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Param search query string false "Search by box code (partial match)"
// @Success 200 {object} utilities.Response{data=BoxesListResponse}
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Router /api/boxes [get]
func (bc *BoxController) GetBoxes(c *gin.Context) {
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	// Parse search parameter
	search := c.Query("search")

	var boxes []models.Box
	var total int64

	// Build query with optional search
	query := bc.DB.Model(&models.Box{})

	if search != "" {
		// Search by box code with partial match
		query = query.Where("code ILIKE ?", "%"+search+"%")
	}

	// Get total count with search filter
	if err := query.Count(&total).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to count boxes", err.Error())
		return
	}

	// Get boxes with pagination, search filter, and order by ID ascending
	if err := query.Order("id ASC").Limit(limit).Offset(offset).Find(&boxes).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve boxes", err.Error())
		return
	}

	// Convert to response format
	boxResponses := make([]models.BoxResponse, len(boxes))
	for i, box := range boxes {
		boxResponses[i] = box.ToBoxResponse()
	}

	response := BoxesListResponse{
		Boxes: boxResponses,
		Pagination: utilities.PaginationResponse{
			Page:  page,
			Limit: limit,
			Total: int(total),
		},
	}

	// Build success message
	message := "Boxes retrieved successfully"
	if search != "" {
		message += " (filtered by code: " + search + ")"
	}

	utilities.SuccessResponse(c, http.StatusOK, message, response)
}

// GetBox godoc
// @Summary Get box by ID
// @Description Get box details by ID (logged-in users only)
// @Tags boxes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Box ID"
// @Success 200 {object} utilities.Response{data=models.BoxResponse}
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Failure 404 {object} utilities.Response
// @Router /api/boxes/{id} [get]
func (bc *BoxController) GetBox(c *gin.Context) {
	boxID := c.Param("id")

	var box models.Box
	if err := bc.DB.First(&box, boxID).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusNotFound, "Box not found", err.Error())
		return
	}

	utilities.SuccessResponse(c, http.StatusOK, "Box retrieved successfully", box.ToBoxResponse())
}

// UpdateBox godoc
// @Summary Update box
// @Description Update box information (logged-in users only)
// @Tags boxes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Box ID"
// @Param request body UpdateBoxRequest true "Update box request"
// @Success 200 {object} utilities.Response{data=models.BoxResponse}
// @Failure 400 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Failure 404 {object} utilities.Response
// @Router /api/boxes/{id} [put]
func (bc *BoxController) UpdateBox(c *gin.Context) {
	boxID := c.Param("id")

	var req UpdateBoxRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utilities.ValidationErrorResponse(c, err)
		return
	}

	var box models.Box
	if err := bc.DB.First(&box, boxID).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusNotFound, "Box not found", err.Error())
		return
	}

	// Check for duplicate box code (excluding current box)
	var existingBox models.Box
	if err := bc.DB.Where("code = ? AND id != ?", req.Code, boxID).First(&existingBox).Error; err == nil {
		utilities.ErrorResponse(c, http.StatusBadRequest, "Box code already exists", "A box with this code already exists")
		return
	}

	// Update box fields
	box.Code = req.Code
	box.Name = req.Name

	if err := bc.DB.Save(&box).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to update box", err.Error())
		return
	}

	utilities.SuccessResponse(c, http.StatusOK, "Box updated successfully", box.ToBoxResponse())
}

// RemoveBox godoc
// @Summary Remove box
// @Description Soft delete a box (logged-in users only)
// @Tags boxes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Box ID"
// @Success 200 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Failure 404 {object} utilities.Response
// @Router /api/boxes/{id} [delete]
func (bc *BoxController) RemoveBox(c *gin.Context) {
	boxID := c.Param("id")

	var box models.Box
	if err := bc.DB.First(&box, boxID).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusNotFound, "Box not found", err.Error())
		return
	}

	if err := bc.DB.Delete(&box).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to remove box", err.Error())
		return
	}

	utilities.SuccessResponse(c, http.StatusOK, "Box removed successfully", nil)
}

// CreateBox godoc
// @Summary Create new box
// @Description Create a new box (logged-in users only)
// @Tags boxes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateBoxRequest true "Create box request"
// @Success 201 {object} utilities.Response{data=models.BoxResponse}
// @Failure 400 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Router /api/boxes [post]
func (bc *BoxController) CreateBox(c *gin.Context) {
	var req CreateBoxRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utilities.ValidationErrorResponse(c, err)
		return
	}

	// convert code to uppercase and trim spaces
	req.Code = strings.ToUpper(strings.TrimSpace(req.Code))

	box := models.Box{
		Code: req.Code,
		Name: req.Name,
	}

	// Check for duplicate box code
	var existingBox models.Box
	if err := bc.DB.Where("code = ?", req.Code).First(&existingBox).Error; err == nil {
		utilities.ErrorResponse(c, http.StatusBadRequest, "Box code already exists", "A box with this code already exists")
		return
	}

	// Create a new box and return the response
	if err := bc.DB.Create(&box).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to create box", err.Error())
		return
	}

	utilities.SuccessResponse(c, http.StatusCreated, "Box created successfully", box.ToBoxResponse())
}

// Request/Response structs
type BoxesListResponse struct {
	Boxes      []models.BoxResponse         `json:"boxes"`
	Pagination utilities.PaginationResponse `json:"pagination"`
}

type UpdateBoxRequest struct {
	Code string `json:"code" binding:"required"`
	Name string `json:"name" binding:"required"`
}

type CreateBoxRequest struct {
	Code string `json:"code" binding:"required"`
	Name string `json:"name" binding:"required"`
}
