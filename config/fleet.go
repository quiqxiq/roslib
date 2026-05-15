package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// RouterEntry mendeskripsikan satu router di fleet — RouterConfig + ID.
type RouterEntry struct {
	ID string
	RouterConfig
}

// FleetConfig adalah konfigurasi multi-router. Cache + Influx + behavior
// flag di-share antar router.
type FleetConfig struct {
	Routers []RouterEntry

	Cache  CacheConfig
	Influx InfluxConfig

	StrictCapability bool
	RegistryPath     string
}

// LoadFleetFromEnv membaca env scheme multi-router:
//
//	ROSLIB_ROUTERS=rb1,rb2,office
//	ROSLIB_ROUTER_RB1_ADDRESS=192.168.88.1:8728
//	ROSLIB_ROUTER_RB1_USERNAME=admin
//	ROSLIB_ROUTER_RB1_PASSWORD=secret
//	...
//
// Cache/Influx/behavior dibaca dari env shared sama seperti LoadFromEnv.
func LoadFleetFromEnv() (*FleetConfig, error) {
	list := os.Getenv("ROSLIB_ROUTERS")
	if list == "" {
		return nil, errors.New("config: ROSLIB_ROUTERS empty — set comma-separated ID list (mis. \"rb1,rb2\")")
	}

	ids := splitAndTrim(list)
	if len(ids) == 0 {
		return nil, errors.New("config: ROSLIB_ROUTERS contains no IDs")
	}

	cfg := &FleetConfig{
		Routers: make([]RouterEntry, 0, len(ids)),
	}

	// Cache/Influx/behavior re-use loader yang sama dari LoadFromEnv —
	// kita panggil dulu, abaikan error router (LoadFromEnv require
	// ROSLIB_ROUTER_ADDRESS yang tidak relevan di fleet mode).
	shared, _ := LoadFromEnv() // ignore error: router-required diabaikan di sini
	if shared != nil {
		cfg.Cache = shared.Cache
		cfg.Influx = shared.Influx
		cfg.StrictCapability = shared.StrictCapability
		cfg.RegistryPath = shared.RegistryPath
	}

	for _, id := range ids {
		envID := strings.ToUpper(strings.ReplaceAll(id, "-", "_"))
		prefix := "ROSLIB_ROUTER_" + envID + "_"

		addr := os.Getenv(prefix + "ADDRESS")
		if addr == "" {
			return nil, fmt.Errorf("config: %sADDRESS empty for router id=%q", prefix, id)
		}

		entry := RouterEntry{
			ID: id,
			RouterConfig: RouterConfig{
				Address:                  addr,
				Username:                 os.Getenv(prefix + "USERNAME"),
				Password:                 os.Getenv(prefix + "PASSWORD"),
				TLS:                      envBool(prefix+"TLS", false),
				InsecureSkipVerify:       envBool(prefix+"INSECURE_SKIP_VERIFY", false),
				DialTimeout:              envDuration("ROSLIB_DIAL_TIMEOUT", 0),
				ListenQueueSize:          envInt("ROSLIB_LISTEN_QUEUE_SIZE", 0),
				ReconnectInitialInterval: envDuration("ROSLIB_RECONNECT_INITIAL", 0),
				ReconnectMaxInterval:     envDuration("ROSLIB_RECONNECT_MAX", 0),
				ReconnectMaxElapsed:      envDuration("ROSLIB_RECONNECT_MAX_ELAPSED", 0),
			},
		}
		cfg.Routers = append(cfg.Routers, entry)
	}

	// Validate cache/influx kebutuhan (sama dengan single-router Validate).
	// Cache.Addr opsional — InMemoryCache default tidak butuh.
	// Influx.Token opsional — InfluxDB3 Core dev-mode (--without-auth).
	if cfg.Influx.Enabled {
		if cfg.Influx.Host == "" {
			return nil, errors.New("config: INFLUX_HOST required when ROSLIB_INFLUX_ENABLED=true")
		}
		if cfg.Influx.Database == "" {
			return nil, errors.New("config: INFLUX_DATABASE required when ROSLIB_INFLUX_ENABLED=true")
		}
	}

	return cfg, nil
}

func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
