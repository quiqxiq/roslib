# roslib-mikhmon

Reimplementasi RouterOS command & script milik [mikhmonv3](https://github.com/laksa19/mikhmonv3) di atas library [`roslib`](../). Tujuan akhir: REST API + SSE service untuk hotspot voucher management. **Iterasi pertama hanya layer command + script (no HTTP yet).**

## Status

| Layer | Status | Test |
|---|---|---|
| `domain/` | ✅ | unit |
| `internal/rosfmt/` | ✅ | unit |
| `internal/tcpmock/` | ✅ | self-test |
| `mikrotik/` (system, hotspot, ppp, network, syslog) | ✅ | integration (live router) |
| `scripts/` (onlogin, onevent, transaction, quickprint) | ✅ | unit (golden file) |
| `workflows/` (cascade) | ✅ | integration |
| `api/` (gin REST + SSE + middleware) | ✅ | live smoke |
| `cmd/server/` (entry point) | ✅ | manual |

Refactor besar (Mei 2026): paket `mikrotik/` di-thin-out — sekarang langsung di atas `*roslib.Device` (bukan `CommandRunner` boundary). Akibatnya muncul method khusus per resource untuk:

- **Streaming**: `hotspot.ActiveStream`, `ppp.ActiveStream`, `syslog.LogStream`, `network.InterfaceTrafficStream`, `network.InterfaceStatsStream`, `network.QueueStatsStream` — wrap `dev.Path(...).Print().Follow/FollowOnly/Interval.Stream()` + inherent streaming.
- **Polling**: `system.MonitorResource`, `system.MonitorScheduler`, `hotspot.MonitorActiveCount`, `hotspot.MonitorUserCount` — wrap `dev.RegisterPoll` (atau goroutine ticker untuk count-only yang hanya emit !done).
- **Cache**: `hotspot.ProfileListCached`, `hotspot.ServerListCached`, `network.PoolListCached`, `system.IdentityCached` — wrap `Print().ExecCached(ctx, ttl)`.

Mock `fake.Runner` dihapus — sub-paket terlalu tipis untuk unit-test (logic ada di `domain/` dan `scripts/` yang sudah pure). Test bergeser ke integration test (build tag `integration`).

## Quickstart

```bash
go build ./...
go test ./...                            # unit (domain + scripts + helpers)
go test -tags=integration ./test/...     # ke router asli (butuh .env)

# Jalankan HTTP server (gin REST + SSE)
cp .env.example .env  # isi ROSLIB_ROUTER_* + HTTP_BIND
go run ./cmd/server
# → http://127.0.0.1:8080/healthz, /api/v1/...
```

Endpoint inventory + format SSE: lihat [docs/API.md](docs/API.md).

## Pemakaian

```go
// Setup persistent connection (1 router = 1 login sepanjang umur app).
cfg, _ := config.LoadFromEnv()
mgr, _, _ := roslib.NewFromConfig(ctx, cfg, log)
defer mgr.CloseAll()
dev, _ := mgr.Get(roslib.DefaultDeviceKey)

// Sub-client per resource (sharing async + tag demux).
hot := hotspot.New(dev)
sys := system.New(dev)

// Snapshot
users, _ := hot.UserList(ctx)

// Mutation
id, _ := hot.UserAdd(ctx, hotspot.UserAddArgs{Name: "voucher-001", Profile: "1day"})

// Live streaming hotspot active (snapshot + delta)
hot.ActiveStream("dashboard-active", func(s *roslib.Sentence) {
    log.Printf("active: user=%s addr=%s uptime=%s", s.Get("user"), s.Get("address"), s.Get("uptime"))
})

// Scheduled poll (CPU/RAM/uptime tiap 5s, interval-group batching otomatis)
sys.MonitorResource("dash-cpu", 5*time.Second, func(s *roslib.Sentence) {
    log.Printf("cpu=%s mem=%s uptime=%s", s.Get("cpu-load"), s.Get("free-memory"), s.Get("uptime"))
})

// Cached read (profile dropdown, dll yang jarang berubah)
profiles, _ := hot.ProfileListCached(ctx, 60*time.Second)
```

## Struktur

Lihat [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md). Ringkas:

```
domain/                shared types
mikrotik/              command wrapper per modul (thin, no business logic)
scripts/               RouterOS script generator (string templating)
workflows/             cascade orchestration
api/                   gin REST + SSE (dto, handler, middleware, sse broker)
cmd/server/            entry point HTTP API
internal/config/       HTTP server config loader
internal/rosfmt/       format helpers
internal/tcpmock/      mock TCP server
internal/testutil/     integration test helper
test/integration/      build tag: integration
```

## Cross-reference ke analisis

Tiap exported function di `mikrotik/` & `scripts/` punya doc comment yang menyebut section di [`../mikhmonv3-analisis.md`](../mikhmonv3-analisis.md), mis. `// UserAdd → §1.6`. Lihat juga [docs/COMMANDS.md](docs/COMMANDS.md), [docs/SCRIPTS.md](docs/SCRIPTS.md), [docs/WORKFLOWS.md](docs/WORKFLOWS.md).

## Konvensi

Lihat [AGENTS.md](AGENTS.md) — dipakai oleh AI agent (Cascade/Claude) untuk konsistensi diff.

## Lisensi

MIT, mengikuti `roslib`.
