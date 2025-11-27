package routes

import (
	"livo-backend/config"
	"livo-backend/controllers"
	"livo-backend/middleware"

	"github.com/gin-gonic/gin"
)

// SetupOutboundRoutes configures outbound-related routes
func SetupOutboundRoutes(api *gin.RouterGroup, cfg *config.Config, outboundController *controllers.OutboundController) {
	// Outbound routes (authenticated)
	outbound := api.Group("/outbounds")
	outbound.Use(middleware.AuthMiddleware(cfg))
	{
		// Public outbound routes
		outbound.GET("", outboundController.GetOutbounds)            // Get all outbounds (with optional search)
		outbound.GET("/:id", outboundController.GetOutbound)         // Get outbound by ID
		outbound.POST("", outboundController.CreateOutbound)         // Create new outbound
		outbound.PUT("/:id", outboundController.UpdateOutbound)      // Update outbound by ID
		outbound.GET("/chart", outboundController.GetChartOutbounds) // Get outbound counts per day for current month
	}
}
