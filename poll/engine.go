package poll

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/go-routeros/routeros/v3"
	"github.com/sirupsen/logrus"
)

// ErrDuplicateID dikembalikan Register jika ID sudah terpakai.
var ErrDuplicateID = errors.New("poll: duplicate config ID")

// ErrEmptyID dikembalikan Register jika ID kosong.
var ErrEmptyID = errors.New("poll: empty config ID")

// ErrZeroInterval dikembalikan Register jika Interval ≤ 0.
var ErrZeroInterval = errors.New("poll: interval must be > 0")

// Engine mengelola sekumpulan poll Config yang dikelompokkan berdasarkan
// interval. Goroutine count proporsional terhadap jumlah interval unik
// (group), bukan jumlah config.
type Engine struct {
	log *logrus.Entry

	mu     sync.RWMutex
	conn   *routeros.Client
	groups map[time.Duration]*group

	// index global ID → interval untuk lookup cepat saat Unregister.
	idx map[string]time.Duration

	ctx    context.Context
	cancel context.CancelFunc
}

// NewEngine membuat engine baru, terikat ke parent context.
// Hentikan engine dengan Close() atau cancel parent.
func NewEngine(parent context.Context, log *logrus.Entry, conn *routeros.Client) *Engine {
	ctx, cancel := context.WithCancel(parent)
	return &Engine{
		log:    log.WithField("component", "poll"),
		conn:   conn,
		groups: make(map[time.Duration]*group),
		idx:    make(map[string]time.Duration),
		ctx:    ctx,
		cancel: cancel,
	}
}

// AttachConn dipanggil oleh device setelah connCommand reconnect.
// Group tidak butuh re-register apa pun — group hanya pegang fungsi
// getter yang baca field conn di sini.
func (e *Engine) AttachConn(c *routeros.Client) {
	e.mu.Lock()
	e.conn = c
	e.mu.Unlock()
}

// Register menambahkan satu Config ke engine. Group dibuat on-demand
// kalau interval-nya belum ada.
func (e *Engine) Register(cfg Config) error {
	if cfg.ID == "" {
		return ErrEmptyID
	}
	if cfg.Interval <= 0 {
		return ErrZeroInterval
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.idx[cfg.ID]; exists {
		return ErrDuplicateID
	}

	g, ok := e.groups[cfg.Interval]
	if !ok {
		g = newGroup(e.log, cfg.Interval, e.connRef())
		g.start(e.ctx)
		e.groups[cfg.Interval] = g
	}
	g.add(cfg)
	e.idx[cfg.ID] = cfg.Interval

	e.log.WithFields(logrus.Fields{
		"poll_id":  cfg.ID,
		"path":     cfg.Path,
		"interval": cfg.Interval.String(),
	}).Info("poll registered")
	return nil
}

// Unregister menghapus Config dengan ID tertentu. Jika group menjadi kosong
// setelah penghapusan, ticker-nya juga dihentikan.
func (e *Engine) Unregister(id string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	interval, ok := e.idx[id]
	if !ok {
		return false
	}
	g := e.groups[interval]
	if g == nil {
		delete(e.idx, id)
		return false
	}
	g.remove(id)
	delete(e.idx, id)

	if g.size() == 0 {
		g.stop()
		delete(e.groups, interval)
	}

	e.log.WithField("poll_id", id).Info("poll unregistered")
	return true
}

// Close menghentikan semua group. Aman dipanggil berkali-kali.
func (e *Engine) Close() {
	e.cancel()
	e.mu.Lock()
	groups := make([]*group, 0, len(e.groups))
	for _, g := range e.groups {
		groups = append(groups, g)
	}
	e.mu.Unlock()
	for _, g := range groups {
		g.stop()
	}
}

// connRef mengembalikan closure yang selalu baca conn terkini.
func (e *Engine) connRef() func() *routeros.Client {
	return func() *routeros.Client {
		e.mu.RLock()
		defer e.mu.RUnlock()
		return e.conn
	}
}
