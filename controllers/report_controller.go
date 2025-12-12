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

type ReportController struct {
	DB *gorm.DB
}

// NewReportController creates a new report controller
func NewReportController(db *gorm.DB) *ReportController {
	return &ReportController{DB: db}
}

// GetBoxReports godoc
// @Summary Get box count reports
// @Description Get box usage count from QC Ribbon and QC Online details with date range filtering, excluding PC/Packing boxes (logged-in users only)
// @Tags reports
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Param start_date query string false "Start date (YYYY-MM-DD format)"
// @Param end_date query string false "End date (YYYY-MM-DD format)"
// @Param search query string false "Search by box code or name (partial match)"
// @Success 200 {object} utilities.Response{data=BoxCountReportsListResponse}
// @Failure 400 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Router /api/reports/boxes-count [get]
func (rc *ReportController) GetBoxReports(c *gin.Context) {
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	// Parse date range parameters
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	// Parse search parameter
	search := c.Query("search")

	// Build date filter conditions
	var ribbonDateFilter, onlineDateFilter string

	if startDate != "" {
		// Validate start date format
		if _, err := time.Parse("2006-01-02", startDate); err != nil {
			utilities.ErrorResponse(c, http.StatusBadRequest, "Invalid start_date format", "start_date must be in YYYY-MM-DD format")
			return
		}
		ribbonDateFilter += fmt.Sprintf(" AND qc_ribbon_details.created_at >= '%s 00:00:00'", startDate)
		onlineDateFilter += fmt.Sprintf(" AND qc_online_details.created_at >= '%s 00:00:00'", startDate)
	}

	if endDate != "" {
		// Validate end date format
		if parsedEndDate, err := time.Parse("2006-01-02", endDate); err != nil {
			utilities.ErrorResponse(c, http.StatusBadRequest, "Invalid end_date format", "end_date must be in YYYY-MM-DD format")
			return
		} else {
			// Add 24 hours to get the start of next day
			nextDay := parsedEndDate.AddDate(0, 0, 1).Format("2006-01-02 00:00:00")
			ribbonDateFilter += fmt.Sprintf(" AND qc_ribbon_details.created_at < '%s'", nextDay)
			onlineDateFilter += fmt.Sprintf(" AND qc_online_details.created_at < '%s'", nextDay)
		}
	}

	var reports []BoxCountReport
	var total int64

	// First, get the data without pagination for counting
	countQuery := rc.DB.Table("boxes").
		Select("boxes.id").
		Joins(fmt.Sprintf(`
			LEFT JOIN (
				SELECT 
					box_id,
					SUM(quantity) as ribbon_count
				FROM qc_ribbon_details
				WHERE deleted_at IS NULL %s
				GROUP BY box_id
			) ribbon_counts ON boxes.id = ribbon_counts.box_id
		`, ribbonDateFilter)).
		Joins(fmt.Sprintf(`
			LEFT JOIN (
				SELECT 
					box_id,
					SUM(quantity) as online_count
				FROM qc_online_details
				WHERE deleted_at IS NULL %s
				GROUP BY box_id
			) online_counts ON boxes.id = online_counts.box_id
		`, onlineDateFilter)).
		Where("boxes.deleted_at IS NULL").
		Where("boxes.code NOT ILIKE ? AND boxes.code NOT ILIKE ? AND boxes.name NOT ILIKE ? AND boxes.name NOT ILIKE ?",
			"%PC%", "%Packing%", "%PC%", "%Packing%").
		Where("(COALESCE(ribbon_counts.ribbon_count, 0) + COALESCE(online_counts.online_count, 0)) > 0")

	// Apply search filter for count if provided
	if search != "" {
		countQuery = countQuery.Where("boxes.code ILIKE ? OR boxes.name ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// Get total count
	if err := countQuery.Count(&total).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to count box reports", err.Error())
		return
	}

	// Build main query for data retrieval
	query := rc.DB.Table("boxes").
		Select(`
			boxes.id as box_id,
			boxes.code as box_code,
			boxes.name as box_name,
			COALESCE(ribbon_counts.ribbon_count, 0) + COALESCE(online_counts.online_count, 0) as total_count,
			COALESCE(ribbon_counts.ribbon_count, 0) as ribbon_count,
			COALESCE(online_counts.online_count, 0) as online_count
		`).
		Joins(fmt.Sprintf(`
			LEFT JOIN (
				SELECT 
					box_id,
					SUM(quantity) as ribbon_count
				FROM qc_ribbon_details
				WHERE deleted_at IS NULL %s
				GROUP BY box_id
			) ribbon_counts ON boxes.id = ribbon_counts.box_id
		`, ribbonDateFilter)).
		Joins(fmt.Sprintf(`
			LEFT JOIN (
				SELECT 
					box_id,
					SUM(quantity) as online_count
				FROM qc_online_details
				WHERE deleted_at IS NULL %s
				GROUP BY box_id
			) online_counts ON boxes.id = online_counts.box_id
		`, onlineDateFilter)).
		Where("boxes.deleted_at IS NULL").
		Where("boxes.code NOT ILIKE ? AND boxes.code NOT ILIKE ? AND boxes.name NOT ILIKE ? AND boxes.name NOT ILIKE ?",
			"%PC%", "%Packing%", "%PC%", "%Packing%").
		Where("(COALESCE(ribbon_counts.ribbon_count, 0) + COALESCE(online_counts.online_count, 0)) > 0")

	// Apply search filter if provided
	if search != "" {
		query = query.Where("boxes.code ILIKE ? OR boxes.name ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// Get reports with pagination
	if err := query.
		Order("total_count DESC, boxes.code ASC").
		Limit(limit).
		Offset(offset).
		Scan(&reports).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve box reports", err.Error())
		return
	}

	// For each box report, get the detailed breakdown
	for i := range reports {
		// Build date filter for details query
		detailDateFilter := "qc_ribbon_details.deleted_at IS NULL" // Specify table name
		if startDate != "" {
			detailDateFilter += fmt.Sprintf(" AND qc_ribbon_details.created_at >= '%s 00:00:00'", startDate)
		}
		if endDate != "" {
			parsedEndDate, _ := time.Parse("2006-01-02", endDate)
			nextDay := parsedEndDate.AddDate(0, 0, 1).Format("2006-01-02 00:00:00")
			detailDateFilter += fmt.Sprintf(" AND qc_ribbon_details.created_at < '%s'", nextDay)
		}

		// Get QC Ribbon details - JOIN with orders and users table
		var ribbonDetails []BoxUsageDetail
		ribbonQuery := rc.DB.Table("qc_ribbon_details").
			Select(`
				qc_ribbons.tracking,
				COALESCE(orders.order_ginee_id, '') as order_id,
				boxes.name as box_name,
				qc_ribbon_details.quantity,
				COALESCE(qc_ribbons.qc_by, 0) as qc_by,
				COALESCE(users.username, '') as username,
				COALESCE(users.full_name, '') as full_name,
				qc_ribbon_details.created_at,
				'QC Ribbon' as source
			`).
			Joins("INNER JOIN qc_ribbons ON qc_ribbons.id = qc_ribbon_details.qc_ribbon_id AND qc_ribbons.deleted_at IS NULL").
			Joins("INNER JOIN boxes ON boxes.id = qc_ribbon_details.box_id AND boxes.deleted_at IS NULL").
			Joins("LEFT JOIN orders ON orders.tracking = qc_ribbons.tracking AND orders.deleted_at IS NULL").
			Joins("LEFT JOIN users ON users.id = qc_ribbons.qc_by AND users.deleted_at IS NULL").
			Where("qc_ribbon_details.box_id = ?", reports[i].BoxID).
			Where(detailDateFilter).
			Order("qc_ribbon_details.created_at DESC")

		if err := ribbonQuery.Scan(&ribbonDetails).Error; err != nil {
			utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve ribbon details", err.Error())
			return
		}

		// Build date filter for online details
		onlineDetailDateFilter := "qc_online_details.deleted_at IS NULL" // Specify table name
		if startDate != "" {
			onlineDetailDateFilter += fmt.Sprintf(" AND qc_online_details.created_at >= '%s 00:00:00'", startDate)
		}
		if endDate != "" {
			parsedEndDate, _ := time.Parse("2006-01-02", endDate)
			nextDay := parsedEndDate.AddDate(0, 0, 1).Format("2006-01-02 00:00:00")
			onlineDetailDateFilter += fmt.Sprintf(" AND qc_online_details.created_at < '%s'", nextDay)
		}

		// Get QC Online details - JOIN with orders and users table
		var onlineDetails []BoxUsageDetail
		onlineQuery := rc.DB.Table("qc_online_details").
			Select(`
				qc_onlines.tracking,
				COALESCE(orders.order_ginee_id, '') as order_id,
				boxes.name as box_name,
				qc_online_details.quantity,
				COALESCE(qc_onlines.qc_by, 0) as qc_by,
				COALESCE(users.username, '') as username,
				COALESCE(users.full_name, '') as full_name,
				qc_online_details.created_at,
				'QC Online' as source
			`).
			Joins("INNER JOIN qc_onlines ON qc_onlines.id = qc_online_details.qc_online_id AND qc_onlines.deleted_at IS NULL").
			Joins("INNER JOIN boxes ON boxes.id = qc_online_details.box_id AND boxes.deleted_at IS NULL").
			Joins("LEFT JOIN orders ON orders.tracking = qc_onlines.tracking AND orders.deleted_at IS NULL").
			Joins("LEFT JOIN users ON users.id = qc_onlines.qc_by AND users.deleted_at IS NULL").
			Where("qc_online_details.box_id = ?", reports[i].BoxID).
			Where(onlineDetailDateFilter).
			Order("qc_online_details.created_at DESC")

		if err := onlineQuery.Scan(&onlineDetails).Error; err != nil {
			utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve online details", err.Error())
			return
		}

		// Combine ribbon and online details
		reports[i].Details = append(ribbonDetails, onlineDetails...)
	}

	response := BoxCountReportsListResponse{
		Reports: reports,
		Pagination: utilities.PaginationResponse{
			Page:  page,
			Limit: limit,
			Total: int(total),
		},
	}

	// Build success message
	message := "Box count reports retrieved successfully"
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

// GetOutboundReports godoc
// @Summary Get outbound reports
// @Description Get outbound reports with date filtering and exact slug search, without pagination (logged-in users only)
// @Tags reports
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param date query string false "Filter by date (YYYY-MM-DD format)"
// @Param search query string false "Search by exact slug match"
// @Success 200 {object} utilities.Response{data=OutboundReportsListResponse}
// @Failure 400 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Router /api/reports/handout-outbounds [get]
func (rc *ReportController) GetOutboundReports(c *gin.Context) {
	// Parse date parameter
	date := c.Query("date")

	// Parse search parameter
	search := c.Query("search")

	var outbounds []models.Outbound
	var total int64

	// Build query for data retrieval
	query := rc.DB.Model(&models.Outbound{})

	// Apply date filter if provided
	if date != "" {
		// Parse date and validate format
		if parsedDate, err := time.Parse("2006-01-02", date); err != nil {
			utilities.ErrorResponse(c, http.StatusBadRequest, "Invalid date format", "date must be in YYYY-MM-DD format")
			return
		} else {
			// Filter for the entire day (from 00:00:00 to 23:59:59)
			startOfDay := parsedDate.Format("2006-01-02 00:00:00")
			endOfDay := parsedDate.AddDate(0, 0, 1).Format("2006-01-02 00:00:00")
			query = query.Where("created_at >= ? AND created_at < ?", startOfDay, endOfDay)
		}
	}

	// Apply search filter if provided (EXACT MATCH)
	if search != "" {
		query = query.Where("expedition_slug = ?", search) // Changed from ILIKE to exact match
	}

	// Get total count with filters
	if err := query.Count(&total).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to count outbound reports", err.Error())
		return
	}

	// Get all outbound records with preloaded relationships (no pagination)
	if err := query.
		Preload("OutboundOperator.UserRoles.Role").
		Preload("OutboundOperator.UserRoles.Assigner").
		Order("id DESC").
		Find(&outbounds).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve outbound reports", err.Error())
		return
	}

	// Convert to response format
	outboundResponses := make([]models.OutboundResponse, len(outbounds))
	for i, outbound := range outbounds {
		outboundResponses[i] = outbound.ToOutboundResponse()
	}

	response := OutboundReportsListResponse{
		Outbounds: outboundResponses,
		Total:     int(total),
	}

	// Build success message
	message := "Outbound reports retrieved successfully"
	var filters []string

	if date != "" {
		filters = append(filters, "date: "+date)
	}

	if search != "" {
		filters = append(filters, "exact slug: "+search) // Updated message
	}

	if len(filters) > 0 {
		message += fmt.Sprintf(" (filtered by %s)", strings.Join(filters, " | "))
	}

	utilities.SuccessResponse(c, http.StatusOK, message, response)
}

// GetReturnReports godoc
// @Summary Get return reports
// @Description Get return reports with date filtering and exact return type search, without pagination (logged-in users only)
// @Tags reports
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param date query string false "Filter by date (YYYY-MM-DD format)"
// @Param search query string false "Search by exact return type match"
// @Success 200 {object} utilities.Response{data=ReturnReportsListResponse}
// @Failure 400 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Router /api/reports/handout-returns [get]
func (rc *ReportController) GetReturnReports(c *gin.Context) {
	// Parse date parameter
	date := c.Query("date")

	// Parse search parameter
	search := c.Query("search")

	var returns []models.Return
	var total int64

	// Build query for data retrieval
	query := rc.DB.Model(&models.Return{})

	// Apply date filter if provided (CHANGED: using updated_at instead of created_at)
	if date != "" {
		// Parse date and validate format
		if parsedDate, err := time.Parse("2006-01-02", date); err != nil {
			utilities.ErrorResponse(c, http.StatusBadRequest, "Invalid date format", "date must be in YYYY-MM-DD format")
			return
		} else {
			// Filter for the entire day (from 00:00:00 to 23:59:59)
			startOfDay := parsedDate.Format("2006-01-02 00:00:00")
			endOfDay := parsedDate.AddDate(0, 0, 1).Format("2006-01-02 00:00:00")
			query = query.Where("updated_at >= ? AND updated_at < ?", startOfDay, endOfDay)
		}
	}

	// Apply search filter if provided (EXACT MATCH)
	if search != "" {
		query = query.Where("return_type = ?", search)
	}

	// Get total count with filters
	if err := query.Count(&total).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to count return reports", err.Error())
		return
	}

	// Get all return records with preloaded relationships
	if err := query.
		Preload("ReturnDetails", "deleted_at IS NULL"). // Load return details (non-deleted only)
		Preload("ReturnDetails.Product").               // Load product info for each detail
		Preload("Channel").                             // Load channel info
		Preload("Store").                               // Load store info
		Preload("CreateOperator").                      // Load create operator info
		Preload("UpdateOperator").                      // Load update operator info
		Order("id DESC").
		Find(&returns).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve return reports", err.Error())
		return
	}

	// Manually load order data for each return using OldTracking
	for i := range returns {
		if returns[i].OldTracking != "" {
			var order models.Order
			if err := rc.DB.
				Preload("OrderDetails").
				Preload("PickOperator").
				Preload("PickOperator.UserRoles.Role").
				Preload("PickOperator.UserRoles.Assigner").
				Where("tracking = ?", returns[i].OldTracking).
				First(&order).Error; err == nil {
				returns[i].Order = &order
			}
		}
	}

	// Convert to response format
	returnResponses := make([]models.ReturnResponse, len(returns))
	for i, ret := range returns {
		returnResponses[i] = ret.ToReturnResponse()
	}

	response := ReturnReportsListResponse{
		Returns: returnResponses,
		Total:   int(total),
	}

	// Build success message
	message := "Return reports retrieved successfully"
	var filters []string

	if date != "" {
		filters = append(filters, "date: "+date)
	}

	if search != "" {
		filters = append(filters, "exact return type: "+search)
	}

	if len(filters) > 0 {
		message += fmt.Sprintf(" (filtered by %s)", strings.Join(filters, " | "))
	}

	utilities.SuccessResponse(c, http.StatusOK, message, response)
}

// GetComplainReports godoc
// @Summary Get complain reports
// @Description Get complain reports with date filtering and exact complain type search, without pagination (logged-in users only)
// @Tags reports
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param date query string false "Filter by date (YYYY-MM-DD format)"
// @Success 200 {object} utilities.Response{data=ComplainReportsListResponse}
// @Failure 400 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Router /api/reports/handout-complains [get]
func (rc *ReportController) GetComplainReports(c *gin.Context) {
	// Parse date parameter
	date := c.Query("date")

	var complains []models.Complain
	var total int64

	// Build query for data retrieval
	query := rc.DB.Model(&models.Complain{})

	// Apply date filter if provided (using updated_at)
	if date != "" {
		// Parse date and validate format
		if parsedDate, err := time.Parse("2006-01-02", date); err != nil {
			utilities.ErrorResponse(c, http.StatusBadRequest, "Invalid date format", "date must be in YYYY-MM-DD format")
			return
		} else {
			// Filter for the entire day (from 00:00:00 to 23:59:59)
			startOfDay := parsedDate.Format("2006-01-02 00:00:00")
			endOfDay := parsedDate.AddDate(0, 0, 1).Format("2006-01-02 00:00:00")
			query = query.Where("updated_at >= ? AND updated_at < ?", startOfDay, endOfDay)
		}
	}

	// Get total count with filters
	if err := query.Count(&total).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to count complain reports", err.Error())
		return
	}

	// Get all complain records with preloaded relationships (no pagination)
	if err := query.Preload("Channel").
		Preload("Store").
		Preload("ProductDetails.Product").
		Preload("UserDetails.Operator.UserRoles.Role").
		Preload("UserDetails.Operator.UserRoles.Assigner").
		Order("id DESC").
		Find(&complains).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve complain reports", err.Error())
		return
	}

	// Convert to response format
	complainResponses := make([]models.ComplainResponse, len(complains))
	for i, comp := range complains {
		complainResponses[i] = comp.ToComplainResponse()
	}

	response := ComplainReportsListResponse{
		Complains: complainResponses,
		Total:     int(total),
	}

	// Build success message
	message := "Complain reports retrieved successfully"
	var filters []string

	if date != "" {
		filters = append(filters, "date: "+date)
	}

	if len(filters) > 0 {
		message += fmt.Sprintf(" (filtered by %s)", strings.Join(filters, " | "))
	}

	utilities.SuccessResponse(c, http.StatusOK, message, response)
}

// Request/Response structs
// BoxUsageDetail represents individual box usage record
type BoxUsageDetail struct {
	Tracking  string    `json:"tracking"`
	OrderID   string    `json:"order_ginee_id"`
	BoxName   string    `json:"box_name"`
	Quantity  int       `json:"quantity"`
	QcBy      uint      `json:"qc_by"`     // Added QC By user ID
	Username  string    `json:"username"`  // Added username
	FullName  string    `json:"full_name"` // Added full name
	CreatedAt time.Time `json:"created_at"`
	Source    string    `json:"source"` // "QC Ribbon" or "QC Online"
}

// BoxCountReport represents box count report
type BoxCountReport struct {
	BoxID       uint             `json:"box_id"`
	BoxCode     string           `json:"box_code"`
	BoxName     string           `json:"box_name"`
	TotalCount  int              `json:"total_count"`
	RibbonCount int              `json:"ribbon_count"`
	OnlineCount int              `json:"online_count"`
	Details     []BoxUsageDetail `json:"details" gorm:"-"` // Added gorm:"-" to ignore this field
}

// BoxCountReportsListResponse represents the response for box count reports
type BoxCountReportsListResponse struct {
	Reports    []BoxCountReport             `json:"reports"`
	Pagination utilities.PaginationResponse `json:"pagination"`
}

// OutboundReportsListResponse represents the response for outbound reports
type OutboundReportsListResponse struct {
	Outbounds []models.OutboundResponse `json:"outbounds"`
	Total     int                       `json:"total"`
}

// ReturnReportsListResponse represents the response for return reports
type ReturnReportsListResponse struct {
	Returns []models.ReturnResponse `json:"returns"`
	Total   int                     `json:"total"`
}

// ComplainReportsListResponse represents the response for complain reports
type ComplainReportsListResponse struct {
	Complains []models.ComplainResponse `json:"complains"`
	Total     int                       `json:"total"`
}
