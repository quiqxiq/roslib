package stream

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-routeros/routeros/v3"
	"github.com/go-routeros/routeros/v3/proto"
	"github.com/sirupsen/logrus"
)

// fakeReply mengimplementasi listenReply untuk test consume() standalone.
type fakeReply struct {
	ch     chan *proto.Sentence
	err    error
	mu     sync.Mutex
	closed bool
}

func newFakeReply() *fakeReply {
	return &fakeReply{ch: make(chan *proto.Sentence, 4)}
}

func (f *fakeReply) Chan() <-chan *proto.Sentence { return f.ch }
func (f *fakeReply) Err() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.err
}
func (f *fakeReply) CancelContext(ctx context.Context) (*routeros.Reply, error) {
	return nil, nil
}

// close menutup channel — meniru !done dari router.
func (f *fakeReply) close(err error) {
	f.mu.Lock()
	if f.closed {
		f.mu.Unlock()
		return
	}
	f.closed = true
	f.err = err
	f.mu.Unlock()
	close(f.ch)
}

func newTestManager() *Manager {
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel) // diam saat test
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		log:       logger.WithField("test", "manager"),
		conn:      nil,
		listeners: make(map[string]*listener),
		ctx:       ctx,
		cancel:    cancel,
	}
}

func attachFakeListener(m *Manager, id string, spec Spec) (*listener, *fakeReply) {
	reply := newFakeReply()
	_, lcancel := context.WithCancel(m.ctx)
	spec.ID = id
	l := &listener{spec: spec, reply: reply, cancel: lcancel}
	m.mu.Lock()
	m.listeners[id] = l
	m.mu.Unlock()
	return l, reply
}

func TestConsumeRemovesEntryOnNaturalClose(t *testing.T) {
	m := newTestManager()
	defer m.Close()

	var finishErr error
	var finishCalled atomic.Bool
	l, reply := attachFakeListener(m, "lst-1", Spec{
		OnFinish: func(id string, err error) {
			finishErr = err
			finishCalled.Store(true)
		},
	})

	done := make(chan struct{})
	go func() {
		m.consume(l)
		close(done)
	}()

	// Kirim 2 sentence lalu close natural (err = nil).
	reply.ch <- &proto.Sentence{Word: "!re"}
	reply.ch <- &proto.Sentence{Word: "!re"}
	reply.close(nil)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("consume() did not return")
	}

	if got := m.Len(); got != 0 {
		t.Fatalf("Len() after natural close = %d; want 0", got)
	}
	if !finishCalled.Load() {
		t.Fatal("OnFinish was not invoked")
	}
	if finishErr != nil {
		t.Fatalf("OnFinish err = %v; want nil", finishErr)
	}
}

func TestConsumeKeepsEntryOnError(t *testing.T) {
	m := newTestManager()
	defer m.Close()

	var finishErr error
	var finishCalled atomic.Bool
	l, reply := attachFakeListener(m, "lst-err", Spec{
		OnFinish: func(id string, err error) {
			finishErr = err
			finishCalled.Store(true)
		},
	})

	done := make(chan struct{})
	go func() {
		m.consume(l)
		close(done)
	}()

	connErr := errors.New("connection drop")
	reply.close(connErr)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("consume() did not return")
	}

	if got := m.Len(); got != 1 {
		t.Fatalf("Len() after error = %d; want 1 (kept for reattach)", got)
	}
	if !finishCalled.Load() {
		t.Fatal("OnFinish was not invoked")
	}
	if !errors.Is(finishErr, connErr) {
		t.Fatalf("OnFinish err = %v; want %v", finishErr, connErr)
	}
}

func TestConsumeSkipsCallbackOnManagerClose(t *testing.T) {
	m := newTestManager()

	var finishCalled atomic.Bool
	l, reply := attachFakeListener(m, "lst-close", Spec{
		OnFinish: func(id string, err error) { finishCalled.Store(true) },
	})

	done := make(chan struct{})
	go func() {
		m.consume(l)
		close(done)
	}()

	// Cancel manager dulu, baru close channel — meniru flow Close().
	m.cancel()
	reply.close(nil)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("consume() did not return")
	}

	if finishCalled.Load() {
		t.Fatal("OnFinish should not fire during Manager Close()")
	}
}

func TestReattachAllSnapshotAfterFinish(t *testing.T) {
	m := newTestManager()
	defer m.Close()

	// Daftar 2 listener — satu finite (close natural), satu long-running.
	lFinite, replyFinite := attachFakeListener(m, "finite", Spec{})
	_, _ = attachFakeListener(m, "long", Spec{})

	doneFinite := make(chan struct{})
	go func() {
		m.consume(lFinite)
		close(doneFinite)
	}()

	replyFinite.close(nil)
	<-doneFinite

	// Setelah natural close, hanya "long" yang harus tersisa di snapshot.
	if got := m.Len(); got != 1 {
		t.Fatalf("Len() after finite finish = %d; want 1", got)
	}
	m.mu.RLock()
	_, hasFinite := m.listeners["finite"]
	_, hasLong := m.listeners["long"]
	m.mu.RUnlock()
	if hasFinite {
		t.Error("finite listener still in map after natural close")
	}
	if !hasLong {
		t.Error("long-running listener removed unexpectedly")
	}
}

func TestRaceFinishVsUnregister(t *testing.T) {
	m := newTestManager()
	defer m.Close()

	// Banyak iterasi untuk membuka jendela race.
	const iterations = 100
	for i := 0; i < iterations; i++ {
		id := "lst-race"
		l, reply := attachFakeListener(m, id, Spec{})

		var wg sync.WaitGroup
		wg.Add(2)

		// Goroutine A: natural close (panggil consume).
		go func() {
			defer wg.Done()
			m.consume(l)
		}()

		// Goroutine B: Unregister bersamaan.
		go func() {
			defer wg.Done()
			m.Unregister(id)
		}()

		// Trigger close — race antara finishListener dan Unregister.
		reply.close(nil)
		wg.Wait()

		// Pasca race, map harus konsisten — entry hilang.
		if _, exists := m.listeners[id]; exists {
			t.Fatalf("iter %d: listener entry leaked", i)
		}
	}
}

func TestRaceFinishVsClose(t *testing.T) {
	m := newTestManager()

	const numListeners = 20
	replies := make([]*fakeReply, numListeners)
	for i := 0; i < numListeners; i++ {
		id := "lst-race-close-" + string(rune('a'+i))
		l, reply := attachFakeListener(m, id, Spec{})
		replies[i] = reply
		go m.consume(l)
	}

	// Race: half close natural, half via Manager.Close.
	go func() {
		for i := 0; i < numListeners/2; i++ {
			replies[i].close(nil)
		}
	}()
	m.Close()

	// Tutup sisa channel agar goroutine consume keluar.
	for i := numListeners / 2; i < numListeners; i++ {
		replies[i].close(nil)
	}

	// Settle.
	time.Sleep(50 * time.Millisecond)

	if got := m.Len(); got != 0 {
		t.Fatalf("after Close(), Len() = %d; want 0", got)
	}
}

func TestFinishListenerPointerEquality(t *testing.T) {
	// Skenario: listener selesai natural, tapi sebelum finishListener jalan,
	// user Unregister + Register ulang dengan ID yang sama. Pointer-equality
	// check di finishListener harus mencegah penghapusan entry baru.
	m := newTestManager()
	defer m.Close()

	const id = "shared-id"
	lOld, replyOld := attachFakeListener(m, id, Spec{})

	// Selesaikan natural.
	doneOld := make(chan struct{})
	go func() {
		m.consume(lOld)
		close(doneOld)
	}()
	replyOld.close(nil)
	<-doneOld

	// Sekarang map kosong (finishListener sudah delete).
	if got := m.Len(); got != 0 {
		t.Fatalf("after old finish, Len() = %d; want 0", got)
	}

	// Register ulang dengan ID yang sama.
	lNew, _ := attachFakeListener(m, id, Spec{})

	// Panggil finishListener untuk listener LAMA — entry baru tidak boleh hilang.
	m.finishListener(lOld)

	if got := m.Len(); got != 1 {
		t.Fatalf("after stale finishListener call, Len() = %d; want 1 (new entry preserved)", got)
	}
	m.mu.RLock()
	cur := m.listeners[id]
	m.mu.RUnlock()
	if cur != lNew {
		t.Fatal("new listener entry was replaced/removed by stale finish call")
	}
}
