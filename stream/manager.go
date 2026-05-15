package stream

import (
	"context"
	"errors"
	"sync"

	"github.com/go-routeros/routeros/v3"
	"github.com/go-routeros/routeros/v3/proto"
	"github.com/quiqxiq/roslib/decode"
	"github.com/sirupsen/logrus"
)

// ErrDuplicateID dikembalikan Register jika listener ID sudah dipakai.
var ErrDuplicateID = errors.New("stream: duplicate listener ID")

// ErrEmptyID dikembalikan Register jika ID kosong.
var ErrEmptyID = errors.New("stream: empty listener ID")

// listenReply abstrak antar-muka subset *routeros.ListenReply yang dipakai
// Manager, agar mudah di-mock di unit test.
type listenReply interface {
	Chan() <-chan *proto.Sentence
	CancelContext(ctx context.Context) (*routeros.Reply, error)
	Err() error
}

// Manager mengelola listener pada satu connStream. Semua listener berbagi
// koneksi yang sama via async + tag demux.
type Manager struct {
	log *logrus.Entry

	mu        sync.RWMutex
	conn      *routeros.Client
	listeners map[string]*listener

	ctx    context.Context
	cancel context.CancelFunc
}

// NewManager membuat Manager baru terikat ke parent context.
func NewManager(parent context.Context, log *logrus.Entry, conn *routeros.Client) *Manager {
	ctx, cancel := context.WithCancel(parent)
	return &Manager{
		log:       log.WithField("component", "stream"),
		conn:      conn,
		listeners: make(map[string]*listener),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Register memulai listener di koneksi saat ini. Spec di-simpan agar
// bisa di-reattach pasca reconnect.
func (m *Manager) Register(spec Spec) error {
	if spec.ID == "" {
		return ErrEmptyID
	}

	m.mu.Lock()
	if _, exists := m.listeners[spec.ID]; exists {
		m.mu.Unlock()
		return ErrDuplicateID
	}
	conn := m.conn
	l := &listener{spec: spec}
	m.listeners[spec.ID] = l
	m.mu.Unlock()

	if err := m.attach(conn, l); err != nil {
		// Roll-back registrasi kalau attach gagal.
		m.mu.Lock()
		delete(m.listeners, spec.ID)
		m.mu.Unlock()
		return err
	}

	m.log.WithFields(logrus.Fields{
		"listener_id": spec.ID,
		"word":        spec.Word,
	}).Info("stream attached")
	return nil
}

// Unregister menghentikan listener via /cancel dan menghapusnya dari registry.
// Mengembalikan false jika ID tidak ditemukan.
func (m *Manager) Unregister(id string) bool {
	m.mu.Lock()
	l, ok := m.listeners[id]
	if !ok {
		m.mu.Unlock()
		return false
	}
	delete(m.listeners, id)
	m.mu.Unlock()

	m.detach(l)
	m.log.WithField("listener_id", id).Info("stream detached")
	return true
}

// ReattachAll dipanggil oleh supervisor connStream setelah reconnect.
// Snapshot semua spec dan daftar ulang di koneksi baru. Listener lama
// sudah otomatis mati ketika koneksi lamanya tutup.
func (m *Manager) ReattachAll(newConn *routeros.Client) {
	m.mu.Lock()
	m.conn = newConn
	snapshot := make([]*listener, 0, len(m.listeners))
	for _, l := range m.listeners {
		snapshot = append(snapshot, l)
	}
	m.mu.Unlock()

	for _, l := range snapshot {
		if err := m.attach(newConn, l); err != nil {
			m.log.WithError(err).WithField("listener_id", l.spec.ID).
				Error("listener reattach failed")
			continue
		}
		m.log.WithField("listener_id", l.spec.ID).Info("listener reattached")
	}
}

// Close menghentikan semua listener. Aman dipanggil berkali-kali.
func (m *Manager) Close() {
	m.cancel()
	m.mu.Lock()
	all := make([]*listener, 0, len(m.listeners))
	for _, l := range m.listeners {
		all = append(all, l)
	}
	m.listeners = make(map[string]*listener)
	m.mu.Unlock()
	for _, l := range all {
		m.detach(l)
	}
}

// Len mengembalikan jumlah listener aktif. Entry yang sudah selesai natural
// (router kirim !done) sudah otomatis dihapus, jadi nilai ini akurat untuk
// monitoring finite-stream cleanup.
func (m *Manager) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.listeners)
}

// ──────────────── internal ────────────────

func (m *Manager) attach(conn *routeros.Client, l *listener) error {
	if conn == nil {
		return errors.New("stream: nil connection")
	}
	lctx, cancel := context.WithCancel(m.ctx)
	l.cancel = cancel

	sentence := l.spec.command()
	queue := l.spec.QueueSize
	var (
		reply *routeros.ListenReply
		err   error
	)
	if queue > 0 {
		reply, err = conn.ListenArgsQueueContext(lctx, sentence, queue)
	} else {
		reply, err = conn.ListenArgsContext(lctx, sentence)
	}
	if err != nil {
		cancel()
		return err
	}
	l.reply = reply
	go m.consume(l)
	return nil
}

func (m *Manager) consume(l *listener) {
	for sen := range l.reply.Chan() {
		if l.spec.Handler != nil {
			l.spec.Handler(decode.Wrap(sen))
		}
	}
	err := l.reply.Err()

	// Manager sedang Close-ing; Close() sudah handle map cleanup + detach.
	// Skip callback supaya tidak fire saat shutdown.
	if m.ctx.Err() != nil {
		return
	}

	if err != nil {
		// Connection error / network drop. Biarkan entry di map agar
		// ReattachAll bisa daftar ulang pasca reconnect.
		m.log.WithError(err).WithField("listener_id", l.spec.ID).
			Warn("listener channel closed with error")
		if l.spec.OnFinish != nil {
			l.spec.OnFinish(l.spec.ID, err)
		}
		return
	}

	// Natural !done dari router. Hapus entry agar ReattachAll skip.
	m.finishListener(l)
	m.log.WithField("listener_id", l.spec.ID).Info("stream finished naturally")
	if l.spec.OnFinish != nil {
		l.spec.OnFinish(l.spec.ID, nil)
	}
}

// finishListener menghapus listener dari map setelah natural completion.
// Pakai pointer-equality untuk hindari race kalau caller meng-Unregister
// lalu Register ulang dengan ID yang sama sebelum consume() exit.
func (m *Manager) finishListener(l *listener) {
	m.mu.Lock()
	if cur, ok := m.listeners[l.spec.ID]; ok && cur == l {
		delete(m.listeners, l.spec.ID)
	}
	m.mu.Unlock()
	if l.cancel != nil {
		l.cancel()
	}
}

func (m *Manager) detach(l *listener) {
	if l == nil {
		return
	}
	if l.reply != nil {
		cancelCtx, cancel := context.WithTimeout(context.Background(), l.spec.cancelTimeout())
		defer cancel()
		if _, err := l.reply.CancelContext(cancelCtx); err != nil && m.ctx.Err() == nil {
			m.log.WithError(err).WithField("listener_id", l.spec.ID).
				Debug("listener cancel returned error")
		}
	}
	if l.cancel != nil {
		l.cancel()
	}
}
