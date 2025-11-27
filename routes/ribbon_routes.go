package routes

import (
	"livo-backend/config"
	"livo-backend/controllers"
	"livo-backend/middleware"

	"github.com/gin-gonic/gin"
)

// SetupMbRibbonRoutes configures mb-ribbon-related routes
func SetupQcRibbonRoutes(api *gin.RouterGroup, cfg *config.Config, qcRibbonController *controllers.QcRibbonController) {
	// Qc-Ribbon routes (authenticated)
	qcRibbon := api.Group("/ribbons/qc-ribbons")
	qcRibbon.Use(middleware.AuthMiddleware(cfg))
	{
		// Public qc-ribbon routes
		qcRibbon.POST("", qcRibbonController.CreateQcRibbon)         // Create new qc-ribbon
		qcRibbon.GET("", qcRibbonController.GetQcRibbons)            // Get all qc-ribbons (with optional search and date filtering)
		qcRibbon.GET("/:id", qcRibbonController.GetQcRibbon)         // Get qc-ribbon by ID
		qcRibbon.GET("/chart", qcRibbonController.GetChartQcRibbons) // Get qc-ribbon counts per day for current month
	}
}
