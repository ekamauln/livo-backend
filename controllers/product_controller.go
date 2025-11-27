package controllers

import (
	"livo-backend/models"
	"livo-backend/utilities"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ProductController struct {
	DB *gorm.DB
}

// NewProductController creates a new product controller
func NewProductController(db *gorm.DB) *ProductController {
	return &ProductController{DB: db}
}

// GetProducts godoc
// @Summary Get all products
// @Description Get list of all products with optional search by SKU.
// @Tags products
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Param search query string false "Search by SKU (partial match)"
// @Success 200 {object} utilities.Response{data=ProductsListResponse}
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Router /api/products [get]
func (pc *ProductController) GetProducts(c *gin.Context) {
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	// Parse search parameter
	search := c.Query("search")

	var products []models.Product
	var total int64

	// Build query with optional search
	query := pc.DB.Model(&models.Product{})

	if search != "" {
		// Search by SKU with partial match
		query = query.Where("sku ILIKE ?", "%"+search+"%")
	}

	// Get total count with search filter
	if err := query.Count(&total).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to count products", err.Error())
		return
	}

	// Get products with pagination, search filter, and order by ID ascending
	if err := query.Order("id ASC").Limit(limit).Offset(offset).Find(&products).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve products", err.Error())
		return
	}

	// Convert to response format
	productResponses := make([]models.ProductResponse, len(products))
	for i, product := range products {
		productResponses[i] = product.ToProductResponse()
	}

	response := ProductsListResponse{
		Products: productResponses,
		Pagination: utilities.PaginationResponse{
			Page:  page,
			Limit: limit,
			Total: int(total),
		},
	}

	// Build success message
	message := "Products retrieved successfully"
	if search != "" {
		message += " (filtered by SKU: " + search + ")"
	}

	utilities.SuccessResponse(c, http.StatusOK, message, response)
}

// GetProduct godoc
// @Summary Get product by ID
// @Description Get specific product information.
// @Tags products
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Product ID"
// @Success 200 {object} utilities.Response{data=models.ProductResponse}
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Failure 404 {object} utilities.Response
// @Router /api/products/{id} [get]
func (pc *ProductController) GetProduct(c *gin.Context) {
	productID := c.Param("id")

	var product models.Product
	if err := pc.DB.First(&product, productID).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusNotFound, "Product not found", err.Error())
		return
	}

	utilities.SuccessResponse(c, http.StatusOK, "Product retrieved successfully", product.ToProductResponse())
}

// UpdateProduct godoc
// @Summary Update product
// @Description Update product information (admin only)
// @Tags products
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Product ID"
// @Param request body UpdateProductRequest true "Update product request"
// @Success 200 {object} utilities.Response{data=models.ProductResponse}
// @Failure 400 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Failure 404 {object} utilities.Response
// @Router /api/products/{id} [put]
func (pc *ProductController) UpdateProduct(c *gin.Context) {
	productID := c.Param("id")

	var req UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utilities.ValidationErrorResponse(c, err)
		return
	}

	var product models.Product
	if err := pc.DB.First(&product, productID).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusNotFound, "Product not found", err.Error())
		return
	}

	// Update product fields
	product.Name = req.Name
	product.Image = req.Image
	product.Variant = req.Variant
	product.Location = req.Location
	product.Barcode = req.Barcode
	if err := pc.DB.Save(&product).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to update product", err.Error())
		return
	}

	utilities.SuccessResponse(c, http.StatusOK, "Product updated successfully", product.ToProductResponse())
}

// RemoveProduct godoc
// @Summary Remove product
// @Description Soft delete a product (admin only)
// @Tags products
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Product ID"
// @Success 200 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Failure 404 {object} utilities.Response
// @Router /api/products/{id} [delete]
func (pc *ProductController) RemoveProduct(c *gin.Context) {
	productID := c.Param("id")

	var product models.Product
	if err := pc.DB.First(&product, productID).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusNotFound, "Product not found", err.Error())
		return
	}

	if err := pc.DB.Delete(&product).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to remove product", err.Error())
		return
	}

	utilities.SuccessResponse(c, http.StatusOK, "Product removed successfully", nil)
}

// CreateProduct godoc
// @Summary Create new product
// @Description Create a new product (admin only)
// @Tags products
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateProductRequest true "Create product request"
// @Success 201 {object} utilities.Response{data=models.ProductResponse}
// @Failure 400 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Router /api/products [post]
func (pc *ProductController) CreateProduct(c *gin.Context) {
	var req CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utilities.ValidationErrorResponse(c, err)
		return
	}

	product := models.Product{
		Sku:      req.Sku,
		Name:     req.Name,
		Image:    req.Image,
		Variant:  req.Variant,
		Location: req.Location,
		Barcode:  req.Barcode,
	}

	// Create a new product and return the response
	if err := pc.DB.Create(&product).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to create product", err.Error())
		return
	}

	utilities.SuccessResponse(c, http.StatusCreated, "Product created successfully", product.ToProductResponse())
}

// Request/Response structs
type ProductsListResponse struct {
	Products   []models.ProductResponse     `json:"products"`
	Pagination utilities.PaginationResponse `json:"pagination"`
}

type UpdateProductRequest struct {
	Name     string `json:"name" binding:"required"`
	Image    string `json:"image" binding:"required"`
	Variant  string `json:"variant" binding:"required"`
	Location string `json:"location"`
	Barcode  string `json:"barcode"`
}

type CreateProductRequest struct {
	Sku      string `json:"sku" binding:"required,alphanum"`
	Name     string `json:"name" binding:"required"`
	Image    string `json:"image" binding:"required"`
	Variant  string `json:"variant" binding:"required"`
	Location string `json:"location"`
	Barcode  string `json:"barcode"`
}
