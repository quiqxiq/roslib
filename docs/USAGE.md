# Usage Guide — `roslib`

Dokumentasi penggunaan per fitur. Bukan API reference (lihat `go doc github.com/quiqxiq/roslib/...` untuk itu) — ini panduan task-oriented.

## Daftar isi

- [Instalasi & koneksi pertama](#instalasi--koneksi-pertama)
- [Loading konfigurasi](#loading-konfigurasi)
- [Snapshot query (Print Exec)](#snapshot-query-print-exec)
- [Mutation: add, set, remove, enable, disable](#mutation-add-set-remove-enable-disable)
- [Streaming (Listener)](#streaming-listener)
  - [Print-follow (log, table-snapshot + delta)](#print-follow-log-table-snapshot--delta)
  - [Inherent streaming (monitor-traffic, ping, torch, sniffer)](#inherent-streaming-monitor-traffic-ping-torch-sniffer)
- [Polling terjadwal](#polling-terjadwal)
- [Cache (InMemory / Redis)](#cache-inmemory--redis)
- [Sink ke InfluxDB3](#sink-ke-influxdb3)
- [Capability validator: strict vs warn vs override](#capability-validator-strict-vs-warn-vs-override)
- [Auto-reconnect & lifecycle](#auto-reconnect--lifecycle)
- [Decoding sentence (typed accessor)](#decoding-sentence-typed-accessor)
- [Debugging tips](#debugging-tips)

---

## Instalasi & koneksi pertama

```bash
go get github.com/quiqxiq/roslib
```

```go
import (
    "context"
    "github.com/quiqxiq/roslib"
    "github.com/sirupsen/logrus"
)

func connect() (*roslib.Device, error) {
    return roslib.New(context.Background(), roslib.Options{
        Address:          "192.168.88.1:8728",
        Username:         "admin",
        Password:         "secret",
        Logger:           logrus.New(),
        StrictCapability: true,
    })
}
```

`New` melakukan dial **dua koneksi** ke router (stream + command), mengaktifkan async mode di keduanya, dan memulai supervisor goroutine untuk auto-reconnect. Cancel parent context atau panggil `dev.Close()` untuk shutdown bersih.

---

## Loading konfigurasi

Library punya loader env stdlib:

```go
import "github.com/quiqxiq/roslib/config"

cfg, err := config.LoadFromEnv()
if err != nil { ... }
dev, influxCli, err := roslib.NewFromConfig(ctx, cfg, logger)
```

Env yang dibaca (lihat `.env.example`):

| Env | Default | Catatan |
|---|---|---|
| `ROSLIB_ROUTER_ADDRESS` | (wajib) | `host:port`, biasa `:8728` |
| `ROSLIB_ROUTER_USERNAME` | (kosong) | |
| `ROSLIB_ROUTER_PASSWORD` | (kosong) | |
| `ROSLIB_ROUTER_TLS` | `false` | `true` untuk port 8729 |
| `ROSLIB_ROUTER_INSECURE_SKIP_VERIFY` | `false` | TLS verify cert |
| `ROSLIB_DIAL_TIMEOUT` | `10s` | per dial attempt |
| `ROSLIB_LISTEN_QUEUE_SIZE` | `100` | channel buffer per listener |
| `ROSLIB_RECONNECT_INITIAL` | `500ms` | initial backoff |
| `ROSLIB_RECONNECT_MAX` | `30s` | max backoff |
| `ROSLIB_RECONNECT_MAX_ELAPSED` | `0` | 0 = retry selamanya |
| `ROSLIB_STRICT_CAPABILITY` | `true` | false = log-warn |
| `ROSLIB_REGISTRY_PATH` | (kosong) | path override JSON registry |
| `ROSLIB_CACHE_ENABLED` | `false` | |
| `ROSLIB_CACHE_ADDR` | (kosong) | `host:port` Redis |
| `ROSLIB_CACHE_PASSWORD` | (kosong) | |
| `ROSLIB_CACHE_DB` | `0` | Redis DB index |
| `ROSLIB_CACHE_TTL` | `30s` | default TTL ExecCached |
| `ROSLIB_INFLUX_ENABLED` | `false` | |
| `INFLUX_HOST` | (kosong) | URL InfluxDB3 |
| `INFLUX_TOKEN` | (kosong) | placeholder kalau auth off |
| `INFLUX_DATABASE` | (kosong) | |
| `INFLUX_ORG` | (kosong) | optional |
| `ROSLIB_INFLUX_MEASUREMENT` | `roslib` | default measurement name |

Pakai `godotenv` di app `main()` kalau butuh `.env` file (library tidak parsing `.env` sendiri).

---

## Snapshot query (Print Exec)

`Exec` mengirim satu RunArgs ke connCommand dan menunggu reply lengkap. Aman dipanggil concurrent — semua command paralel di socket yang sama via tag demux.

```go
reply, err := dev.Path("/ip/address").Print().Exec(ctx)
for _, row := range reply.Rows {
    fmt.Println(row.Get("address"), row.Get("interface"))
}
```

Flag tambahan tersedia sebagai method chain:

```go
dev.Path("/ip/route").Print().Detail().Exec(ctx)
dev.Path("/interface").Print().Stats().Exec(ctx)
dev.Path("/log").Print().Count().Exec(ctx)
dev.Path("/ip/route").Print().Flag("terse").Exec(ctx) // flag bebas
```

Filter (`?key=value`):

```go
dev.Path("/ip/firewall/filter").
    Print().
    Where("chain", "input").
    WherePair(roslib.WhereNot("disabled", "true")).
    Exec(ctx)
```

Named parameter (`=key=value`):

```go
dev.Path("/ip/dhcp-server/lease").Print().With("server", "dhcp1").Exec(ctx)
```

Reply structure:

```go
type Reply struct {
    Raw  *routeros.Reply       // raw dari go-routeros — kalau butuh sentinel Done
    Rows []*decode.Sentence    // wrapper typed accessor
}
```

---

## Mutation: add, set, remove, enable, disable

```go
// Add
_, err := dev.Path("/ip/address").Add(ctx,
    roslib.NewPair("address", "10.0.0.1/24"),
    roslib.NewPair("interface", "ether1"),
    roslib.NewPair("comment", "via roslib"),
)

// Set (numbers = .id atau index)
_, err = dev.Path("/ip/firewall/filter").Set(ctx, "*A",
    roslib.NewPair("action", "drop"),
)

// Remove
_, err = dev.Path("/ip/address").Remove(ctx, "*B")

// Enable/Disable
_, err = dev.Path("/interface").Enable(ctx, "ether2")
_, err = dev.Path("/interface").Disable(ctx, "ether2")
```

Validasi strict: command harus berkelas `Mutation` di registry. Misuse (mis. memanggil `Add` pada path streaming) langsung ditolak.

Untuk command yang tidak punya helper khusus, pakai `Run`:

```go
reply, err := dev.Path("/system/script").Run(ctx, "run",
    roslib.NewPair("number", "0"))
```

---

## Streaming (Listener)

Library punya **dua jalur streaming**, sesuai class di registry:

### Print-follow (log, table-snapshot + delta)

Untuk command yang punya arg `follow`/`follow-only`/`interval` (508 path di v7.20.8 — log, address-list, firewall counter, dst):

```go
// FollowOnly = hanya event baru
_ = dev.Path("/log").Print().FollowOnly().Stream("log-tail", func(s *roslib.Sentence) {
    logger.WithField("msg", s.Get("message")).Info("router log")
})

// Follow = snapshot dulu, lalu emit perubahan
_ = dev.Path("/ip/hotspot/active").Print().Follow().Stream("hotspot",
    func(s *roslib.Sentence) {
        logger.WithFields(logrus.Fields{
            "user":   s.Get("user"),
            "uptime": s.Get("uptime"),
        }).Info("hotspot user")
    })
```

Tweak optional:

```go
dev.Path("/log").Print().FollowOnly().
    QueueSize(500).
    CancelTimeout(3 * time.Second).
    Stream("log-tail", handler)
```

### Inherent streaming (monitor-traffic, ping, torch, sniffer)

Untuk command yang **selalu** streaming tanpa butuh kata `follow` (50+ path: `/interface/monitor-traffic`, `/tool/ping`, `/tool/torch`, `/tool/sniffer/*`, `/interface/**/monitor`, …):

```go
dev.Path("/interface/monitor-traffic").
    With("interface", "ether1").
    With("once", "no").
    Stream("nic-1", func(s *roslib.Sentence) {
        logger.WithFields(logrus.Fields{
            "rx-bps": s.Get("rx-bits-per-second"),
            "tx-bps": s.Get("tx-bits-per-second"),
        }).Info("nic")
    })

dev.Path("/tool/ping").
    With("address", "8.8.8.8").
    With("count", "5").
    Stream("ping-google", func(s *roslib.Sentence) {
        fmt.Printf("ping seq=%s time=%s\n", s.Get("seq"), s.Get("time"))
    })

dev.Path("/tool/torch").
    With("interface", "ether1").
    Stream("torch-1", torchHandler)
```

Flag bebas (tanpa `=`):

```go
dev.Path("/tool/ping").Arg("arp-ping").With("address", "192.168.1.1").
    Stream("arp-ping-1", handler)
```

Unregister kapan saja:

```go
dev.UnregisterStream("nic-1")
```

Listener berbagi **satu** connStream lewat tag demux. Reconnect otomatis akan `ReattachAll` seluruh listener.

---

## Polling terjadwal

`PollEngine` mengelompokkan poll berdasarkan interval — N command dengan 3 interval berbeda = 3 goroutine ticker, bukan N.

```go
// Interval 5s — group A.
_ = dev.RegisterPoll(roslib.PollConfig{
    ID:       "sys-resource",
    Path:     "/system/resource",
    Args:     []string{"print"},
    Interval: 5 * time.Second,
    Handler:  func(s *roslib.Sentence) { /* sink */ },
})
_ = dev.RegisterPoll(roslib.PollConfig{
    ID:       "interface-stats",
    Path:     "/interface",
    Args:     []string{"print", "stats"},
    Interval: 5 * time.Second, // group A juga — fan-out concurrent di tick yang sama
    Handler:  func(s *roslib.Sentence) {},
})

// Interval 30s — group B baru.
_ = dev.RegisterPoll(roslib.PollConfig{
    ID:       "package-check",
    Path:     "/system/package/update",
    Args:     []string{"check-for-updates"},
    Interval: 30 * time.Second,
    Handler:  func(s *roslib.Sentence) {},
})

// Stop kapan saja
dev.UnregisterPoll("sys-resource")
```

Args konvensi: `Args[0]` = action (default `"print"`), `Args[1:]` = flag-word seperti `"detail"`/`"stats"`. Filter `?k=v` via `Where`, named param `=k=v` via `Pairs`.

Drop policy:

- `Timeout` per command (default = `Interval`) — kalau router lambat, command di-cancel dan tick berikutnya tetap jalan.
- `MaxInFlight > 0` — skip tick kalau jumlah command in-flight ≥ limit (cegah thundering herd).

```go
roslib.PollConfig{
    ID: "slow-q", Path: "/queue/simple", Args: []string{"print", "stats"},
    Interval: 5 * time.Second,
    Timeout: 3 * time.Second, // batas lebih ketat dari interval
    MaxInFlight: 2,
    Handler: handler,
}
```

---

## Cache (InMemory / Redis)

`PrintBuilder.ExecCached(ctx, ttl)` cek cache dulu, hit → decode langsung dari cache (JSON), miss → hit router lalu simpan.

```go
dev, _ := roslib.New(ctx, roslib.Options{
    ...,
    Cache:    cache.NewInMemory(),
    CacheTTL: 30 * time.Second,
})

reply, _ := dev.Path("/ip/address").Print().ExecCached(ctx, 30*time.Second)
// panggilan kedua dalam 30 detik → langsung dari memory, tidak hit router
```

Key kanonik:

```go
key := cache.KeyOf([]string{"/ip/address/print"})
// → "roslib:5b3c..." (sha256 hex)
```

### Redis (build-tag `redis`)

`cache/redis.go` adalah skeleton yang **tidak dikompilasi** secara default supaya `go-redis` bukan dependency wajib library inti.

```bash
go get github.com/redis/go-redis/v9
go build -tags=redis ./...
```

```go
import "github.com/quiqxiq/roslib/cache"

cli, _ := cache.NewRedis(cache.RedisOptions{
    Addr: "127.0.0.1:6379",
    DB:   0,
})
opts.Cache = cli
```

---

## Sink ke InfluxDB3

`metrics/influx/Writer` mengubah `*decode.Sentence` → `*influxdb3.Point`:

```go
import "github.com/quiqxiq/roslib/metrics/influx"

writer := influx.NewWriter(influxCli, "system_resource",
    func(s *roslib.Sentence) map[string]string {
        return map[string]string{
            "board": s.Get("board-name"),
            "ver":   s.Get("version"),
        }
    },
    func(s *roslib.Sentence) map[string]any {
        return map[string]any{
            "cpu_load":     s.IntOr("cpu-load", 0),
            "free_memory":  s.BytesOr("free-memory", 0),
            "uptime_secs":  int64(s.DurationOr("uptime", 0).Seconds()),
        }
    },
)

// Wire ke poll
dev.RegisterPoll(roslib.PollConfig{
    ID: "sys", Path: "/system/resource", Args: []string{"print"},
    Interval: 5 * time.Second,
    Handler:  influx.PollSink(writer, logger.WithField("sink", "influx")),
})

// Wire ke stream
_ = dev.Path("/log").Print().FollowOnly().Stream("log",
    influx.StreamSink(logWriter, logger))
```

### InfluxDB BatchedWriter

Kalau throughput tinggi (mis. monitor-traffic 1Hz × 10 interface = 10 point/det), pakai batcher:

```go
bw := influx.NewBatchedWriter(writer, /*maxSize*/ 500, /*interval*/ 2*time.Second)
bw.Start(ctx)
defer bw.Close(ctx) // flush final

dev.RegisterPoll(roslib.PollConfig{
    ID: "sys", Path: "/system/resource", Args: []string{"print"},
    Interval: time.Second,
    Handler:  influx.BatchedPollSink(bw),
})
```

### Reader

`Reader.Query` tipis di atas SDK upstream — user iterate sendiri:

```go
reader := influx.NewReader(influxCli)
iter, err := reader.Query(ctx,
    "SELECT time, cpu_load, free_memory FROM system_resource ORDER BY time DESC LIMIT 10")
if err != nil { return err }

for iter.Next() {
    row := iter.Value() // map[string]any
    fmt.Println(row)
}
```

InfluxQL juga didukung lewat opsi:

```go
import "github.com/InfluxCommunity/influxdb3-go/v2/influxdb3"

iter, _ := reader.Query(ctx, "SELECT * FROM measurement",
    influxdb3.WithQueryType(influxdb3.InfluxQL))
```

---

## Capability validator: strict vs warn vs override

Library punya registry **RouterOS 7.20.8** yang di-embed (541 endpoint).

### Strict (default)

```go
opts.StrictCapability = true
```

- Command word tidak ada di registry → `ErrUnknownCommand`
- Arg tidak ada di Command.Args → `ErrUnknownArg`
- Class mismatch (mis. `.Exec()` di path streaming) → `ErrInvalidClass`

Semua return error Go sebelum sentence dikirim ke router.

### Warn

```go
opts.StrictCapability = false
```

Validator log-warn lewat `device.Logger()` lalu **tetap kirim** sentence. Berguna kalau punya router versi non-7.x atau path eksperimental.

### Override registry

Untuk versi RouterOS lain (mis. v6 atau v7.21 dengan command baru):

```go
import "github.com/quiqxiq/roslib/capability"

// File path
reg, err := capability.Load(capability.LoadOptions{Path: "routeros_6.49.json"})
opts.Registry = reg

// Inline bytes
reg, err = capability.Load(capability.LoadOptions{Bytes: myJSONBytes})
opts.Registry = reg

// Disable validasi total
opts.Registry = nil
```

Format JSON sama dengan `capability/assets/mikrotik/routeros_7.20.8.json` — tree `_type=path/dir/cmd/arg`.

---

## Auto-reconnect & lifecycle

Setiap koneksi punya supervisor goroutine yang baca `<-chan error` dari `AsyncContext`. Kalau channel emit error (TCP putus, `!fatal`, dst):

1. Tutup koneksi lama (idempotent).
2. Backoff exponential (default 500ms → 30s, retry selamanya).
3. Redial + `AsyncContext` baru.
4. Update pointer di RouterDevice (mutex).
5. **connCommand reconnect** → `PollEngine.AttachConn(new)` (pointer swap).
6. **connStream reconnect** → `StreamManager.ReattachAll(new)` (re-register tiap listener).

Tunable:

```go
opts.ReconnectInitialInterval = time.Second
opts.ReconnectMaxInterval     = 60 * time.Second
opts.ReconnectMaxElapsed      = 5 * time.Minute  // 0 = unlimited
opts.DialTimeout              = 5 * time.Second
```

Shutdown:

```go
dev.Close() // graceful — cancel context, hentikan poll engine + stream manager, tutup keduanya
```

`Close` aman dipanggil multiple times.

---

## Decoding sentence (typed accessor)

`*decode.Sentence` (alias `*roslib.Sentence`) membungkus `*proto.Sentence` dengan helper:

```go
s.Get(key)              // raw string ("" kalau absent)
s.Has(key)              // bool
s.Word()                // "!re" / "!done" / "!trap" / "!fatal"

// Typed dengan error
val, err := s.Bool("disabled")
n, err   := s.Int("cpu-load")
f, err   := s.Float("temperature")
d, err   := s.Duration("uptime")    // "1w2d3h" atau "00:01:23.456"
b, err   := s.Bytes("free-memory")  // "1024K", "2M", "3G" → int64
t, err   := s.Time("creation-time") // ISO 8601 atau "Jan/02/2006 15:04:05"

// Typed dengan default fallback
n  := s.IntOr("cpu-load", 0)
d  := s.DurationOr("uptime", 0)
b  := s.BytesOr("free-memory", 0)
```

---

## Debugging tips

### Logging internal go-routeros

Library otomatis pasang `slog.Handler` yang redirect ke logrus. Semua log internal (sentence yang dikirim/diterima, tag, dst) bisa di-stream:

```go
logger.SetLevel(logrus.DebugLevel)
```

### Print sentence sebelum kirim

Builder tidak mengekspos sentence langsung. Untuk inspect, pakai `query.BuildSentence` manual:

```go
import "github.com/quiqxiq/roslib/query"

words := query.BuildSentence("/ip/address/print", nil,
    []query.Pair{},
    []query.WherePair{query.Where("interface", "ether1")},
)
fmt.Println(words)
// → [/ip/address/print ?interface=ether1]
```

### Race detection

```bash
go test -race ./...
```

Library bersih dari race menurut detector (verified pada CI).

### Connection state

```go
dev.CommandConn() // *routeros.Client saat ini (bisa berubah pasca-reconnect)
dev.Streams()     // *stream.Manager
dev.Polls()       // *poll.Engine
```

---

## Catatan tentang RouterOS v6

Registry yang di-embed adalah v7.20.8. Beberapa path berbeda di v6:

- v6 `/ping` ↔ v7 `/tool/ping`
- v6 sebagian `/tool/*` tidak ada (langsung di root)

Mitigasi:

1. **`StrictCapability=false`** — terima semua sentence; router yang validate.
2. **Custom registry** dari JSON v6 (kalau punya).
3. **Mayoritas path stabil** — `/ip/address/*`, `/interface/*`, `/system/resource`, `/log`, dst sama di v6/v7.

Lihat detail di [docs/integration-report.md](integration-report.md).
