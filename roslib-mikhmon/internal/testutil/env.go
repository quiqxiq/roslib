package testutil

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/quiqxiq/roslib"
	"github.com/sirupsen/logrus"
)

// RequireIntegration men-skip test kalau env ROSLIB_ROUTER_ADDRESS kosong.
// Wajib dipanggil di awal setiap test integration.
func RequireIntegration(t *testing.T) {
	t.Helper()
	if os.Getenv("ROSLIB_ROUTER_ADDRESS") == "" {
		t.Skip("integration test: set ROSLIB_ROUTER_ADDRESS to enable")
	}
}

// IntegrationPrefix mengembalikan prefix untuk resource yang dibuat saat
// test. Default "mikhmon-it"; bisa di-override via MIKHMON_IT_PREFIX.
func IntegrationPrefix() string {
	if p := os.Getenv("MIKHMON_IT_PREFIX"); p != "" {
		return p
	}
	return "mikhmon-it"
}

// UniqueName mengembalikan "<prefix>-<test-suffix>-<rand>" yang aman
// dipakai sebagai nama resource sekaligus mudah di-cleanup manual.
func UniqueName(t *testing.T, suffix string) string {
	t.Helper()
	name := strings.ReplaceAll(t.Name(), "/", "_")
	name = strings.ToLower(name)
	if len(name) > 32 {
		name = name[:32]
	}
	rb := make([]byte, 4)
	_, _ = rand.Read(rb)
	rnd := hex.EncodeToString(rb)
	parts := []string{IntegrationPrefix(), name, suffix, rnd}
	out := ""
	for _, p := range parts {
		if p == "" {
			continue
		}
		if out != "" {
			out += "-"
		}
		out += p
	}
	return out
}

// NewClient men-dial router via env (lewat roslib.NewFromConfig) dan
// mengembalikan *roslib.Device siap pakai. Cleanup mgr otomatis via
// t.Cleanup — caller cukup pass dev langsung ke sub-package mikrotik
// (mis. hotspot.New(dev), system.New(dev)).
func NewClient(t *testing.T) *roslib.Device {
	t.Helper()
	RequireIntegration(t)

	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	cfg, err := loadConfigFromEnv()
	if err != nil {
		t.Fatalf("testutil: load config: %v", err)
	}
	mgr, _, err := roslib.NewFromConfig(context.Background(), cfg, logger)
	if err != nil {
		t.Fatalf("testutil: connect router: %v", err)
	}
	t.Cleanup(func() { mgr.CloseAll() })

	dev, err := mgr.Get(roslib.DefaultDeviceKey)
	if err != nil {
		t.Fatalf("testutil: acquire device: %v", err)
	}
	return dev
}

// DefaultTimeout adalah context timeout yang masuk akal untuk kebanyakan
// command integration test.
const DefaultTimeout = 10 * time.Second

// Context mengembalikan context dengan DefaultTimeout dan cancel
// otomatis di t.Cleanup.
func Context(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), DefaultTimeout)
	t.Cleanup(cancel)
	return ctx
}
