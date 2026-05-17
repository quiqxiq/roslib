package model

import "time"

// Transaction mencatat setiap penjualan voucher hotspot.
// Menggantikan penyimpanan di /system/script RouterOS.
type Transaction struct {
	ID       uint           `gorm:"primaryKey"`
	DeviceID uint           `gorm:"index;not null"`
	Device   MikrotikDevice `gorm:"foreignKey:DeviceID"`

	SaleDate  string `gorm:"index;size:12"` // "jan/02/2006"
	SaleTime  string `gorm:"size:8"`         // "15:04:05"
	SaleMonth string `gorm:"index;size:8"`   // "jan2025" — untuk filter laporan bulanan

	Username  string `gorm:"index;not null;size:128"`
	Price     int
	SellPrice int
	IP        string `gorm:"size:45"`
	MAC       string `gorm:"size:17"`
	Validity  string `gorm:"size:16"`
	Profile   string `gorm:"index;size:64"`
	Comment   string `gorm:"size:255"`

	CreatedAt time.Time
}
