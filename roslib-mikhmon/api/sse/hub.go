package sse

import "sync"

// Hub adalah registry broker by-topic. Caller (handler/stream.go) panggil
// GetOrCreate untuk dapat broker — kalau topic belum ada, buat baru
// dengan onStart/onStop yang diberikan.
type Hub struct {
	mu      sync.Mutex
	brokers map[string]*Broker
}

// NewHub membuat hub kosong. Thread-safe untuk concurrent access dari
// banyak handler goroutine.
func NewHub() *Hub {
	return &Hub{brokers: make(map[string]*Broker)}
}

// GetOrCreate kembalikan broker untuk topic. Kalau belum ada, buat baru.
// startFn diberi pointer broker yang baru dibuat agar bisa pakai
// b.Publish sebagai handler stream backend.
//
// Catatan: hub memegang broker selamanya (tidak GC). Kalau topic
// dinamis (mis. log:<topics>), hati-hati dengan growth — untuk MVP
// jumlah topic kecil (< 50 unique).
func (h *Hub) GetOrCreate(topic string, startFn func(b *Broker) error, stopFn func()) *Broker {
	h.mu.Lock()
	defer h.mu.Unlock()

	if b, ok := h.brokers[topic]; ok {
		return b
	}
	var b *Broker
	b = NewBroker(topic, func() error { return startFn(b) }, stopFn)
	h.brokers[topic] = b
	return b
}
