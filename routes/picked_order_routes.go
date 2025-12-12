package routes

import (
	"livo-backend/config"
	"livo-backend/controllers"
	"livo-backend/middleware"

	"github.com/gin-gonic/gin"
)

// SetupPickedOrderRoutes configures pick order-related routes
func SetupPickedOrderRoutes(api *gin.RouterGroup, cfg *config.Config, pickedOrderController *controllers.PickedOrderController) {
	// Pick Order routes (authenticated)
	pickedOrders := api.Group("/picked-orders")
	pickedOrders.Use(middleware.AuthMiddleware(cfg))
	{
		// Public pick order routes
		pickedOrders.GET("", pickedOrderController.GetPickedOrders) // Get all pick orders (with optional search and date filtering)
		pickedOrders.GET("/:id", pickedOrderController.GetPickedOrder)
	}
}
