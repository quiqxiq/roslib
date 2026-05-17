// Command server adalah entry point HTTP API + SSE roslib-mikhmon.
//
// Lifecycle:
//
//  1. Load config HTTP + DB via env (lihat .env.example).
//  2. Buka koneksi PostgreSQL + jalankan AutoMigrate.
//  3. Build stores (device, transaction, profile_config).
//  4. Build DeviceManager — load semua device dari DB, dial koneksi RouterOS.
//  5. Build ExpiryService — spawn per-device goroutine pemeriksa expiry.
//  6. Build api.Server (gin engine + middleware + routes + SSE hub).
//  7. http.Server.ListenAndServe + graceful shutdown via SIGINT/SIGTERM.
package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	roslibcfg "github.com/quiqxiq/roslib/config"
	roslibinflux "github.com/quiqxiq/roslib/metrics/influx"
	"github.com/quiqxiq/roslib-mikhmon/api"
	"github.com/quiqxiq/roslib-mikhmon/api/sse"
	"github.com/quiqxiq/roslib-mikhmon/internal/config"
	"github.com/quiqxiq/roslib-mikhmon/service/devmgr"
	"github.com/quiqxiq/roslib-mikhmon/service/expiry"
	"github.com/quiqxiq/roslib-mikhmon/service/metrics"
	"github.com/quiqxiq/roslib-mikhmon/store"
	"github.com/quiqxiq/roslib-mikhmon/store/model"
	"github.com/sirupsen/logrus"
)

func main() {
	log := logrus.New()
	log.SetLevel(logrus.InfoLevel)
	log.SetFormatter(&logrus.JSONFormatter{})

	// Best-effort load .env — kalau tidak ada, lanjut pakai env system.
	dotenvPath := os.Getenv("DOTENV_FILE")
	if dotenvPath == "" {
		dotenvPath = ".env"
	}
	if err := godotenv.Load(dotenvPath); err == nil {
		log.WithField("file", dotenvPath).Info("loaded .env")
	}

	httpCfg, err := config.LoadHTTPFromEnv()
	if err != nil {
		log.WithError(err).Fatal("load http config")
	}
	dbCfg := config.LoadDBFromEnv()

	rootCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// ── Database ──────────────────────────────────────────────────────────
	db, err := store.Open(dbCfg.DSN())
	if err != nil {
		log.WithError(err).Fatal("open database")
	}
	if err := store.Migrate(db); err != nil {
		log.WithError(err).Fatal("migrate database")
	}
	log.Info("database ready")

	// ── Stores ────────────────────────────────────────────────────────────
	deviceStore := store.NewDeviceStore(db)
	txStore := store.NewTransactionStore(db)
	profileStore := store.NewProfileConfigStore(db)

	// ── Device Manager ────────────────────────────────────────────────────
	devMgr := devmgr.New(deviceStore, log)
	if err := devMgr.Start(rootCtx); err != nil {
		log.WithError(err).Fatal("start device manager")
	}

	// ── Expiry Service ────────────────────────────────────────────────────
	expSvc := expiry.New(devMgr, deviceStore, profileStore, txStore, log)
	if err := expSvc.Start(rootCtx); err != nil {
		log.WithError(err).Fatal("start expiry service")
	}

	// ── InfluxDB3 + Metrics Service ───────────────────────────────────────
	influxCfg := roslibcfg.InfluxConfig{
		Enabled:  os.Getenv("INFLUX_ENABLED") == "true",
		Host:     os.Getenv("INFLUX_HOST"),
		Token:    os.Getenv("INFLUX_TOKEN"),
		Database: os.Getenv("INFLUX_DATABASE"),
	}
	var metricsSvc *metrics.Service
	var influxReader *roslibinflux.Reader
	if influxCfg.Enabled {
		influxCli, err := roslibinflux.NewClient(influxCfg)
		if err != nil {
			log.WithError(err).Fatal("init influx client")
		}
		defer influxCli.Close()
		influxReader = roslibinflux.NewReader(influxCli)
		metricsSvc = metrics.New(influxCli, devMgr, log)
		metricsSvc.Start(rootCtx)
		log.Info("influx metrics started")
	}

	// Wire device lifecycle hooks — expiry + metrics keduanya
	devMgr.OnDeviceConnected = func(d model.MikrotikDevice) {
		expSvc.StartDevice(d)
		if metricsSvc != nil {
			metricsSvc.StartDevice(d)
		}
	}
	devMgr.OnDeviceRemoved = func(slug string) {
		expSvc.StopDevice(slug)
		if metricsSvc != nil {
			metricsSvc.StopDevice(slug)
		}
	}

	// ── HTTP Server ───────────────────────────────────────────────────────
	deps := &api.Deps{
		Logger:       log,
		HTTPConfig:   httpCfg,
		DB:           db,
		DeviceStore:  deviceStore,
		TxStore:      txStore,
		ProfileStore: profileStore,
		DevMgr:       devMgr,
		Hub:          sse.NewHub(),
		InfluxReader: influxReader,
	}

	handler := api.NewServer(deps)

	httpSrv := &http.Server{
		Addr:        httpCfg.Bind,
		Handler:     handler,
		ReadTimeout: httpCfg.ReadTimeout,
		// WriteTimeout sengaja 0: SSE butuh long-lived connection.
		WriteTimeout: 0,
		IdleTimeout:  httpCfg.IdleTimeout,
	}

	serverErr := make(chan error, 1)
	go func() {
		log.WithField("bind", httpCfg.Bind).Info("http server listening")
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
		close(serverErr)
	}()

	select {
	case <-rootCtx.Done():
		log.Info("shutdown signal received")
	case err := <-serverErr:
		if err != nil {
			log.WithError(err).Fatal("http server crashed")
		}
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), httpCfg.ShutdownGrace)
	defer cancelShutdown()
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		log.WithError(err).Warn("http shutdown error")
	}
	log.Info("server stopped")
}
