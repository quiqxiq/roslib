package model

import (
	"time"

	"gorm.io/gorm"
)

// MikrotikDevice menyimpan konfigurasi koneksi ke router MikroTik.
// Status diupdate otomatis oleh DeviceManager setiap 30 detik.
type MikrotikDevice struct {
	ID          uint   `gorm:"primaryKey"`
	Slug        string `gorm:"uniqueIndex;not null;size:64"` // identifier di URL: /devices/:slug
	DisplayName string `gorm:"not null;size:128"`
	Address     string `gorm:"not null;size:255"` // "192.168.88.1:8728"
	Username    string `gorm:"not null;size:64"`
	Password    string `gorm:"not null"`
	UseTLS      bool   `gorm:"default:false"`

	// Status koneksi — diupdate oleh DeviceManager
	Status    string     `gorm:"default:'disconnected';size:32"` // connected | disconnected | error
	LastSeen  *time.Time
	LastError string `gorm:"size:512"`

	// Config expiry service
	ExpiryCheckInterval string `gorm:"default:'2m';size:16"`
	Active              bool   `gorm:"default:true"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}
