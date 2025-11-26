package routes

import (
	"livo-backend/config"
	"livo-backend/controllers"
	"livo-backend/middleware"

	"github.com/gin-gonic/gin"
)

// SetupUserManagerRoutes configures user manager-related routes
func SetupUserManagerRoutes(api *gin.RouterGroup, cfg *config.Config, userManagerController *controllers.UserManagerController) {
	// User manager routes (authenticated + role-based)
	userManager := api.Group("/user-manager")
	userManager.Use(middleware.AuthMiddleware(cfg))
	{
		// Get all roles - public to all authenticated users (no role restriction)
		userManager.GET("/roles", userManagerController.GetRoles)

		// Get all users - public to all authenticated users (no role restriction)
		userManager.GET("/users", userManagerController.GetUsers)
		userManager.GET("/users/:id", userManagerController.GetUser)                     // Get user by ID
		userManager.PUT("/users/:id/password", userManagerController.UpdateUserPassword) // Update user password
		userManager.PUT("/users/:id/profile", userManagerController.UpdateUserProfile)   // Update user profile

		// User management (coordinator only)
		users := userManager.Group("/users")
		users.Use(middleware.RequireCoordinatorRoles())
		{
			users.PUT("/:id/status", userManagerController.UpdateUserStatus)     // Update user status (active/inactive)
			users.POST("", userManagerController.CreateUser)                     // Create new user
			users.DELETE("/:id", userManagerController.DeleteUser)               // Delete user
		}

		// Role assignment (coordinator only)
		roleAssignment := userManager.Group("/users/:id/roles") // Assign or remove roles to/from a user
		roleAssignment.Use(middleware.RequireCoordinatorRoles())
		{
			roleAssignment.POST("", userManagerController.AssignRole)   // Assign role to user
			roleAssignment.DELETE("", userManagerController.RemoveRole) // Remove role from user
		}
	}
}
