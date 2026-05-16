// Package config menyediakan struct konfigurasi terpusat untuk roslib
// beserta loader dari environment variable.
//
// User app boleh memuat .env duluan (mis. lewat godotenv) sebelum
// memanggil LoadFromEnv — paket ini sengaja tidak men-pin loader env
// agar tetap stdlib-only.
package config

import (
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/quiqxiq/roslib/device"
	"github.com/sirupsen/logrus"
)

// Config menampung seluruh setelan: router + observability toggle.
type Config struct {
	Router RouterConfig
	Cache  CacheConfig
	Influx InfluxConfig
}

// RouterConfig menampung field koneksi RouterOS.
type RouterConfig struct {
	Address  string
	Username string
	Password string

	TLS                bool
	InsecureSkipVerify bool

	DialTimeout              time.Duration
	ListenQueueSize          int
	ReconnectInitialInterval time.Duration
	ReconnectMaxInterval     time.Duration
	ReconnectMaxElapsed      time.Duration
}

// CacheConfig: toggle + parameter koneksi Redis (atau cache lain).
type CacheConfig struct {
	Enabled    bool
	Addr       string
	Password   string
	DB         int
	DefaultTTL time.Duration
}

// InfluxConfig: toggle + parameter koneksi InfluxDB3.
type InfluxConfig struct {
	Enabled            bool
	Host               string
	Token              string
	Database           string
	Organization       string
	DefaultMeasurement string
}

// LoadFromEnv membaca seluruh ROSLIB_* dan INFLUX_* env var,
// mengisi defaults untuk yang kosong.
func LoadFromEnv() (*Config, error) {
	cfg := &Config{
		Router: RouterConfig{
			Address:                  os.Getenv("ROSLIB_ROUTER_ADDRESS"),
			Username:                 os.Getenv("ROSLIB_ROUTER_USERNAME"),
			Password:                 os.Getenv("ROSLIB_ROUTER_PASSWORD"),
			TLS:                      envBool("ROSLIB_ROUTER_TLS", false),
			InsecureSkipVerify:       envBool("ROSLIB_ROUTER_INSECURE_SKIP_VERIFY", false),
			DialTimeout:              envDuration("ROSLIB_DIAL_TIMEOUT", 10*time.Second),
			ListenQueueSize:          envInt("ROSLIB_LISTEN_QUEUE_SIZE", 100),
			ReconnectInitialInterval: envDuration("ROSLIB_RECONNECT_INITIAL", 500*time.Millisecond),
			ReconnectMaxInterval:     envDuration("ROSLIB_RECONNECT_MAX", 30*time.Second),
			ReconnectMaxElapsed:      envDuration("ROSLIB_RECONNECT_MAX_ELAPSED", 0),
		},
		Cache: CacheConfig{
			Enabled:    envBool("ROSLIB_CACHE_ENABLED", false),
			Addr:       os.Getenv("ROSLIB_CACHE_ADDR"),
			Password:   os.Getenv("ROSLIB_CACHE_PASSWORD"),
			DB:         envInt("ROSLIB_CACHE_DB", 0),
			DefaultTTL: envDuration("ROSLIB_CACHE_TTL", 30*time.Second),
		},
		Influx: InfluxConfig{
			Enabled:            envBool("ROSLIB_INFLUX_ENABLED", false),
			Host:               os.Getenv("INFLUX_HOST"),
			Token:              os.Getenv("INFLUX_TOKEN"),
			Database:           os.Getenv("INFLUX_DATABASE"),
			Organization:       os.Getenv("INFLUX_ORG"),
			DefaultMeasurement: envString("ROSLIB_INFLUX_MEASUREMENT", "roslib"),
		},
	}
	return cfg, cfg.Validate()
}

// Validate cek precondition minimal.
//
// Catatan: ROSLIB_CACHE_ADDR hanya wajib kalau implementasi Redis dipakai
// (build-tag `redis`). Default InMemoryCache tidak butuh ADDR — caller
// tetap bisa enable cache tanpa set ADDR.
//
// Untuk Influx, TOKEN opsional supaya kompatibel dengan InfluxDB3 Core
// dev-mode (`--without-auth`). HOST + DATABASE tetap wajib.
func (c *Config) Validate() error {
	if c.Router.Address == "" {
		return errors.New("config: ROSLIB_ROUTER_ADDRESS is required")
	}
	if c.Influx.Enabled {
		if c.Influx.Host == "" {
			return errors.New("config: INFLUX_HOST required when ROSLIB_INFLUX_ENABLED=true")
		}
		if c.Influx.Database == "" {
			return errors.New("config: INFLUX_DATABASE required when ROSLIB_INFLUX_ENABLED=true")
		}
	}
	return nil
}

// ToDeviceOptions menerjemahkan RouterConfig + logger menjadi device.Options.
// TLS config (mis. cert pool) sengaja tidak di-materialize di sini — caller
// bisa override hasilnya kalau butuh TLS lebih kaya.
func (c *Config) ToDeviceOptions(log *logrus.Logger) device.Options {
	opts := device.Options{
		Address:                  c.Router.Address,
		Username:                 c.Router.Username,
		Password:                 c.Router.Password,
		Logger:                   log,
		ListenQueueSize:          c.Router.ListenQueueSize,
		DialTimeout:              c.Router.DialTimeout,
		ReconnectInitialInterval: c.Router.ReconnectInitialInterval,
		ReconnectMaxInterval:     c.Router.ReconnectMaxInterval,
		ReconnectMaxElapsed:      c.Router.ReconnectMaxElapsed,
	}
	if c.Router.TLS {
		opts.TLS = newTLSConfig(c.Router.InsecureSkipVerify)
	}
	return opts
}

// ──────────────── env helpers ────────────────

func envString(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	switch v {
	case "1", "true", "True", "TRUE", "yes", "on":
		return true
	case "0", "false", "False", "FALSE", "no", "off":
		return false
	}
	return def
}

func envInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func envDuration(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}
