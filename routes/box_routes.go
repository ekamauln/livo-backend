package routes

import (
	"livo-backend/config"
	"livo-backend/controllers"
	"livo-backend/middleware"

	"github.com/gin-gonic/gin"
)

// SetupBoxRoutes configures box-related routes
func SetupBoxRoutes(api *gin.RouterGroup, cfg *config.Config, boxController *controllers.BoxController) {
	// Box routes (authenticated)
	box := api.Group("/boxes")
	box.Use(middleware.AuthMiddleware(cfg))
	{
		// Public box routes
		box.POST("", boxController.CreateBox)       // Create new box
		box.GET("", boxController.GetBoxes)         // Get all boxes (with optional search)
		box.GET("/:id", boxController.GetBox)       // Get box by ID
		box.PUT("/:id", boxController.UpdateBox)    // Update box by ID
		box.DELETE("/:id", boxController.RemoveBox) // Delete box by ID
	}
}
