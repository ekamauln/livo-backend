package controllers

import (
	"livo-backend/models"
	"livo-backend/utilities"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UserManagerController struct {
	DB *gorm.DB
}

// NewUserManagerController creates a new user manager controller
func NewUserManagerController(db *gorm.DB) *UserManagerController {
	return &UserManagerController{DB: db}
}

// GetUsers godoc
// @Summary Get all users with search capability
// @Description Get list of all users. Optional search by username or full name.
// @Tags user-manager
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Param search query string false "Search by username or full name"
// @Success 200 {object} utilities.Response{data=UsersListResponse}
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Router /api/user-manager/users [get]
func (umc *UserManagerController) GetUsers(c *gin.Context) {
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	// Parse search parameter
	search := strings.TrimSpace(c.Query("search"))

	var users []models.User
	var total int64

	// Build base query
	query := umc.DB.Model(&models.User{})

	// Add search conditions if search parameter is provided
	if search != "" {
		searchCondition := "username ILIKE ? OR full_name ILIKE ?"
		searchPattern := "%" + search + "%"
		query = query.Where(searchCondition, searchPattern, searchPattern)
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to count users", err.Error())
		return
	}

	// Get users with pagination and order by ID ascending
	if err := query.Order("id ASC").Preload("UserRoles.Role").Preload("UserRoles.Assigner").Limit(limit).Offset(offset).Find(&users).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve users", err.Error())
		return
	}

	// Convert to response format
	userResponses := make([]models.UserResponse, len(users))
	for i, user := range users {
		userResponses[i] = user.ToUserResponse()
	}

	response := UsersListResponse{
		Users: userResponses,
		Pagination: utilities.PaginationResponse{
			Page:  page,
			Limit: limit,
			Total: int(total),
		},
	}

	utilities.SuccessResponse(c, http.StatusOK, "Users retrieved successfully", response)
}

// GetUser godoc
// @Summary Get user by ID
// @Description Get specific user information.
// @Tags user-manager
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Success 200 {object} utilities.Response{data=models.UserResponse}
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Failure 404 {object} utilities.Response
// @Router /api/user-manager/users/{id} [get]
func (umc *UserManagerController) GetUser(c *gin.Context) {
	userID := c.Param("id")

	var user models.User
	if err := umc.DB.Preload("UserRoles.Role").First(&user, userID).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusNotFound, "User not found", err.Error())
		return
	}

	utilities.SuccessResponse(c, http.StatusOK, "User retrieved successfully", user.ToUserResponse())
}

// UpdateUserStatus godoc
// @Summary Update user status
// @Description Activate or deactivate a user.
// @Tags user-manager
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Param request body UpdateUserStatusRequest true "Update status request"
// @Success 200 {object} utilities.Response{data=models.UserResponse}
// @Failure 400 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Failure 404 {object} utilities.Response
// @Router /api/user-manager/users/{id}/status [put]
func (umc *UserManagerController) UpdateUserStatus(c *gin.Context) {
	userID := c.Param("id")

	var req UpdateUserStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utilities.ValidationErrorResponse(c, err)
		return
	}

	var user models.User
	if err := umc.DB.First(&user, userID).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusNotFound, "User not found", err.Error())
		return
	}

	user.IsActive = req.IsActive
	if err := umc.DB.Save(&user).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to update user status", err.Error())
		return
	}

	// Load user with roles
	umc.DB.Preload("UserRoles.Role").First(&user, user.ID)

	utilities.SuccessResponse(c, http.StatusOK, "User status updated successfully", user.ToUserResponse())
}

// AssignRole godoc
// @Summary Assign role to user
// @Description Assign a role to a user.
// @Tags user-manager
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Param request body AssignRoleRequest true "Assign role request"
// @Success 200 {object} utilities.Response{data=models.UserResponse}
// @Failure 400 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Failure 404 {object} utilities.Response
// @Router /api/user-manager/users/{id}/roles [post]
func (umc *UserManagerController) AssignRole(c *gin.Context) {
	userID := c.Param("id")

	var req AssignRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utilities.ValidationErrorResponse(c, err)
		return
	}

	// Find target user
	var user models.User
	if err := umc.DB.Preload("UserRoles.Role").First(&user, userID).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusNotFound, "User not found", err.Error())
		return
	}

	// Find role
	var role models.Role
	if err := umc.DB.Where("name = ?", req.RoleName).First(&role).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusNotFound, "Role not found", err.Error())
		return
	}

	// Check if user already has this role
	for _, userRole := range user.UserRoles {
		if userRole.RoleID == role.ID {
			utilities.ErrorResponse(c, http.StatusConflict, "User already has this role", "role already assigned")
			return
		}
	}

	// Check permission hierarchy (get current user's roles)
	currentUserRoles, _ := c.Get("roles")
	currentRoles := currentUserRoles.([]string)

	// Get current user's highest role level
	hierarchy := models.GetRoleHierarchy()
	currentMaxLevel := 0
	for _, roleName := range currentRoles {
		if level, exists := hierarchy[roleName]; exists && level > currentMaxLevel {
			currentMaxLevel = level
		}
	}

	// Check if current user can assign this role
	targetRoleLevel, exists := hierarchy[req.RoleName]
	if !exists || currentMaxLevel < targetRoleLevel {
		utilities.ErrorResponse(c, http.StatusForbidden, "Insufficient permissions to assign this role", "permission denied")
		return
	}

	// Get current user ID from context
	currentUserID, exists := c.Get("user_id")
	if !exists {
		utilities.ErrorResponse(c, http.StatusUnauthorized, "User not authenticated", "user_id not found in context")
		return
	}

	// Assign role
	userRole := models.UserRole{
		UserID:     user.ID,
		RoleID:     role.ID,
		AssignedBy: currentUserID.(uint),
	}

	if err := umc.DB.Create(&userRole).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to assign role", err.Error())
		return
	}

	// Reload user with updated roles
	umc.DB.Preload("UserRoles.Role").First(&user, user.ID)

	utilities.SuccessResponse(c, http.StatusOK, "Role assigned successfully", user.ToUserResponse())
}

// RemoveRole godoc
// @Summary Remove role from user
// @Description Remove a role from a user.
// @Tags user-manager
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Param request body RemoveRoleRequest true "Remove role request"
// @Success 200 {object} utilities.Response{data=models.UserResponse}
// @Failure 400 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Failure 404 {object} utilities.Response
// @Router /api/user-manager/users/{id}/roles [delete]
func (umc *UserManagerController) RemoveRole(c *gin.Context) {
	userID := c.Param("id")

	var req RemoveRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utilities.ValidationErrorResponse(c, err)
		return
	}

	// Find role
	var role models.Role
	if err := umc.DB.Where("name = ?", req.RoleName).First(&role).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusNotFound, "Role not found", err.Error())
		return
	}

	// Check permission hierarchy
	currentUserRoles, _ := c.Get("roles")
	currentRoles := currentUserRoles.([]string)

	hierarchy := models.GetRoleHierarchy()
	currentMaxLevel := 0
	for _, roleName := range currentRoles {
		if level, exists := hierarchy[roleName]; exists && level > currentMaxLevel {
			currentMaxLevel = level
		}
	}

	targetRoleLevel, exists := hierarchy[req.RoleName]
	if !exists || currentMaxLevel < targetRoleLevel {
		utilities.ErrorResponse(c, http.StatusForbidden, "Insufficient permissions to remove this role", "permission denied")
		return
	}

	// Remove role
	if err := umc.DB.Where("user_id = ? AND role_id = ?", userID, role.ID).Delete(&models.UserRole{}).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to remove role", err.Error())
		return
	}

	// Reload user with updated roles
	var user models.User
	umc.DB.Preload("UserRoles.Role").Preload("UserRoles.Assigner").First(&user, userID)

	utilities.SuccessResponse(c, http.StatusOK, "Role removed successfully", user.ToUserResponse())
}

// CreateUser godoc
// @Summary Create a new user
// @Description Create a new user account.
// @Tags user-manager
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateUserRequest true "Create user request"
// @Success 201 {object} utilities.Response{data=models.UserResponse}
// @Failure 400 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Failure 409 {object} utilities.Response
// @Router /api/user-manager/users [post]
func (umc *UserManagerController) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utilities.ValidationErrorResponse(c, err)
		return
	}

	// Check if user already exists
	var existingUser models.User
	if err := umc.DB.Where("username = ? OR email = ?", req.Username, req.Email).First(&existingUser).Error; err == nil {
		utilities.ErrorResponse(c, http.StatusConflict, "User already exists", "username or email already taken")
		return
	}

	// Hash password
	hashedPassword, err := utilities.HashPassword(req.Password)
	if err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to hash password", err.Error())
		return
	}

	// Get current user ID for audit trail
	currentUserID, exists := c.Get("user_id")
	if !exists {
		utilities.ErrorResponse(c, http.StatusUnauthorized, "User not authenticated", "user_id not found in context")
		return
	}

	// Create user
	user := models.User{
		Username: req.Username,
		Email:    req.Email,
		Password: hashedPassword,
		FullName: req.FullName,
		IsActive: req.IsActive,
	}

	if err := umc.DB.Create(&user).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to create user", err.Error())
		return
	}

	// Assign initial role if specified
	if req.InitialRole != "" {
		// Check permission hierarchy for role assignment
		currentUserRoles, _ := c.Get("roles")
		currentRoles := currentUserRoles.([]string)

		hierarchy := models.GetRoleHierarchy()
		currentMaxLevel := 0
		for _, roleName := range currentRoles {
			if level, exists := hierarchy[roleName]; exists && level > currentMaxLevel {
				currentMaxLevel = level
			}
		}

		// Check if current user can assign this role
		targetRoleLevel, exists := hierarchy[req.InitialRole]
		if !exists {
			utilities.ErrorResponse(c, http.StatusBadRequest, "Invalid role specified", "role not found")
			return
		}

		if currentMaxLevel < targetRoleLevel {
			utilities.ErrorResponse(c, http.StatusForbidden, "Insufficient permissions to assign this role", "permission denied")
			return
		}

		// Find and assign the role
		var role models.Role
		if err := umc.DB.Where("name = ?", req.InitialRole).First(&role).Error; err != nil {
			utilities.ErrorResponse(c, http.StatusBadRequest, "Role not found", err.Error())
			return
		}

		userRole := models.UserRole{
			UserID:     user.ID,
			RoleID:     role.ID,
			AssignedBy: currentUserID.(uint),
		}

		if err := umc.DB.Create(&userRole).Error; err != nil {
			utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to assign role", err.Error())
			return
		}
	} else {
		// Assign guest role by default
		var guestRole models.Role
		if err := umc.DB.Where("name = ?", "guest").First(&guestRole).Error; err == nil {
			userRole := models.UserRole{
				UserID:     user.ID,
				RoleID:     guestRole.ID,
				AssignedBy: currentUserID.(uint),
			}
			umc.DB.Create(&userRole)
		}
	}

	// Load user with roles
	umc.DB.Preload("UserRoles.Role").Preload("UserRoles.Assigner").First(&user, user.ID)

	utilities.SuccessResponse(c, http.StatusCreated, "User created successfully", user.ToUserResponse())
}

// DeleteUser godoc
// @Summary Delete a user
// @Description Delete a user account.
// @Tags user-manager
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Success 200 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Failure 404 {object} utilities.Response
// @Router /api/user-manager/users/{id} [delete]
func (umc *UserManagerController) DeleteUser(c *gin.Context) {
	userID := c.Param("id")

	// Find user to be deleted
	var user models.User
	if err := umc.DB.Preload("UserRoles.Role").First(&user, userID).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusNotFound, "User not found", err.Error())
		return
	}

	// Prevent deletion of current user
	currentUserID, exists := c.Get("user_id")
	if !exists {
		utilities.ErrorResponse(c, http.StatusUnauthorized, "User not authenticated", "user_id not found in context")
		return
	}

	if user.ID == currentUserID.(uint) {
		utilities.ErrorResponse(c, http.StatusForbidden, "Cannot delete your own account", "self-deletion not allowed")
		return
	}

	// Check permission hierarchy - can only delete users with lower roles
	currentUserRoles, _ := c.Get("roles")
	currentRoles := currentUserRoles.([]string)

	hierarchy := models.GetRoleHierarchy()
	currentMaxLevel := 0
	for _, roleName := range currentRoles {
		if level, exists := hierarchy[roleName]; exists && level > currentMaxLevel {
			currentMaxLevel = level
		}
	}

	// Get target user's highest role level
	targetMaxLevel := 0
	for _, userRole := range user.UserRoles {
		if level, exists := hierarchy[userRole.Role.Name]; exists && level > targetMaxLevel {
			targetMaxLevel = level
		}
	}

	// Check if current user has permission to delete target user
	if currentMaxLevel <= targetMaxLevel {
		utilities.ErrorResponse(c, http.StatusForbidden, "Insufficient permissions to delete this user", "permission denied")
		return
	}

	// Start transaction to ensure data consistency
	tx := umc.DB.Begin()

	// Delete all user roles first (due to foreign key constraints)
	if err := tx.Where("user_id = ?", user.ID).Delete(&models.UserRole{}).Error; err != nil {
		tx.Rollback()
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to delete user roles", err.Error())
		return
	}

	// Delete the user (soft delete)
	if err := tx.Delete(&user).Error; err != nil {
		tx.Rollback()
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to delete user", err.Error())
		return
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to commit transaction", err.Error())
		return
	}

	utilities.SuccessResponse(c, http.StatusOK, "User deleted successfully", nil)
}

// UpdateUserPassword godoc
// @Summary Update user password
// @Description Update a user's password.
// @Tags user-manager
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Param request body UpdateUserPasswordRequest true "Update password request"
// @Success 200 {object} utilities.Response{data=models.UserResponse}
// @Failure 400 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Failure 404 {object} utilities.Response
// @Router /api/user-manager/users/{id}/password [put]
func (umc *UserManagerController) UpdateUserPassword(c *gin.Context) {
	userID := c.Param("id")

	var req UpdateUserPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utilities.ValidationErrorResponse(c, err)
		return
	}

	// Find user to be updated
	var user models.User
	if err := umc.DB.Preload("UserRoles.Role").First(&user, userID).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusNotFound, "User not found", err.Error())
		return
	}

	// Check permission hierarchy - can only update users with lower or equal roles
	currentUserRoles, _ := c.Get("roles")
	currentRoles := currentUserRoles.([]string)

	hierarchy := models.GetRoleHierarchy()
	currentMaxLevel := 0
	for _, roleName := range currentRoles {
		if level, exists := hierarchy[roleName]; exists && level > currentMaxLevel {
			currentMaxLevel = level
		}
	}

	// Get target user's highest role level
	targetMaxLevel := 0
	for _, userRole := range user.UserRoles {
		if level, exists := hierarchy[userRole.Role.Name]; exists && level > targetMaxLevel {
			targetMaxLevel = level
		}
	}

	// Check if current user has permission to update target user
	if currentMaxLevel < targetMaxLevel {
		utilities.ErrorResponse(c, http.StatusForbidden, "Insufficient permissions to update this user", "permission denied")
		return
	}

	// Hash new password
	hashedPassword, err := utilities.HashPassword(req.NewPassword)
	if err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to hash password", err.Error())
		return
	}

	// Update password
	user.Password = hashedPassword
	if err := umc.DB.Save(&user).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to update password", err.Error())
		return
	}

	// Clear refresh token to force re-login
	user.RefreshToken = ""
	umc.DB.Save(&user)

	// Load user with roles for response
	umc.DB.Preload("UserRoles.Role").Preload("UserRoles.Assigner").First(&user, user.ID)

	utilities.SuccessResponse(c, http.StatusOK, "Password updated successfully", user.ToUserResponse())
}

// UpdateUserProfile godoc
// @Summary Update user profile
// @Description Update user's full name and email
// @Tags user-manager
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Param request body UpdateUserProfileRequest true "Update profile request"
// @Success 200 {object} utilities.Response{data=models.UserResponse}
// @Failure 400 {object} utilities.Response
// @Failure 401 {object} utilities.Response
// @Failure 403 {object} utilities.Response
// @Failure 404 {object} utilities.Response
// @Failure 409 {object} utilities.Response
// @Router /api/user-manager/users/{id}/profile [put]
func (umc *UserManagerController) UpdateUserProfile(c *gin.Context) {
	userID := c.Param("id")

	var req UpdateUserProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utilities.ValidationErrorResponse(c, err)
		return
	}

	// Find user to be updated
	var user models.User
	if err := umc.DB.Preload("UserRoles.Role").First(&user, userID).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusNotFound, "User not found", err.Error())
		return
	}

	// Check if email already exists (if email is being changed)
	if req.Email != "" && req.Email != user.Email {
		var existingUser models.User
		if err := umc.DB.Where("email = ? AND id != ?", req.Email, user.ID).First(&existingUser).Error; err == nil {
			utilities.ErrorResponse(c, http.StatusConflict, "Email already exists", "email already taken by another user")
			return
		}
	}

	// Check permission hierarchy - can only update users with lower or equal roles
	currentUserRoles, _ := c.Get("roles")
	currentRoles := currentUserRoles.([]string)

	hierarchy := models.GetRoleHierarchy()
	currentMaxLevel := 0
	for _, roleName := range currentRoles {
		if level, exists := hierarchy[roleName]; exists && level > currentMaxLevel {
			currentMaxLevel = level
		}
	}

	// Get target user's highest role level
	targetMaxLevel := 0
	for _, userRole := range user.UserRoles {
		if level, exists := hierarchy[userRole.Role.Name]; exists && level > targetMaxLevel {
			targetMaxLevel = level
		}
	}

	// Check if current user has permission to update target user
	if currentMaxLevel < targetMaxLevel {
		utilities.ErrorResponse(c, http.StatusForbidden, "Insufficient permissions to update this user", "permission denied")
		return
	}

	// Update fields if provided
	if req.FullName != "" {
		user.FullName = req.FullName
	}
	if req.Email != "" {
		user.Email = req.Email
	}

	// Save changes
	if err := umc.DB.Save(&user).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to update user profile", err.Error())
		return
	}

	// Load user with roles for response
	umc.DB.Preload("UserRoles.Role").Preload("UserRoles.Assigner").First(&user, user.ID)

	utilities.SuccessResponse(c, http.StatusOK, "User profile updated successfully", user.ToUserResponse())
}

// GetRoles godoc
// @Summary Get all roles
// @Description Get list of all available roles.
// @Tags user-manager
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Success 200 {object} utilities.Response{data=[]models.Role}
// @Failure 401 {object} utilities.Response
// @Router /api/user-manager/roles [get]
func (umc *UserManagerController) GetRoles(c *gin.Context) {
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	var roles []models.Role
	var total int64

	// Get total count
	umc.DB.Model(&models.Role{}).Count(&total)

	if err := umc.DB.Limit(limit).Offset(offset).Find(&roles).Error; err != nil {
		utilities.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve roles", err.Error())
		return
	}

	// Convert to response format
	roleResponses := make([]models.RoleListResponse, len(roles))
	for i, role := range roles {
		roleResponses[i] = role.ToRoleListResponse()
	}

	response := RoleListResponse{
		Roles: roleResponses,
		Pagination: utilities.PaginationResponse{
			Page:  page,
			Limit: limit,
			Total: int(total),
		},
	}

	utilities.SuccessResponse(c, http.StatusOK, "Roles retrieved successfully", response)
}

// Request/Response structs
type UsersListResponse struct {
	Users      []models.UserResponse    `json:"users"`
	Pagination utilities.PaginationResponse `json:"pagination"`
}

type RoleListResponse struct {
	Roles      []models.RoleListResponse `json:"roles"`
	Pagination utilities.PaginationResponse  `json:"pagination"`
}

type CreateUserRequest struct {
	Username    string `json:"username" binding:"required,min=3,max=50" example:"john_doe"`
	Email       string `json:"email" binding:"required,email" example:"john@example.com"`
	Password    string `json:"password" binding:"required,min=6" example:"password123"`
	FullName    string `json:"full_name" binding:"required" example:"John Doe"`
	IsActive    bool   `json:"is_active" example:"true"`
	InitialRole string `json:"initial_role,omitempty" example:"picker"`
}

type UpdateUserStatusRequest struct {
	IsActive bool `json:"is_active" example:"true"`
}

type UpdateUserPasswordRequest struct {
	NewPassword string `json:"new_password" binding:"required,min=6" example:"newpassword123"`
}

type UpdateUserProfileRequest struct {
	FullName string `json:"full_name,omitempty" binding:"omitempty,min=1" example:"John Doe Updated"`
	Email    string `json:"email,omitempty" binding:"omitempty,email" example:"newemail@example.com"`
}

type AssignRoleRequest struct {
	RoleName string `json:"role_name" binding:"required" example:"manager"`
}

type RemoveRoleRequest struct {
	RoleName string `json:"role_name" binding:"required" example:"manager"`
}
