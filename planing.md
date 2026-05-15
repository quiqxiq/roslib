# roslib — Planning Arsitektur (Revisi)

Library wrapper di atas `github.com/go-routeros/routeros/v3` untuk akses MikroTik RouterOS, dirancang khusus untuk memanfaatkan **async mode + tag-based multiplexing** secara penuh sehingga overhead koneksi minimal dan throughput command maksimal.

---

## 1. Konfirmasi API `go-routeros/v3`

Berdasarkan dokumentasi resmi di [pkg.go.dev/github.com/go-routeros/routeros/v3](https://pkg.go.dev/github.com/go-routeros/routeros/v3) (v3.0.1, MIT license), API yang relevan untuk desain ini:

| API | Kegunaan untuk roslib |
|---|---|
| `DialContext(ctx, addr, user, pass)` | Buat koneksi persistent dengan cancellation |
| `DialTLSContext(ctx, addr, user, pass, tlsCfg)` | Idem, dengan TLS |
| `AsyncContext(ctx) <-chan error` | **Aktifkan multiplexing via tag**, return channel error untuk monitor koneksi mati |
| `IsAsync() bool` | Cek apakah koneksi sudah async |
| `RunArgsContext(ctx, []string) (*Reply, error)` | Kirim command, tunggu reply. Di mode async tidak blocking command lain |
| `ListenArgsContext(ctx, []string) (*ListenReply, error)` | Long-running listen (follow), return segera, hasil via channel |
| `ListenArgsQueueContext(ctx, []string, queueSize)` | Sama, dengan queue size custom |
| `ListenReply.Chan() <-chan *proto.Sentence` | Channel sentence untuk listener |
| `ListenReply.CancelContext(ctx)` | Stop satu listener tanpa kill koneksi |
| `ListenReply.Err() error` | Error pertama saat memproses sentence |
| `Client.Queue int` | Default queue size untuk Listen |
| `Client.SetLogHandler(LogHandler)` | `LogHandler = slog.Handler` → butuh adapter ke logrus |
| `Close() error` | Tutup koneksi |
| `DeviceError`, `UnknownReplyError` | Tipe error spesifik dari device |

**Implikasi penting:**

- `AsyncContext` adalah satu-satunya jalur ke multiplexing tag. Setiap koneksi yang dipakai untuk concurrent command **wajib** dipanggil `AsyncContext`.
- Channel `<-chan error` dari `AsyncContext` adalah **sinyal koneksi mati** — kita pakai ini sebagai trigger reconnect, bukan polling status.
- `SetLogHandler` menerima `slog.Handler`, bukan logger arbiter. Untuk pakai **logrus** kita butuh **adapter slog → logrus**.
- `ListenReply.Cancel*` memungkinkan kita stop listener individual tanpa menutup koneksi — penting untuk dynamic stream registration.

---

## 2. Tech Stack

| Komponen | Library | Alasan |
|---|---|---|
| Bahasa | Go 1.21+ | `slog` stdlib, generics, context |
| RouterOS client | `github.com/go-routeros/routeros/v3` | API resmi, support async + listen |
| Logging | `github.com/sirupsen/logrus` | Sesuai requirement |
| Slog → logrus adapter | `github.com/samber/slog-logrus/v2` | Bridge `slog.Handler` ke logrus untuk `SetLogHandler` |
| Backoff reconnect | `github.com/cenkalti/backoff/v4` | Exponential backoff yang battle-tested |
| Cache (opsional, build tag `redis`) | `github.com/redis/go-redis/v9` | Cache hasil query yang sering dipakai; default in-memory/noop |
| Metrics sink (opsional) | `github.com/InfluxCommunity/influxdb3-go/v2` | Sink poll/stream → InfluxDB3 time series (SQL/InfluxQL) |
| UUID (opsional) | `github.com/google/uuid` | ID listener & poll registration |
| Capability registry | `//go:embed` JSON RouterOS 7.20.8 | Validasi command/arg + auto-routing streaming vs one-shot |
| Config loader | stdlib `os` (env var) | Zero-dep — user app bebas wrap `godotenv` di main |
| Testing | `github.com/stretchr/testify` | Assertion & mocking |

Stdlib yang dipakai berat: `context`, `sync`, `time`, `errors`, `fmt`, `strings`.

---

## 3. Konsep Arsitektur

### 3.1 Model 2 koneksi persistent per router

```
┌──────────────────────────────────────────────────────────────────────┐
│                    RouterDevice (per IP router)                      │
│                                                                      │
│  connStream  *routeros.Client  ← AsyncContext aktif                  │
│  │                                                                   │
│  │  EKSKLUSIF untuk: Listen (follow / follow-only)                   │
│  │  Semua active listener berjalan concurrent di sini via tag.       │
│  │  TIDAK pernah dipakai untuk RunArgs apapun.                       │
│  │  Alasan isolasi: Listen long-lived; jika digabung dengan query    │
│  │  dan koneksi putus → semua listener ikut hilang state.            │
│  │                                                                   │
│  connCommand *routeros.Client  ← AsyncContext aktif                  │
│     SHARED untuk: query, mutation, dan poll.                         │
│     Async mode + tag demux dari go-routeros membuat ratusan command  │
│     bisa berjalan concurrent di satu koneksi tanpa antri.            │
│     Pisah poll dari query = tidak ada manfaat nyata, hanya overhead. │
└──────────────────────────────────────────────────────────────────────┘
```

**Kenapa 2 koneksi, bukan 3?**

Pertanyaan "perlu pisah pool dari query?" muncul karena reflek pemikiran sync — bayangkan poll sibuk pegang koneksi, query harus antri. Tapi di async mode go-routeros, **tidak ada antrian**: setiap `RunArgsContext` di-tag, kirim, dan reply masuk ke waiter yang sesuai. Throughput satu koneksi async = throughput TCP-nya, bukan dibatasi oleh satu command at a time. Pisah koneksi untuk poll = bayar 1 koneksi TCP ekstra untuk benefit nol.

**Kenapa Listen tetap pisah?**

`Listen` adalah long-running connection state — satu listener artinya satu tag yang "open" terus. Kalau digabung di `connCommand`:

- Saat reconnect (koneksi putus), listener state hilang. Untuk re-register kita harus simpan semua listener spec — fine.
- Tapi RunArgs yang sedang in-flight juga ikut gagal — fine, retry mudah.
- Masalahnya: dengan listener banyak (misal 10+) di satu koneksi yang sama dengan query/poll yang ratusan call/detik, **debugging dan reasoning** jadi sulit. Pisah = isolasi konsern + lebih predictable.

---

### 3.2 Cara polling bekerja (redesign)

**Masalah di rancangan lama** (dan di banyak implementasi pool naïf):

- Tiap path punya ticker sendiri.
- 50 path = 50 goroutine ticker + 50 channel `<-chan time.Time`.
- Boros goroutine, dan tidak leverage async (tiap tick kirim 1 command, padahal connCommand bisa fire 50 sekaligus).

**Desain baru — Interval Group Batching:**

```
PollEngine
│
├─ Group [interval=5s]
│   ├─ poll: /system/resource print
│   ├─ poll: /system/health print
│   └─ poll: /interface print stats
│       └─ 1 ticker, 1 goroutine.
│         Tiap tick → fire semua 3 command concurrent di connCommand
│         via 3 goroutine pendek yang panggil RunArgsContext.
│         Hasil masing-masing dispatch ke handler-nya.
│
├─ Group [interval=10s]
│   ├─ poll: /queue/simple print stats
│   └─ poll: /ip/dhcp-server/lease print
│
└─ Group [interval=30s]
    └─ poll: /system/package/update check-for-updates
```

**Mekanisme satu tick:**

```
tick! (group 5s)
   │
   ├─ goroutine: RunArgsContext("/system/resource/print")  ─┐
   ├─ goroutine: RunArgsContext("/system/health/print")    ─┤  Tiga-tiganya jalan
   └─ goroutine: RunArgsContext("/interface/print", "stats")┘  concurrent via
                                                              tag di connCommand.
   reply tag=A → handler /system/resource
   reply tag=B → handler /system/health
   reply tag=C → handler /interface
```

**Properti penting desain ini:**

1. **Goroutine count proporsional dengan jumlah interval group, bukan jumlah command.** 50 command dengan 3 interval berbeda = 3 ticker goroutine, bukan 50.
2. **Fan-out di tiap tick** memanfaatkan async penuh — tidak ada serialisasi.
3. **Slow command tidak menghambat command lain** — masing-masing punya goroutine pendek sendiri yang menunggu reply.
4. **Mendaftarkan command baru = tambah ke registry**, tidak buat goroutine baru kecuali interval-nya unik (group baru).
5. **Drop policy per group**: kalau satu tick belum selesai dan tick berikutnya datang, kita bisa pilih skip atau queue. Default: skip + log warning (mencegah unbounded growth saat router slow).

---

### 3.3 Reconnection strategy

`AsyncContext(ctx)` return `<-chan error`. Channel ini emit error kalau:

- TCP putus
- Server kirim `!fatal`
- Context dibatalkan

Setiap koneksi (connStream, connCommand) punya **supervisor goroutine** yang nunggu di channel ini. Kalau error:

```
error datang di asyncErr channel
   │
   ├─ Close() koneksi lama (idempotent)
   ├─ Backoff exponential (cenkalti/backoff)
   ├─ DialContext + AsyncContext lagi
   ├─ Update pointer di RouterDevice (under mutex)
   ├─ Trigger callback ke StreamManager / PollEngine:
   │    - StreamManager.ReattachAll(newConn) → re-register semua listener
   │    - PollEngine.AttachConn(newConn) → ticker tinggal pakai conn baru
   └─ Resume supervisor di koneksi baru
```

PollEngine tidak butuh "re-register" karena tiap tick dia panggil `conn.RunArgsContext` baru — cukup swap pointer conn. StreamManager wajib re-register karena setiap listener adalah long-running call yang harus dibuat ulang di koneksi baru.

---

## 4. Struktur Folder

```text
roslib/
│
├── roslib.go                  ← public entry: New() & NewFromConfig() → *Device
│
├── device/
│   ├── device.go              ← RouterDevice struct + lifecycle (New/Close), Path()
│   ├── connect.go             ← dial + AsyncContext untuk 2 koneksi
│   ├── supervisor.go          ← goroutine monitor <-chan error, reconnect + backoff
│   ├── options.go             ← Options (addr, creds, TLS, Registry, Cache, Strict…)
│   ├── log_adapter.go         ← slog.Handler → logrus bridge (samber/slog-logrus)
│   └── validation.go          ← validatePollConfig & validateStreamSpec
│
├── builder/
│   ├── executor.go            ← Executor interface (impl. oleh device)
│   ├── path.go                ← PathBuilder: With/Arg/Print/Stream (inherent)
│   ├── print.go               ← PrintBuilder: Detail/Stats/Where/Exec/ExecCached
│   ├── write.go               ← Add / Set / Remove / Enable / Disable / Run
│   ├── stream.go              ← .Follow() / .FollowOnly() → StreamBuilder.Stream
│   └── validate.go            ← hook validasi lazy (strict vs warn)
│
├── capability/
│   ├── registry.go            ← Class, Command, Registry, Lookup/RequireClass
│   ├── loader.go              ← //go:embed JSON + Load(LoadOptions)
│   ├── streaming.go           ← whitelist & klasifikasi otomatis
│   └── assets/mikrotik/routeros_7.20.8.json (2.6 MB, 541 endpoint)
│
├── stream/
│   ├── manager.go             ← StreamManager: registry listener di connStream
│   └── listener.go            ← Spec blueprint (Word, Args, …) + reattach
│
├── poll/
│   ├── engine.go              ← PollEngine: kumpulan IntervalGroup + AttachConn
│   ├── group.go               ← IntervalGroup: 1 ticker, fan-out semua command
│   └── config.go              ← PollConfig (path, args, where, interval, handler)
│
├── decode/
│   ├── sentence.go            ← *proto.Sentence → typed wrapper
│   └── types.go               ← parser bool/duration/bytes/datetime RouterOS
│
├── query/
│   └── query.go               ← Pair, WherePair, BuildSentence (shared types)
│
├── cache/
│   ├── cache.go               ← interface Cache + NoopCache + InMemoryCache
│   ├── key.go                 ← KeyOf(sentence) → sha256 cache key
│   └── redis.go               ← Redis impl di build tag `redis`
│
├── metrics/influx/
│   ├── client.go              ← NewClient(cfg) wrap influxdb3.New
│   ├── writer.go              ← Writer + BatchedWriter
│   ├── reader.go              ← Reader tipis di atas Query
│   └── handler.go             ← PollSink / StreamSink / Batched* helpers
│
├── config/
│   ├── config.go              ← LoadFromEnv + ToDeviceOptions
│   └── tls.go                 ← helper *tls.Config minimal
│
├── examples/
│   ├── usage/                 ← contoh API umum
│   ├── streaming/             ← demo monitor-traffic, ping, torch, .../monitor
│   └── influx/                ← full flow: env → device → poll → InfluxDB → reader
│
├── .env.example               ← daftar env yang di-recognize config.LoadFromEnv
│
└── resources/                 ← typed wrapper opsional (di-scope-out)
```

---

## 5. Implementasi Kunci

### 5.1 Logger adapter — logrus ke slog.Handler

`Client.SetLogHandler` butuh `slog.Handler`, sedangkan kita pakai logrus. Adapter:

```go
// device/log_adapter.go
package device

import (
    "log/slog"

    "github.com/sirupsen/logrus"
    slogrus "github.com/samber/slog-logrus/v2"
)

// NewSlogHandlerFromLogrus membuat slog.Handler yang menulis ke logrus.Logger.
// Hasilnya bisa di-pass ke routeros.Client.SetLogHandler.
func NewSlogHandlerFromLogrus(l *logrus.Logger) slog.Handler {
    return slogrus.Option{
        Level:  slog.LevelDebug, // biarkan logrus yang filter level
        Logger: l,
    }.NewLogrusHandler()
}
```

Pemakaian saat dial:

```go
client, _ := routeros.DialContext(ctx, addr, user, pass)
client.SetLogHandler(NewSlogHandlerFromLogrus(opts.Logger))
client.AsyncContext(ctx)
```

Semua log internal go-routeros akan masuk ke logrus user, dengan field structured ter-mapping otomatis.

---

### 5.2 RouterDevice

```go
// device/device.go
package device

import (
    "context"
    "sync"

    "github.com/go-routeros/routeros/v3"
    "github.com/sirupsen/logrus"
)

type RouterDevice struct {
    opts DeviceOptions
    log  *logrus.Entry

    mu          sync.RWMutex
    connStream  *routeros.Client
    connCommand *routeros.Client

    streams *stream.Manager
    polls   *poll.Engine

    ctx    context.Context
    cancel context.CancelFunc
}

func New(parent context.Context, opts DeviceOptions) (*RouterDevice, error) {
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

    d.streams = stream.NewManager(d.log, d.connStream)
    d.polls   = poll.NewEngine(d.log, d.connCommand)

    // Supervisor per koneksi: nunggu <-chan error, reconnect kalau perlu
    go d.superviseStream()
    go d.superviseCommand()

    d.log.Info("router device ready")
    return d, nil
}

func (d *RouterDevice) Close() error {
    d.cancel()
    d.mu.Lock()
    defer d.mu.Unlock()
    if d.connStream != nil  { _ = d.connStream.Close()  }
    if d.connCommand != nil { _ = d.connCommand.Close() }
    return nil
}
```

---

### 5.3 Dial + AsyncContext

```go
// device/connect.go
func (d *RouterDevice) dialBoth() error {
    var err error

    if d.connStream, err = d.dialOne("stream"); err != nil {
        return fmt.Errorf("dial connStream: %w", err)
    }

    if d.connCommand, err = d.dialOne("command"); err != nil {
        _ = d.connStream.Close()
        return fmt.Errorf("dial connCommand: %w", err)
    }
    return nil
}

func (d *RouterDevice) dialOne(role string) (*routeros.Client, error) {
    log := d.log.WithField("conn", role)
    log.Debug("dialing")

    var (
        c   *routeros.Client
        err error
    )
    if d.opts.TLS != nil {
        c, err = routeros.DialTLSContext(d.ctx, d.opts.Address, d.opts.Username, d.opts.Password, d.opts.TLS)
    } else {
        c, err = routeros.DialContext(d.ctx, d.opts.Address, d.opts.Username, d.opts.Password)
    }
    if err != nil {
        return nil, err
    }

    c.Queue = d.opts.ListenQueueSize       // default 100 misalnya
    c.SetLogHandler(NewSlogHandlerFromLogrus(d.opts.Logger))
    c.AsyncContext(d.ctx)                  // aktifkan multiplexing — wajib

    log.Info("connection established (async mode)")
    return c, nil
}
```

---

### 5.4 Supervisor — auto-reconnect via `<-chan error`

```go
// device/supervisor.go
func (d *RouterDevice) superviseCommand() {
    for {
        // AsyncContext sudah dipanggil saat dial. Kita perlu re-ambil channel-nya
        // saat dial. Simpan reference-nya di field, atau panggil ulang setelah reconnect.
        // (Implementasi nyata: dialOne mengembalikan juga channel error-nya.)

        select {
        case <-d.ctx.Done():
            return
        case err := <-d.cmdAsyncErr:
            if err == nil {
                return // context closed normally
            }
            d.log.WithError(err).Warn("connCommand died, reconnecting")
            d.reconnectCommand()
        }
    }
}

func (d *RouterDevice) reconnectCommand() {
    bo := backoff.NewExponentialBackOff()
    bo.MaxElapsedTime = 0 // retry selamanya, sampai ctx cancel

    operation := func() error {
        c, err := d.dialOne("command")
        if err != nil {
            d.log.WithError(err).Warn("redial connCommand failed")
            return err
        }
        d.mu.Lock()
        _ = d.connCommand.Close()
        d.connCommand = c
        d.mu.Unlock()

        d.polls.AttachConn(c)  // poll engine swap pointer
        d.log.Info("connCommand reconnected")
        return nil
    }
    _ = backoff.Retry(operation, backoff.WithContext(bo, d.ctx))
}

// superviseStream serupa, tapi panggil d.streams.ReattachAll(newConn) bukan AttachConn.
```

---

### 5.5 PollEngine — interval group batching

```go
// poll/config.go
package poll

import "time"

type PollConfig struct {
    ID       string
    Path     string         // "/system/resource"
    Args     []string       // ["print"] atau ["print", "stats"]
    Where    []WherePair
    Interval time.Duration
    Handler  func(*Sentence)
}

// poll/engine.go
type Engine struct {
    log  *logrus.Entry
    mu   sync.RWMutex
    conn *routeros.Client          // connCommand
    grps map[time.Duration]*group  // key: interval
}

func NewEngine(log *logrus.Entry, conn *routeros.Client) *Engine {
    return &Engine{
        log:  log,
        conn: conn,
        grps: make(map[time.Duration]*group),
    }
}

// Register: tambah config ke group sesuai interval. Group dibuat on-demand.
func (e *Engine) Register(ctx context.Context, cfg PollConfig) {
    e.mu.Lock()
    g, ok := e.grps[cfg.Interval]
    if !ok {
        g = newGroup(e.log, cfg.Interval, e.connRef())
        e.grps[cfg.Interval] = g
        g.start(ctx)
    }
    g.add(cfg)
    e.mu.Unlock()

    e.log.WithFields(logrus.Fields{
        "poll_id":  cfg.ID,
        "path":     cfg.Path,
        "interval": cfg.Interval,
    }).Info("poll registered")
}

func (e *Engine) Unregister(id string) {
    e.mu.Lock()
    defer e.mu.Unlock()
    for _, g := range e.grps {
        if g.remove(id) {
            e.log.WithField("poll_id", id).Info("poll unregistered")
            return
        }
    }
}

// AttachConn: dipanggil setelah connCommand reconnect.
// Tidak perlu re-register apapun — group hanya pegang pointer ke fungsi getter.
func (e *Engine) connRef() func() *routeros.Client {
    return func() *routeros.Client {
        e.mu.RLock()
        defer e.mu.RUnlock()
        return e.conn
    }
}

func (e *Engine) AttachConn(c *routeros.Client) {
    e.mu.Lock()
    e.conn = c
    e.mu.Unlock()
}
```

```go
// poll/group.go
type group struct {
    log      *logrus.Entry
    interval time.Duration
    getConn  func() *routeros.Client

    mu      sync.RWMutex
    configs map[string]PollConfig

    inFlight atomic.Int32  // counter command yang masih jalan untuk drop policy
}

func newGroup(log *logrus.Entry, interval time.Duration, getConn func() *routeros.Client) *group {
    return &group{
        log:      log.WithField("group_interval", interval),
        interval: interval,
        getConn:  getConn,
        configs:  make(map[string]PollConfig),
    }
}

func (g *group) add(cfg PollConfig)        { g.mu.Lock(); g.configs[cfg.ID] = cfg; g.mu.Unlock() }
func (g *group) remove(id string) bool {
    g.mu.Lock()
    defer g.mu.Unlock()
    if _, ok := g.configs[id]; ok {
        delete(g.configs, id)
        return true
    }
    return false
}

func (g *group) start(ctx context.Context) {
    go func() {
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

// tick: fan-out semua config concurrent ke connCommand.
// Async mode di go-routeros handle tag demux — semua RunArgsContext
// berjalan paralel di satu koneksi tanpa antri.
func (g *group) tick(ctx context.Context) {
    g.mu.RLock()
    snapshot := make([]PollConfig, 0, len(g.configs))
    for _, c := range g.configs {
        snapshot = append(snapshot, c)
    }
    g.mu.RUnlock()

    conn := g.getConn()
    if conn == nil {
        g.log.Warn("tick skipped: no active connection")
        return
    }

    for _, cfg := range snapshot {
        cfg := cfg
        go func() {
            g.inFlight.Add(1)
            defer g.inFlight.Add(-1)

            args := buildSentence(cfg.Path+"/print", cfg.Args, cfg.Where)

            tickCtx, cancel := context.WithTimeout(ctx, g.interval) // jangan lebih lama dari interval
            defer cancel()

            reply, err := conn.RunArgsContext(tickCtx, args)
            if err != nil {
                g.log.WithError(err).WithField("poll_id", cfg.ID).Warn("poll failed")
                return
            }
            for _, re := range reply.Re {
                cfg.Handler(decode(re))
            }
        }()
    }
}
```

**Catatan policy:**

- `context.WithTimeout(ctx, g.interval)` mencegah satu poll bocor lebih lama dari interval-nya. Kalau router lambat, command di-cancel dan tick berikutnya tetap jalan.
- `inFlight` counter bisa dipakai untuk drop policy lanjutan (misal: kalau ada > N command jalan, skip tick — protect dari thundering herd saat router sangat lambat).

---

### 5.6 StreamManager — kelola listener di connStream

```go
// stream/manager.go
type Manager struct {
    log  *logrus.Entry
    mu   sync.RWMutex
    conn *routeros.Client
    lst  map[string]*Listener
}

type Listener struct {
    ID      string
    Path    string
    Args    []string
    Handler func(*Sentence)

    reply  *routeros.ListenReply
    cancel context.CancelFunc
}

func (m *Manager) Register(ctx context.Context, l *Listener) error {
    lctx, cancel := context.WithCancel(ctx)
    l.cancel = cancel

    sentence := append([]string{l.Path + "/print"}, l.Args...)
    reply, err := m.conn.ListenArgsContext(lctx, sentence)
    if err != nil {
        cancel()
        return err
    }
    l.reply = reply

    m.mu.Lock()
    m.lst[l.ID] = l
    m.mu.Unlock()

    go m.consume(l)
    m.log.WithFields(logrus.Fields{"listener_id": l.ID, "path": l.Path}).Info("stream attached")
    return nil
}

func (m *Manager) consume(l *Listener) {
    for sentence := range l.reply.Chan() {
        l.Handler(decode(sentence))
    }
    if err := l.reply.Err(); err != nil {
        m.log.WithError(err).WithField("listener_id", l.ID).Warn("listener channel closed with error")
    }
}

func (m *Manager) Unregister(ctx context.Context, id string) {
    m.mu.Lock()
    l, ok := m.lst[id]
    delete(m.lst, id)
    m.mu.Unlock()
    if !ok { return }

    if _, err := l.reply.CancelContext(ctx); err != nil {
        m.log.WithError(err).Warn("listener cancel failed")
    }
    l.cancel()
}

// ReattachAll: setelah connStream reconnect, daftarkan ulang semua listener.
func (m *Manager) ReattachAll(ctx context.Context, newConn *routeros.Client) {
    m.mu.Lock()
    m.conn = newConn
    snapshot := make([]*Listener, 0, len(m.lst))
    for _, l := range m.lst {
        snapshot = append(snapshot, l)
    }
    m.mu.Unlock()

    for _, l := range snapshot {
        lctx, cancel := context.WithCancel(ctx)
        l.cancel = cancel
        sentence := append([]string{l.Path + "/print"}, l.Args...)
        reply, err := newConn.ListenArgsContext(lctx, sentence)
        if err != nil {
            m.log.WithError(err).WithField("listener_id", l.ID).Error("reattach failed")
            continue
        }
        l.reply = reply
        go m.consume(l)
        m.log.WithField("listener_id", l.ID).Info("listener reattached")
    }
}
```

---

### 5.7 Capability Registry — validasi lazy di builder

JSON `capability/assets/mikrotik/routeros_7.20.8.json` (2.6 MB, 541 endpoint)
di-embed lewat `//go:embed`. Loader parsing dilakukan sekali pakai
`sync.OnceValues`, hasilnya `*Registry` flat — index `Cmds[word]` ke
`*Command` lengkap dengan `Class` dan set `Args`.

Klasifikasi otomatis saat parse:

| Class | Aturan |
|---|---|
| `ClassMutation` | action ∈ {add, remove, set, enable, disable, comment, move, edit, reset, reset-counters, reset-mac-address, unset} |
| `ClassStreaming` | exact match whitelist (`/tool/ping`, `/tool/torch`, `/tool/flood-ping`, `/tool/traceroute`, `/tool/bandwidth-test`, `/interface/monitor-traffic`, …) ATAU prefix `/tool/sniffer/` ATAU action `monitor` ATAU non-print yang punya arg `interval` |
| `ClassStreamablePrint` | action `print` & punya arg `follow`/`follow-only`/`interval` |
| `ClassOneShot` | sisanya |

Builder memanggil validator di terminal call:

- `Exec()` → Class ∈ {OneShot, StreamablePrint, Mutation}
- `Stream()` (direct dari `PathBuilder`) → Class == Streaming
- `Stream()` (dari `PrintBuilder.Follow()`) → Class == StreamablePrint
- `Add/Set/Remove/Enable/Disable` → Class == Mutation
- `Run()` → cek args saja (class apa pun)

Strict mode (`Options.StrictCapability=true`, default) → return `error`.
Strict=false → log-warn lewat `device.Logger()` + command tetap dikirim.

Override JSON (untuk RouterOS versi lain) lewat `Options.RegistryPath` atau
`config.Config.RegistryPath`.

---

### 5.8 Cache wire-up — `PrintBuilder.ExecCached`

`cache.Cache` adalah interface dengan `Get/Set` + TTL. Default
`NoopCache` (selalu miss). Implementasi `InMemoryCache` untuk testing.
Skeleton `RedisCache` di file build-tag `redis` — `go-redis` tidak wajib
di go.mod inti.

```go
func (p *PrintBuilder) ExecCached(ctx context.Context, ttl time.Duration) (*Reply, error) {
    sentence := p.command()
    key := cache.KeyOf(sentence) // sha256 dari kata-kata sentence
    if data, hit, _ := p.exec.Cache().Get(ctx, key); hit {
        return decodeCached(data), nil
    }
    raw, err := p.exec.RunCommand(ctx, sentence)
    if err != nil { return nil, err }
    encoded, _ := json.Marshal(toCached(raw))
    _ = p.exec.Cache().Set(ctx, key, encoded, ttl)
    return wrapReply(raw), nil
}
```

Encoding JSON sengaja (bukan gob) supaya debug-friendly di Redis CLI.

---

### 5.9 InfluxDB3 sink — Writer/Reader + Poll/Stream helper

`metrics/influx/Writer` bertanggung jawab konversi `*decode.Sentence`
→ `*influxdb3.Point` lewat fungsi mapper user untuk tag & field:

```go
w := influx.NewWriter(cli, "system_resource",
    func(s *decode.Sentence) map[string]string {
        return map[string]string{"board": s.Get("board-name")}
    },
    func(s *decode.Sentence) map[string]any {
        return map[string]any{
            "cpu_load": s.IntOr("cpu-load", 0),
            "uptime":   int64(s.DurationOr("uptime", 0).Seconds()),
        }
    },
)

dev.RegisterPoll(roslib.PollConfig{
    ID: "sys-resource", Path: "/system/resource",
    Args: []string{"print"}, Interval: 5 * time.Second,
    Handler: influx.PollSink(w, logger.WithField("sink", "influx")),
})
```

`BatchedWriter` opsional: buffer N point atau flush per interval, satu
goroutine internal. Reader tipis di atas `Client.Query` — user iterate
sendiri `*QueryIterator`.

---

### 5.10 Config loader — `config.LoadFromEnv`

Struct `Config{Router, Cache, Influx, StrictCapability, RegistryPath}`
diisi dari env var ROSLIB_*/INFLUX_*. Validator memastikan field wajib
ada saat toggle on (mis. `INFLUX_TOKEN` wajib kalau `ROSLIB_INFLUX_ENABLED=true`).

Helper `cfg.ToDeviceOptions(log)` menerjemahkan ke `device.Options`.
`roslib.NewFromConfig(ctx, cfg, log)` adalah konstruktor end-to-end:
load registry → build cache (NoopCache kalau off) → dial router →
build `*influxdb3.Client` (kalau influx on).

---

## 6. Contoh Pemakaian

```go
log := logrus.New()
log.SetLevel(logrus.InfoLevel)
log.SetFormatter(&logrus.JSONFormatter{})

ctx := context.Background()

// ── Opsi A: literal Options ─────────────────────────────────────
device, err := roslib.New(ctx, roslib.Options{
    Address:          "192.168.88.1:8728",
    Username:         "admin",
    Password:         "secret",
    Logger:           log,
    ListenQueueSize:  100,
    StrictCapability: true, // tolak command/arg tidak dikenal
})
if err != nil { log.Fatal(err) }
defer device.Close()

// ── Opsi B: dari env (config.LoadFromEnv → NewFromConfig) ───────
// cfg, _ := config.LoadFromEnv()
// device, influxCli, _ := roslib.NewFromConfig(ctx, cfg, log)
// defer device.Close()
// if influxCli != nil { defer influxCli.Close() }

// ── Query / Mutation → connCommand (concurrent via tag demux) ───
go func() {
    reply, _ := device.Path("/ip/address").Print().Exec(ctx)
    fmt.Println("address rows:", len(reply.Rows))
}()
go func() {
    reply, _ := device.Path("/ip/route").Print().Detail().Exec(ctx)
    fmt.Println("route rows:", len(reply.Rows))
}()

device.Path("/ip/address").Add(ctx,
    roslib.NewPair("address", "10.0.0.1/24"),
    roslib.NewPair("interface", "ether1"),
)

// ── Query dengan cache (TTL 30s) ────────────────────────────────
reply, _ := device.Path("/ip/address").Print().ExecCached(ctx, 30*time.Second)

// ── Stream print-follow → connStream ────────────────────────────
device.Path("/log").Print().FollowOnly().Stream("log-tail", func(s *roslib.Sentence) {
    log.WithField("msg", s.Get("message")).Info("router log")
})
device.Path("/ip/hotspot/active").Print().Follow().Stream("hotspot-active", func(s *roslib.Sentence) {
    log.WithFields(logrus.Fields{"user": s.Get("user"), "uptime": s.Get("uptime")}).Info("hotspot")
})

// ── Stream inherently-streaming (TANPA Print) ──────────────────
device.Path("/interface/monitor-traffic").
    With("interface", "ether1").
    Stream("nic-1", func(s *roslib.Sentence) {
        log.WithFields(logrus.Fields{
            "rx-bps": s.Get("rx-bits-per-second"),
            "tx-bps": s.Get("tx-bits-per-second"),
        }).Info("nic")
    })

device.Path("/tool/ping").
    With("address", "8.8.8.8").
    With("count", "5").
    Stream("ping-google", func(s *roslib.Sentence) {
        log.WithField("time", s.Get("time")).Info("ping reply")
    })

// ── Poll → di-batch oleh interval-group ─────────────────────────
device.RegisterPoll(roslib.PollConfig{
    ID:       "system-resource",
    Path:     "/system/resource",
    Args:     []string{"print"},
    Interval: 5 * time.Second,
    Handler:  influx.PollSink(writer, log.WithField("sink", "influx")),
})
device.RegisterPoll(roslib.PollConfig{
    ID:       "interface-stats",
    Path:     "/interface",
    Args:     []string{"print", "stats"},
    Interval: 5 * time.Second,
    Handler:  func(s *roslib.Sentence) { _ = s.Get("rx-byte") },
})

// Misuse → strict validator menolak.
_, err = device.Path("/interface/monitor-traffic").Print().Exec(ctx)
// err: capability: /interface/monitor-traffic has class Streaming,
//      expected OneShot|StreamablePrint|Mutation — use .Stream() ...

// Unregister kapan saja
device.UnregisterPoll("system-resource")
device.UnregisterStream("nic-1")
```

---

## 7. Ringkasan Perubahan dari Rancangan Sebelumnya

| Aspek | Rancangan lama | Rancangan baru |
|---|---|---|
| Model koneksi | Pool banyak koneksi (acquire/release per command) | **2 koneksi persistent per router** (stream + command) |
| Concurrent query | Antri atau butuh koneksi tambahan | **AsyncContext + tag demux** di connCommand: ratusan command paralel |
| Stream listener | Dedicated conn per listener | **Satu connStream, semua listener concurrent via tag** |
| Polling — scheduler | Ticker per path (50 path = 50 goroutine ticker) | **Interval group**: poll dengan interval sama digabung, 1 ticker per group |
| Polling — dispatch | Sequential per tick (1 command per tick) | **Fan-out concurrent**: tiap tick fire semua command group via async tag |
| Reconnect — stream | Re-dial per listener (manual & rumit) | **`StreamManager.ReattachAll(newConn)`** re-register otomatis |
| Reconnect — command | Manual handling per call | **Supervisor goroutine** nunggu `<-chan error` dari `AsyncContext` + exponential backoff |
| Logging | Tidak spesifik | **logrus** sebagai logger user; adapter slog→logrus untuk `SetLogHandler` go-routeros |
| Overhead koneksi | Tinggi (banyak TCP socket per router) | **Minimal: 2 TCP socket per router**, lifetime panjang |
| Overhead goroutine polling | O(N) — N = jumlah command poll | **O(G)** — G = jumlah interval unik (biasanya jauh lebih kecil) |
| Drop policy poll | Tidak ada / implicit | **Per-tick timeout = interval** + opsi inFlight counter |
| Validasi command | Runtime error dari router (lambat) | **Capability registry embed (JSON 7.20.8)** — validasi local, klasifikasi otomatis streaming vs one-shot |
| API streaming non-print | Tidak dibedakan dari print-follow | **`Path(p).Stream(id, h)` langsung** untuk monitor-traffic/ping/torch/sniffer tanpa kata `follow` |
| Konfigurasi | Literal `Options{}` per panggilan | **Package `config/` + `LoadFromEnv` + `NewFromConfig`** terpusat |
| Cache hasil query | Tidak ada | **`PrintBuilder.ExecCached(ctx, ttl)`** + interface Cache + Noop/InMemory/Redis-skeleton |
| Observability sink | Disebut tapi belum jadi | **`metrics/influx/`** Writer+Reader+BatchedWriter+PollSink/StreamSink helper |

---

## 8. Hal yang Sengaja Diluar Scope (untuk dipikir nanti)

- **Multi-router orchestration** (1 service handle banyak router): bisa di lapisan atas, `map[routerID]*RouterDevice`.
- **Distributed locking** untuk mutation yang harus serial cross-instance: di luar library, pakai Redis/etcd.
- **Rate limiting per router** untuk mencegah API overload: bisa ditambah middleware di builder layer.
- **Metrics expose Prometheus** untuk health connect / reconnect count / poll latency: gampang ditambah di supervisor & poll engine.
- **Multi-version registry** — saat ini hanya RouterOS 7.20.8 yang di-embed. User pakai `RegistryPath` untuk firmware lain; bundling multi-version belum tersedia.