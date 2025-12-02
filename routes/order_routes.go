package routes

import (
	"livo-backend/config"
	"livo-backend/controllers"
	"livo-backend/middleware"

	"github.com/gin-gonic/gin"
)

// SetupOrderRoutes configures order-related routes
func SetupOrderRoutes(api *gin.RouterGroup, cfg *config.Config, orderController *controllers.OrderController) {
	// Order routes (authenticated)
	order := api.Group("/orders")
	order.Use(middleware.AuthMiddleware(cfg))
	{
		// Public order routes
		order.GET("", orderController.GetOrders)                                  // Get all orders (with optional search and date filtering)
		order.GET("/:id", orderController.GetOrder)                               // Get specific order by ID (full details)
		order.POST("/bulk", orderController.BulkCreateOrders)                     // Create multiple orders
		order.PUT("/:id", orderController.UpdateOrder)                            // Update order details
		order.PUT("/:id/complained", orderController.UpdateOrderComplainedStatus) // Update order complained status
	}

	// Order management routes (admin only)
	order.Use(middleware.RequireAdminRoles())
	{
		order.POST("/:id/duplicate", orderController.DuplicateOrder) // Duplicate an order
		order.PUT("/:id/cancel", orderController.CancelOrder)        // Cancel an order
	}

	// Order management routes (coordinator only)
	order.Use(middleware.RequireCoordinatorRoles())
	{
		order.PUT("/:id/assign-picker", orderController.AssignPicker) // Assign picker to order
	}
}

// SetupMobileOrderRoutes configures mobile order-related routes
func SetupMobileOrderRoutes(api *gin.RouterGroup, cfg *config.Config, mobileOrderController *controllers.MobileOrderController) {
	// Mobile order routes (authenticated)
	mobileOrder := api.Group("/mobile/orders")
	mobileOrder.Use(middleware.AuthMiddleware(cfg))
	{
		// Mobile order routes
		mobileOrder.GET("", mobileOrderController.GetMobileOrders)                  // Get all orders for pickers with search capability
		mobileOrder.GET(":id", mobileOrderController.GetMobileOrder)                // Get specific order by ID
		mobileOrder.GET(":id/pick", mobileOrderController.PickingOrder)             // Pick order
		mobileOrder.GET("/my-picking", mobileOrderController.GetMyPickingOrders)    // Get my ongoing picking orders
		mobileOrder.GET(":id/complete", mobileOrderController.CompletePickingOrder) // Complete order
		mobileOrder.PUT(":id/pending", mobileOrderController.PendingPickOrders)     // Pending picking order
	}
}
