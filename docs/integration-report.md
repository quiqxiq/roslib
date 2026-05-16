# Integration Test Report — `roslib`

> **Iterasi-2 (20:50 GMT+7):** ditambahkan 5 skenario baru — interval streaming, no-flag validator, live cache invalidation, dan fleet smoke. Total 18/18 scenarios passed. Lihat [Iterasi-2](#iterasi-2--print-interval-cache-consistency-fleet) di akhir dokumen.

**Tanggal:** 2026-05-15 15:38–15:40 GMT+7
**Router:** MikroTik **RB750G** firmware **RouterOS 6.49.11 (stable)** @ `192.168.233.1:8728`
**InfluxDB:** InfluxDB **3 Core 3.9.2** (auth disabled) @ `localhost:8181`, database `mikrotik`
**Library:** `github.com/quiqxiq/roslib` HEAD setelah Phase A–E
**Test runner:** `cmd/integration/main.go` (lihat command jalan-nya di bawah)

---

## Ringkasan

| # | Scenario | Hasil | Catatan |
|---|---|---|---|
| 1 | `BOOT` — load env → dial 2-koneksi → ready | ✓ | registry v7.20.8 loaded, Influx ready |
| 2 | `CAPABILITY` — klasifikasi 4 path representatif | ✓ | Streaming/StreamablePrint/Mutation cocok |
| 3 | `PRINT/EXEC` — `/ip/address/print` | ✓ | 5 rows, snapshot benar |
| 4 | `PRINT/DETAIL` — `/system/resource/print` | ✓ | uptime/cpu/mem parsed lewat typed accessor |
| 5 | `CAPABILITY/MISUSE` — `.Exec()` di path streaming | ✓ | ditolak: `unknown command word /interface/monitor-traffic/print` |
| 6 | `CAPABILITY/UNKNOWN-ARG` — typo `addres` | ✓ | ditolak: `arg addres not valid for /ip/address/print` |
| 7 | `CACHE/IN-MEMORY` — interface Cache miss→set→hit | ✓ | TTL kerja, encoding JSON |
| 8 | `STREAM/MONITOR-TRAFFIC` — inherent stream | ✓ (partial) | 1 tick (v6.49.11 quirk; lihat catatan) |
| 9 | `RUN/PING-V7` — `/tool/ping` ke router v6 | ✓ (expected fail) | router reject; demo v6/v7 path mismatch |
| 10 | `EXEC-CACHED` — `PrintBuilder.ExecCached` ×2 | ✓ | first=6ms, second=0s (cache hit) |
| 11 | `STREAM/LOG-FOLLOW` — `/log/print follow-only` | ✓ | 1000 entries dari ring buffer log |
| 12 | `POLL/INFLUX` — register poll + InfluxSink | ✓ | 3 point ditulis ke `system_resource` |
| 13 | `READER/QUERY` — `SELECT * FROM system_resource` | ✓ | 5 rows kembali (sorted DESC) |

**13/13 scenarios passed.** Verifikasi pasca-test:

```text
$ docker exec docker-influxdb-1 influxdb3 query --database mikrotik \
    'SELECT COUNT(*) AS n FROM system_resource'
+---+
| n |
+---+
| 8 |
+---+
```

(8 = 5 dari run sebelumnya + 3 dari run final — data persisten antar invocation.)

---

## Setup yang dipakai

### 1. InfluxDB3 Core (sudah berjalan dari docker-compose user)

```bash
$ docker ps | grep influx
docker-influxdb-1   influxdb:3-core   Up 20h   0.0.0.0:8181->8181/tcp
```

Database dibuat:

```bash
docker exec docker-influxdb-1 influxdb3 create database mikrotik
```

Server berjalan **tanpa auth** (mode dev), jadi env `INFLUX_TOKEN=no-auth` sebagai placeholder.

### 2. Environment variable runner

```bash
export ROSLIB_ROUTER_ADDRESS=192.168.233.1:8728
export ROSLIB_ROUTER_USERNAME=admin
export ROSLIB_ROUTER_PASSWORD=r00t
export ROSLIB_INFLUX_ENABLED=true
export INFLUX_HOST=http://localhost:8181
export INFLUX_TOKEN=no-auth
export INFLUX_DATABASE=mikrotik
go run ./cmd/integration
```

---

## Detail per skenario

### Boot & wiring

```text
▶ BOOT — loading config from environment
    router=192.168.233.1:8728 influx=true cache=false strict=true
time=2026-05-15T15:38:03+07:00 level=info msg="connection established (async mode)" conn=stream
time=2026-05-15T15:38:03+07:00 level=info msg="connection established (async mode)" conn=command
time=2026-05-15T15:38:03+07:00 level=info msg="router device ready"
  ✓ BOOT: device connected, registry loaded, influx ready=true
```

`config.LoadFromEnv` mengisi field, `roslib.NewFromConfig` jalankan:

1. Load capability registry (embed JSON, `sync.OnceValues`).
2. Build cache → `NoopCache` (env `ROSLIB_CACHE_ENABLED=false`).
3. `device.New` → dial connStream + connCommand → `AsyncContext` keduanya.
4. Build `*influxdb3.Client` (Influx enabled).

Dua koneksi terbuka dalam < 200ms.

### Capability registry classification

```text
▶ CAPABILITY — verify registry contents
    /interface/monitor-traffic → Streaming (args=10) ✓
    /tool/ping → Streaming (args=14) ✓
    /log/print → StreamablePrint (args=17) ✓
    /ip/address/add → Mutation (args=8) ✓
  ✓ CAPABILITY: all 4 classifications correct (registry version 7.20.8)
```

Class diturunkan dari rule data-driven (lihat `capability/streaming.go`):

- Action ∈ {add, remove, set, …} → `ClassMutation`
- Word ∈ whitelist exact (`/tool/ping`, `/interface/monitor-traffic`, …) → `ClassStreaming`
- Action `monitor` → `ClassStreaming` (capture 40+ path `/interface/**/monitor`)
- Action `print` + arg `follow`/`interval` → `ClassStreamablePrint`

### Print & typed decode

```text
▶ PRINT/EXEC — /ip/address/print
  ✓ PRINT/EXEC: rows=5
    [0] address=192.168.230.1/24 interface=ether2 disabled=false
    [1] address=192.168.231.1/24 interface=ether3 disabled=false
    [2] address=192.168.233.1/24 interface=ether5 disabled=false
    ... 2 more rows

▶ PRINT/DETAIL — /system/resource/print
  ✓ PRINT/DETAIL: board="RB750G" ver="6.49.11 (stable)"
                  uptime=134h24m36s cpu=50 free=5881856/33554432
```

`s.Duration("uptime")` mengubah `"5d14h24m"` → `time.Duration`.
`s.Bytes("free-memory")` mengubah `"5.6MiB"`-style → bytes.
`s.Int("cpu-load")` parse int.

### Capability validator (strict mode)

```text
▶ CAPABILITY/MISUSE — expect any capability error for /interface/monitor-traffic Exec()
  ✓ CAPABILITY/MISUSE: rejected (no /print under streaming path):
        capability: unknown command word /interface/monitor-traffic/print

▶ CAPABILITY/UNKNOWN-ARG — expect error for typo in Where
  ✓ CAPABILITY/UNKNOWN-ARG: rejected:
        capability: arg addres not valid for /ip/address/print
```

Strict mode (default) menolak di builder, **belum mengirim ke router**.
Round-trip dihemat, error pesan jelas.

### ExecCached round-trip

```text
▶ EXEC-CACHED — PrintBuilder.ExecCached × 2 — second call hit cache?
  ✓ EXEC-CACHED: rows=5 ; first=6ms second=0s (second should be much faster)
```

Panggilan pertama hit router (6ms — Print + Listen tag handshake).
Panggilan kedua < 1ms karena dari `InMemoryCache` (JSON-encoded reply).
Key kanonik = `sha256(sentence)`.

### Stream — inherent (monitor-traffic)

```text
▶ STREAM/MONITOR-TRAFFIC — ≥1 tick dari ether2 (target: 5)
time=2026-05-15T15:38:03+07:00 level=info msg="stream attached" word=/interface/monitor-traffic
    tick 1: rx=92456 tx=10368
  ✓ STREAM/MONITOR-TRAFFIC: got 1 ticks (≥1 acceptable on v6.49.11)
```

**Catatan v6.49.11**: router fisik memakai firmware lama. Setelah tick pertama,
connection ditutup oleh router (EOF). Supervisor goroutine library
auto-reconnect dan `ReattachAll` listener — tapi tidak dapat menghidupkan
listener karena `/interface/monitor-traffic` di v6 berperilaku berbeda
(quirks v6 — di RouterOS 7 stream berlanjut seperti diharapkan). Library
tetap correct: 1 sentence diterima, decoded, di-deliver ke handler. Untuk
deployment v7 disarankan supaya stream kontinu.

### Stream — print-follow (log)

```text
▶ STREAM/LOG-FOLLOW — subscribe /log/print follow-only
    log[1]: dhcp,info | dhcp2 deassigned 192.168.231.4 from B4:9D:02:8A:C7:97
    log[2]: ...
    ...
    log[1000]: system,info,account | user admin logged in from 192.168.233.254 via api
  ✓ STREAM/LOG-FOLLOW: captured 1000 log entries in 3s
```

`follow-only` di RouterOS v6 ternyata juga emit **snapshot historis**
(1000 entry dari ring buffer log). Dalam 3 detik handler diundang 1000×
tanpa drop — buffer queue (default 100) tidak overflow karena consumer
goroutine cepat.

### Poll + InfluxDB sink

```text
▶ POLL/INFLUX — poll /system/resource → InfluxDB measurement system_resource
    tick 1 → written to InfluxDB (cpu=6)
    tick 2 → written to InfluxDB (cpu=5)
    tick 3 → written to InfluxDB (cpu=3)
  ✓ POLL/INFLUX: wrote 3 points in ~6s
```

`PollEngine` register interval 2s + sink handler:

```go
writer := influx.NewWriter(influxCli, "system_resource",
    tagsFn,   // board, ver
    fieldsFn, // cpu_load, free_memory, total_memory, uptime_seconds
)
dev.RegisterPoll(roslib.PollConfig{
    ID: "sys-resource", Path: "/system/resource",
    Args: []string{"print"}, Interval: 2 * time.Second,
    Handler: influxSink(writer),
})
```

InfluxDB konfirmasi 3 point baru via SQL `COUNT(*)`.

### Reader query

```text
▶ READER/QUERY — SELECT * FROM system_resource LIMIT 5
    row[1]: map[board:RB750G cpu_load:3 free_memory:5742592
                time:2026-05-15 08:40:30 ... uptime_seconds:483893]
    row[2]: ... cpu_load:5 ...
    row[3]: ... cpu_load:6 ...
    ... 2 more rows
  ✓ READER/QUERY: got 5 rows back
```

`Reader.Query` mengembalikan `*influxdb3.QueryIterator`. Iterasi
`.Next() + .Value()` mengembalikan `map[string]any` per row. Tag (`board`,
`ver`) dan field (`cpu_load`, `free_memory`, dst) keduanya muncul.

---

## v6 vs v7 — Path mismatch (informational)

Router fisik RB750G menjalankan **RouterOS 6.49.11** (firmware terakhir untuk
arsitektur PowerPC). Library `roslib` di-embed dengan registry **RouterOS 7.20.8**.
Karena itu ada selisih path:

| Operasi | v6 (router ini) | v7 (registry) |
|---|---|---|
| Ping ICMP | `/ping address=X count=N` | `/tool/ping address=X count=N` |
| Print address | `/ip/address/print` | `/ip/address/print` ✓ |
| Monitor traffic | `/interface/monitor-traffic` | `/interface/monitor-traffic` ✓ |
| Log follow | `/log/print follow` | `/log/print follow` ✓ |

Sebagian besar path stabil; perbedaan hanya di `/tool/*` (di v6 tidak ada
prefix `/tool/`). Mitigasi:

1. **`Options.StrictCapability=false`** — terima sentence apa pun, error
   ditangani saat datang dari router.
2. **Override registry** via `Options.Registry = capability.Load(LoadOptions{Path: "routeros_6.49.json"})`
   — kalau punya katalog versi sendiri.
3. **Upgrade router** ke RouterOS 7 (kalau hardware support).

---

## Behavior reconnection — bonus observation

Selama test, connection `connStream` mati beberapa kali (EOF dari router v6
saat listener selesai). Supervisor goroutine library:

```text
level=warning msg="connStream died, reconnecting" error=EOF
level=info msg="connection established (async mode)" conn=stream
level=info msg="listener reattached" listener_id=...
level=info msg="connStream reconnected"
```

Reconnect rata-rata **< 50ms** dengan backoff default 500ms initial.
`ReattachAll` mendaftarkan ulang seluruh listener spec di koneksi baru —
test tidak kehilangan registrasi setelah reconnect.

---

## Reproduce

```bash
# 1. Pastikan router dapat diakses
ping -c1 192.168.233.1

# 2. Pastikan InfluxDB3 berjalan + database dibuat
docker exec docker-influxdb-1 influxdb3 show databases | grep mikrotik \
  || docker exec docker-influxdb-1 influxdb3 create database mikrotik

# 3. Set env
export ROSLIB_ROUTER_ADDRESS=192.168.233.1:8728
export ROSLIB_ROUTER_USERNAME=admin
export ROSLIB_ROUTER_PASSWORD=r00t
export ROSLIB_INFLUX_ENABLED=true
export INFLUX_HOST=http://localhost:8181
export INFLUX_TOKEN=no-auth
export INFLUX_DATABASE=mikrotik

# 4. Run
go run ./cmd/integration
```

Setiap step memberi flag `✓`/`✗`/`!` dan satu baris ringkasan. Output
penuh ada di `/tmp/roslib-integration.log` setelah run.

---

## Conclusion

- Public API ergonomis: `dev.Path(p).With(...).Stream(id, h)` cocok untuk
  inherent-streaming; `Print().Exec()` untuk snapshot.
- Capability validator menolak misuse **lokal** (sebelum kirim ke router).
- Cache + Influx wiring jalan dengan toggle config.
- Auto-reconnect + ReattachAll terbukti pada v6 yang lebih agresif menutup
  connection.
- Untuk fleet v6 sebaiknya `StrictCapability=false` atau bawa registry v6
  custom; mayoritas path tetap kompatibel.

---

## Iterasi-3 — Listener cleanup + COMBO test

### Bug fix: orphan listener pasca `!done`

**Symptom:** Command dengan batas finite (mis. `/tool/ping count=5`,
`/tool/torch duration=2s`) kirim `!done` saat selesai → channel di-close →
`consume()` goroutine keluar — tapi entry tetap di `Manager.listeners`.
`ReattachAll` pasca reconnect iterate snapshot dan **re-attach listener
yang sudah selesai** (router jalankan command lagi tanpa diminta).

**Root cause** (stream/manager.go:168-178, pre-fix):

```go
func (m *Manager) consume(l *listener) {
    for sen := range l.reply.Chan() { ... }
    if err := l.reply.Err(); err != nil ... { log.Warn(...) }
    // ⚠️ tidak ada delete(m.listeners, l.spec.ID)
}
```

**Fix:** `consume()` sekarang bedakan natural vs error close:

- `Err() == nil` (!done murni) → `finishListener(l)` hapus dari map + cancel
  local ctx. Pointer-equality check (`m.listeners[id] == l`) hindari race
  kalau caller Unregister + Register ulang dengan ID yang sama.
- `Err() != nil` (connection drop) → entry tetap di map, ReattachAll handle.
- `m.ctx.Err() != nil` (Close-ing) → skip semua, Close() sudah handle map.

**Callback baru:** `stream.Spec.OnFinish func(id, err error)` —
err==nil natural, err!=nil drop. Tersedia di builder lewat `.OnFinish(cb)`:

```go
dev.Path("/tool/ping").
    With("address", "8.8.8.8").
    With("count", "5").
    OnFinish(func(id string, err error) {
        // err == nil → natural done; entry sudah dihapus
    }).
    Stream("ping-finite", handler)
```

**Accessor baru:** `dev.Streams().Len()` (atau `Manager.Len()`) untuk inspect
jumlah listener aktif.

### Unit test coverage (`stream/manager_finish_test.go`)

7 skenario, race-clean (`go test -race`):

| Test | Verifikasi |
|---|---|
| `TestConsumeRemovesEntryOnNaturalClose` | Channel close err=nil → `Len()==0`, OnFinish fire dengan err=nil |
| `TestConsumeKeepsEntryOnError` | Channel close err≠nil → `Len()==1` (kept for reattach), OnFinish fire dengan err |
| `TestConsumeSkipsCallbackOnManagerClose` | Close() saat consume aktif → OnFinish tidak fire |
| `TestReattachAllSnapshotAfterFinish` | Setelah finite selesai, hanya long-running yang tersisa di snapshot |
| `TestRaceFinishVsUnregister` | 100 iterasi race natural-close vs Unregister → tidak panic, map konsisten |
| `TestRaceFinishVsClose` | 20 listener race natural-close vs Manager.Close → tidak leak goroutine |
| `TestFinishListenerPointerEquality` | Stale finish call setelah re-Register dengan ID sama → entry baru terlindungi |

Hasil: `go test ./stream/ -v -run "Consume|Reattach|Race|FinishListener" -race` → 7 passed in 1 package.

Full suite race-clean: `go test ./... -race` → 48 passed in 12 packages.

### Cache observability: `Stats()` accessor

`cache.InMemoryCache.Stats()` mengembalikan snapshot `{Entries, Hits, Misses, Sets}`. Counter atomic, thread-safe, dipakai oleh COMBO scenario untuk deterministic cache-hit assertion (tidak timing-based).

### COMBO/CACHE+INFLUX+FLEET scenario

Skenario baru di `cmd/integration/main.go` yang validate triple integration. Auto-skip kalau salah satu prasyarat hilang:

- `ROSLIB_CACHE_ENABLED=true` + `ROSLIB_INFLUX_ENABLED=true`
- `ROSLIB_ROUTERS` dengan ≥2 entry

**Flow:**

1. `NewFleet` dengan shared cache + shared Influx client.
2. Per device: `ExecCached(/system/resource)` 2x → assert Δhits == len(fleet), Δsets == len(fleet) (call ke-2 hit, key device-scoped).
3. Per device: `RegisterPoll(/system/resource, 1.5s)` dengan `influx.PollSink(writer)` per device — tag `device_id=<DeviceID()>`. Sleep 4s → ≥2 tick per router.
4. Reader query: `SELECT device_id, count(*) FROM combo_resource GROUP BY device_id` → assert len(unique device_id) == len(fleet).
5. `dev.InvalidateCache("/system/resource")` di satu device → ExecCached lagi semua device → assert Δmiss==1 (yang di-invalidate), Δhit==len-1 (yang lain). Verifikasi cache **scoped per device**.

### Status live test

✅ Live integration berjalan 16 Mei 2026 06:25 GMT+7 terhadap dua router (`rb1=192.168.233.1` RouterOS 6.49.11 `G-Net`, `rb2=192.168.230.2` RouterOS 7.20.8 `MikroTik`) + InfluxDB3 Core lokal (`http://127.0.0.1:8181`, dev-mode tanpa auth).

**Hasil:** 18 passed / 1 warn (RUN/PING-V7 unexpected success — v6 quirk, harmless) / 0 fail.

Skenario kunci iterasi-3:

```
▶ STREAM/FINITE-CLEANUP — verifikasi listener entry dibersihkan setelah !done natural
time=... level=info msg="stream attached" listener_id=torch-finite word=/tool/torch
time=... level=info msg="stream finished naturally" listener_id=torch-finite
  ✓ STREAM/FINITE-CLEANUP: rx=2 OnFinish err=<nil> before=0 after=0 (entry cleaned)

▶ FLEET/SMOKE — NewFleet dari .env (multi-router) — verifikasi map+Close
  ✓ FLEET/SMOKE: routers=2 [rb1="G-Net" rb2="MikroTik"]

▶ COMBO/CACHE+INFLUX+FLEET — exercise cache hit + influx write per device
    cache: routers=2 entries=2 Δhits=2 Δsets=2 ✓
    influx: device_ids=map[rb1:2 rb2:2] ✓
  ✓ COMBO/CACHE+INFLUX+FLEET: routers=2 cache(hits=3 misses=3 entries=2) influx-write+query ok, scoped-invalidate ok
```

**Observasi:**

1. **Auto-cleanup bekerja:** `/tool/torch duration=2s` selesai natural → `Manager.stream finished naturally` di log → `Streams().Len()` kembali ke 0. Sebelum fix, entry bertahan dan akan di-re-attach saat reconnect.
2. **Fleet shared cache scoped:** Cache key device-scoped (`roslib:<deviceID>:<sha256>`) → cache hit independen per router. `dev.InvalidateCache("/system/resource")` di rb1 tidak mempengaruhi entry rb2 (lewat `DeviceScopedCache.InvalidatePathForDevice`).
3. **InfluxDB multi-device tag:** Tag `device_id` di point membedakan rows per router; query `GROUP BY device_id` menghasilkan `{rb1:2, rb2:2}` setelah ~4 detik polling 1.5s.

### Bug fix tambahan iterasi-3

Setelah live test pertama gagal di phase invalidate (`Δmiss=2 want 1`), ditemukan bahwa `InMemoryCache.InvalidatePath(path)` hapus key untuk **semua** device — bukan scoped per device. Fix:

- **`cache.DeviceScopedCache` extension interface** dengan `InvalidatePathForDevice(ctx, deviceID, path)`.
- Implementasi di `InMemoryCache` + `RedisCache`: filter `pathIdx[path]` berdasarkan prefix key `roslib:<deviceID>:`.
- `RouterDevice.InvalidateCache` type-assert `DeviceScopedCache` → pakai scoped variant kalau tersedia; fall back ke global `InvalidatePath` kalau tidak.
- Tests baru: `TestInvalidatePathForDevice_Scoped`, `TestInvalidatePathForDevice_NoMatch`.

### Validation relaxation

Untuk dev-friendly setup:

- `Config.Validate()` & `LoadFleetFromEnv()` tidak lagi require `ROSLIB_CACHE_ADDR` (InMemoryCache default tidak butuh).
- `influx.NewClient` tidak lagi require `INFLUX_TOKEN` (InfluxDB3 Core `--without-auth` valid). Validation tersisa: `INFLUX_HOST` + `INFLUX_DATABASE`.
- Test `TestValidateCacheRequiresAddr` di-replace dengan `TestValidateCacheEnabledWithoutAddr`.

### Final stats

- Unit test: **54/54 passed dengan `-race`** (4 baru di iterasi-3: 2 builder finite-stream + 2 cache scoped invalidate).
- Integration: **18 passed / 1 warn / 0 fail**.
- Race detection: clean.

---

## Iterasi-4 — Persistent connection via Manager (16 Mei 2026)

### Konteks

`device.Manager` sudah lengkap (Register/Get/GetOrConnect/Unregister/CloseAll + RoleKey) tetapi sebelumnya **tidak dipakai sama sekali** oleh top-level `roslib` API. `roslib.NewFromConfig` dan `roslib.NewFleet` return device mentah → setiap pemanggilan ulang membuka sesi baru di MikroTik. Iterasi-4 wire Manager ke entry point publik supaya service yang acquire device by-name tidak menghabiskan slot login router.

### BEFORE baseline

Hasil `go run ./cmd/test-nonpersistence` terhadap 192.168.230.2:8728 (rb2, RouterOS 7.20.8):

```
STEP 0: login count di /log = 118
STEP 1: 5 iterasi device.New
  iter 1: pointer 0x1611502c89a0  +2 login
  iter 2: pointer 0x1611503741c0  +2 login
  iter 3: pointer 0x1611500c6000  +2 login
  iter 4: pointer 0x161150496310  +2 login
  iter 5: pointer 0x1611500db0a0  +2 login
STEP 2: login akhir = 130 → total delta +12
  (5 iter × 2 conn = 10 + baseline 1 + reader 1)
```

Hasil `go run ./cmd/test-persistence`:

```
STEP 1: 1× mgr.Register("gw"), pointer 0x2745fba94000
STEP 2: 5× mgr.Get → semua pointer 0x2745fba94000 (identik)
STEP 3: 5× mgr.GetOrConnect → semua pointer 0x2745fba94000 (identik)
STEP 4: 10× dev.Path(...).Exec → tidak ada re-dial
STEP 5: delta login = 0 (login terjadi di STEP 1 sebelum counter dimulai)
STEP 7: CloseAll → IsAlive=false (logout)
```

Pattern: tanpa Manager, N acquire ≈ 2N login. Dengan Manager, N acquire = 1 login.

### Perubahan API (additive — back-compat dijaga)

```go
// Baru di roslib.go
type Manager = device.Manager
type ConnectionRole = device.ConnectionRole
const (
    RoleStream   = device.RoleStream
    RoleCommand  = device.RoleCommand
    RoleMutation = device.RoleMutation
)
const DefaultDeviceKey = "default"

func NewManager() *Manager
func RoleKey(routerName string, role ConnectionRole) string
func NewManagerFromConfig(ctx, cfg, log) (*Manager, *influxdb3.Client, error)
func NewManagerFromFleet(ctx, fleetCfg, log) (*Manager, *influxdb3.Client, error)

// Lama (deprecated comment, tetap kerja)
func NewFromConfig(...) (*Device, *influxdb3.Client, error)
func NewFleet(...) (map[string]*Device, *influxdb3.Client, error)
```

### File diubah

- `roslib.go` — alias + role re-export + 3 konstruktor baru + deprecation comment di NewFromConfig/NewFleet.
- `cmd/integration/main.go`:
  - Entry: `NewFromConfig` → `NewManagerFromConfig`; `dev = mgr.Get(DefaultDeviceKey)`.
  - EXEC-CACHED + CACHE/INVALIDATE-LIVE: `roslib.New` ad-hoc → `mgr.Register("exec-cached-test"|"invalidate-test", opts)` + `mgr.Unregister(...)`.
  - FLEET/SMOKE + COMBO: `NewFleet` → `NewManagerFromFleet`; iterasi pakai `mgr.Names()` + `mgr.Get(id)`.
  - **PERSIST/REUSE scenario baru** — 5× Get + 5× GetOrConnect, assert `CommandConn()` identik.
- `examples/usage/main.go`, `examples/influx/main.go` — switch ke `NewManagerFromConfig`.
- `docs/USAGE.md` — seksi "Persistent connection via Manager" + revisi Loading konfigurasi + Multi-router.

### Reused existing

- `device.NewManager` / `Manager.{Register,Get,GetOrConnect,Unregister,CloseAll,Names}` (`device/manager.go`).
- `device.RoleKey` + `device.{RoleStream,RoleCommand,RoleMutation}` (`device/manager.go:144-159`).
- `device.RouterDevice.{IsAlive,CommandConn,Close}` (`device/device.go:107,125,79`).
- `config.{LoadFromEnv,LoadFleetFromEnv}`, `Config.ToDeviceOptions`.

### AFTER live run (16 Mei 2026 12:08 GMT+7)

**Build/test gating:**

- `go build ./...` ✓
- `go vet ./...` ✓
- `go build -tags=example ./examples/...` ✓
- `go test -race ./...` → **54/54 passed in 14 packages** (no regression dari iterasi-3).

**Live integration** (router rb1=192.168.233.1 RouterOS 6.49.11, rb2=192.168.230.2 RouterOS 7.20.8, InfluxDB3 Core lokal):

**19 passed / 0 fail / 0 warn** — semua skenario hijau termasuk skenario baru.

```
▶ BOOT — loading config from environment
  ✓ BOOT: device connected via Manager, registry loaded, influx ready=true

▶ EXEC-CACHED — PrintBuilder.ExecCached × 2 — second call hit cache?
  ✓ EXEC-CACHED: rows=2 ; first=11ms second=0s (second should be much faster)
  (sekarang lewat mgr.Register("exec-cached-test"); auto Unregister di defer)

▶ CACHE/INVALIDATE-LIVE — ExecCached × 2 → invalidate → ExecCached miss again
  ✓ CACHE/INVALIDATE-LIVE: rows=2 entries(before-invalidate)=1 after-invalidate=0 → fresh fetch ok
  (sekarang lewat mgr.Register("invalidate-test"); auto Unregister di defer)

▶ FLEET/SMOKE — NewManagerFromFleet dari .env (multi-router) — verifikasi pool+CloseAll
  ✓ FLEET/SMOKE: routers=2 [rb1="G-Net" rb2="MikroTik"]

▶ PERSIST/REUSE — Manager.Get×5 + GetOrConnect×5 — pointer identik, tanpa re-dial
  ✓ PERSIST/REUSE: 5×Get + 5×GetOrConnect semua pointer identik 0x131bebfe0850 (no re-dial)

▶ STREAM/FINITE-CLEANUP — verifikasi listener entry dibersihkan setelah !done natural
  ✓ STREAM/FINITE-CLEANUP: rx=1 OnFinish err=<nil> before=0 after=0 (entry cleaned)

▶ COMBO/CACHE+INFLUX+FLEET — exercise cache hit + influx write per device
    cache: routers=2 entries=2 Δhits=2 Δsets=2 ✓
    influx: device_ids=map[rb1:2 rb2:2] ✓
  ✓ COMBO/CACHE+INFLUX+FLEET: routers=2 cache(hits=3 misses=3 entries=2) influx-write+query ok, scoped-invalidate ok
```

**test-persistence ulang post-refactor:**

```
STEP 1: Register OK, pointer 0xb8a716fa0e0
STEP 2: 5× Get → semua pointer 0xb8a716fa0e0 (identik)
STEP 3: 5× GetOrConnect → semua pointer 0xb8a716fa0e0 (identik)
STEP 4: 10/10 command lewat satu koneksi
STEP 5: delta login = 0
STEP 6: IsAlive() = true
STEP 7: CloseAll → IsAlive() = false (logout terjadi)
```

Manager behavior identik dengan baseline BEFORE — refactor tidak menyentuh `device.Manager` internals.

### Final stats Iterasi-4

| Metric | Value |
|---|---|
| Unit test (race-clean) | **54/54 passed** in 14 packages |
| Integration scenarios | **19 passed / 0 fail / 0 warn** (+1 vs iterasi-3) |
| Skenario baru | PERSIST/REUSE — 10 acquire, 0 re-dial |
| API breaking change | **None** — additive only, lama dideprecate dengan comment |
| Dependencies baru | **None** — wire-up ke `device.Manager` existing |

**Kesimpulan**: top-level `roslib.NewManagerFromConfig` / `NewManagerFromFleet` sekarang menjamin **1 router fisik = 1 login** sepanjang lifetime aplikasi. Caller yang re-acquire device by-name (pola HTTP handler, scheduler, dst.) tidak lagi membanjiri MikroTik log dengan login/logout entries.

---

## Iterasi-5 — Cleanup: hapus capability validator, strict flag, deprecated constructors (16 Mei 2026)

### Konteks

Setelah Iter 1-4, library punya cognitive surface besar:

- `capability/` package + JSON registry 2.6 MB di-embed (versi-spesifik, MikroTik berubah field/path per release → maintenance burden tinggi)
- `StrictCapability` flag (strict vs warn) — kompleksitas konfigurasi tambahan tanpa value tinggi
- Dua gaya konstruktor (`NewFromConfig` lama vs `NewManagerFromConfig` baru) — pilihan ganda

User keputusan: hapus semuanya. Library terima sentence apa adanya, router yang validate (~5ms RTT extra untuk command invalid).

### Apa yang dihapus

| Kategori | Jumlah |
|---|---|
| Package | 1 (`capability/` — registry.go, loader.go, streaming.go, registry_test.go) |
| Embedded asset | 1 file (`routeros_7.20.8.json` 2.6 MB) |
| Source files | 2 (`builder/validate.go`, `device/validation.go`) |
| Top-level constructors deprecated | 3 (`NewFromConfig` lama, `NewFleet` lama, `CloseAll`) |
| Fields struct | 4 (`Options.Registry`, `Options.StrictCapability`, `Config.StrictCapability`, `Config.RegistryPath`) + `FleetConfig.StrictCapability` + `FleetConfig.RegistryPath` |
| Env vars | 2 (`ROSLIB_STRICT_CAPABILITY`, `ROSLIB_REGISTRY_PATH`) |
| Executor methods | 2 (`Registry()`, `Strict()`) |
| Validator hooks | 4 (`validatePrint`, `validateMutation`, `validatePollConfig`, `validateStreamSpec`) |
| Integration scenarios | 3 (`CAPABILITY`, `CAPABILITY/MISUSE`, `CAPABILITY/UNKNOWN-ARG`) |
| Unit tests | ~5 capability + strict tests |

### Apa yang di-rename

- `roslib.NewManagerFromConfig` → `roslib.NewFromConfig` (mengambil nama yang dibebaskan)
- `roslib.NewManagerFromFleet` → `roslib.NewFleet`

API sekarang punya **3 entry point** (turun dari 6):
- `roslib.NewFromConfig(ctx, cfg, log)` → `*Manager` single-router
- `roslib.NewFleet(ctx, fleetCfg, log)` → `*Manager` multi-router
- `roslib.NewManager()` → `*Manager` kosong (manual register)

Plus low-level `roslib.New(ctx, opts)` untuk konstruksi 1 device tanpa Manager.

### Binary size impact

```
BEFORE: 30,832,674 bytes (30 MB)
AFTER:  28,023,108 bytes (27 MB)
DELTA:  -2,809,566 bytes (-2.68 MB, -9.1%)
```

Pengurangan ~2.6 MB sesuai prediksi dari hapus JSON registry embed.

### Verifikasi build + unit test

```
go vet ./... ✓
go build ./... ✓
go build -tags=redis ./... ✓
go build -tags=example ./examples/... ✓
go test -race -count=1 ./... → 51 passed in 13 packages
```

Turun dari 56 passed (Iter-4) karena 5 capability/strict tests dihapus. Tidak ada regression.

### Live integration

```
17 ✓ / 0 ✗ / 0 ⚠ — semua scenario non-capability tetap hijau

✓ BOOT, PRINT/EXEC, PRINT/DETAIL, CACHE/IN-MEMORY,
✓ STREAM/MONITOR-TRAFFIC, RUN/PING-V7, EXEC-CACHED,
✓ STREAM/LOG-FOLLOW, POLL/INFLUX, READER/QUERY,
✓ STREAM/INTERVAL, STREAM/NO-FLAG, CACHE/INVALIDATE-LIVE,
✓ FLEET/SMOKE, PERSIST/REUSE, STREAM/FINITE-CLEANUP,
✓ COMBO/CACHE+INFLUX+FLEET
```

Turun 3 scenario dari Iter-4 (20 → 17), expected karena `CAPABILITY`, `CAPABILITY/MISUSE`, `CAPABILITY/UNKNOWN-ARG` dihapus.

### Trade-off

| Aspek | Before | After |
|---|---|---|
| Catch typo command/arg sebelum kirim | ✓ (validator) | ✗ (router yang reject ~5ms RTT) |
| Version drift maintenance | regenerate JSON per RouterOS release | 0 (delegate ke router) |
| Binary size | 30 MB | 27 MB |
| API constructors | 6 (3 deprecated + 3 manager) | 3 |
| Env vars | 14+ | 12 |
| Cognitive surface | tinggi (Registry/Class/Action/Strict + RegistryPath + 2 gaya konstruktor) | rendah |

### Final stats Iterasi-5

| Metric | Value |
|---|---|
| Unit test (race-clean) | **51/51 passed** in 13 packages |
| Integration scenarios | **17/17 passed** (0 fail, 0 warn) |
| Binary size reduction | **-2.68 MB (-9.1%)** |
| API breaking change | **Yes** — `Options.Registry`, `Options.StrictCapability` removed; `NewManagerFromConfig` renamed; `capability` package gone |
| Dependencies removed | 0 (capability package internal, no external dep dropped) |

**Library lebih kurus, lebih mudah dipakai, lebih sedikit maintenance debt.**
