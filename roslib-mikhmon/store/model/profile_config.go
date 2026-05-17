package model

import "time"

// HotspotProfileConfig menyimpan konfigurasi mikhmon-specific per profile hotspot.
// Digunakan expiry service untuk menentukan aksi saat user expired.
type HotspotProfileConfig struct {
	ID          uint           `gorm:"primaryKey"`
	DeviceID    uint           `gorm:"uniqueIndex:uniq_dev_prof;not null"`
	Device      MikrotikDevice `gorm:"foreignKey:DeviceID"`
	ProfileName string         `gorm:"uniqueIndex:uniq_dev_prof;not null;size:64"`

	// Mode expired: 0 (none) | rem (remove) | ntf (notice) | remc (remove+record) | ntfc (notice+record)
	ExpiryMode string `gorm:"default:'0';size:8"`
	Validity   string `gorm:"size:16"` // "30d", "1d", dll
	Price      int
	SellPrice  int
	LockMAC    bool `gorm:"default:false"`

	CreatedAt time.Time
	UpdatedAt time.Time
}
