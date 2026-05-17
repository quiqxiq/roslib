package devmgr

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/quiqxiq/roslib"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik/hotspot"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik/network"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik/ppp"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik/syslog"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik/system"
	"github.com/quiqxiq/roslib-mikhmon/store"
	"github.com/quiqxiq/roslib-mikhmon/store/model"
	"github.com/quiqxiq/roslib-mikhmon/workflows"
	"github.com/sirupsen/logrus"
)

// ClientSet adalah kumpulan klien yang sudah terikat ke satu roslib.Device.
type ClientSet struct {
	DeviceID uint
	Dev      *roslib.Device
	Hot      *hotspot.Client
	Sys      *system.Client
	Net      *network.Client
	PPP      *ppp.Client
	Log      *syslog.Client
	WF       *workflows.Clients
}

// Manager memuat device dari DB, mengelola koneksi, dan memperbarui status.
type Manager struct {
	store  store.DeviceStore
	log    *logrus.Logger
	mu     sync.RWMutex
	active map[string]*ClientSet // key = device slug
	ctx    context.Context       // root context, hidup sepanjang server

	// Hook callbacks — dipanggil setelah koneksi berhasil / device dihapus.
	// Set sebelum memanggil Start(). Thread-safe: hanya dibaca setelah Set.
	OnDeviceConnected func(d model.MikrotikDevice)
	OnDeviceRemoved   func(slug string)
}

func New(ds store.DeviceStore, log *logrus.Logger) *Manager {
	return &Manager{
		store:  ds,
		log:    log,
		active: make(map[string]*ClientSet),
	}
}

// Start memuat semua active device dari DB, mendial koneksi, lalu menjalankan
// loop status update tiap 30 detik di background.
func (m *Manager) Start(ctx context.Context) error {
	m.ctx = ctx
	devices, err := m.store.List(ctx)
	if err != nil {
		return err
	}
	for _, d := range devices {
		if err := m.connect(d); err != nil {
			m.log.WithError(err).Warnf("devmgr: failed to connect %s", d.Slug)
		}
	}
	return nil
}

// Get mengembalikan ClientSet by slug. Error jika tidak ditemukan.
func (m *Manager) Get(slug string) (*ClientSet, error) {
	m.mu.RLock()
	cs, ok := m.active[slug]
	m.mu.RUnlock()
	if !ok {
		return nil, errors.New("device not found or disconnected: " + slug)
	}
	return cs, nil
}

// ListActive mengembalikan snapshot slug→ClientSet untuk semua device yang sedang terhubung.
func (m *Manager) ListActive() map[string]*ClientSet {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make(map[string]*ClientSet, len(m.active))
	for k, v := range m.active {
		out[k] = v
	}
	return out
}

// Add mendaftarkan device baru (dipanggil setelah POST /devices).
// Pakai m.ctx (root context) bukan request context — koneksi harus hidup
// sepanjang server, bukan hanya selama HTTP request.
func (m *Manager) Add(_ context.Context, d model.MikrotikDevice) error {
	return m.connect(d)
}

// Remove mendiskoneksi device dan menghapusnya dari map.
func (m *Manager) Remove(slug string) {
	m.mu.Lock()
	cs, ok := m.active[slug]
	if ok {
		delete(m.active, slug)
	}
	m.mu.Unlock()

	if !ok {
		return
	}
	if m.OnDeviceRemoved != nil {
		m.OnDeviceRemoved(slug)
	}
	cs.Dev.Close()
}

// connect mendial roslib.Device dan membuat ClientSet.
// Selalu pakai m.ctx agar koneksi hidup sepanjang server.
func (m *Manager) connect(d model.MikrotikDevice) error {
	dev, err := roslib.New(m.ctx, roslib.Options{
		Address:        d.Address,
		Username:       d.Username,
		Password:       d.Password,
		Logger:         m.log,
		OnStatusChange: m.makeStatusHook(d),
	})
	if err != nil {
		now := time.Now()
		_ = m.store.UpdateStatus(m.ctx, d.ID, "error", err.Error(), &now)
		return err
	}

	cs := &ClientSet{
		DeviceID: d.ID,
		Dev:      dev,
		Hot:      hotspot.New(dev),
		Sys:      system.New(dev),
		Net:      network.New(dev),
		PPP:      ppp.New(dev),
		Log:      syslog.New(dev),
	}
	cs.WF = workflows.New(dev)

	m.mu.Lock()
	m.active[d.Slug] = cs
	m.mu.Unlock()

	now := time.Now()
	_ = m.store.UpdateStatus(m.ctx, d.ID, "connected", "", &now)

	if m.OnDeviceConnected != nil {
		m.OnDeviceConnected(d)
	}
	return nil
}

// makeStatusHook mengembalikan callback OnStatusChange yang update DB
// secara real-time setiap kali supervisor mendeteksi perubahan koneksi.
func (m *Manager) makeStatusHook(d model.MikrotikDevice) func(string, string) {
	dbStatus := map[string]string{
		"connected": "connected",
		"error":     "error",
		"closed":    "disconnected",
	}
	log := m.log.WithField("device", d.Slug)
	return func(status, errMsg string) {
		s, ok := dbStatus[status]
		if !ok {
			s = status
		}
		now := time.Now()
		_ = m.store.UpdateStatus(m.ctx, d.ID, s, errMsg, &now)
		log.WithField("status", s).Info("devmgr: device status changed")
	}
}
