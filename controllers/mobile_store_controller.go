package controllers

import (
	"livo-backend/models"
	"livo-backend/utilities"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type MobileStoreController struct {
	DB *gorm.DB
}

// NewMobileStoreController creates a new mobile store controller
func NewMobileStoreController(db *gorm.DB) *MobileStoreController {
	return &MobileStoreController{DB: db}
}

// GetMobileStores godoc
// @Summary Get all stores
// @Description Get list of all stores.
// @Tags stores
// @Accept json
// @Produce json
// @Param search query string false "Search by store tracking (partial match)"
// @Success 200 {object} utilities.Response{data=MobileStoresListResponse}
// @Router /api/mobile/stores [get]
func (smc *MobileStoreController) GetMobileStores(c *gin.Context) {
	// Parse search parameter
	search := c.Query("search")

	var stores []models.Store
	var total int64

	// Build query with optional search
	query := smc.DB.Model(&models.Store{})

	if search != "" {
		// Search by store mobile tracking with partial match
		query = query.Where("code ILIKE ? OR name ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// Get total count with search filter
	if err := query.Count(&total).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to count store mobiles", err.Error())
		return
	}

	// Execute query to get store mobiles
	if err := query.Order("id ASC").Find(&stores).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve stores", err.Error())
		return
	}

	// Convert to response format
	storeResponses := make([]models.StoreResponse, len(stores))
	for i, store := range stores {
		storeResponses[i] = store.ToStoreResponse()
	}

	response := StoreMobilesListResponse{
		Stores: storeResponses,
		Total:  int(total),
	}

	// Build success message
	message := "Store mobiles retrieved successfully"
	if search != "" {
		message += " (filtered by code or name: " + search + ")"
	}

	utilities.SuccessResponse(c, http.StatusOK, message, response)
}

// Request/Response structs
type StoreMobilesListResponse struct {
	Stores []models.StoreResponse `json:"stores"`
	Total  int                    `json:"total"`
}
