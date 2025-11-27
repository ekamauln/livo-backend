package routes

import (
	"livo-backend/config"
	"livo-backend/controllers"
	"livo-backend/middleware"

	"github.com/gin-gonic/gin"
)

// SetupChannelRoutes configures channel-related routes
func SetupChannelRoutes(api *gin.RouterGroup, cfg *config.Config, channelController *controllers.ChannelController) {
	// Channel routes (authenticated)
	channel := api.Group("/channels")
	channel.Use(middleware.AuthMiddleware(cfg))
	{
		// Public channel routes
		channel.GET("", channelController.GetChannels)          // Get all channels (with optional search)
		channel.GET("/:id", channelController.GetChannel)       // Get channel by ID
		channel.POST("", channelController.CreateChannel)       // Create new channel
		channel.PUT("/:id", channelController.UpdateChannel)    // Update channel by ID
		channel.DELETE("/:id", channelController.RemoveChannel) // Delete channel by ID
	}
}

func SetupMobileChannelRoutes(api *gin.RouterGroup, cfg *config.Config, mobileChannelController *controllers.MobileChannelController) {
	// Mobile channel routes (public)
	mobileChannel := api.Group("/mobile/channels")
	{
		// Public mobile channel routes
		mobileChannel.GET("", mobileChannelController.GetMobileChannels) // Get all channels for mobile (with optional search)
	}
}
