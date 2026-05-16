// Package roslib adalah wrapper di atas github.com/go-routeros/routeros/v3
// yang dirancang untuk memanfaatkan async mode + tag-based multiplexing
// secara penuh: 2 koneksi persisten per router (stream + command),
// interval-group batching untuk polling, dan auto-reconnect berbasis
// channel error dari AsyncContext.
//
// Entry point (rekomendasi pakai Manager — persistent connection):
//
//   - roslib.NewManagerFromConfig(ctx, cfg, log) — single router via Manager
//   - roslib.NewManagerFromFleet(ctx, fleetCfg, log) — multi router via Manager
//   - roslib.NewManager() — Manager kosong, Register manual
//
// Entry point lama (tetap di-maintain untuk back-compat):
//
//   - roslib.New(ctx, Options{...}) — konstruktor manual single device
//   - roslib.NewFromConfig(ctx, cfg, log) — single device dari Config
//   - roslib.NewFleet(ctx, fleetCfg, log) — map[string]*Device
//
// Dari Device, chain fluent device.Path(p)... atau register poll/stream
// langsung lewat RegisterPoll / RegisterStream.
package roslib

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/InfluxCommunity/influxdb3-go/v2/influxdb3"
	cachepkg "github.com/quiqxiq/roslib/cache"
	"github.com/quiqxiq/roslib/capability"
	"github.com/quiqxiq/roslib/config"
	"github.com/quiqxiq/roslib/decode"
	"github.com/quiqxiq/roslib/device"
	"github.com/quiqxiq/roslib/metrics/influx"
	"github.com/quiqxiq/roslib/poll"
	"github.com/quiqxiq/roslib/query"
	"github.com/quiqxiq/roslib/stream"
	"github.com/sirupsen/logrus"
)

// ──────────── re-export tipe penting ────────────

// Device adalah handle ke satu router. Alias untuk *device.RouterDevice.
type Device = device.RouterDevice

// Options adalah parameter konfigurasi New.
type Options = device.Options

// Config adalah konfigurasi terpusat (env-loadable).
type Config = config.Config

// PollConfig mendeskripsikan satu poll periodik.
type PollConfig = poll.Config

// PollHandler dipanggil setiap !re sentence dari hasil poll.
type PollHandler = poll.Handler

// StreamSpec mendeskripsikan listener long-running di koneksi stream.
type StreamSpec = stream.Spec

// StreamHandler dipanggil setiap !re sentence dari listener.
type StreamHandler = stream.Handler

// Sentence adalah wrapper typed di atas proto.Sentence (lihat decode).
type Sentence = decode.Sentence

// Pair adalah named parameter "=key=value".
type Pair = query.Pair

// WherePair adalah filter "?key=value" untuk Print.
type WherePair = query.WherePair

// Cache adalah interface store key/value untuk hasil query.
type Cache = cachepkg.Cache

// Registry adalah capability registry hasil parse JSON RouterOS.
type Registry = capability.Registry

// Manager mengelola pool RouterDevice persisten yang dapat di-acquire ulang
// by-name tanpa re-dial. Alias untuk *device.Manager.
type Manager = device.Manager

// ConnectionRole adalah label peran koneksi untuk pola multi-koneksi per
// router fisik (mis. memisahkan queue antara stream dan mutation).
type ConnectionRole = device.ConnectionRole

const (
	// RoleStream label koneksi khusus listener long-running.
	RoleStream = device.RoleStream
	// RoleCommand label koneksi khusus query/exec one-shot.
	RoleCommand = device.RoleCommand
	// RoleMutation label koneksi khusus add/set/remove dengan timeout lebih lama.
	RoleMutation = device.RoleMutation
)

// RoleKey menghasilkan compound key "routerName:role" untuk dipakai di
// Manager.Register / Get / GetOrConnect saat caller butuh beberapa koneksi
// terpisah ke satu router fisik.
func RoleKey(routerName string, role ConnectionRole) string {
	return device.RoleKey(routerName, role)
}

// ──────────── helper konstruktor ────────────

// NewManager membuat Manager kosong siap pakai. Caller register device
// satu-satu via mgr.Register(ctx, name, opts), atau pakai
// NewManagerFromConfig / NewManagerFromFleet untuk sekali setup.
func NewManager() *Manager { return device.NewManager() }

// New men-dial 2 koneksi async ke router, mengaktifkan supervisor untuk
// auto-reconnect, dan mengembalikan *Device siap pakai.
func New(parent context.Context, opts Options) (*Device, error) {
	return device.New(parent, opts)
}

// NewFromConfig mengkombinasikan config + logger menjadi Device:
//
//   - load capability registry (embed atau RegistryPath kalau di-set)
//   - build cache (NoopCache kalau Cache.Enabled false)
//   - dial router
//
// InfluxClient dikembalikan terpisah supaya caller bisa menutupnya sendiri.
// Caller juga boleh mengabaikannya (nil kalau Influx.Enabled false).
//
// Deprecated: pakai NewManagerFromConfig supaya device dapat re-acquire
// tanpa re-dial. Konstruktor ini tetap di-maintain untuk back-compat.
func NewFromConfig(parent context.Context, cfg *Config, log *logrus.Logger) (*Device, *influxdb3.Client, error) {
	if cfg == nil {
		return nil, nil, errors.New("roslib: nil config")
	}
	if log == nil {
		return nil, nil, errors.New("roslib: nil logger")
	}
	if err := cfg.Validate(); err != nil {
		return nil, nil, err
	}

	reg, err := capability.Load(capability.LoadOptions{Path: cfg.RegistryPath})
	if err != nil {
		return nil, nil, err
	}

	var cache cachepkg.Cache = cachepkg.NoopCache{}
	if cfg.Cache.Enabled {
		cache = cachepkg.NewInMemory() // user-app boleh override pakai Redis di build-tag
	}

	opts := cfg.ToDeviceOptions(log)
	opts.Registry = reg
	opts.Cache = cache
	opts.CacheTTL = cfg.Cache.DefaultTTL
	opts.StrictCapability = cfg.StrictCapability

	dev, err := device.New(parent, opts)
	if err != nil {
		return nil, nil, err
	}

	var influxClient *influxdb3.Client
	if cfg.Influx.Enabled {
		influxClient, err = influx.NewClient(cfg.Influx)
		if err != nil {
			_ = dev.Close()
			return nil, nil, err
		}
	}
	return dev, influxClient, nil
}

// NewFleet membangun map[deviceID]*Device dari FleetConfig.
//
// Registry, cache, dan influx client di-share antar device — efisien
// untuk service yang manage banyak router. InfluxClient dikembalikan
// terpisah supaya caller bisa menutupnya saat shutdown.
//
// Jika satu router gagal dial, fleet rollback (close yang sudah dial)
// dan return error — pendekatan all-or-nothing supaya state konsisten.
// Caller yang butuh partial-fleet (skip router error) boleh wrap sendiri.
//
// Deprecated: pakai NewManagerFromFleet supaya device dapat re-acquire
// tanpa re-dial. Konstruktor ini tetap di-maintain untuk back-compat.
func NewFleet(parent context.Context, cfg *config.FleetConfig, log *logrus.Logger) (map[string]*Device, *influxdb3.Client, error) {
	if cfg == nil {
		return nil, nil, errors.New("roslib: nil fleet config")
	}
	if log == nil {
		return nil, nil, errors.New("roslib: nil logger")
	}
	if len(cfg.Routers) == 0 {
		return nil, nil, errors.New("roslib: fleet has no routers")
	}

	reg, err := capability.Load(capability.LoadOptions{Path: cfg.RegistryPath})
	if err != nil {
		return nil, nil, err
	}

	var sharedCache cachepkg.Cache = cachepkg.NoopCache{}
	if cfg.Cache.Enabled {
		sharedCache = cachepkg.NewInMemory()
	}

	var influxClient *influxdb3.Client
	if cfg.Influx.Enabled {
		influxClient, err = influx.NewClient(cfg.Influx)
		if err != nil {
			return nil, nil, err
		}
	}

	fleet := make(map[string]*Device, len(cfg.Routers))
	for _, entry := range cfg.Routers {
		opts := Options{
			ID:                       entry.ID,
			Address:                  entry.Address,
			Username:                 entry.Username,
			Password:                 entry.Password,
			Logger:                   log,
			ListenQueueSize:          entry.ListenQueueSize,
			DialTimeout:              entry.DialTimeout,
			ReconnectInitialInterval: entry.ReconnectInitialInterval,
			ReconnectMaxInterval:     entry.ReconnectMaxInterval,
			ReconnectMaxElapsed:      entry.ReconnectMaxElapsed,
			Registry:                 reg,
			Cache:                    sharedCache,
			CacheTTL:                 cfg.Cache.DefaultTTL,
			StrictCapability:         cfg.StrictCapability,
		}
		dev, derr := device.New(parent, opts)
		if derr != nil {
			CloseAll(fleet)
			if influxClient != nil {
				_ = influxClient.Close()
			}
			return nil, nil, fmt.Errorf("dial router %q: %w", entry.ID, derr)
		}
		fleet[entry.ID] = dev
	}

	return fleet, influxClient, nil
}

// CloseAll loop semua device di fleet dan panggil Close. Aman dipanggil
// dengan map nil atau kosong.
func CloseAll(fleet map[string]*Device) {
	for _, dev := range fleet {
		if dev != nil {
			_ = dev.Close()
		}
	}
}

// fleetEntryOptions assemble device.Options untuk satu entry fleet, dengan
// shared registry/cache/StrictCapability sudah di-inject.
func fleetEntryOptions(entry config.RouterEntry, log *logrus.Logger, reg *capability.Registry, sharedCache cachepkg.Cache, cacheTTL time.Duration, strict bool) Options {
	return Options{
		ID:                       entry.ID,
		Address:                  entry.Address,
		Username:                 entry.Username,
		Password:                 entry.Password,
		Logger:                   log,
		ListenQueueSize:          entry.ListenQueueSize,
		DialTimeout:              entry.DialTimeout,
		ReconnectInitialInterval: entry.ReconnectInitialInterval,
		ReconnectMaxInterval:     entry.ReconnectMaxInterval,
		ReconnectMaxElapsed:      entry.ReconnectMaxElapsed,
		Registry:                 reg,
		Cache:                    sharedCache,
		CacheTTL:                 cacheTTL,
		StrictCapability:         strict,
	}
}

// DefaultDeviceKey adalah key yang dipakai NewManagerFromConfig saat
// register single-router. Caller pakai ini untuk acquire device:
//
//	dev, _ := mgr.Get(roslib.DefaultDeviceKey)
const DefaultDeviceKey = "default"

// NewManagerFromConfig membangun Manager terisi 1 RouterDevice dari Config
// (single-router mode). Key device = DefaultDeviceKey ("default"). Caller
// dapat re-acquire device tanpa dial ulang lewat mgr.Get(key) atau
// mgr.GetOrConnect(ctx, key, opts).
//
// InfluxClient dikembalikan terpisah (nil kalau disabled) — caller wajib
// Close() saat shutdown.
//
// Pola pakai:
//
//	mgr, influxCli, err := roslib.NewManagerFromConfig(ctx, cfg, log)
//	if err != nil { return err }
//	defer mgr.CloseAll()
//	if influxCli != nil { defer influxCli.Close() }
//	dev, _ := mgr.Get("default")
//	// ... dev.Path("/system/resource").Print().Exec(ctx) ...
func NewManagerFromConfig(parent context.Context, cfg *Config, log *logrus.Logger) (*Manager, *influxdb3.Client, error) {
	if cfg == nil {
		return nil, nil, errors.New("roslib: nil config")
	}
	if log == nil {
		return nil, nil, errors.New("roslib: nil logger")
	}
	if err := cfg.Validate(); err != nil {
		return nil, nil, err
	}

	reg, err := capability.Load(capability.LoadOptions{Path: cfg.RegistryPath})
	if err != nil {
		return nil, nil, err
	}

	var cache cachepkg.Cache = cachepkg.NoopCache{}
	if cfg.Cache.Enabled {
		cache = cachepkg.NewInMemory()
	}

	opts := cfg.ToDeviceOptions(log)
	opts.Registry = reg
	opts.Cache = cache
	opts.CacheTTL = cfg.Cache.DefaultTTL
	opts.StrictCapability = cfg.StrictCapability

	var influxClient *influxdb3.Client
	if cfg.Influx.Enabled {
		influxClient, err = influx.NewClient(cfg.Influx)
		if err != nil {
			return nil, nil, err
		}
	}

	mgr := NewManager()
	if err := mgr.Register(parent, DefaultDeviceKey, opts); err != nil {
		if influxClient != nil {
			_ = influxClient.Close()
		}
		return nil, nil, err
	}
	return mgr, influxClient, nil
}

// NewManagerFromFleet membangun Manager terisi semua RouterDevice dari
// FleetConfig (multi-router mode). Setiap entry di-register dengan key =
// entry.ID. Registry + cache + influx client di-share antar device.
//
// Jika satu router gagal dial, mgr.CloseAll() dipanggil dan error
// dikembalikan — all-or-nothing supaya state konsisten.
//
// Pola pakai:
//
//	mgr, influxCli, err := roslib.NewManagerFromFleet(ctx, fleetCfg, log)
//	if err != nil { return err }
//	defer mgr.CloseAll()
//	for _, name := range mgr.Names() {
//	    dev, _ := mgr.Get(name)
//	    // ...
//	}
func NewManagerFromFleet(parent context.Context, cfg *config.FleetConfig, log *logrus.Logger) (*Manager, *influxdb3.Client, error) {
	if cfg == nil {
		return nil, nil, errors.New("roslib: nil fleet config")
	}
	if log == nil {
		return nil, nil, errors.New("roslib: nil logger")
	}
	if len(cfg.Routers) == 0 {
		return nil, nil, errors.New("roslib: fleet has no routers")
	}

	reg, err := capability.Load(capability.LoadOptions{Path: cfg.RegistryPath})
	if err != nil {
		return nil, nil, err
	}

	var sharedCache cachepkg.Cache = cachepkg.NoopCache{}
	if cfg.Cache.Enabled {
		sharedCache = cachepkg.NewInMemory()
	}

	var influxClient *influxdb3.Client
	if cfg.Influx.Enabled {
		influxClient, err = influx.NewClient(cfg.Influx)
		if err != nil {
			return nil, nil, err
		}
	}

	mgr := NewManager()
	for _, entry := range cfg.Routers {
		opts := fleetEntryOptions(entry, log, reg, sharedCache, cfg.Cache.DefaultTTL, cfg.StrictCapability)
		if rerr := mgr.Register(parent, entry.ID, opts); rerr != nil {
			mgr.CloseAll()
			if influxClient != nil {
				_ = influxClient.Close()
			}
			return nil, nil, fmt.Errorf("dial router %q: %w", entry.ID, rerr)
		}
	}
	return mgr, influxClient, nil
}

// NewPair adalah helper konstruktor Pair.
func NewPair(key, value string) Pair { return query.NewPair(key, value) }

// Where adalah helper konstruktor WherePair (operator =).
func Where(key, value string) WherePair { return query.Where(key, value) }

// WhereNot adalah helper konstruktor WherePair (operator !=).
func WhereNot(key, value string) WherePair { return query.WhereNot(key, value) }
