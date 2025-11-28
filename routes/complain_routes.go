package routes

import (
	"livo-backend/config"
	"livo-backend/controllers"
	"livo-backend/middleware"

	"github.com/gin-gonic/gin"
)

// SetupComplainRoutes configures complain-related routes
func SetupComplainRoutes(api *gin.RouterGroup, cfg *config.Config, complainController *controllers.ComplainController) {
	// Complain routes (authenticated)
	complain := api.Group("/complains")
	complain.Use(middleware.AuthMiddleware(cfg))
	{
		// Public complain routes
		complain.POST("", complainController.CreateComplain)                     // Create new complain
		complain.GET("", complainController.GetComplains)                        // Get all complains (with optional search)
		complain.GET("/:id", complainController.GetComplain)                     // Get complain by ID
		complain.PUT("/:id/solution", complainController.UpdateSolutionComplain) // Update complain solution and total fee
		complain.PUT("/:id/check", complainController.UpdateCheckComplain)       // Update complain checked status
	}
}
