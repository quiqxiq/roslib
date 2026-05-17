package hotspot

import (
	"context"
	"errors"
	"time"
)

// errMonitorIDInUse dikembalikan kalau ID sudah dipakai monitor lain.
var errMonitorIDInUse = errors.New("hotspot: monitor id already registered")

// MonitorActiveCount → poll /ip/hotspot/active count-only="" tiap interval
// (analisis §1.8). Sidebar counter mikhmon dashboard.
//
// Implementasi pakai manual goroutine + ticker karena RouterOS API
// emit hanya !done (dengan =ret=N) untuk count-only — PollEngine roslib
// hanya panggil handler untuk !re. ID di-track di Client agar
// StopMonitor(id) bisa cancel goroutine.
func (c *Client) MonitorActiveCount(id string, interval time.Duration, h func(int)) error {
	return c.startTicker(id, interval, func(ctx context.Context) {
		if n, err := c.ActiveCount(ctx); err == nil {
			h(n)
		}
	})
}

// MonitorUserCount → poll /ip/hotspot/user count-only="" tiap interval.
// Counter total user terdaftar.
func (c *Client) MonitorUserCount(id string, interval time.Duration, h func(int)) error {
	return c.startTicker(id, interval, func(ctx context.Context) {
		if n, err := c.UserCount(ctx); err == nil {
			h(n)
		}
	})
}

// startTicker mendaftarkan goroutine ticker dengan ID tertentu. Goroutine
// menjalankan fn tiap interval sampai cancel dipanggil via StopMonitor(id).
func (c *Client) startTicker(id string, interval time.Duration, fn func(ctx context.Context)) error {
	if id == "" || interval <= 0 {
		return errors.New("hotspot: invalid id or interval")
	}
	c.tickersMu.Lock()
	if _, exists := c.tickers[id]; exists {
		c.tickersMu.Unlock()
		return errMonitorIDInUse
	}
	ctx, cancel := context.WithCancel(context.Background())
	c.tickers[id] = cancel
	c.tickersMu.Unlock()

	go func() {
		defer func() {
			c.tickersMu.Lock()
			delete(c.tickers, id)
			c.tickersMu.Unlock()
		}()
		// First tick segera supaya UI tidak nunggu interval pertama.
		fn(ctx)
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				fn(ctx)
			}
		}
	}()
	return nil
}

// StopMonitor menghentikan ticker atau poll dengan ID tersebut.
// Cek manual-ticker dulu, lalu fallback ke dev.UnregisterPoll.
// Return true bila ditemukan dan dihentikan.
func (c *Client) StopMonitor(id string) bool {
	c.tickersMu.Lock()
	cancel, ok := c.tickers[id]
	if ok {
		cancel()
		delete(c.tickers, id)
		c.tickersMu.Unlock()
		return true
	}
	c.tickersMu.Unlock()
	return c.dev.UnregisterPoll(id)
}
