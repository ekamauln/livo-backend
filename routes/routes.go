package routes

import (
	"fmt"
	"livo-backend/config"
	"livo-backend/controllers"
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// SetupRoutes configures all routes for the application
func SetupRoutes(cfg *config.Config, authController *controllers.AuthController, userManagerController *controllers.UserManagerController, boxController *controllers.BoxController, channelController *controllers.ChannelController, mobileChannelController *controllers.MobileChannelController, expeditionController *controllers.ExpeditionController) *gin.Engine {
	// Set Gin mode
	gin.SetMode(cfg.GinMode)

	router := gin.Default()

	// CORS middleware
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	corsConfig.AllowHeaders = append(corsConfig.AllowHeaders, "Authorization")
	router.Use(cors.New(corsConfig))
	corsConfig = cors.Config{
		AllowOrigins: strings.Split(cfg.CORSAllowedOrigins, ","),
		AllowMethods: strings.Split(cfg.CORSAllowedMethods, ","),
		AllowHeaders: []string{
			"Origin",
			"Content-Length",
			"Content-Type",
			"Authorization",
			"Accept",
			"X-Requested-With",
		},
		ExposeHeaders: []string{
			"Content-Length",
			"Content-Type",
		},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
	router.Use(cors.New(corsConfig))

	// Set trusted proxies for security
	router.SetTrustedProxies([]string{"127.0.0.1", "192.168.31.52", "192.168.31.53", "192.168.31.54", "192.168.31.55", "::1"})

	// Serve static files from static directory
	router.Static("/static", "./static")

	// Specifically handle favicon requests
	router.GET("/favicon.ico", func(c *gin.Context) {
		c.File("./static/favicon.ico")
	})

	// Swagger documentation (keep original endpoint for compatibility)
	router.GET("/swagger/*any", func(c *gin.Context) {
		// Dynamic URL based on the request
		scheme := "http"
		if c.Request.TLS != nil {
			scheme = "https"
		}
		host := c.Request.Host
		url := ginSwagger.URL(fmt.Sprintf("%s://%s/swagger/doc.json", scheme, host))
		ginSwagger.WrapHandler(swaggerFiles.Handler, url)(c)
	})

	// RapiDoc documentation (new primary documentation)
	router.GET("/docs", func(c *gin.Context) {
		// Dynamic URLs based on the request
		scheme := "http"
		if c.Request.TLS != nil {
			scheme = "https"
		}
		host := c.Request.Host
		baseURL := fmt.Sprintf("%s://%s", scheme, host)
		specURL := fmt.Sprintf("%s/swagger/doc.json", baseURL)

		html := `<!DOCTYPE html>
<html>
<head>
    <title>Livotech Backend Service API Documentation</title>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <script type="module" src="https://unpkg.com/rapidoc@9.3.4/dist/rapidoc-min.js"></script>
</head>
<body>
    <rapi-doc 
        spec-url="` + specURL + `"
        theme="dark"
        render-style="focused"
        schema-style="table"
        default-schema-tab="schema"
        show-header="true"
        show-info="true"
        allow-authentication="true"
        allow-server-selection="false"
        allow-api-list-style-selection="false"
        show-components="true"
        schema-description-expanded="true"
        default-api-server="` + baseURL + `"
        api-key-name="Authorization"
        api-key-location="header"
        api-key-value=""
        layout="row"
        sort-tags="true"
        nav-bg-color="#1e293b"
        nav-text-color="#f1f5f9"
        nav-hover-bg-color="#334155"
        nav-hover-text-color="#ffffff"
        nav-accent-color="#3b82f6"
        primary-color="#3b82f6"
        bg-color="#0f172a"
        text-color="#f1f5f9"
        header-color="#1e293b"
        regular-color="#64748b"
        font-size="default"
        update-route="false"
        route-prefix="#"
        sort-endpoints-by="method"
        goto-path=""
        fill-request-fields-with-example="true"
        persist-auth="true"
        use-path-in-nav-bar="false"
        nav-item-spacing="default"
        show-method-in-nav-bar="as-colored-block"
        response-area-height="40%"
        show-curl-before-try="true"
        schema-expand-level="1"
        schema-hide-read-only="never"
        fetch-credentials="omit"
        match-paths=""
        match-type="includes"
    >
        <div slot="overview">
            <h2>Welcome to Livotech Backend Service API</h2>
            <p>A comprehensive user management backend service with JWT authentication and role-based access control.</p>
            <p><strong>Authentication:</strong> This API uses Bearer token authentication. Include your JWT token in the Authorization header with the format: <code>Bearer your-token-here</code></p>
        </div>
    </rapi-doc>
</body>
</html>`
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
	})

	// Redirect root to docs for better UX
	router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/docs")
	})

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":    "ok",
			"message":   "Livotech Backend Service is running",
			"timestamp": time.Now().Format("02 January 2006 - 15:04:05"),
		})
	})

	// API routes
	api := router.Group("/api")

	// Setup route groups
	SetupAuthRoutes(api, cfg, authController)
	SetupUserManagerRoutes(api, cfg, userManagerController)
	SetupBoxRoutes(api, cfg, boxController)
	SetupChannelRoutes(api, cfg, channelController)
	SetupMobileChannelRoutes(api, cfg, mobileChannelController)
	SetupExpeditionRoutes(api, cfg, expeditionController)

	return router
}
