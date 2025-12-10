package routes

import (
	"livo-backend/config"
	"livo-backend/controllers"
	"livo-backend/middleware"

	"github.com/gin-gonic/gin"
)

// SetupReportRoutes configures report-related routes
func SetupReportRoutes(api *gin.RouterGroup, cfg *config.Config, reportController *controllers.ReportController) {
	// Report routes (authenticated)
	report := api.Group("/reports")
	report.Use(middleware.AuthMiddleware(cfg))
	{
		// Public report routes
		report.GET("/boxes-count", reportController.GetBoxReports)            // Get box count reports
		report.GET("/handout-outbounds", reportController.GetOutboundReports) // Get handout outbound reports
	}
}
