package model

import (
	"time"

	"gorm.io/gorm"
)

// User adalah tabel autentikasi operator. Implementasi login belum ada.
type User struct {
	ID        uint           `gorm:"primaryKey"`
	Username  string         `gorm:"uniqueIndex;not null;size:64"`
	Email     string         `gorm:"uniqueIndex;size:128"`
	Password  string         `gorm:"not null"`
	Role      string         `gorm:"default:'operator';size:32"`
	Active    bool           `gorm:"default:true"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}
