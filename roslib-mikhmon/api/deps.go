// Package api menyediakan HTTP layer (gin) di atas mikrotik/* + workflows/.
// Entry point pakai cmd/server/main.go yang inject Deps lalu panggil NewServer.
package api

import (
	roslibinflux "github.com/quiqxiq/roslib/metrics/influx"
	"github.com/quiqxiq/roslib-mikhmon/api/sse"
	"github.com/quiqxiq/roslib-mikhmon/internal/config"
	"github.com/quiqxiq/roslib-mikhmon/service/devmgr"
	"github.com/quiqxiq/roslib-mikhmon/store"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Deps dipakai oleh cmd/server/main.go untuk meng-assemble handler.
type Deps struct {
	Logger     *logrus.Logger
	HTTPConfig *config.HTTPConfig
	DB         *gorm.DB

	// Stores
	DeviceStore  store.DeviceStore
	TxStore      store.TransactionStore
	ProfileStore store.ProfileConfigStore

	// Device Manager — mengelola koneksi ke semua router
	DevMgr *devmgr.Manager

	// SSE Hub — shared real-time broker
	Hub *sse.Hub

	// InfluxReader untuk history query API. Nil jika INFLUX_ENABLED=false.
	InfluxReader *roslibinflux.Reader
}
