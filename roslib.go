// Package roslib adalah wrapper di atas github.com/go-routeros/routeros/v3
// yang dirancang untuk memanfaatkan async mode + tag-based multiplexing
// secara penuh: 2 koneksi persisten per router (stream + command),
// interval-group batching untuk polling, dan auto-reconnect berbasis
// channel error dari AsyncContext.
//
// Entry point:
//
//   - roslib.New(ctx, Options{...}) — konstruktor manual
//   - roslib.NewFromConfig(ctx, cfg, log) — dari config.Config (env loader)
//
// Dari Device, chain fluent device.Path(p)... atau register poll/stream
// langsung lewat RegisterPoll / RegisterStream.
package roslib

import (
	"context"
	"errors"

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

// ──────────── helper konstruktor ────────────

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

// NewPair adalah helper konstruktor Pair.
func NewPair(key, value string) Pair { return query.NewPair(key, value) }

// Where adalah helper konstruktor WherePair (operator =).
func Where(key, value string) WherePair { return query.Where(key, value) }

// WhereNot adalah helper konstruktor WherePair (operator !=).
func WhereNot(key, value string) WherePair { return query.WhereNot(key, value) }
