package middleware

import (
	"livo-backend/config"
	"livo-backend/utilities"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware validates JWT token
func AuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			utilities.ErrorResponse(c, http.StatusUnauthorized, "Authorization header is required", "missing authorization header")
			c.Abort()
			return
		}

		bearerToken := strings.Split(authHeader, " ")
		if len(bearerToken) != 2 || bearerToken[0] != "Bearer" {
			utilities.ErrorResponse(c, http.StatusUnauthorized, "Invalid authorization header format", "invalid bearer token format")
			c.Abort()
			return
		}

		claims, err := utilities.ValidateToken(bearerToken[1], cfg.JWTSecret)
		if err != nil {
			utilities.ErrorResponse(c, http.StatusUnauthorized, "Invalid token", err.Error())
			c.Abort()
			return
		}

		// Set user claims in context
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("roles", claims.Roles)
		c.Next()
	}
}
