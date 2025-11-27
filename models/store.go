package models

import (
	"time"

	"gorm.io/gorm"
)

type Store struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Code      string         `gorm:"unique;not null" json:"code" example:"AX"`
	Name      string         `gorm:"not null;unique" json:"name" example:"AXON"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type StoreResponse struct {
	ID      uint      `json:"id"`
	Code    string    `json:"code"`
	Name    string    `json:"name"`
	Created time.Time `json:"created_at"`
	Updated time.Time `json:"updated_at"`
}

// ToStoreResponse converts Store model to StoreResponse
func (s *Store) ToStoreResponse() StoreResponse {
	return StoreResponse{
		ID:      s.ID,
		Code:    s.Code,
		Name:    s.Name,
		Created: s.CreatedAt,
		Updated: s.UpdatedAt,
	}
}

// ToStoreMobileResponse converts Store model to StoreResponse for mobile use
func (s *Store) ToStoreMobileResponse() StoreResponse {
	return StoreResponse{
		ID:      s.ID,
		Code:    s.Code,
		Name:    s.Name,
		Created: s.CreatedAt,
		Updated: s.UpdatedAt,
	}
}
