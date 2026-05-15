package config

import (
	"testing"
	"time"
)

func TestLoadFromEnvDefaults(t *testing.T) {
	t.Setenv("ROSLIB_ROUTER_ADDRESS", "10.0.0.1:8728")
	t.Setenv("ROSLIB_ROUTER_USERNAME", "admin")
	t.Setenv("ROSLIB_ROUTER_PASSWORD", "secret")

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Router.DialTimeout != 10*time.Second {
		t.Errorf("DialTimeout default = %v, want 10s", cfg.Router.DialTimeout)
	}
	if cfg.Router.ListenQueueSize != 100 {
		t.Errorf("ListenQueueSize default = %d, want 100", cfg.Router.ListenQueueSize)
	}
	if !cfg.StrictCapability {
		t.Error("StrictCapability default should be true")
	}
	if cfg.Cache.Enabled || cfg.Influx.Enabled {
		t.Error("Cache & Influx default should be disabled")
	}
}

func TestLoadFromEnvCustom(t *testing.T) {
	t.Setenv("ROSLIB_ROUTER_ADDRESS", "10.0.0.1:8728")
	t.Setenv("ROSLIB_ROUTER_USERNAME", "admin")
	t.Setenv("ROSLIB_ROUTER_PASSWORD", "secret")
	t.Setenv("ROSLIB_DIAL_TIMEOUT", "20s")
	t.Setenv("ROSLIB_STRICT_CAPABILITY", "false")
	t.Setenv("ROSLIB_CACHE_ENABLED", "true")
	t.Setenv("ROSLIB_CACHE_ADDR", "127.0.0.1:6379")
	t.Setenv("ROSLIB_CACHE_TTL", "1m")
	t.Setenv("ROSLIB_INFLUX_ENABLED", "true")
	t.Setenv("INFLUX_HOST", "https://cloud.influxdata.com")
	t.Setenv("INFLUX_TOKEN", "test-token")
	t.Setenv("INFLUX_DATABASE", "mydb")

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Router.DialTimeout != 20*time.Second {
		t.Errorf("DialTimeout = %v, want 20s", cfg.Router.DialTimeout)
	}
	if cfg.StrictCapability {
		t.Error("StrictCapability = true, expected false")
	}
	if !cfg.Cache.Enabled {
		t.Error("Cache should be enabled")
	}
	if cfg.Cache.DefaultTTL != time.Minute {
		t.Errorf("DefaultTTL = %v, want 1m", cfg.Cache.DefaultTTL)
	}
	if !cfg.Influx.Enabled {
		t.Error("Influx should be enabled")
	}
}

func TestValidateMissingAddress(t *testing.T) {
	cfg := &Config{}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for missing ROSLIB_ROUTER_ADDRESS")
	}
}

// Cache enabled tanpa ADDR adalah valid (InMemory default). Test ini
// memverifikasi validation tidak salah-reject use case ini.
func TestValidateCacheEnabledWithoutAddr(t *testing.T) {
	cfg := &Config{
		Router: RouterConfig{Address: "x:8728"},
		Cache:  CacheConfig{Enabled: true},
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("InMemory cache should not require ADDR; got %v", err)
	}
}

func TestValidateInfluxRequiresHostTokenDB(t *testing.T) {
	cfg := &Config{
		Router: RouterConfig{Address: "x:8728"},
		Influx: InfluxConfig{Enabled: true, Host: "h", Token: "t"},
	}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error when Influx enabled but Database empty")
	}
}
