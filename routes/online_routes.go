package routes

import (
	"livo-backend/config"
	"livo-backend/controllers"
	"livo-backend/middleware"

	"github.com/gin-gonic/gin"
)

// SetupQcOnlineRoutes configures qc-online-related routes
func SetupQcOnlineRoutes(api *gin.RouterGroup, cfg *config.Config, qcOnlineController *controllers.QcOnlineController) {
	// Qc-Online routes (authenticated)
	qcOnline := api.Group("/onlines/qc-onlines")
	qcOnline.Use(middleware.AuthMiddleware(cfg))
	{
		// Public qc-online routes
		qcOnline.GET("", qcOnlineController.GetQcOnlines)            // Get all qc-onlines (with optional search and date filtering)
		qcOnline.GET("/:id", qcOnlineController.GetQcOnline)         // Get qc-online by ID
		qcOnline.POST("", qcOnlineController.CreateQcOnline)         // Create new qc-online
		qcOnline.GET("/chart", qcOnlineController.GetChartQcOnlines) // Get qc-online counts per day for current month
	}
}

// SetupOnlineFlowRoutes configures online-flow-related routes
func SetupOnlineFlowRoutes(api *gin.RouterGroup, cfg *config.Config, onlineFlowController *controllers.OnlineFlowController) {
	// Online-flow routes (authenticated)
	onlineFlow := api.Group("/onlines/online-flows")
	onlineFlow.Use(middleware.AuthMiddleware(cfg))
	{
		// Public online-flow routes
		onlineFlow.GET("", onlineFlowController.GetOnlineFlows)          // Get all online flows (with optional search and date filtering)
		onlineFlow.GET("/:tracking", onlineFlowController.GetOnlineFlow) // Get online flow by tracking number
	}
}
