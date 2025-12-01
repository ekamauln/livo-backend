package controllers

import (
	"livo-backend/models"
	"livo-backend/utilities"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UserController struct {
	DB *gorm.DB
}

// NewUserController creates a new user controller
func NewUserController(db *gorm.DB) *UserController {
	return &UserController{DB: db}
}

// GetProfile godoc
// @Summary Get user profile
// @Description Get current user's profile information
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} utilities.Response{data=models.UserResponse}
// @Failure 401 {object} utilities.Response
// @Router /api/user/profile [get]
func (uc *UserController) GetProfile(c *gin.Context) {
	userID := c.GetUint("user_id")

	var user models.User
	if err := uc.DB.Preload("UserRoles.Role").First(&user, userID).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusNotFound, "User not found", err.Error())
		return
	}

	utilities.SuccessResponse(c, http.StatusOK, "Profile retrieved successfully", user.ToUserResponse())
}

// UpdateProfile godoc
// @Summary Update user profile
// @Description Update current user's profile information
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body UpdateProfileRequest true "Update profile request"
// @Success 200 {object} utilities.Response{data=models.UserResponse}
// @Failure 400 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Router /api/user/profile [put]
func (uc *UserController) UpdateProfile(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utilities.ValidationErrorResponse(c, err)
		return
	}

	var user models.User
	if err := uc.DB.First(&user, userID).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusNotFound, "User not found", err.Error())
		return
	}

	// Update user fields
	if req.FullName != "" {
		user.FullName = req.FullName
	}
	if req.Email != "" {
		// Check if email is already taken by another user
		var existingUser models.User
		if err := uc.DB.Where("email = ? AND id != ?", req.Email, userID).First(&existingUser).Error; err == nil {
			utilities.ErrorResponse(c, http.StatusConflict, "Email already taken", "email already exists")
			return
		}
		user.Email = req.Email
	}

	// Update password if provided
	if req.Password != "" {
		hashedPassword, err := utilities.HashPassword(req.Password)
		if err != nil {
			utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to hash password", err.Error())
			return
		}
		user.Password = hashedPassword
	}

	if err := uc.DB.Save(&user).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to update profile", err.Error())
		return
	}

	// Load user with roles
	uc.DB.Preload("UserRoles.Role").Preload("UserRoles.Assigner").First(&user, user.ID)

	utilities.SuccessResponse(c, http.StatusOK, "Profile updated successfully", user.ToUserResponse())
}

// UpdateProfileRequest represents the update profile request
type UpdateProfileRequest struct {
	FullName string `json:"full_name,omitempty" example:"John Doe"`
	Email    string `json:"email,omitempty" example:"john@example.com"`
	Password string `json:"password,omitempty" example:"newpassword123"`
}
