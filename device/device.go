package device

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/go-routeros/routeros/v3"
	"github.com/quiqxiq/roslib/builder"
	"github.com/quiqxiq/roslib/cache"
	"github.com/quiqxiq/roslib/poll"
	"github.com/quiqxiq/roslib/stream"
	"github.com/sirupsen/logrus"
)

// RouterDevice mewakili satu router. Memegang 2 koneksi persisten + handle
// ke StreamManager (untuk listener) dan PollEngine (untuk command periodik).
type RouterDevice struct {
	opts Options
	log  *logrus.Entry

	mu          sync.RWMutex
	connStream  *routeros.Client
	connCommand *routeros.Client

	// asyncErr channel untuk supervisor — di-reset setiap kali reconnect.
	streamAsyncErr  <-chan error
	commandAsyncErr <-chan error

	streams *stream.Manager
	polls   *poll.Engine

	ctx    context.Context
	cancel context.CancelFunc

	// supervisorWG melacak supervisor goroutines agar CloseAndWait dapat
	// memastikan tidak ada goroutine yang masih emit OnStatusChange
	// callback setelah caller pikir device sudah benar-benar di-tutup.
	supervisorWG sync.WaitGroup

	closeOnce sync.Once
}

// New men-dial 2 koneksi async ke router, mengaktifkan supervisor untuk
// auto-reconnect, dan mengembalikan RouterDevice siap pakai.
func New(parent context.Context, opts Options) (*RouterDevice, error) {
	if opts.Logger == nil {
		return nil, errors.New("device: Options.Logger is required")
	}
	if opts.Address == "" {
		return nil, errors.New("device: Options.Address is required")
	}
	if opts.Cache == nil {
		opts.Cache = cache.NoopCache{}
	}

	ctx, cancel := context.WithCancel(parent)
	d := &RouterDevice{
		opts:   opts,
		log:    opts.Logger.WithField("router", opts.Address),
		ctx:    ctx,
		cancel: cancel,
	}

	if err := d.dialBoth(); err != nil {
		cancel()
		return nil, err
	}

	d.streams = stream.NewManager(ctx, d.log, d.connStream)
	d.polls = poll.NewEngine(ctx, d.log, d.connCommand)

	d.supervisorWG.Add(2)
	go func() {
		defer d.supervisorWG.Done()
		d.superviseStream()
	}()
	go func() {
		defer d.supervisorWG.Done()
		d.superviseCommand()
	}()

	d.log.Info("router device ready")
	return d, nil
}

// Close menutup kedua koneksi, menghentikan StreamManager + PollEngine,
// dan membatalkan semua context. Aman dipanggil berkali-kali.
func (d *RouterDevice) notifyStatus(status, errMsg string) {
	if d.opts.OnStatusChange != nil {
		d.opts.OnStatusChange(status, errMsg)
	}
}

func (d *RouterDevice) Close() error {
	d.closeOnce.Do(d.shutdown)
	return nil
}

// CloseAndWait sama dengan Close, lalu menunggu semua supervisor
// goroutine benar-benar exit sebelum return. Dipakai caller yang akan
// segera membuat Device baru dengan address/credentials yang sama
// (mis. reconfigure dari UI) — mencegah race di mana supervisor lama
// masih emit OnStatusChange ke handler baru.
//
// Idempotent (boleh dipanggil setelah Close); pemanggilan kedua hanya
// tunggu WaitGroup (no-op kalau sudah zero).
func (d *RouterDevice) CloseAndWait() error {
	d.closeOnce.Do(d.shutdown)
	d.supervisorWG.Wait()
	return nil
}

func (d *RouterDevice) shutdown() {
	d.notifyStatus("closed", "")
	d.cancel()
	if d.streams != nil {
		d.streams.Close()
	}
	if d.polls != nil {
		d.polls.Close()
	}
	d.mu.Lock()
	if d.connStream != nil {
		_ = d.connStream.Close()
	}
	if d.connCommand != nil {
		_ = d.connCommand.Close()
	}
	d.mu.Unlock()
	d.log.Info("router device closed")
}

// IsAlive melaporkan apakah device masih aktif: context belum cancel
// dan kedua koneksi tidak nil.
//
// Dipakai oleh Manager.GetOrConnect untuk memutuskan apakah perlu
// buat koneksi baru. Tidak menjamin koneksi TCP sedang up — itu
// tanggung jawab supervisor. Cukup untuk guard di sisi Manager.
func (d *RouterDevice) IsAlive() bool {
	if d.ctx.Err() != nil {
		return false
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.connStream != nil && d.connCommand != nil
}

// Context mengembalikan context internal device. Cancel pada parent akan
// otomatis mem-propagate ke seluruh subsistem.
func (d *RouterDevice) Context() context.Context { return d.ctx }

// Logger mengembalikan entry logger device (sudah di-attach field router).
func (d *RouterDevice) Logger() *logrus.Entry { return d.log }

// CommandConn mengembalikan koneksi command saat ini. Pointer bisa berubah
// pasca reconnect — caller wajib panggil ulang untuk operasi baru.
func (d *RouterDevice) CommandConn() *routeros.Client {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.connCommand
}

// RunCommand mengirim sentence ke koneksi command (async + tag demux).
// Memenuhi builder.Executor agar fluent API bekerja.
func (d *RouterDevice) RunCommand(ctx context.Context, sentence []string) (*routeros.Reply, error) {
	conn := d.CommandConn()
	if conn == nil {
		return nil, errors.New("device: command connection not ready")
	}
	return conn.RunArgsContext(ctx, sentence)
}

// Path memulai chain fluent untuk path RouterOS tertentu.
//
//	d.Path("/ip/address").Print().Where("interface","ether1").Exec(ctx)
//	d.Path("/ip/address").Add(ctx, query.NewPair("address","10.0.0.1/24"), ...)
//	d.Path("/log").Print().FollowOnly().Stream("log-tail", handler)
//	d.Path("/interface/monitor-traffic").With("interface","ether1").Stream("nic", handler)
func (d *RouterDevice) Path(path string) *builder.PathBuilder {
	return builder.New(d, path)
}

// ──────────────── builder.Executor interface ────────────────

// Cache mengembalikan instance cache (NoopCache kalau disabled).
func (d *RouterDevice) Cache() cache.Cache { return d.opts.Cache }

// CacheTTL adalah default TTL untuk ExecCached saat caller pakai 0.
func (d *RouterDevice) CacheTTL() time.Duration { return d.opts.CacheTTL }

// DeviceID identifier device untuk scope cache.
// Pakai Options.ID kalau di-set, fallback ke Address.
func (d *RouterDevice) DeviceID() string {
	if d.opts.ID != "" {
		return d.opts.ID
	}
	return d.opts.Address
}

// InvalidateCache menghapus seluruh entry cache yang ter-asosiasi dengan
// path RouterOS tertentu pada device ini. Aman dipanggil bila cache adalah
// NoopCache (no-op).
//
// Gunakan ini setelah mutation eksternal (mis. operator edit di WinBox)
// atau setelah mutation panggilan library yang menurut user invalidasi
// dibutuhkan — library TIDAK auto-invalidate (per design).
func (d *RouterDevice) InvalidateCache(ctx context.Context, path string) error {
	if d.opts.Cache == nil {
		return nil
	}
	// Fleet mode dengan shared cache: pakai scoped invalidate supaya tidak
	// nge-bust entry milik device lain. Fall back ke global invalidate kalau
	// implementasi cache tidak support scoping.
	if scoped, ok := d.opts.Cache.(cache.DeviceScopedCache); ok {
		return scoped.InvalidatePathForDevice(ctx, d.DeviceID(), path)
	}
	return d.opts.Cache.InvalidatePath(ctx, path)
}

// Streams mengembalikan StreamManager untuk operasi listener.
func (d *RouterDevice) Streams() *stream.Manager { return d.streams }

// Polls mengembalikan PollEngine untuk operasi poll.
func (d *RouterDevice) Polls() *poll.Engine { return d.polls }

// RegisterPoll meneruskan ke PollEngine.
func (d *RouterDevice) RegisterPoll(cfg poll.Config) error {
	return d.polls.Register(cfg)
}

// UnregisterPoll meneruskan ke PollEngine. Helper convenience.
func (d *RouterDevice) UnregisterPoll(id string) bool {
	return d.polls.Unregister(id)
}

// RegisterStream meneruskan ke StreamManager.
func (d *RouterDevice) RegisterStream(spec stream.Spec) error {
	return d.streams.Register(spec)
}

// UnregisterStream meneruskan ke StreamManager. Helper convenience.
func (d *RouterDevice) UnregisterStream(id string) bool {
	return d.streams.Unregister(id)
}
