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

type OnlineFlowController struct {
	DB *gorm.DB
}

// NewOnlineFlowController creates a new online flow controller
func NewOnlineFlowController(db *gorm.DB) *OnlineFlowController {
	return &OnlineFlowController{DB: db}
}

// GetOnlineFlows godoc
// @Summary Get all online flows
// @Description Get all online flows with pagination and search, primary tracking from qc-online.
// @Tags onlines
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Param start_date query string false "Start date (YYYY-MM-DD format)"
// @Param end_date query string false "End date (YYYY-MM-DD format)"
// @Param search query string false "Search by tracking number"
// @Success 200 {object} utilities.Response{data=OnlineFlowsListResponse}
// @Failure 400 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Router /api/onlines/online-flows [get]
func (ofc *OnlineFlowController) GetOnlineFlows(c *gin.Context) {
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	// Parse date range parameters
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	// Parse search parameter (optional)
	search := c.Query("search")

	var trackingNumbers []string
	var total int64

	// Get tracking numbers primarily from mb_onlines
	query := ofc.DB.Model(&models.QcOnline{}).Select("DISTINCT tracking").Where("tracking IS NOT NULL AND tracking != ''")

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

	// Build online flows for each tracking
	var onlineFlows []OnlineFlowResponse
	for _, tracking := range trackingNumbers {
		flow := ofc.buildOnlineFlow(tracking)
		onlineFlows = append(onlineFlows, flow)
	}

	response := OnlineFlowsListResponse{
		OnlineFlows: onlineFlows,
		Pagination: utilities.PaginationResponse{
			Page:  page,
			Limit: limit,
			Total: int(total),
		},
	}

	// Build success message
	message := "Online flows retrieved successfully"
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

// GetOnlineFlow godoc
// @Summary Get online flow tracking
// @Description Get the complete flow tracking through online process (qc-online -> outbound -> order).
// @Tags onlines
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param tracking path string true "Tracking number"
// @Success 200 {object} utilities.Response{data=OnlineFlowResponse}
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Failure 404 {object} utilities.Response
// @Router /api/onlines/online-flows/{tracking} [get]
func (ofc *OnlineFlowController) GetOnlineFlow(c *gin.Context) {
	tracking := c.Param("tracking")

	if tracking == "" {
		utilities.ErrorResponse(c, http.StatusBadRequest, "Invalid tracking", "Tracking number is required")
		return
	}

	flow := ofc.buildOnlineFlow(tracking)

	// CHANGED: Check if qc-online exists (since it's the primary source)
	if flow.QcOnline == nil {
		utilities.ErrorResponse(c, http.StatusNotFound, "Tracking not found", "No qc-online record found for the specified tracking number")
		return
	}

	utilities.SuccessResponse(c, http.StatusOK, "Online flow retrieved successfully", flow)
}

// Helper function to build online flow for a tracking number
func (ofc *OnlineFlowController) buildOnlineFlow(tracking string) OnlineFlowResponse {
	var response OnlineFlowResponse
	response.Tracking = tracking

	// 1. Query QC Online (PRIMARY SOURCE)
	var qcOnline models.QcOnline
	if err := ofc.DB.Preload("QcOperator.UserRoles.Role").Preload("QcOperator.UserRoles.Assigner").Where("tracking = ?", tracking).First(&qcOnline).Error; err == nil {
		var operator *OnlineOperatorFlowInfo
		if qcOnline.QcOperator != nil {
			operator = &OnlineOperatorFlowInfo{
				ID:       qcOnline.QcOperator.ID,
				Username: qcOnline.QcOperator.Username,
				FullName: qcOnline.QcOperator.FullName,
			}
		}

		response.QcOnline = &QcOnlineFlowInfo{
			Operator:  operator,
			CreatedAt: qcOnline.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	// 2. Query Outbound
	var outbound models.Outbound
	if err := ofc.DB.Preload("OutboundOperator.UserRoles.Role").Preload("OutboundOperator.UserRoles.Assigner").Where("tracking = ?", tracking).First(&outbound).Error; err == nil {
		var operator *OnlineOperatorFlowInfo
		if outbound.OutboundOperator != nil {
			operator = &OnlineOperatorFlowInfo{
				ID:       outbound.OutboundOperator.ID,
				Username: outbound.OutboundOperator.Username,
				FullName: outbound.OutboundOperator.FullName,
			}
		}

		response.Outbound = &OnlineOutboundFlowInfo{
			Operator:        operator,
			Expedition:      outbound.Expedition,
			ExpeditionColor: outbound.ExpeditionColor,
			CreatedAt:       outbound.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	// 3. Query Order (LAST)
	var order models.Order
	if err := ofc.DB.Where("tracking = ?", tracking).First(&order).Error; err == nil {
		response.Order = &OnlineOrderFlowInfo{
			Tracking:         order.Tracking,
			ProcessingStatus: order.ProcessingStatus,
			OrderGineeID:     order.OrderGineeID,
			Complained:       order.Complained,
			CreatedAt:        order.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	return response
}

// Request/Response structs - REORDERED to match flow
type OnlineFlowsListResponse struct {
	OnlineFlows []OnlineFlowResponse         `json:"online_flows"`
	Pagination  utilities.PaginationResponse `json:"pagination"`
}

// REORDERED: qc-online -> outbound -> order
type OnlineFlowResponse struct {
	Tracking string                  `json:"tracking"`
	QcOnline *QcOnlineFlowInfo       `json:"qc_online,omitempty"`
	Outbound *OnlineOutboundFlowInfo `json:"outbound,omitempty"`
	Order    *OnlineOrderFlowInfo    `json:"order,omitempty"`
}

type QcOnlineFlowInfo struct {
	Operator  *OnlineOperatorFlowInfo `json:"operator,omitempty"`
	CreatedAt string                  `json:"created_at"`
}

type OnlineOrderFlowInfo struct {
	Tracking         string `json:"tracking"`
	ProcessingStatus string `json:"processing_status"`
	OrderGineeID     string `json:"order_ginee_id"`
	Complained       bool   `json:"complained"`
	CreatedAt        string `json:"created_at"`
}

type OnlineOutboundFlowInfo struct {
	Operator        *OnlineOperatorFlowInfo `json:"operator,omitempty"`
	Expedition      string                  `json:"expedition"`
	ExpeditionColor string                  `json:"expedition_color"`
	CreatedAt       string                  `json:"created_at"`
}

type OnlineOperatorFlowInfo struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	FullName string `json:"full_name"`
}
