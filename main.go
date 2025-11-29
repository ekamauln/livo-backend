package main

import (
	"fmt"
	"livo-backend/config"
	"livo-backend/controllers"
	_ "livo-backend/docs" // This is required for Swagger
	"livo-backend/migrations"
	"livo-backend/routes"
	"log"
)

// @title Livotech Backend Service API
// @version 1.0
// @description A comprehensive user management backend service with JWT authentication and role-based access control
// @contact.name API Support
// @contact.email support@livotech.com
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.
func main() {
	log.Println("🚀 Starting Livotech Backend Service...")

	// Load configuration
	log.Println("📝 Loading configuration...")
	cfg := config.LoadConfig()
	log.Println("✓ Configuration loaded successfully")

	// Connect to database with retry logic
	log.Println("🔌 Connecting to database...")
	config.ConnectDatabase(cfg)

	// Run migrations
	log.Println("🔄 Running database migrations...")
	db := config.GetDB()
	migrations.AutoMigrate(db) // No error handling needed, it's handled inside the function

	// Initialize controllers
	log.Println("🎮 Initializing controllers...")
	authController := controllers.NewAuthController(db, cfg)
	userManagerController := controllers.NewUserManagerController(db)
	boxController := controllers.NewBoxController(db)
	channelController := controllers.NewChannelController(db)
	mobileChannelController := controllers.NewMobileChannelController(db)
	expeditionController := controllers.NewExpeditionController(db)
	productController := controllers.NewProductController(db)
	storeController := controllers.NewStoreController(db)
	mobileStoreController := controllers.NewMobileStoreController(db)
	qcRibbonController := controllers.NewQcRibbonController(db)
	ribbonFlowController := controllers.NewRibbonFlowController(db)
	qcOnlineController := controllers.NewQcOnlineController(db)
	onlineFlowController := controllers.NewOnlineFlowController(db)
	pickedOrderController := controllers.NewPickedOrderController(db)
	outboundController := controllers.NewOutboundController(db)
	returnController := controllers.NewReturnController(db)
	mobileReturnController := controllers.NewMobileReturnController(db)
	complainController := controllers.NewComplainController(db)
	orderController := controllers.NewOrderController(db)
	mobileOrderController := controllers.NewMobileOrderController(db)
	log.Println("✓ Controllers initialized successfully")

	// Setup routes
	log.Println("🛣️  Setting up routes...")
	router := routes.SetupRoutes(cfg, authController, userManagerController, boxController, channelController, mobileChannelController, expeditionController, productController, storeController, mobileStoreController, qcRibbonController, ribbonFlowController, qcOnlineController, onlineFlowController, pickedOrderController, outboundController, returnController, mobileReturnController, complainController, orderController, mobileOrderController)
	log.Println("✓ Routes configured successfully")

	// Build API URL from config
	apiURL := fmt.Sprintf("http://%s:%s", cfg.APIHost, cfg.Port)

	// Start server
	log.Println("════════════════════════════════════════════════════════════")
	log.Printf("✓ Server ready on port %s", cfg.Port)
	log.Printf("📊 Health check: %s/health", apiURL)
	log.Printf("📚 API documentation: %s/docs", apiURL)
	log.Printf("📖 Swagger UI: %s/swagger/index.html", apiURL)
	log.Println("════════════════════════════════════════════════════════════")

	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatal("❌ Failed to start server:", err)
	}
}
