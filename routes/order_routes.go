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
		order.GET("", orderController.GetOrders)                                         // Get all orders (with optional search and date filtering)
		order.GET("/:id", orderController.GetOrder)                                      // Get specific order by ID (full details)
		order.POST("/bulk", orderController.BulkCreateOrders)                            // Create multiple orders
		order.PUT("/:id", orderController.UpdateOrder)                                   // Update order details
		order.PUT("/:id/complained", orderController.UpdateOrderComplainedStatus)        // Update order complained status
		order.PUT("/:id/qc-process", orderController.QCProcessStatusOrder)               // Update order QC process status
		order.PUT("/:id/picking-completed", orderController.PickingCompletedStatusOrder) // Update order picking complete
	}

	// Order management routes (admin only)
	order.Use(middleware.RequireAdminRoles())
	{
		order.POST("/:id/duplicate", orderController.DuplicateOrder) // Duplicate an order
		order.PUT("/:id/cancel", orderController.CancelOrder)        // Cancel an order
	}

	// Order management routes (coordinator only)
	orderCoordinator := api.Group("/orders")
	orderCoordinator.Use(middleware.AuthMiddleware(cfg))
	orderCoordinator.Use(middleware.RequireCoordinatorRoles())
	{
		orderCoordinator.PUT("/:id/pending-pick", orderController.PendingPickOrders) // Pending an picked orders
		orderCoordinator.GET("/assigned", orderController.GetAssignedOrders)         // Get all assigned orders for current date
		orderCoordinator.PUT("/:id/assign-picker", orderController.AssignPicker)     // Assign picker to order
	}
}

// SetupMobileOrderRoutes configures mobile order-related routes
func SetupMobileOrderRoutes(api *gin.RouterGroup, cfg *config.Config, mobileOrderController *controllers.MobileOrderController) {
	// Mobile order routes (authenticated)
	mobileOrder := api.Group("/mobile/orders")
	mobileOrder.Use(middleware.AuthMiddleware(cfg))
	{
		// Mobile order routes
		mobileOrder.GET("", mobileOrderController.GetMyPickingOrders)                // Get my ongoing picking orders
		mobileOrder.GET(":id", mobileOrderController.GetMyPickingOrder)              // Get my ongoing picking order
		mobileOrder.PUT(":id/pending-pick", mobileOrderController.PendingPickOrders) // Pending picking order
		mobileOrder.PUT(":id/complete", mobileOrderController.CompletePickingOrder)  // Complete order
	}
	mobileOrderCoordinator := api.Group("/mobile/orders")
	mobileOrderCoordinator.Use(middleware.AuthMiddleware(cfg))
	mobileOrderCoordinator.Use(middleware.RequireCoordinatorRoles())
	{
		mobileOrderCoordinator.POST("/bulk-assign-picker", mobileOrderController.BulkAssignPicker) // Bulk assign pickers to orders
		mobileOrderCoordinator.GET("/picked-orders", mobileOrderController.GetMobilePickedOrders)  // Get picked orders for coordinator
	}
}
