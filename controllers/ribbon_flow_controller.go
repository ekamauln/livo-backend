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

type RibbonFlowController struct {
	DB *gorm.DB
}

// NewRibbonFlowController creates a new ribbon flow controller
func NewRibbonFlowController(db *gorm.DB) *RibbonFlowController {
	return &RibbonFlowController{DB: db}
}

// GetRibbonFlows godoc
// @Summary Get all ribbon flows
// @Description Get all ribbon flows with pagination and search, primary tracking from qc-ribbon.
// @Tags ribbons
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Param start_date query string false "Start date (YYYY-MM-DD or YYYY-M-D format)"
// @Param end_date query string false "End date (YYYY-MM-DD or YYYY-M-D format)"
// @Param search query string false "Search by tracking number"
// @Success 200 {object} utilities.Response{data=RibbonFlowsListResponse}
// @Failure 400 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Router /api/ribbons/ribbon-flows [get]
func (rfc *RibbonFlowController) GetRibbonFlows(c *gin.Context) {
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	// ADDED: Parse date range parameters
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	// Parse search parameter (optional)
	search := c.Query("search")

	var trackingNumbers []string
	var total int64

	// Get tracking numbers primarily from qc_ribbons
	query := rfc.DB.Model(&models.QcRibbon{}).Select("DISTINCT tracking").Where("tracking IS NOT NULL AND tracking != ''")

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

	// Add search filter if provided
	if search != "" {
		query = query.Where("tracking ILIKE ?", "%"+search+"%")
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to count tracking numbers", err.Error())
		return
	}

	// Get paginated tracking numbers
	if err := query.Order("tracking").Limit(limit).Offset(offset).Pluck("tracking", &trackingNumbers).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve tracking numbers", err.Error())
		return
	}

	// Build ribbon flows for each tracking
	var ribbonFlows []RibbonFlowResponse
	for _, tracking := range trackingNumbers {
		flow := rfc.buildRibbonFlow(tracking)
		ribbonFlows = append(ribbonFlows, flow)
	}

	response := RibbonFlowsListResponse{
		RibbonFlows: ribbonFlows,
		Pagination: utilities.PaginationResponse{
			Page:  page,
			Limit: limit,
			Total: int(total),
		},
	}

	// Build success message with date filters
	message := "Ribbon flows retrieved successfully"
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

// GetRibbonFlow godoc
// @Summary Get ribbon flow tracking
// @Description Get the complete flow tracking through ribbon process (qc-ribbon -> outbound -> order).
// @Tags ribbons
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param tracking path string true "Tracking number"
// @Success 200 {object} utilities.Response{data=RibbonFlowResponse}
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Failure 404 {object} utilities.Response
// @Router /api/ribbons/ribbon-flow/{tracking} [get]
func (rfc *RibbonFlowController) GetRibbonFlow(c *gin.Context) {
	tracking := c.Param("tracking")

	if tracking == "" {
		utilities.ErrorResponse(c, http.StatusBadRequest, "Invalid tracking", "Tracking number is required")
		return
	}

	flow := rfc.buildRibbonFlow(tracking)

	// CHANGED: Check if qc-ribbon exists (since it's the primary source)
	if flow.QcRibbon == nil {
		utilities.ErrorResponse(c, http.StatusNotFound, "Tracking not found", "No qc-ribbon record found for the specified tracking number")
		return
	}

	utilities.SuccessResponse(c, http.StatusOK, "Ribbon flow retrieved successfully", flow)
}

// Helper function to build ribbon flow for a tracking number
func (rfc *RibbonFlowController) buildRibbonFlow(tracking string) RibbonFlowResponse {
	var response RibbonFlowResponse
	response.Tracking = tracking

	// 1. Query QC Ribbon (PRIMARY SOURCE)
	var qcRibbon models.QcRibbon
	if err := rfc.DB.Preload("User").Where("tracking = ?", tracking).First(&qcRibbon).Error; err == nil {
		var operator *RibbonOperatorFlowInfo
		if qcRibbon.QcOperator != nil {
			operator = &RibbonOperatorFlowInfo{
				ID:       qcRibbon.QcOperator.ID,
				Username: qcRibbon.QcOperator.Username,
				FullName: qcRibbon.QcOperator.FullName,
			}
		}

		response.QcRibbon = &QcRibbonFlowInfo{
			Operator:  operator,
			CreatedAt: qcRibbon.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	// 2. Query Outbound
	var outbound models.Outbound
	if err := rfc.DB.Preload("User").Where("tracking = ?", tracking).First(&outbound).Error; err == nil {
		var operator *RibbonOperatorFlowInfo
		if outbound.OutboundOperator != nil {
			operator = &RibbonOperatorFlowInfo{
				ID:       outbound.OutboundOperator.ID,
				Username: outbound.OutboundOperator.Username,
				FullName: outbound.OutboundOperator.FullName,
			}
		}

		response.Outbound = &RibbonOutboundFlowInfo{
			Operator:        operator,
			Expedition:      outbound.Expedition,
			ExpeditionColor: outbound.ExpeditionColor, // ADDED
			CreatedAt:       outbound.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	// 3. Query Order (LAST)
	var order models.Order
	if err := rfc.DB.Where("tracking = ?", tracking).First(&order).Error; err == nil {
		response.Order = &RibbonOrderFlowInfo{
			Tracking:     order.Tracking,
			OrderGineeID: order.OrderGineeID,
			Complained:   order.Complained,
			CreatedAt:    order.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	return response
}

// Request/Response structs - REORDERED to match flow
type RibbonFlowsListResponse struct {
	RibbonFlows []RibbonFlowResponse         `json:"ribbon_flows"`
	Pagination  utilities.PaginationResponse `json:"pagination"`
}

// FIXED: Use unique pagination response to avoid conflicts
type RibbonFlowPaginationResponse struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
	Total int `json:"total"`
}

// REORDERED: qc-ribbon -> outbound -> order
type RibbonFlowResponse struct {
	Tracking string                  `json:"tracking"`
	QcRibbon *QcRibbonFlowInfo       `json:"qc_ribbon,omitempty"`
	Outbound *RibbonOutboundFlowInfo `json:"outbound,omitempty"`
	Order    *RibbonOrderFlowInfo    `json:"order,omitempty"`
}

type QcRibbonFlowInfo struct {
	Operator  *RibbonOperatorFlowInfo `json:"operator,omitempty"`
	CreatedAt string                  `json:"created_at"`
}

type OutboundFlowInfo struct {
	Operator        *RibbonOperatorFlowInfo `json:"operator,omitempty"`
	Expedition      string                  `json:"expedition"`
	ExpeditionColor string                  `json:"expedition_color"`
	CreatedAt       string                  `json:"created_at"`
}

type RibbonOutboundFlowInfo struct {
	Operator        *RibbonOperatorFlowInfo `json:"operator,omitempty"`
	Expedition      string                  `json:"expedition"`
	ExpeditionColor string                  `json:"expedition_color"`
	CreatedAt       string                  `json:"created_at"`
}

type RibbonOrderFlowInfo struct {
	Tracking     string `json:"tracking"`
	OrderGineeID string `json:"order_ginee_id"`
	Complained   bool   `json:"complained"`
	CreatedAt    string `json:"created_at"`
}

type RibbonOperatorFlowInfo struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	FullName string `json:"full_name"`
}
