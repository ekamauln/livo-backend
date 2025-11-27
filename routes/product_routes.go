package routes

import (
	"livo-backend/config"
	"livo-backend/controllers"
	"livo-backend/middleware"

	"github.com/gin-gonic/gin"
)

// SetupProductRoutes configures product-related routes
func SetupProductRoutes(api *gin.RouterGroup, cfg *config.Config, productController *controllers.ProductController) {
	// Product routes (authenticated)
	product := api.Group("/products")
	product.Use(middleware.AuthMiddleware(cfg))
	{
		// Public product routes
		product.GET("", productController.GetProducts)    // Get all products (with optional search)
		product.GET("/:id", productController.GetProduct) // Get product by ID

		// Admin product management routes (coordinator roles)
		productAdmin := product.Group("")
		productAdmin.Use(middleware.RequireCoordinatorRoles())
		{
			productAdmin.POST("", productController.CreateProduct)       // Create new product
			productAdmin.PUT("/:id", productController.UpdateProduct)    // Update product by ID
			productAdmin.DELETE("/:id", productController.RemoveProduct) // Delete product by ID
		}
	}
}
