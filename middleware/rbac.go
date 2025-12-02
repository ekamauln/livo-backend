package middleware

import (
	"livo-backend/utilities"
	"net/http"

	"github.com/gin-gonic/gin"
)

// RequireRoles middleware checks if user has any of the required roles
func RequireRoles(requiredRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roles, exists := c.Get("roles")
		if !exists {
			utilities.ErrorResponse(c, http.StatusUnauthorized, "No roles found in token", "missing roles")
			c.Abort()
			return
		}

		userRoles, ok := roles.([]string)
		if !ok {
			utilities.ErrorResponse(c, http.StatusInternalServerError, "Invalid roles format", "roles format error")
			c.Abort()
			return
		}

		// Check if user has any of the required roles
		hasRole := false
		for _, requiredRole := range requiredRoles {
			for _, userRole := range userRoles {
				if userRole == requiredRole {
					hasRole = true
					break
				}
			}
			if hasRole {
				break
			}
		}

		if !hasRole {
			utilities.ErrorResponse(c, http.StatusForbidden, "Insufficient permissions", "access denied")
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireCoordinatorRoles for endpoints that require coordinator role
func RequireCoordinatorRoles() gin.HandlerFunc {
	return RequireRoles("superadmin", "coordinator")
}

// RequireAdminRoles for endpoints that require admin role
func RequireAdminRoles() gin.HandlerFunc {
	return RequireRoles("superadmin", "admin")
}
