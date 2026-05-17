package store

import (
	"context"
	"errors"

	"github.com/quiqxiq/roslib-mikhmon/store/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ProfileConfigStore interface {
	Get(ctx context.Context, deviceID uint, profileName string) (model.HotspotProfileConfig, error)
	Upsert(ctx context.Context, cfg *model.HotspotProfileConfig) error
	ListByDevice(ctx context.Context, deviceID uint) ([]model.HotspotProfileConfig, error)
	Delete(ctx context.Context, id uint) error
}

type gormProfileConfigStore struct{ db *gorm.DB }

func NewProfileConfigStore(db *gorm.DB) ProfileConfigStore {
	return &gormProfileConfigStore{db: db}
}

func (s *gormProfileConfigStore) Get(ctx context.Context, deviceID uint, profileName string) (model.HotspotProfileConfig, error) {
	var cfg model.HotspotProfileConfig
	err := s.db.WithContext(ctx).
		Where("device_id = ? AND profile_name = ?", deviceID, profileName).
		First(&cfg).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Default: mode "rem" (remove expired user)
		return model.HotspotProfileConfig{
			DeviceID:    deviceID,
			ProfileName: profileName,
			ExpiryMode:  "rem",
		}, nil
	}
	return cfg, err
}

func (s *gormProfileConfigStore) Upsert(ctx context.Context, cfg *model.HotspotProfileConfig) error {
	return s.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "device_id"}, {Name: "profile_name"}},
			DoUpdates: clause.AssignmentColumns([]string{"expiry_mode", "validity", "price", "sell_price", "lock_mac", "updated_at"}),
		}).Create(cfg).Error
}

func (s *gormProfileConfigStore) ListByDevice(ctx context.Context, deviceID uint) ([]model.HotspotProfileConfig, error) {
	var cfgs []model.HotspotProfileConfig
	err := s.db.WithContext(ctx).Where("device_id = ?", deviceID).Find(&cfgs).Error
	return cfgs, err
}

func (s *gormProfileConfigStore) Delete(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&model.HotspotProfileConfig{}, id).Error
}
