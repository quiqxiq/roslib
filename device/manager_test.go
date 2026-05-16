package device

import (
	"context"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

// unreachableAddr menyediakan address TCP yang sengaja tidak listen — dial
// pasti gagal cepat (RST atau timeout). Reservasi port via net.Listen kemudian
// Close supaya host:port valid tapi tidak ada server.
func unreachableAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	_ = l.Close()
	// Jeda kecil untuk memastikan TIME_WAIT tidak konflik.
	time.Sleep(10 * time.Millisecond)
	return addr
}

func newSilentLogger() *logrus.Logger {
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	return log
}

// TestGetOrConnectDialFailRemovesStaleEntry memastikan saat dial gagal di
// slow path GetOrConnect, entry stale (dari attempt sebelumnya) sudah
// dihapus dari map — Get(name) selanjutnya konsisten not-found.
func TestGetOrConnectDialFailRemovesStaleEntry(t *testing.T) {
	mgr := NewManager()
	log := newSilentLogger()

	host, portStr, _ := net.SplitHostPort(unreachableAddr(t))
	port, _ := strconv.Atoi(portStr)
	_ = port

	badOpts := Options{
		Address:                  net.JoinHostPort(host, portStr),
		Username:                 "admin",
		Password:                 "x",
		Logger:                   log,
		DialTimeout:              200 * time.Millisecond,
		ReconnectInitialInterval: 50 * time.Millisecond,
		ReconnectMaxInterval:     100 * time.Millisecond,
		ReconnectMaxElapsed:      100 * time.Millisecond,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Attempt 1: dial gagal (first attempt, slow path masuk ke branch
	// "tidak ada stale entry"). Tidak ada entry sebelumnya, jadi map
	// harus tetap kosong setelah error.
	if _, err := mgr.GetOrConnect(ctx, "ghost", badOpts); err == nil {
		t.Fatal("expected dial fail")
	}

	mgr.mu.RLock()
	_, present := mgr.devices["ghost"]
	mgr.mu.RUnlock()
	if present {
		t.Error("ghost entry leaked into map after first failed dial")
	}
	if _, gerr := mgr.Get("ghost"); gerr == nil {
		t.Error("expected Get to fail (no entry) after first failed dial")
	}
}

// TestGetOrConnectDialFailAfterStaleEntryRemoved memastikan skenario: entry
// pernah ada (alive=false), GetOrConnect re-dial gagal, entry dibersihkan.
// Tidak memakai dial baru — kita inject stale dengan Register manual + Close.
func TestGetOrConnectDialFailAfterStaleEntryRemoved(t *testing.T) {
	mgr := NewManager()
	log := newSilentLogger()

	// Buat fake stale device: kita perlu RouterDevice yang IsAlive=false.
	// Cara paling murah: Register dengan opts unreachable → gagal. Manager
	// tidak akan simpan entry. Sebagai gantinya, langsung suntik stale
	// device via field map (white-box).
	addr := unreachableAddr(t)
	staleCtx, staleCancel := context.WithCancel(context.Background())
	staleCancel() // ctx canceled → IsAlive=false
	stale := &RouterDevice{
		opts:   Options{Address: addr, Logger: log},
		log:    log.WithField("router", addr),
		ctx:    staleCtx,
		cancel: staleCancel,
	}

	mgr.mu.Lock()
	mgr.devices["zombie"] = stale
	mgr.mu.Unlock()

	// Sanity: stale device IS in map but IsAlive=false.
	if stale.IsAlive() {
		t.Fatal("setup: stale should be IsAlive=false")
	}
	if _, gerr := mgr.Get("zombie"); gerr != nil {
		t.Fatalf("setup: Get should find stale; got %v", gerr)
	}

	// GetOrConnect dengan bad opts → dial gagal. Stale harus dibersihkan.
	badOpts := Options{
		Address:                  addr,
		Username:                 "admin",
		Password:                 "x",
		Logger:                   log,
		DialTimeout:              200 * time.Millisecond,
		ReconnectInitialInterval: 50 * time.Millisecond,
		ReconnectMaxInterval:     100 * time.Millisecond,
		ReconnectMaxElapsed:      100 * time.Millisecond,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if _, err := mgr.GetOrConnect(ctx, "zombie", badOpts); err == nil {
		t.Fatal("expected dial fail")
	}

	// Stale harus sudah dihapus.
	mgr.mu.RLock()
	_, stillThere := mgr.devices["zombie"]
	mgr.mu.RUnlock()
	if stillThere {
		t.Error("stale entry not cleaned after dial fail")
	}
	if _, gerr := mgr.Get("zombie"); gerr == nil {
		t.Error("Get(zombie) should error after stale cleanup")
	}
}
