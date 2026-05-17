package store

import (
	"context"

	"github.com/quiqxiq/roslib-mikhmon/store/model"
	"gorm.io/gorm"
)

type TransactionStore interface {
	Create(ctx context.Context, tx *model.Transaction) error
	ListByDevice(ctx context.Context, deviceID uint, month string) ([]model.Transaction, error)
	ListByDeviceDate(ctx context.Context, deviceID uint, date string) ([]model.Transaction, error)
}

type gormTransactionStore struct{ db *gorm.DB }

func NewTransactionStore(db *gorm.DB) TransactionStore { return &gormTransactionStore{db: db} }

func (s *gormTransactionStore) Create(ctx context.Context, tx *model.Transaction) error {
	return s.db.WithContext(ctx).Create(tx).Error
}

func (s *gormTransactionStore) ListByDevice(ctx context.Context, deviceID uint, month string) ([]model.Transaction, error) {
	var txs []model.Transaction
	q := s.db.WithContext(ctx).Where("device_id = ?", deviceID)
	if month != "" {
		q = q.Where("sale_month = ?", month)
	}
	err := q.Order("created_at desc").Find(&txs).Error
	return txs, err
}

func (s *gormTransactionStore) ListByDeviceDate(ctx context.Context, deviceID uint, date string) ([]model.Transaction, error) {
	var txs []model.Transaction
	err := s.db.WithContext(ctx).
		Where("device_id = ? AND sale_date = ?", deviceID, date).
		Order("created_at desc").Find(&txs).Error
	return txs, err
}
