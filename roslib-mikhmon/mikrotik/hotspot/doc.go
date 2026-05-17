// Package hotspot membungkus path /ip/hotspot/* RouterOS yang dipakai mikhmon:
// server, user, profile, active, host, cookie, ip-binding.
//
// Cross-ref: analisis §1.6 – §1.9.
package hotspot

import (
	"context"
	"sync"

	"github.com/quiqxiq/roslib"
)

// Client adalah handle untuk semua command /ip/hotspot/*.
//
// Field tickers menyimpan goroutine ticker yang dispawn oleh
// MonitorActiveCount / MonitorUserCount (lihat monitor.go). Method-method
// itu pakai manual ticker karena RouterOS hanya emit !done untuk count-only,
// sedangkan PollEngine roslib hanya panggil handler untuk !re.
type Client struct {
	dev *roslib.Device

	tickersMu sync.Mutex
	tickers   map[string]context.CancelFunc
}

// New membuat Client baru.
func New(dev *roslib.Device) *Client {
	return &Client{
		dev:     dev,
		tickers: make(map[string]context.CancelFunc),
	}
}
