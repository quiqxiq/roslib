package poll

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-routeros/routeros/v3"
	"github.com/quiqxiq/roslib/decode"
	"github.com/sirupsen/logrus"
)

// group menampung sekumpulan Config yang berbagi interval yang sama.
// Satu group = satu ticker = satu goroutine.
type group struct {
	log      *logrus.Entry
	interval time.Duration

	// getConn diisi oleh Engine — fungsi getter agar group selalu
	// mengambil koneksi terkini (penting saat reconnect).
	getConn func() *routeros.Client

	mu      sync.RWMutex
	configs map[string]*pollState

	cancel context.CancelFunc
	done   chan struct{}
}

// pollState melacak runtime state per Config di dalam group.
type pollState struct {
	cfg      Config
	inFlight atomic.Int32
}

func newGroup(log *logrus.Entry, interval time.Duration, getConn func() *routeros.Client) *group {
	return &group{
		log:      log.WithField("group_interval", interval.String()),
		interval: interval,
		getConn:  getConn,
		configs:  make(map[string]*pollState),
		done:     make(chan struct{}),
	}
}

func (g *group) add(cfg Config) {
	g.mu.Lock()
	g.configs[cfg.ID] = &pollState{cfg: cfg}
	g.mu.Unlock()
}

func (g *group) remove(id string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if _, ok := g.configs[id]; ok {
		delete(g.configs, id)
		return true
	}
	return false
}

func (g *group) size() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.configs)
}

// start meluncurkan goroutine ticker. Aman dipanggil sekali saja.
func (g *group) start(parent context.Context) {
	ctx, cancel := context.WithCancel(parent)
	g.cancel = cancel
	go func() {
		defer close(g.done)
		t := time.NewTicker(g.interval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				g.tick(ctx)
			}
		}
	}()
}

// stop menghentikan ticker dan menunggu goroutine keluar.
func (g *group) stop() {
	if g.cancel != nil {
		g.cancel()
	}
	<-g.done
}

// tick: fan-out semua config concurrent ke koneksi command via async mode.
// Tag demux di go-routeros membuat semua RunArgsContext berjalan paralel
// di satu koneksi tanpa antri.
func (g *group) tick(ctx context.Context) {
	g.mu.RLock()
	snapshot := make([]*pollState, 0, len(g.configs))
	for _, p := range g.configs {
		snapshot = append(snapshot, p)
	}
	g.mu.RUnlock()

	if len(snapshot) == 0 {
		return
	}

	conn := g.getConn()
	if conn == nil {
		g.log.Warn("tick skipped: no active connection")
		return
	}

	for _, st := range snapshot {
		// Drop policy: kalau in-flight sudah melampaui limit, skip tick ini
		// untuk poll bersangkutan. Mencegah unbounded growth saat router slow.
		if st.cfg.MaxInFlight > 0 && st.inFlight.Load() >= int32(st.cfg.MaxInFlight) {
			g.log.WithFields(logrus.Fields{
				"poll_id":   st.cfg.ID,
				"in_flight": st.inFlight.Load(),
			}).Warn("tick skipped: max in-flight reached")
			continue
		}
		go g.runOne(ctx, conn, st)
	}
}

func (g *group) runOne(ctx context.Context, conn *routeros.Client, st *pollState) {
	st.inFlight.Add(1)
	defer st.inFlight.Add(-1)

	timeout := st.cfg.effectiveTimeout()
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	sentence := st.cfg.command()
	reply, err := conn.RunArgsContext(runCtx, sentence)
	if err != nil {
		// Context cancel saat shutdown bukan kondisi error yang perlu di-warn.
		if runCtx.Err() != nil && ctx.Err() != nil {
			return
		}
		g.log.WithError(err).WithFields(logrus.Fields{
			"poll_id": st.cfg.ID,
			"path":    st.cfg.Path,
		}).Warn("poll failed")
		return
	}
	if st.cfg.Handler == nil {
		return
	}
	for _, re := range reply.Re {
		st.cfg.Handler(decode.Wrap(re))
	}
}
