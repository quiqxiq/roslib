# Workflows Mapping

Peta dari analisis §4 (Alur Proses Lengkap) ke fungsi Go di paket `workflows/`.

## §4.1 Delete User — Cascade Cleanup

```go
import "github.com/quiqxiq/roslib-mikhmon/workflows"

wf := workflows.New(runner)
err := workflows.DeleteUser(ctx, wf, userID)
```

Step yang dijalankan (mengikuti analisis):

1. `/ip/hotspot/user/print ?.id=$uid` → dapat nama
2. `/system/script/print ?name=$name` → list script transaksi
3. `/system/scheduler/print ?name=$name` → list scheduler
4. `/system/script/remove .id` (loop)
5. `/system/scheduler/remove .id` (loop)
6. `/ip/hotspot/user/remove .id`

User yang sudah hilang sebelum step 1 → step 2-5 di-skip, step 6 tetap dicoba (idempotent).

### Bulk delete (separator `~`)

```go
err := workflows.BulkDeleteUsersFromString(ctx, wf, "*1~*2~*3")
// atau:
ids := workflows.ParseBulkIDs("*1~*2~*3")
err := workflows.BulkDeleteUsers(ctx, wf, ids)
```

Tidak fail-fast — kembalikan `*BulkDeleteUsersErr` yang berisi map ID → error.

## §4.2 Delete IP Binding — 9 Step Cascade

```go
err := workflows.DeleteBinding(ctx, wf, bindingID)
// atau jika hanya tahu MAC:
err := workflows.DeleteBindingByMAC(ctx, wf, "AA:BB:CC:DD:EE:FF")
```

Step:

1. lookup binding by ID → dapat MAC
2. `/ip/hotspot/ip-binding/remove`
3. `/queue/simple/print ?name=$mac`
4. `/queue/simple/remove .id` (loop)
5. `/system/scheduler/print ?name=$mac`
6. `/system/scheduler/remove .id` (loop)
7. `/ip/arp/print ?mac-address=$mac`
8. `/ip/arp/remove .id` (loop)
9. `/ip/dhcp-server/lease/print ?mac-address=$mac` + `/remove`

## §4.3 Delete User Profile — 3 Step Cascade

```go
err := workflows.DeleteProfile(ctx, wf, profileID, profileName)
```

Step:

1. `/system/scheduler/print ?name=<profileName>` → cari monitor scheduler
2. `/ip/hotspot/user/profile/remove .id`
3. `/system/scheduler/remove .id`

`profileName` boleh kosong (skip step 1+3).

## §4.4 Kick User Active — 4 Step Cascade

```go
err := workflows.KickActive(ctx, wf, activeID)
```

Step:

1. `/ip/hotspot/active/print ?.id=$uid` → dapat nama user
2. `/ip/hotspot/cookie/print ?user=$name`
3. `/ip/hotspot/cookie/remove .id` (loop)
4. `/ip/hotspot/active/remove .id`

## Catatan Konsistensi

RouterOS API tidak transactional. Workflow adalah **fail-fast** — kalau step ke-N gagal, step 1..N-1 yang sudah eksekusi tidak di-rollback. Caller bertanggung jawab retry / manual cleanup.

Untuk operasi yang sensitif (mis. ProfileRemove), workflow tetap mencoba menghapus scheduler terkait meskipun ProfileRemove gagal — supaya tidak ada orphan scheduler kalau profile sudah ke-delete oleh operator manual.
