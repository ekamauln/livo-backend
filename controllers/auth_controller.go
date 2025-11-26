package controllers

import (
	"livo-backend/config"
	"livo-backend/models"
	"livo-backend/utilities"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AuthController struct {
	DB     *gorm.DB
	Config *config.Config
}

// RegisterRequest represents the registration request
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50" example:"john_doe"`
	Email    string `json:"email" binding:"required,email" example:"john@example.com"`
	Password string `json:"password" binding:"required,min=6" example:"password123"`
	FullName string `json:"full_name" binding:"required" example:"John Doe"`
}

// LoginRequest represents the login request
type LoginRequest struct {
	Username string `json:"username" binding:"required" example:"john_doe"`
	Password string `json:"password" binding:"required" example:"password123"`
}

// LoginResponse represents the login response
type LoginResponse struct {
	AccessToken  string              `json:"access_token"`
	RefreshToken string              `json:"refresh_token"`
	User         models.UserResponse `json:"user"`
}

// RefreshTokenRequest represents the refresh token request
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// NewAuthController creates a new auth controller
func NewAuthController(db *gorm.DB, config *config.Config) *AuthController {
	return &AuthController{
		DB:     db,
		Config: config,
	}
}

// Register godoc
// @Summary Register a new user
// @Description Register a new user account
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "Registration request"
// @Success 201 {object} utilities.Response{data=models.UserResponse}
// @Failure 400 {object} utilities.Response
// @Failure 409 {object} utilities.Response
// @Router /api/auth/register [post]
func (ac *AuthController) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utilities.ValidationErrorResponse(c, err)
		return
	}

	// Check if user already exists
	var existingUser models.User
	if err := ac.DB.Where("username = ? OR email = ?", req.Username, req.Email).First(&existingUser).Error; err == nil {
		utilities.ErrorResponse(c, http.StatusConflict, "User already exists", "username or email already taken")
		return
	}

	// Hash password
	hashedPassword, err := utilities.HashPassword(req.Password)
	if err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to hash password", err.Error())
		return
	}

	// Create user
	user := models.User{
		Username: req.Username,
		Email:    req.Email,
		Password: hashedPassword,
		FullName: req.FullName,
		IsActive: true,
	}

	if err := ac.DB.Create(&user).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to create user", err.Error())
		return
	}

	// Assign guest role by default
	var guestRole models.Role
	if err := ac.DB.Where("name = ?", "guest").First(&guestRole).Error; err == nil {
		userRole := models.UserRole{
			UserID:     user.ID,
			RoleID:     guestRole.ID,
			AssignedBy: 1,
		}
		ac.DB.Create(&userRole)
	}

	// Load user with roles
	ac.DB.Preload("UserRoles.Role").First(&user, user.ID)

	utilities.SuccessResponse(c, http.StatusCreated, "User registered successfully", user.ToUserResponse())
}

// Login godoc
// @Summary Login user
// @Description Authenticate user and return JWT tokens
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login request"
// @Success 200 {object} utilities.Response{data=LoginResponse}
// @Failure 400 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Router /api/auth/login [post]
func (ac *AuthController) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utilities.ValidationErrorResponse(c, err)
		return
	}

	// Find user
	var user models.User
	if err := ac.DB.Preload("UserRoles.Role").Where("username = ?", req.Username).First(&user).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusUnauthorized, "Invalid credentials", "user not found")
		return
	}

	// Check password
	if !utilities.CheckPasswordHash(req.Password, user.Password) {
		utilities.ErrorResponse(c, http.StatusUnauthorized, "Invalid credentials", "incorrect password")
		return
	}

	// Check if user is active
	if !user.IsActive {
		utilities.ErrorResponse(c, http.StatusUnauthorized, "Account is inactive", "user account is disabled")
		return
	}

	// Extract roles
	roles := make([]string, len(user.UserRoles))
	for i, userRole := range user.UserRoles {
		roles[i] = userRole.Role.Name
	}

	// Generate tokens
	accessToken, refreshToken, err := utilities.GenerateTokens(
		user.ID,
		user.Username,
		roles,
		ac.Config.JWTSecret,
		ac.Config.JWTExpireHours,
		ac.Config.RefreshTokenExpireDays,
	)
	if err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to generate tokens", err.Error())
		return
	}

	// Save refresh token
	user.RefreshToken = refreshToken
	ac.DB.Save(&user)

	response := LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user.ToUserResponse(),
	}

	utilities.SuccessResponse(c, http.StatusOK, "Login successful", response)
}

// RefreshToken godoc
// @Summary Refresh access token
// @Description Generate new access token using refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RefreshTokenRequest true "Refresh token request"
// @Success 200 {object} utilities.Response{data=LoginResponse}
// @Failure 400 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Router /api/auth/refresh [post]
func (ac *AuthController) RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utilities.ValidationErrorResponse(c, err)
		return
	}

	// Validate refresh token
	claims, err := utilities.ValidateRefreshToken(req.RefreshToken, ac.Config.JWTSecret)
	if err != nil {
		utilities.ErrorResponse(c, http.StatusUnauthorized, "Invalid refresh token", err.Error())
		return
	}

	// Find user
	var user models.User
	if err := ac.DB.Preload("UserRoles.Role").Preload("UserRoles.Assigner").Where("id = ? AND refresh_token = ?", claims.UserID, req.RefreshToken).First(&user).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusUnauthorized, "Invalid refresh token", "refresh token not found")
		return
	}

	// Extract roles
	roles := make([]string, len(user.UserRoles))
	for i, userRole := range user.UserRoles {
		roles[i] = userRole.Role.Name
	}

	// Generate new tokens
	accessToken, refreshToken, err := utilities.GenerateTokens(
		user.ID,
		user.Username,
		roles,
		ac.Config.JWTSecret,
		ac.Config.JWTExpireHours,
		ac.Config.RefreshTokenExpireDays,
	)
	if err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to generate tokens", err.Error())
		return
	}

	// Update refresh token
	user.RefreshToken = refreshToken
	ac.DB.Save(&user)

	response := LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user.ToUserResponse(),
	}

	utilities.SuccessResponse(c, http.StatusOK, "Token refreshed successfully", response)
}

// Logout godoc
// @Summary Logout user
// @Description Logout user by invalidating refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Router /api/auth/logout [post]
func (ac *AuthController) Logout(c *gin.Context) {
	userID := c.GetUint("user_id")

	// Clear refresh token
	if err := ac.DB.Model(&models.User{}).Where("id = ?", userID).Update("refresh_token", "").Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to logout", err.Error())
		return
	}

	utilities.SuccessResponse(c, http.StatusOK, "Logout successful", nil)
}
