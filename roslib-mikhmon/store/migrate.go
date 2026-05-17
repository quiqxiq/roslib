package store

import (
	"github.com/quiqxiq/roslib-mikhmon/store/model"
	"gorm.io/gorm"
)

// Migrate menjalankan AutoMigrate untuk semua tabel.
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.User{},
		&model.MikrotikDevice{},
		&model.Transaction{},
		&model.HotspotProfileConfig{},
	)
}
