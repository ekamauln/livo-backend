package models

import (
	"time"

	"gorm.io/gorm"
)

// UserRole represents the many-to-many relationship between users and roles
type UserRole struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	UserID     uint           `gorm:"not null" json:"user_id"`
	RoleID     uint           `gorm:"not null" json:"role_id"`
	AssignedBy uint           `gorm:"not null" json:"assigned_by"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationship
	Assigner User `gorm:"foreignKey:AssignedBy" json:"assigner"`
	User     User `gorm:"foreignKey:UserID" json:"user"`
	Role     Role `gorm:"foreignKey:RoleID" json:"role"`
}
