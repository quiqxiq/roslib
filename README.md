# roslib

Wrapper Go di atas [`github.com/go-routeros/routeros/v3`](https://pkg.go.dev/github.com/go-routeros/routeros/v3) untuk MikroTik RouterOS API, dirancang khusus untuk **async mode + tag multiplexing**: 2 koneksi persisten per router (stream + command), interval-group batching untuk polling, auto-reconnect via `<-chan error`, persistent connection pool via Manager, plus hooks ke cache & InfluxDB3 sebagai contoh observability sink.

> **Status:** sudah teruji terhadap router fisik MikroTik RB750G RouterOS 6.49.11 dan InfluxDB 3 Core. Lihat [docs/integration-report.md](docs/integration-report.md).

## Highlight

- ✅ **2 koneksi persisten per router** (stream + command) — bukan pool, bukan dial-per-call.
- ✅ **AsyncContext + tag demux**: ratusan `RunArgs`/`Listen` paralel di satu socket.
- ✅ **Interval-group polling**: 50 command, 3 interval unik → 3 goroutine ticker, bukan 50.
- ✅ **Auto-reconnect** dengan exponential backoff + `ReattachAll` untuk listener.
- ✅ **Persistent Manager**: `mgr.Get(name)` reuse koneksi — 1 router = 1 login sepanjang lifetime.
- ✅ **Fluent builder**: `dev.Path("/ip/address").Print().Where(...).Exec(ctx)` dst.
- ✅ **Cache + Influx + Config opt-in** lewat env var; toggle independen.

## Install

```bash
go get github.com/quiqxiq/roslib
```

Library butuh Go 1.21+ (untuk `log/slog`, `sync.OnceValues`, generics, `//go:embed`).

## Quickstart

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/quiqxiq/roslib"
    "github.com/sirupsen/logrus"
)

func main() {
    logger := logrus.New()
    ctx := context.Background()

    dev, err := roslib.New(ctx, roslib.Options{
        Address:  "192.168.88.1:8728",
        Username: "admin",
        Password: "secret",
        Logger:   logger,
    })
    if err != nil {
        log.Fatal(err)
    }
    defer dev.Close()

    // Snapshot.
    reply, _ := dev.Path("/ip/address").Print().Exec(ctx)
    log.Printf("address rows: %d", len(reply.Rows))

    // Real-time stream.
    _ = dev.Path("/interface/monitor-traffic").
        With("interface", "ether1").
        Stream("nic-1", func(s *roslib.Sentence) {
            log.Printf("rx=%s tx=%s",
                s.Get("rx-bits-per-second"),
                s.Get("tx-bits-per-second"))
        })

    // Background polling.
    _ = dev.RegisterPoll(roslib.PollConfig{
        ID:       "sys-resource",
        Path:     "/system/resource",
        Args:     []string{"print"},
        Interval: 5 * time.Second,
        Handler:  func(s *roslib.Sentence) { /* ... */ },
    })

    time.Sleep(30 * time.Second)
}
```

## Loading dari env (recommended untuk service)

Library punya loader stdlib (zero new dep). Set env, lalu `NewFromConfig`:

```bash
cp .env.example .env
# edit .env: minimal isi ROSLIB_ROUTER_ADDRESS, USERNAME, PASSWORD
```

```go
cfg, _ := config.LoadFromEnv()
mgr, influxCli, _ := roslib.NewFromConfig(ctx, cfg, logger)
defer mgr.CloseAll()
if influxCli != nil { defer influxCli.Close() }

dev, _ := mgr.Get(roslib.DefaultDeviceKey)
// pemanggilan mgr.Get() ulang reuse koneksi — 0 dial tambahan
```

User app bebas memuat `.env` duluan (mis. via `godotenv`) sebelum panggil `LoadFromEnv`.

## Pattern singkat

| Kebutuhan | API |
|---|---|
| Snapshot query | `dev.Path(p).Print().Where(...).Exec(ctx)` |
| Snapshot dengan cache | `dev.Path(p).Print().ExecCached(ctx, ttl)` |
| Stream print-follow | `dev.Path(p).Print().Follow().Stream(id, h)` (atau `FollowOnly()`) |
| Stream inherent (monitor-traffic, ping, …) | `dev.Path(p).With(k, v).Stream(id, h)` |
| Add/Set/Remove/Enable/Disable | `dev.Path(p).Add(ctx, pairs...)` dst |
| Command bebas | `dev.Path(p).Run(ctx, action, pairs...)` |
| Poll terjadwal | `dev.RegisterPoll(PollConfig{...})` |
| Unregister | `dev.UnregisterPoll(id)` / `dev.UnregisterStream(id)` |

Detail dan contoh lengkap: [docs/USAGE.md](docs/USAGE.md).

## Folder structure

```
roslib/
├── roslib.go               public facade — New(), NewFromConfig(), NewFleet(), tipe alias
├── device/                 RouterDevice + Manager (persistent pool) + supervisor
├── builder/                fluent API (Path → Print/Run/Add/Set/Stream)
├── stream/                 StreamManager + Spec blueprint + ReattachAll
├── poll/                   PollEngine + IntervalGroup batching
├── decode/                 Sentence wrapper + typed accessors
├── query/                  Pair, WherePair, BuildSentence
├── cache/                  Cache iface + Noop + InMemory + Redis (build-tag)
├── metrics/influx/         InfluxDB3 Writer/Reader/BatchedWriter/Sink helper
├── config/                 LoadFromEnv + LoadFleetFromEnv + ToDeviceOptions
├── cmd/integration/        runner test ke router asli (lihat report)
└── examples/               contoh usage / streaming / influx
```

## Cache (opsional)

Default `NoopCache`. Untuk `InMemoryCache`:

```go
opts.Cache = cache.NewInMemory()
opts.CacheTTL = 30 * time.Second
```

Skeleton Redis ada di `cache/redis.go` (build tag `redis`):

```bash
go get github.com/redis/go-redis/v9
go build -tags=redis ./...
```

Pemakaian:

```go
reply, err := dev.Path("/ip/address").Print().ExecCached(ctx, 30*time.Second)
```

## InfluxDB3 sink (opsional)

```go
writer := influx.NewWriter(influxCli, "system_resource",
    func(s *roslib.Sentence) map[string]string {
        return map[string]string{"board": s.Get("board-name")}
    },
    func(s *roslib.Sentence) map[string]any {
        return map[string]any{
            "cpu_load":    s.IntOr("cpu-load", 0),
            "free_memory": s.BytesOr("free-memory", 0),
        }
    },
)
dev.RegisterPoll(roslib.PollConfig{
    ID: "sys", Path: "/system/resource",
    Args: []string{"print"}, Interval: 5 * time.Second,
    Handler: influx.PollSink(writer, logger.WithField("sink", "influx")),
})

// Read back
reader := influx.NewReader(influxCli)
iter, _ := reader.Query(ctx, "SELECT * FROM system_resource ORDER BY time DESC LIMIT 10")
for iter.Next() {
    fmt.Println(iter.Value())
}
```

`BatchedWriter` tersedia kalau write-rate tinggi (lihat [USAGE](docs/USAGE.md#influxdb-batched-writer)).

## Test infrastruktur (Docker)

Library punya `test/docker-compose.yaml` minimal untuk InfluxDB3 Core lokal. Kalau sudah punya container sendiri, skip — tinggal kasih `INFLUX_HOST` ke endpoint yang ada.

```bash
cd test && docker compose up -d
```

## Test live ke router

```bash
export ROSLIB_ROUTER_ADDRESS=192.168.88.1:8728
export ROSLIB_ROUTER_USERNAME=admin
export ROSLIB_ROUTER_PASSWORD=secret
export ROSLIB_INFLUX_ENABLED=true
export INFLUX_HOST=http://localhost:8181
export INFLUX_TOKEN=no-auth
export INFLUX_DATABASE=mikrotik
go run ./cmd/integration
```

Output detail ada di [docs/integration-report.md](docs/integration-report.md).

## Test unit (tanpa router)

```bash
go test ./...
go test -race ./...
```

Unit test pada 13 paket — cache TTL + scoped invalidate, config loader, stream lifecycle, writer point-build, device manager persistence, dst.

## Dokumentasi tambahan

- **[docs/USAGE.md](docs/USAGE.md)** — detail per fitur (fluent API, polling, streaming, cache, influx, config).
- **[docs/integration-report.md](docs/integration-report.md)** — laporan test live + screenshot hasil.
- **[planing.md](planing.md)** — design doc lengkap (arsitektur, trade-off, perbandingan rancangan lama vs baru).

## Lisensi

Library ini mengikuti lisensi MIT (sama dengan upstream `go-routeros/v3`).
