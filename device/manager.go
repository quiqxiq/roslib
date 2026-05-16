package device

import (
	"context"
	"fmt"
	"sync"
)

// Manager mengelola kumpulan RouterDevice yang persisten.
// Setiap key bisa berupa nama router biasa ("office-gw") atau
// compound key dengan role ("office-gw:stream") untuk skenario
// multi-koneksi per router fisik.
//
// Contoh pemakaian:
//
//	mgr := device.NewManager()
//	_ = mgr.Register(ctx, "gw-1", opts)
//	dev, _ := mgr.Get("gw-1")         // ambil yang sudah ada
//	dev, _ := mgr.GetOrConnect(ctx, "gw-1", opts)  // buat kalau belum ada
type Manager struct {
	mu      sync.RWMutex
	devices map[string]*RouterDevice
}

// NewManager membuat Manager kosong siap pakai.
func NewManager() *Manager {
	return &Manager{
		devices: make(map[string]*RouterDevice),
	}
}

// Register membuat koneksi baru dan menyimpannya dengan key name.
// Error jika name sudah terdaftar (dan koneksi masih hidup).
// Gunakan Unregister lalu Register lagi jika ingin ganti konfigurasi.
func (m *Manager) Register(ctx context.Context, name string, opts Options) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if existing, ok := m.devices[name]; ok && existing.ctx.Err() == nil {
		return fmt.Errorf("manager: device %q already registered and still alive", name)
	}

	dev, err := New(ctx, opts)
	if err != nil {
		return fmt.Errorf("manager: register %q: %w", name, err)
	}
	m.devices[name] = dev
	return nil
}

// Get mengembalikan device yang sudah terdaftar.
// Error jika name belum terdaftar.
// Tidak memeriksa apakah koneksi masih hidup — caller bisa cek lewat IsAlive().
func (m *Manager) Get(name string) (*RouterDevice, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	dev, ok := m.devices[name]
	if !ok {
		return nil, fmt.Errorf("manager: device %q not registered", name)
	}
	return dev, nil
}

// GetOrConnect mengembalikan device yang ada jika masih hidup,
// atau membuat koneksi baru jika belum terdaftar / sudah mati.
//
// Ini adalah method utama yang dipakai di production: tidak perlu
// tahu apakah device sudah pernah diregister sebelumnya.
//
// Thread-safe via double-checked locking.
func (m *Manager) GetOrConnect(ctx context.Context, name string, opts Options) (*RouterDevice, error) {
	// fast path: sudah ada dan masih hidup
	m.mu.RLock()
	dev, ok := m.devices[name]
	m.mu.RUnlock()
	if ok && dev.IsAlive() {
		return dev, nil
	}

	// slow path: perlu buat baru
	m.mu.Lock()
	defer m.mu.Unlock()

	// cek ulang setelah acquire write lock (double-checked locking)
	dev, ok = m.devices[name]
	if ok && dev.IsAlive() {
		return dev, nil
	}

	// tutup koneksi lama yang sudah mati (kalau ada) sebelum buat baru
	if ok {
		_ = dev.Close()
	}

	newDev, err := New(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("manager: connect %q: %w", name, err)
	}
	m.devices[name] = newDev
	return newDev, nil
}

// Unregister menutup koneksi dan menghapus device dari map.
// No-op jika name tidak ditemukan.
func (m *Manager) Unregister(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if dev, ok := m.devices[name]; ok {
		_ = dev.Close()
		delete(m.devices, name)
	}
}

// CloseAll menutup semua koneksi dan mengosongkan map.
// Dipanggil saat aplikasi shutdown.
func (m *Manager) CloseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, dev := range m.devices {
		_ = dev.Close()
		delete(m.devices, name)
	}
}

// Names mengembalikan daftar semua key yang terdaftar saat ini.
func (m *Manager) Names() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.devices))
	for name := range m.devices {
		names = append(names, name)
	}
	return names
}

// ─────────────── Multi-koneksi per router (stream/query/mutation) ───────────────

// ConnectionRole mendefinisikan peran koneksi dalam skenario
// multi-koneksi ke satu router fisik.
type ConnectionRole string

const (
	RoleStream   ConnectionRole = "stream"
	RoleCommand  ConnectionRole = "command"
	RoleMutation ConnectionRole = "mutation"
)

// RoleKey menghasilkan compound key "routerName:role" untuk dipakai
// sebagai key di Register/Get/GetOrConnect.
//
//	key := device.RoleKey("office-gw", device.RoleStream)
//	// → "office-gw:stream"
func RoleKey(routerName string, role ConnectionRole) string {
	return routerName + ":" + string(role)
}