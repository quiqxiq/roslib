# Testing Guide

Tiga lapis test:

## 1. Unit Test (default)

Pakai `mikrotik/fake.Runner` — in-memory, deterministik, cepat.

```bash
go test ./...
go test -race ./...
go test -cover ./...
```

Tiap test mengikuti pola:

```go
r := fake.New()
r.On("/path/print", "?name=foo").Reply(fake.Row{".id": "*1", "name": "foo"})
c := system.New(r)

got, err := c.SomeOp(ctx, "foo")
require.NoError(t, err)
assert.Equal(t, expected, got)
assert.Equal(t, []string{"/path/print", "?name=foo"}, r.LastCall())
```

`fake.Runner` API:

| Metode | Tujuan |
|---|---|
| `On(words...)` | daftar stub baru (prefix match) |
| `.Reply(rows...)` | set Rows reply |
| `.ReplyDone()` | set !done kosong (untuk add/set/remove) |
| `.ReplyDoneWith(row)` | set !done dengan field (mis. `=ret=*ID`) |
| `.ReplyError(err)` | return error saat matched |
| `.Times(n)` | batasi berapa kali stub bisa matched |
| `Default(reply)` | fallback kalau tidak ada match |
| `Calls()` | snapshot semua sentence yang masuk |
| `LastCall()` | sentence terakhir |

## 2. Hybrid Test (TCP mock)

Pakai `internal/tcpmock` untuk verifikasi encoding length-prefix word.

```go
srv, _ := tcpmock.Start()
defer srv.Close()
srv.Script([]string{"!done", "=ret=*42"})
// dial srv.Addr(), kirim sentence, verify srv.Received()
```

Tipikal: tes karakter spesial (`"`, `\`, `$`, multibyte) di sentence value.

## 3. Integration Test (ke router asli)

Build tag: `integration`. Skip otomatis kalau `ROSLIB_ROUTER_ADDRESS` kosong.

```bash
cp .env.example .env
# isi ROSLIB_ROUTER_ADDRESS, USERNAME, PASSWORD
export $(grep -v '^#' .env | xargs)
go test -tags=integration -count=1 ./test/integration/...
```

Konvensi resource yang dibuat:

- Nama pakai `testutil.UniqueName(t, "suffix")` → `mikhmon-it-<test>-<rand>`.
- Selalu defer cleanup via `t.Cleanup(func(){ _ = client.XxxRemove(ctx, id) })`.

`-count=1` mencegah cache hasil — penting untuk integration test.

## Golden File Update (scripts/)

Generator script (`scripts/onlogin`, `scripts/onevent`) di-test dengan golden file.

```bash
# regenerate file di scripts/<modul>/testdata/golden/
go test ./scripts/onlogin ./scripts/onevent -update

# review diff lalu commit:
git diff scripts/*/testdata/
git add scripts/*/testdata/ && git commit
```

Test tanpa `-update` akan FAIL kalau output beda dari golden file.

## Coverage Target

- Unit test: ≥ 80% pada `mikrotik/`, `scripts/`, `workflows/`.
- Integration test: smoke per modul + workflow cascade end-to-end.

```bash
make test-cover
```
