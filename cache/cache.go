// Package cache mendefinisikan interface Cache untuk hasil query RouterOS
// + dua implementasi default (Noop, InMemory). Implementasi Redis tersedia
// di file build-tag terpisah (cache/redis.go, tag `redis`) sehingga go-redis
// tidak menjadi dependency wajib library inti.
package cache

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// Stats merangkum counter cache untuk observability + assertion di test.
// Hits + Misses + Sets di-track via atomic, Entries dihitung saat panggilan.
type Stats struct {
	Entries int64
	Hits    int64
	Misses  int64
	Sets    int64
}

// Cache adalah store key/value sederhana dengan TTL + invalidasi
// berdasarkan path RouterOS.
//
// Get/Set: contract standar.
// InvalidatePath: hapus seluruh entry yang berasal dari path tertentu.
// Implementasi yang tidak punya tracking path (mis. NoopCache) boleh
// melakukan no-op atau menghapus seluruh entry — yang penting konsisten
// dengan kontrak: setelah InvalidatePath(path), tidak ada Get(key) yang
// pernah di-Set untuk path itu boleh hit.
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, bool, error)
	Set(ctx context.Context, key string, val []byte, ttl time.Duration) error

	// InvalidatePath menghapus semua entry yang ter-asosiasi dengan path.
	// Idempotent.
	InvalidatePath(ctx context.Context, path string) error
}

// PathAwareCache adalah extension interface untuk impl yang bisa track
// path saat Set. ExecCached akan pakai SetForPath kalau cache memenuhi
// interface ini; kalau tidak, fallback ke Set (InvalidatePath jadi best
// effort — biasanya no-op).
type PathAwareCache interface {
	Cache
	SetForPath(ctx context.Context, path, key string, val []byte, ttl time.Duration) error
}

// DeviceScopedCache adalah extension interface untuk impl yang bisa
// invalidate per (device, path). Pakai ini di fleet mode supaya
// dev.InvalidateCache hanya hapus entry milik device tersebut.
type DeviceScopedCache interface {
	Cache
	InvalidatePathForDevice(ctx context.Context, deviceID, path string) error
}

// ──────────────── NoopCache ────────────────

// NoopCache selalu miss & no-op pada Set. Default kalau Config.Cache.Enabled
// false, sehingga caller tidak perlu nil-check.
type NoopCache struct{}

func (NoopCache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	return nil, false, nil
}
func (NoopCache) Set(ctx context.Context, key string, val []byte, ttl time.Duration) error {
	return nil
}
func (NoopCache) InvalidatePath(ctx context.Context, path string) error { return nil }

// ──────────────── InMemoryCache ────────────────

// InMemoryCache adalah implementasi map + mutex untuk testing dan
// deployment tanpa Redis. TTL dievaluasi saat Get (lazy expiry).
//
// Selain map data utama, ada secondary index pathIdx[path] = set{key} untuk
// mendukung InvalidatePath. Index hanya terisi saat caller pakai
// SetForPath; Set biasa tidak track path apa pun.
type InMemoryCache struct {
	mu      sync.RWMutex
	data    map[string]inMemoryEntry
	pathIdx map[string]map[string]struct{}
	keyPath map[string]string // reverse: key → path (untuk cleanup pas expiry)

	hits   atomic.Int64
	misses atomic.Int64
	sets   atomic.Int64
}

type inMemoryEntry struct {
	val []byte
	exp time.Time // zero = no expiry
}

// NewInMemory mengembalikan InMemoryCache siap pakai.
func NewInMemory() *InMemoryCache {
	return &InMemoryCache{
		data:    make(map[string]inMemoryEntry),
		pathIdx: make(map[string]map[string]struct{}),
		keyPath: make(map[string]string),
	}
}

func (c *InMemoryCache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	c.mu.RLock()
	e, ok := c.data[key]
	c.mu.RUnlock()
	if !ok {
		c.misses.Add(1)
		return nil, false, nil
	}
	if !e.exp.IsZero() && time.Now().After(e.exp) {
		c.mu.Lock()
		c.removeKeyLocked(key)
		c.mu.Unlock()
		c.misses.Add(1)
		return nil, false, nil
	}
	c.hits.Add(1)
	return e.val, true, nil
}

func (c *InMemoryCache) Set(ctx context.Context, key string, val []byte, ttl time.Duration) error {
	var exp time.Time
	if ttl > 0 {
		exp = time.Now().Add(ttl)
	}
	c.mu.Lock()
	c.data[key] = inMemoryEntry{val: val, exp: exp}
	c.mu.Unlock()
	c.sets.Add(1)
	return nil
}

// SetForPath identik Set tapi juga record key ke pathIdx[path] supaya
// InvalidatePath(path) bisa hapus.
func (c *InMemoryCache) SetForPath(ctx context.Context, path, key string, val []byte, ttl time.Duration) error {
	var exp time.Time
	if ttl > 0 {
		exp = time.Now().Add(ttl)
	}
	c.mu.Lock()
	c.data[key] = inMemoryEntry{val: val, exp: exp}
	if path != "" {
		set, ok := c.pathIdx[path]
		if !ok {
			set = make(map[string]struct{})
			c.pathIdx[path] = set
		}
		set[key] = struct{}{}
		c.keyPath[key] = path
	}
	c.mu.Unlock()
	c.sets.Add(1)
	return nil
}

// InvalidatePath hapus semua entry yang terdaftar untuk path tersebut.
// Scope: GLOBAL — entry untuk semua device dengan path ini akan dihapus.
// Untuk invalidate per-device, pakai InvalidatePathForDevice.
// Aman dipanggil kalau path tidak ada.
func (c *InMemoryCache) InvalidatePath(ctx context.Context, path string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	set, ok := c.pathIdx[path]
	if !ok {
		return nil
	}
	for key := range set {
		delete(c.data, key)
		delete(c.keyPath, key)
	}
	delete(c.pathIdx, path)
	return nil
}

// InvalidatePathForDevice menghapus entry path hanya untuk device tertentu.
// Cocok untuk fleet mode dengan shared cache instance — invalidate di satu
// device tidak mempengaruhi device lain.
//
// Implementasi: filter pathIdx[path] berdasarkan prefix key
// "roslib:<deviceID>:". Lihat cache.KeyOf untuk format key.
func (c *InMemoryCache) InvalidatePathForDevice(ctx context.Context, deviceID, path string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	set, ok := c.pathIdx[path]
	if !ok {
		return nil
	}
	prefix := "roslib:" + deviceID + ":"
	removed := make([]string, 0, len(set))
	for key := range set {
		if !startsWith(key, prefix) {
			continue
		}
		delete(c.data, key)
		delete(c.keyPath, key)
		removed = append(removed, key)
	}
	for _, key := range removed {
		delete(set, key)
	}
	if len(set) == 0 {
		delete(c.pathIdx, path)
	}
	return nil
}

func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

// Len mengembalikan jumlah entry; berguna untuk asersi di test.
func (c *InMemoryCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.data)
}

// Stats mengembalikan snapshot counter cache. Aman dipanggil dari goroutine
// mana pun. Entries dihitung saat dipanggil (bukan counter); Hits/Misses/Sets
// di-track atomik per operasi.
func (c *InMemoryCache) Stats() Stats {
	c.mu.RLock()
	entries := int64(len(c.data))
	c.mu.RUnlock()
	return Stats{
		Entries: entries,
		Hits:    c.hits.Load(),
		Misses:  c.misses.Load(),
		Sets:    c.sets.Load(),
	}
}

// removeKeyLocked menghapus key dari semua struktur. Caller harus pegang
// write lock.
func (c *InMemoryCache) removeKeyLocked(key string) {
	delete(c.data, key)
	if path, ok := c.keyPath[key]; ok {
		delete(c.keyPath, key)
		if set, sok := c.pathIdx[path]; sok {
			delete(set, key)
			if len(set) == 0 {
				delete(c.pathIdx, path)
			}
		}
	}
}
