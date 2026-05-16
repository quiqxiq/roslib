# Usage Guide — `roslib`

Dokumentasi penggunaan per fitur. Bukan API reference (lihat `go doc github.com/quiqxiq/roslib/...` untuk itu) — ini panduan task-oriented.

## Daftar isi

- [Instalasi & koneksi pertama](#instalasi--koneksi-pertama)
- [Loading konfigurasi](#loading-konfigurasi)
- [Multi-router (fleet)](#multi-router-fleet)
- [Snapshot query (Print Exec)](#snapshot-query-print-exec)
- [Mutation: add, set, remove, enable, disable](#mutation-add-set-remove-enable-disable)
- [Streaming (Listener)](#streaming-listener)
  - [Print-follow (log, table-snapshot + delta)](#print-follow-log-table-snapshot--delta)
  - [Print + Interval (queue counter, interface stats)](#print--interval-queue-counter-interface-stats)
  - [Inherent streaming (monitor-traffic, ping, torch, sniffer)](#inherent-streaming-monitor-traffic-ping-torch-sniffer)
- [Polling terjadwal](#polling-terjadwal)
- [Cache (InMemory / Redis)](#cache-inmemory--redis)
- [Cache invalidation](#cache-invalidation)
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

Library punya loader env stdlib. **Rekomendasi pakai `NewManagerFromConfig`** — Manager memegang koneksi persisten by-name, sehingga acquire ulang tidak re-dial:

```go
import "github.com/quiqxiq/roslib/config"

cfg, err := config.LoadFromEnv()
if err != nil { ... }

mgr, influxCli, err := roslib.NewManagerFromConfig(ctx, cfg, logger)
if err != nil { ... }
defer mgr.CloseAll()
if influxCli != nil { defer influxCli.Close() }

dev, _ := mgr.Get(roslib.DefaultDeviceKey)
// pemanggilan ulang mgr.Get(...) reuse pointer yang sama — 0 login tambahan
```

Pola lama (deprecated, tetap kerja) — `NewFromConfig` return `*Device` mentah, caller harus tracking lifecycle sendiri:

```go
dev, influxCli, err := roslib.NewFromConfig(ctx, cfg, logger) // deprecated
defer dev.Close()
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

## Multi-router (fleet)

Library menyediakan helper minimal: tidak ada package `fleet/`, hanya `map[string]*Device` + loader env multi-router + helper close massal.

### Env scheme

```bash
ROSLIB_ROUTERS=rb1,rb2,office-gw          # comma-separated ID list
ROSLIB_ROUTER_RB1_ADDRESS=192.168.88.1:8728
ROSLIB_ROUTER_RB1_USERNAME=admin
ROSLIB_ROUTER_RB1_PASSWORD=secret
ROSLIB_ROUTER_RB2_ADDRESS=192.168.88.2:8728
ROSLIB_ROUTER_RB2_USERNAME=admin
ROSLIB_ROUTER_RB2_PASSWORD=secret
ROSLIB_ROUTER_OFFICE_GW_ADDRESS=10.0.0.1:8728   # "office-gw" → OFFICE_GW di env name
ROSLIB_ROUTER_OFFICE_GW_USERNAME=admin
ROSLIB_ROUTER_OFFICE_GW_PASSWORD=secret

# Cache/Influx/strict toggle tetap shared (lihat single-router env).
```

ID dash di-translate ke underscore di nama env: `office-gw` → `ROSLIB_ROUTER_OFFICE_GW_*`.

### Konstruksi

Rekomendasi pakai `NewManagerFromFleet` — semua router di-register ke satu Manager, acquire ulang tidak re-dial:

```go
import "github.com/quiqxiq/roslib/config"

fleetCfg, _ := config.LoadFleetFromEnv()
mgr, influxCli, _ := roslib.NewManagerFromFleet(ctx, fleetCfg, logger)
defer mgr.CloseAll()
if influxCli != nil { defer influxCli.Close() }

// Akses per ID
rb1, _ := mgr.Get("rb1")
reply, _ := rb1.Path("/ip/address").Print().Exec(ctx)

// Iterasi semua device
for _, id := range mgr.Names() {
    dev, _ := mgr.Get(id)
    go func(id string, d *roslib.Device) {
        rep, _ := d.Path("/system/resource").Print().Exec(ctx)
        log.Printf("[%s] uptime=%s", id, rep.Rows[0].Get("uptime"))
    }(id, dev)
}
```

Pola lama (deprecated, tetap kerja) — `NewFleet` return `map[string]*Device`:

```go
fleet, influxCli, _ := roslib.NewFleet(ctx, fleetCfg, logger) // deprecated
defer roslib.CloseAll(fleet)
```

### Cache + multi-router

Cache instance dibagi antar router (kalau enabled). Key cache otomatis di-prefix dengan device ID, jadi sentence yang sama dari dua router tidak konflik. Pemeriksaan ada di `TestExecCachedDeviceScoping`.

```text
roslib:rb1:<hash>   ← cache untuk rb1
roslib:rb2:<hash>   ← cache untuk rb2, isi bisa berbeda
```

### Atomic dial

`NewManagerFromFleet` dial semua router sekuensial. Kalau satu gagal, semua yang sudah berhasil di-close (rollback) dan return error. Caller yang butuh "best-effort" (skip router yang gagal) boleh konstruk `Manager` kosong lewat `roslib.NewManager()` lalu `Register` per router dengan error handling sendiri.

---

## Persistent connection via Manager

`*device.RouterDevice` sudah persistent secara internal (2 koneksi async + supervisor reconnect), tapi setiap panggilan `device.New` atau `roslib.New[FromConfig]` membuka sesi baru di MikroTik. Untuk service yang sering re-acquire device (mis. HTTP handler yang pegang ID router dari path), pakai Manager supaya 1 router = 1 login sepanjang umur aplikasi.

### Pola dasar

```go
mgr, influxCli, _ := roslib.NewManagerFromConfig(ctx, cfg, log)
defer mgr.CloseAll()

// Setiap HTTP request misalnya:
func handle(w http.ResponseWriter, r *http.Request) {
    dev, err := mgr.Get(roslib.DefaultDeviceKey) // tidak dial — reuse
    if err != nil { http.Error(w, err.Error(), 500); return }
    reply, _ := dev.Path("/system/resource").Print().Exec(r.Context())
    json.NewEncoder(w).Encode(reply.Rows)
}
```

| Method | Perilaku |
|---|---|
| `mgr.Register(ctx, name, opts)` | Dial sekali, store dengan key. Error kalau name sudah ada dan masih alive. |
| `mgr.Get(name)` | Reuse device yang ada. Error kalau belum di-register. **Tidak** verifikasi alive — caller cek `dev.IsAlive()` kalau perlu. |
| `mgr.GetOrConnect(ctx, name, opts)` | Reuse kalau alive, dial baru kalau belum ada atau sudah mati. Double-checked locking. **Workhorse production.** |
| `mgr.Unregister(name)` | Close + hapus dari pool. |
| `mgr.CloseAll()` | Close semua, kosongkan pool. |
| `mgr.Names()` | List semua key terdaftar. |

### Bukti behaviour

Lihat `cmd/test-persistence/main.go` (pola Manager) dan `cmd/test-nonpersistence/main.go` (pola `device.New` per-acquire). Dijalankan langsung ke router:

```bash
go run ./cmd/test-nonpersistence   # 5 iterasi → +10 login di /log
go run ./cmd/test-persistence      # 10 acquire → +0..1 login (pointer identik)
```

### Power-user: koneksi terpisah per role

Kalau workload streaming bisa membanjiri queue command (misal `/tool/sniffer` + heavy query), pisahkan koneksi pakai `RoleKey` helper:

```go
streamKey := roslib.RoleKey("rb1", roslib.RoleStream)
cmdKey    := roslib.RoleKey("rb1", roslib.RoleCommand)
mutKey    := roslib.RoleKey("rb1", roslib.RoleMutation)

mgr.Register(ctx, streamKey, streamOpts) // queue besar, no timeout
mgr.Register(ctx, cmdKey, cmdOpts)       // timeout 10s
mgr.Register(ctx, mutKey, mutOpts)       // timeout 30s

streamDev, _ := mgr.Get(streamKey)
cmdDev, _    := mgr.Get(cmdKey)
mutDev, _    := mgr.Get(mutKey)
```

Tiap key = `RouterDevice` independen dengan 2 koneksi internal sendiri (jadi 6 koneksi total ke router fisik yang sama). Tidak otomatis — Manager hanya menyediakan key namespace. Default `NewManagerFromConfig` / `NewManagerFromFleet` hanya register 1 device per router.

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

### Print + Interval (queue counter, interface stats)

Beberapa `print` tidak punya event-stream sendiri tapi mendukung polling via `interval=<d>`. Pola tipikal: `/queue/simple/print stats interval=1s`, `/interface/print stats interval=2s`.

```go
// Counter per queue, update tiap 1 detik.
dev.Path("/queue/simple").Print().Stats().
    Interval(1 * time.Second).
    Stream("q-stats", func(s *roslib.Sentence) {
        logger.WithFields(logrus.Fields{
            "name":  s.Get("name"),
            "bytes": s.Get("bytes"),
            "rate":  s.Get("rate"),
        }).Info("queue")
    })

// Counter interface dengan rate, update tiap 2 detik.
dev.Path("/interface").Print().Stats().Rate().
    Interval(2 * time.Second).
    Stream("nic-stats", handler)

// Bisa dikombinasi dengan Follow / FollowOnly (event-driven + keep-alive poll).
dev.Path("/ip/firewall/filter").Print().
    Follow().Interval(2 * time.Second).
    Stream("fw-flow", handler)
```

**Validator**: `.Stream()` tanpa salah satu dari `Follow()`/`FollowOnly()`/`Interval()` akan return `ErrNoStreamFlag` — listener tanpa flag streaming akan langsung close oleh RouterOS.

Helper flag print yang umum dipakai dengan Interval:

| Method | Flag |
|---|---|
| `.Stats()` | `stats` — counter byte/packet |
| `.Bytes()` | `bytes` — counter byte saja |
| `.Packets()` | `packets` — counter packet saja |
| `.Rate()` | `rate` — bit/byte rate per detik |
| `.Detail()` | `detail` — semua field |
| `.Proplist("f1","f2")` | `proplist=f1,f2` — projection |
| `.Count()` | `count-only` — jumlah row saja |

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

### Finite-stream auto-cleanup + `OnFinish`

Command dengan batas finite (mis. `/tool/ping count=5`, `/tool/torch duration=2s`) akan kirim `!done` saat selesai. Sejak iterasi-3, library:

1. **Auto-clean** entry dari `Manager.listeners` saat natural close — `ReattachAll` pasca reconnect tidak lagi re-attach listener yang sudah selesai.
2. **Tetap simpan** entry kalau channel close karena connection error (Err != nil) — supaya `ReattachAll` bisa daftar ulang.

Pasang callback `OnFinish` untuk tahu kapan listener selesai (natural atau error):

```go
done := make(chan error, 1)
dev.Path("/tool/ping").
    With("address", "8.8.8.8").
    With("count", "5").
    OnFinish(func(id string, err error) {
        // err == nil → natural !done dari router (entry sudah dihapus)
        // err != nil → connection drop (entry tetap, ReattachAll akan handle)
        done <- err
    }).
    Stream("ping-finite", handler)

if err := <-done; err == nil {
    fmt.Println("ping selesai natural")
}
```

Cek jumlah listener aktif:

```go
fmt.Println("active listeners:", dev.Streams().Len())
```

`OnFinish` juga tersedia di chain `Print().Follow()/FollowOnly()/Interval()`:

```go
dev.Path("/log").Print().Follow().
    OnFinish(func(id string, err error) { /* ... */ }).
    Stream("log-follow", handler)
```

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

## Cache invalidation

Library **tidak** auto-invalidate setelah mutation (per design — predictable & explicit). User panggil manual saat tahu state berubah.

### Single device

```go
// Setelah panggilan mutation library:
_, _ = dev.Path("/ip/address").Add(ctx, roslib.NewPair("address", "10.0.0.1/24"))
_ = dev.InvalidateCache(ctx, "/ip/address")

// Atau setelah perubahan eksternal (WinBox, SSH, operator lain):
_ = dev.InvalidateCache(ctx, "/ip/address")
reply, _ := dev.Path("/ip/address").Print().ExecCached(ctx, 30*time.Second) // fresh
```

### Lewat cache instance langsung

```go
import "context"

_ = sharedCache.InvalidatePath(context.Background(), "/ip/address")
```

### Yang dijamin

- Entry yang di-Set lewat **`ExecCached`** ter-track ke `pathIdx` dan dihapus oleh `InvalidatePath(path)`.
- Entry yang di-Set lewat `Set` biasa **tidak ter-track** dan tidak ikut hilang (sengaja — escape hatch).
- `NoopCache.InvalidatePath` no-op.
- Path tidak ada di cache → `InvalidatePath` no-op (idempotent).

### Test konsistensi yang ada

| Test | Lokasi |
|---|---|
| `TestInvalidatePath_InMemory` | Set → Invalidate → Miss |
| `TestInvalidatePath_Partial` | Invalidate satu path, path lain tetap |
| `TestInvalidatePath_NotFound` | Idempotent untuk path absent |
| `TestInvalidatePath_AfterExpiry` | Tidak bocor reference saat TTL expire |
| `TestConcurrentSetInvalidate` | Race detector clean |
| `TestSetWithoutPath` | Set tanpa path tidak ter-affect |
| `TestExecCachedInvalidate` | ExecCached × 2 → invalidate → miss lagi |
| `TestExecCachedDeviceScoping` | Cache dipisah per device di fleet |
| Live `CACHE/INVALIDATE-LIVE` | Verifikasi end-to-end di router fisik |

Run: `go test ./cache/ ./builder/ -v -run "Invalidate|ExecCached"`.

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
