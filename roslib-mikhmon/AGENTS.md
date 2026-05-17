# AGENTS.md — Konvensi untuk AI Pair-Programming

Dokumen ini panduan untuk agent (Cascade, Claude Code, dll) yang membantu mengembangkan `roslib-mikhmon`. Tujuannya: diff kecil, konteks minimal, predictable.

## Aturan Wajib

1. **Tidak ada business logic di `mikrotik/`**. Paket ini hanya thin wrapper di atas `*roslib.Device`. Keputusan (kapan cascade, mode mana) ada di `workflows/`.
2. **Tidak ada IO di `domain/` & `scripts/`**. Murni tipe/string templating. Mudah di-test tanpa router.
3. **Setiap exported symbol wajib doc comment** yang menyebut cross-reference ke analisis (mis. `// UserAdd → §1.6`).
4. **Format signature konsisten**: `func (c *Client) <Op>(ctx context.Context, <args>) (<result>, error)`. ctx pertama, error terakhir.
5. **Single file per resource**: mudah grep, mudah konteks AI minimal.
6. **Sub-paket Client `{dev *roslib.Device}`** — konstruktor `New(dev *roslib.Device) *Client`. Tidak ada `mikrotik.CommandRunner` boundary lagi. Semua method panggil `c.dev.Path(...)` chain (snapshot/mutation/stream) atau `c.dev.RegisterPoll(...)` (poll).
7. **Streaming/polling/cache per resource**: file `<resource>_stream.go` untuk wrap `Print().Follow/FollowOnly/Interval.Stream()` atau inherent `Path().Stream()`; file `monitor.go` untuk wrap `RegisterPoll` atau goroutine ticker; suffix `Cached` untuk wrap `Print().ExecCached(ctx, ttl)`.
8. **HTTP handler tipis** — `api/handler/<resource>.go` ekspose `Register(g *gin.RouterGroup)` method. Pattern: extract input dari `*gin.Context` → call sub-client/workflow → wrap envelope via `WriteOK/WriteList/WriteCreated/WriteNoContent/WriteErr/WriteValidationErr`. Tidak ada business logic — itu hidup di `mikrotik/`/`workflows/`.
9. **DTO terpisah dari domain** — `api/dto/<resource>.go`. JSON snake_case, validator tag `binding:"..."` hanya di sini, password & field sensitif tidak di-expose di response struct. Mapper `FromDomain*` & `ToArgs()` di file yang sama.
10. **SSE pakai broker shared** — handler stream pakai `sse.Hub.GetOrCreate(topic, startFn, stopFn)` untuk auto fan-out N client tanpa duplikasi backend stream. `sse.Stream(c, broker)` pump ke writer.

## Naming

| Kategori | Pola | Contoh |
|---|---|---|
| Function command | `<Resource><Verb>` | `UserAdd`, `UserSetExpiry`, `SchedulerByName`, `BindingRemove` |
| Filter alternatif | `<Resource>By<Field>` | `UserByName`, `UserByID`, `ScriptByComment` |
| Count | `<Resource>Count` | `UserCount`, `ActiveCount` |
| Workflow cascade | imperative | `DeleteUser`, `DeleteBinding`, `KickActive` |
| Script builder | `Build<Purpose>` | `BuildExpiryScript`, `BuildExpiryMonitor` |
| Test | `Test<Type>_<Op>_<Scenario>` | `TestClient_UserAdd_Success`, `TestClient_UserAdd_Duplicate` |

## File Layout per Resource

Setiap file di `mikrotik/<modul>/<resource>.go` mengikuti urutan:

```go
package <modul>

// 1. const path
// 2. struct domain (kalau perlu, mostly di-import dari domain/)
// 3. method receiver *Client, urutan: List → ByX → Count → Add → Set → Remove
// 4. helper private (huruf kecil)
```

## Test Convention

- **Unit test hanya untuk `domain/`, `scripts/`, dan `internal/`** — yaitu paket tanpa IO. `mikrotik/` sub-paket terlalu tipis untuk unit-test setelah refactor (semua call delegate ke roslib builder, yang punya unit test sendiri).
- Test sub-paket `mikrotik/` pakai integration test di `test/integration/` dengan build tag `//go:build integration`.
- Test name `TestIntegration_<Modul>_<Op>` atau `TestIntegration_<Modul><Op>Stream` untuk varian streaming.
- Tiap fitur baru di sub-paket `mikrotik/` → minimal 1 integration test happy-path.
- Untuk script generator: golden file di `testdata/golden/<scenario>.txt`. Update via flag `-update`.

## Integration Test

- Build tag: `//go:build integration`.
- Selalu pakai `testutil.RequireIntegration(t)` di awal.
- Resource yang dibuat harus prefix `mikhmon-it-<test-name>-<random>` agar mudah di-cleanup manual.
- Selalu defer cleanup via `t.Cleanup(...)`.

## Workflow Diff yang Diharapkan dari AI

Kalau diminta tambah command baru:

1. Buat file di `mikrotik/<modul>/<resource>.go` dengan signature standar.
2. Tambah method baru ke struct `Client` (atau resource sub-client).
3. Tulis unit test di file `_test.go` sebelahnya.
4. Update `docs/COMMANDS.md` dengan baris baru (mapping analisis → fungsi).
5. Tambah integration smoke test kalau belum ada.

Semua step di atas harus muncul di satu PR.

## Yang Harus Dihindari

- ❌ Helper "utils" generik tanpa konteks.
- ❌ Interface yang hanya satu implementasi.
- ❌ Sentence assembly manual (`"=key=" + val`, `"?key=" + val`) — selalu pakai builder `dev.Path(p).Print().Where(k, v)` / `Add/Set/Remove` atau `roslib.NewPair`.
- ❌ Manual ticker untuk polling kalau bisa pakai `dev.RegisterPoll(...)` (PollEngine punya interval grouping). Manual ticker hanya kalau RouterOS API tidak emit `!re` (mis. count-only yang hanya emit `!done`).
- ❌ Ganti naming function tanpa update doc + test bersamaan.
- ❌ Add dependency baru tanpa diskusi (cek `go.mod` ringan: roslib + logrus + testify saja).
- ❌ Tulis test yang skip kalau env kosong, kecuali untuk `test/integration/`.
