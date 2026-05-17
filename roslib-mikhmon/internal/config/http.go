// Package config menyimpan loader env untuk HTTP server roslib-mikhmon.
// roslib internal config (router credentials, cache, influx) tetap di
// github.com/quiqxiq/roslib/config — paket ini hanya tambahan untuk
// HTTP server layer.
package config

import (
	"os"
	"strings"
	"time"
)

// HTTPConfig mengontrol perilaku http.Server di cmd/server.
type HTTPConfig struct {
	Bind          string        // mis. "127.0.0.1:8080" atau ":8080"
	ReadTimeout   time.Duration // default 10s
	IdleTimeout   time.Duration // default 60s
	ShutdownGrace time.Duration // grace period saat SIGINT/SIGTERM, default 10s
	CORSOrigins   []string      // default ["*"]; production override via CORS_ALLOWED_ORIGINS
}

// LoadHTTPFromEnv membaca env HTTP_* + CORS_ALLOWED_ORIGINS. Default-nya
// aman untuk local-only deployment.
func LoadHTTPFromEnv() (*HTTPConfig, error) {
	return &HTTPConfig{
		Bind:          getenv("HTTP_BIND", "127.0.0.1:8080"),
		ReadTimeout:   getenvDur("HTTP_READ_TIMEOUT", 10*time.Second),
		IdleTimeout:   getenvDur("HTTP_IDLE_TIMEOUT", 60*time.Second),
		ShutdownGrace: getenvDur("HTTP_SHUTDOWN_GRACE", 10*time.Second),
		CORSOrigins:   getenvList("CORS_ALLOWED_ORIGINS", []string{"*"}),
	}, nil
}

func getenv(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func getenvDur(key string, def time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}

func getenvList(key string, def []string) []string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return def
	}
	return out
}
