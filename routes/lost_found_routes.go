package routes

import (
	"livo-backend/config"
	"livo-backend/controllers"
	"livo-backend/middleware"

	"github.com/gin-gonic/gin"
)

// SetupLostFoundRoutes configures lost and found related routes
func SetupLostFoundRoutes(api *gin.RouterGroup, cfg *config.Config, lostFoundController *controllers.LostFoundController) {
	// Lost and found routes (authenticated)
	lostFound := api.Group("/lost-founds")
	lostFound.Use(middleware.AuthMiddleware(cfg))
	{
		lostFound.POST("/", lostFoundController.CreateLostFound)
		lostFound.GET("/", lostFoundController.GetLostFounds)
		lostFound.GET("/:id", lostFoundController.GetLostFound)
		lostFound.PUT("/:id", lostFoundController.UpdateLostFound)
		lostFound.DELETE("/:id", lostFoundController.RemoveLostFound)
	}
}
