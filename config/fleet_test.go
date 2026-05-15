package config

import (
	"testing"
)

func TestLoadFleetFromEnv_Empty(t *testing.T) {
	// Pastikan tidak ada ROSLIB_ROUTERS — Setenv kosong.
	t.Setenv("ROSLIB_ROUTERS", "")
	if _, err := LoadFleetFromEnv(); err == nil {
		t.Error("expected error when ROSLIB_ROUTERS empty")
	}
}

func TestLoadFleetFromEnv_Single(t *testing.T) {
	t.Setenv("ROSLIB_ROUTERS", "rb1")
	t.Setenv("ROSLIB_ROUTER_RB1_ADDRESS", "192.168.88.1:8728")
	t.Setenv("ROSLIB_ROUTER_RB1_USERNAME", "admin")
	t.Setenv("ROSLIB_ROUTER_RB1_PASSWORD", "secret")

	cfg, err := LoadFleetFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Routers) != 1 {
		t.Fatalf("routers=%d, want 1", len(cfg.Routers))
	}
	if cfg.Routers[0].ID != "rb1" {
		t.Errorf("ID=%q", cfg.Routers[0].ID)
	}
	if cfg.Routers[0].Address != "192.168.88.1:8728" {
		t.Errorf("Address=%q", cfg.Routers[0].Address)
	}
}

func TestLoadFleetFromEnv_Multiple(t *testing.T) {
	t.Setenv("ROSLIB_ROUTERS", "rb1,rb2,office")
	t.Setenv("ROSLIB_ROUTER_RB1_ADDRESS", "10.0.0.1:8728")
	t.Setenv("ROSLIB_ROUTER_RB1_USERNAME", "admin")
	t.Setenv("ROSLIB_ROUTER_RB1_PASSWORD", "p1")
	t.Setenv("ROSLIB_ROUTER_RB2_ADDRESS", "10.0.0.2:8728")
	t.Setenv("ROSLIB_ROUTER_RB2_USERNAME", "admin")
	t.Setenv("ROSLIB_ROUTER_RB2_PASSWORD", "p2")
	t.Setenv("ROSLIB_ROUTER_OFFICE_ADDRESS", "10.0.0.3:8728")
	t.Setenv("ROSLIB_ROUTER_OFFICE_USERNAME", "user")
	t.Setenv("ROSLIB_ROUTER_OFFICE_PASSWORD", "p3")

	cfg, err := LoadFleetFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Routers) != 3 {
		t.Fatalf("routers=%d, want 3", len(cfg.Routers))
	}

	want := map[string]string{
		"rb1":    "10.0.0.1:8728",
		"rb2":    "10.0.0.2:8728",
		"office": "10.0.0.3:8728",
	}
	for _, r := range cfg.Routers {
		if w, ok := want[r.ID]; !ok || w != r.Address {
			t.Errorf("router %s: address=%q, want %q", r.ID, r.Address, w)
		}
	}
}

func TestLoadFleetFromEnv_MissingAddress(t *testing.T) {
	t.Setenv("ROSLIB_ROUTERS", "rb1,rb2")
	t.Setenv("ROSLIB_ROUTER_RB1_ADDRESS", "10.0.0.1:8728")
	// RB2 address sengaja kosong.
	if _, err := LoadFleetFromEnv(); err == nil {
		t.Error("expected error when one router missing address")
	}
}

func TestLoadFleetFromEnv_WithCacheAndInflux(t *testing.T) {
	t.Setenv("ROSLIB_ROUTERS", "rb1")
	t.Setenv("ROSLIB_ROUTER_RB1_ADDRESS", "10.0.0.1:8728")
	t.Setenv("ROSLIB_ROUTER_RB1_USERNAME", "admin")
	t.Setenv("ROSLIB_ROUTER_RB1_PASSWORD", "p")
	t.Setenv("ROSLIB_CACHE_ENABLED", "true")
	t.Setenv("ROSLIB_CACHE_ADDR", "127.0.0.1:6379")
	t.Setenv("ROSLIB_INFLUX_ENABLED", "true")
	t.Setenv("INFLUX_HOST", "http://localhost:8181")
	t.Setenv("INFLUX_TOKEN", "no-auth")
	t.Setenv("INFLUX_DATABASE", "mikrotik")

	cfg, err := LoadFleetFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Cache.Enabled || cfg.Cache.Addr != "127.0.0.1:6379" {
		t.Errorf("cache not loaded correctly: %+v", cfg.Cache)
	}
	if !cfg.Influx.Enabled || cfg.Influx.Database != "mikrotik" {
		t.Errorf("influx not loaded correctly: %+v", cfg.Influx)
	}
}

func TestLoadFleetFromEnv_DashID(t *testing.T) {
	t.Setenv("ROSLIB_ROUTERS", "office-gw")
	t.Setenv("ROSLIB_ROUTER_OFFICE_GW_ADDRESS", "10.0.0.1:8728")
	t.Setenv("ROSLIB_ROUTER_OFFICE_GW_USERNAME", "admin")
	t.Setenv("ROSLIB_ROUTER_OFFICE_GW_PASSWORD", "p")

	cfg, err := LoadFleetFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Routers[0].ID != "office-gw" {
		t.Errorf("ID=%q, want office-gw", cfg.Routers[0].ID)
	}
}
