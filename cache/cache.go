// Package cache mendefinisikan interface Cache untuk hasil query RouterOS
// + dua implementasi default (Noop, InMemory). Implementasi Redis tersedia
// di file build-tag terpisah (cache/redis.go, tag `redis`) sehingga go-redis
// tidak menjadi dependency wajib library inti.
package cache

import (
	"context"
	"sync"
	"time"
)

// Cache adalah store key/value sederhana dengan TTL.
//
// Get mengembalikan (value, hit, error). hit=false ⇒ miss (bukan error).
// Set menyimpan value dengan TTL relatif; ttl=0 berarti tidak expire
// (atau pakai default impl, implementation-defined).
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, bool, error)
	Set(ctx context.Context, key string, val []byte, ttl time.Duration) error
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

// ──────────────── InMemoryCache ────────────────

// InMemoryCache adalah implementasi map + mutex untuk testing dan
// deployment tanpa Redis. TTL dievaluasi saat Get (lazy expiry).
type InMemoryCache struct {
	mu   sync.RWMutex
	data map[string]inMemoryEntry
}

type inMemoryEntry struct {
	val []byte
	exp time.Time // zero = no expiry
}

// NewInMemory mengembalikan InMemoryCache siap pakai.
func NewInMemory() *InMemoryCache {
	return &InMemoryCache{data: make(map[string]inMemoryEntry)}
}

func (c *InMemoryCache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	c.mu.RLock()
	e, ok := c.data[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false, nil
	}
	if !e.exp.IsZero() && time.Now().After(e.exp) {
		c.mu.Lock()
		delete(c.data, key)
		c.mu.Unlock()
		return nil, false, nil
	}
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
	return nil
}

// Len mengembalikan jumlah entry; berguna untuk asersi di test.
func (c *InMemoryCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.data)
}
