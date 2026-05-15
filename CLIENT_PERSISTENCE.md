# Persistent Connection ke MikroTik

Dokumen ini menjelaskan bagaimana `pkg/mikrotik/client` mempertahankan satu sesi koneksi ke MikroTik selama aplikasi berjalan, sehingga setiap eksekusi command tidak membuat sesi baru.

---

## Mengapa Hanya Ada Satu Login di Log MikroTik

Di log MikroTik, saat aplikasi berjalan hanya terlihat **satu entry login** di awal, dan **satu entry logout** saat aplikasi dihentikan — berapapun banyak command yang dieksekusi di tengahnya. Ini bukan kebetulan, melainkan hasil desain yang disengaja.

---

## Arsitektur Koneksi

```
Manager (manager.go)
  └─ map[name]*Client          ← satu entry per router yang terdaftar
        └─ Client (client.go)
              └─ *routeros.Client   ← SATU TCP connection per router
                    └─ Async Mode (tag multiplexing)
                          ├─ goroutine watchAsync  ← penjaga koneksi
                          └─ goroutine reconnect   ← pemulih koneksi (jika putus)
```

---

## Alur Lifecycle Koneksi

```
App Start
  │
  └─ Manager.Register() / GetOrConnect()
        └─ Client.Connect()
              ├─ dial()              → TCP connect + /login  ← LOG: "logged in"
              ├─ conn.AsyncContext() → aktifkan tag multiplexing
              └─ go watchAsync()     → goroutine penjaga berjalan di background

App Berjalan (semua command reuse c.conn yang sama)
  ├─ Run("/ip/address/print")        → getConn() → reuse conn
  ├─ RunMany([cmd1, cmd2, cmd3])     → getConn() → reuse conn, concurrent via tags
  ├─ ListenArgs("/tool/traffic")     → getConn() → reuse conn, streaming
  └─ ... tidak ada dial atau login baru

Jika Jaringan Terputus (opsional)
  └─ watchAsync() deteksi error → go reconnect()
        └─ dial() ulang dengan exponential backoff  ← LOG: "logged in" (ke-2)
              1s → 2s → 4s → 8s → ... maks 30s

App Stop / Shutdown
  └─ Manager.CloseAll()
        └─ Client.Close()
              └─ conn.Close()  ← LOG: "logged out"
```

---

## Mekanisme Detail

### 1. Login Terjadi Sekali — `dial()`

```go
// client.go
func (c *Client) dial(ctx context.Context) (*routeros.Client, error) {
    addr := fmt.Sprintf("%s:%d", c.config.Host, c.config.Port)
    if c.config.UseTLS {
        return routeros.DialTLSContext(ctx, addr, c.config.Username, c.config.Password, nil)
    }
    return routeros.DialContext(ctx, addr, c.config.Username, c.config.Password)
}
```

`routeros.DialContext()` melakukan dua hal dalam satu panggilan: **TCP connect** dan **/login** ke RouterOS API. Setelah ini selesai, sesi sudah terbuka.

`dial()` hanya dipanggil dari dua tempat:
- `Connect()` — saat startup awal
- `reconnect()` — hanya jika koneksi benar-benar putus

### 2. Satu Koneksi untuk Banyak Command — Async Mode

```go
// client.go
func (c *Client) Connect(ctx context.Context) error {
    conn, err := c.dial(ctx)           // login sekali
    // ...
    errCh := conn.AsyncContext(c.asyncCtx)  // aktifkan multiplexing
    conn.Queue = DefaultQueueSize
    c.conn = conn                      // simpan, tidak pernah diganti kecuali reconnect
    go c.watchAsync(errCh)             // goroutine penjaga berjalan
    return nil
}
```

`conn.AsyncContext()` mengaktifkan **tag multiplexing** dari library `go-routeros/v3`:
- Satu TCP socket menangani banyak command serentak
- Setiap command diberi tag unik secara otomatis
- Response di-route balik berdasarkan tag tersebut
- Tidak perlu locking tambahan di sisi kode ini

### 3. Setiap Command Reuse Koneksi — `getConn()`

```go
// client.go
func (c *Client) getConn() (*routeros.Client, error) {
    c.mu.RLock()
    conn := c.conn
    c.mu.RUnlock()
    if conn == nil {
        return nil, fmt.Errorf("not connected to mikrotik (%s)", c.config.Host)
    }
    return conn, nil
}
```

Semua method eksekusi memanggil `getConn()` yang hanya mengambil `c.conn` yang sudah ada:

| Method | Keterangan |
|---|---|
| `Run(sentence...)` | Command tunggal, gunakan timeout dari config |
| `RunContext(ctx, sentence...)` | Command tunggal dengan context kustom |
| `RunMany(ctx, commands)` | Banyak command serentak via goroutine, satu koneksi |
| `ListenArgs(args)` | Streaming command (follow mode) |
| `ListenManyArgsContext(ctx, commands, size)` | Banyak stream serentak, fan-in ke satu channel |
| `RunRaw(ctx, args)` | Command bebas, return `[]map[string]string` |
| `ListenRaw(ctx, args, resultChan)` | Stream bebas ke channel eksternal |

Tidak satu pun dari method di atas melakukan dial atau login baru.

### 4. Auto-Reconnect — Jaga Sesi Tetap Hidup

```go
// client.go
func (c *Client) watchAsync(errCh <-chan error) {
    err := <-errCh  // blokir sampai async loop mati
    // ...
    go c.reconnect()
}

func (c *Client) reconnect() {
    backoff := reconnectBaseDelay  // mulai 1 detik
    for {
        conn, err := c.dial(dialCtx)  // login ulang hanya jika koneksi mati
        if err == nil {
            conn.AsyncContext(c.asyncCtx)
            c.conn = conn
            go c.watchAsync(errCh)
            return
        }
        time.Sleep(backoff)
        if backoff < reconnectMaxDelay {
            backoff *= 2  // 1s → 2s → 4s → 8s → ... maks 30s
        }
    }
}
```

Goroutine `watchAsync` berjalan di background sepanjang waktu. Jika async loop mati (jaringan putus, router restart), ia memicu `reconnect()` dengan exponential backoff. Login kedua hanya terjadi dalam kondisi ini.

### 5. Logout Saat `Close()`

```go
// client.go
func (c *Client) Close() {
    c.mu.Lock()
    c.closed = true
    conn := c.conn
    c.conn = nil
    c.mu.Unlock()

    c.asyncCancel()   // stop semua listener
    if conn != nil {
        conn.Close()  // kirim /quit ke RouterOS → muncul logout di log
    }
}
```

`conn.Close()` dari library `go-routeros` mengirim `/quit` ke RouterOS API, yang memunculkan entry **logout** di log MikroTik. Dipanggil dari:

| Pemanggil | Kondisi |
|---|---|
| `Manager.CloseAll()` | Saat aplikasi shutdown |
| `Manager.Unregister(name)` | Hapus satu router secara eksplisit |
| `Manager.GetOrConnect()` | Hanya jika client lama disconnect, sebelum buat yang baru |
| `Manager.TestConnection()` | Koneksi uji coba, langsung ditutup setelah tes |

---

## Manager — Pengelola Banyak Router

`Manager` menyimpan semua `*Client` dalam sebuah map:

```go
// manager.go
type Manager struct {
    clients map[string]*Client  // key = nama router
    mu      sync.RWMutex
    logger  *zap.Logger
}
```

Tiga cara mendapatkan client:

| Method | Perilaku |
|---|---|
| `Register(ctx, name, cfg)` | Buat koneksi baru, error jika nama sudah ada |
| `Get(name)` | Ambil client yang ada, error jika belum terdaftar |
| `GetOrConnect(ctx, name, cfg)` | Ambil yang ada jika masih konek, buat baru jika tidak |

`GetOrConnect()` adalah method yang paling sering dipakai di production karena menangani kedua kasus (sudah ada / belum ada) dengan aman tanpa race condition — menggunakan pola double-checked locking.

---

## Kunci Desain Persistence

| Mekanisme | Penjelasan |
|---|---|
| `c.conn` disimpan di struct | Koneksi hidup selama `Client` hidup, bukan per-request |
| `conn.AsyncContext()` | Satu TCP socket tangani banyak command via tag — tidak perlu socket baru |
| `getConn()` dipanggil semua method | Semua path ambil koneksi yang sudah ada, tidak pernah dial ulang |
| `watchAsync` goroutine | Deteksi koneksi putus secara pasif, tanpa polling |
| `reconnect()` dengan exponential backoff | Pulih otomatis jika jaringan terputus |
| `Close()` satu-satunya jalan keluar | Logout hanya terjadi saat shutdown atau unregister eksplisit |

---

## Satu Router, Banyak Koneksi Persistent

Arsitektur di atas mendukung skenario lanjutan: **satu router fisik memiliki beberapa koneksi persistent sekaligus**, masing-masing untuk peran berbeda (streaming, query, mutation).

### Masalah: Queue Bersama pada Satu Koneksi

Satu `Client` dengan async mode sudah bisa menangani concurrency via tag multiplexing — tetapi semua command berbagi satu buffer `conn.Queue`:

```
Satu koneksi shared (conn.Queue = 100):
  ├─ streaming data terus masuk  ──► bisa menghabiskan semua slot
  ├─ query response menunggu     ──► timeout karena queue penuh
  └─ mutation response menunggu  ──► sama
```

Streaming yang berjalan terus-menerus (misal `/tool/traffic`, `/interface/listen`) dapat menghabiskan seluruh slot queue dan membuat command query/mutation gagal timeout.

### Solusi: Compound Key di Manager

`Manager.clients` adalah `map[string]*Client` dengan key berupa string bebas. Ini sudah cukup untuk mendaftarkan tiga koneksi ke router yang sama menggunakan konvensi `"namaRouter:role"`:

```go
manager.Register(ctx, "router-alpha:stream",   cfgStream)
manager.Register(ctx, "router-alpha:query",    cfgQuery)
manager.Register(ctx, "router-alpha:mutation", cfgMutation)
```

Ketiganya dial ke `Host:Port` yang sama, masing-masing punya sesi sendiri di log MikroTik, dan semuanya tetap persistent. Tidak ada perubahan kode pada `Manager` maupun `Client`.

Untuk type safety, cukup tambahkan helper di layer yang memanggil `Register`:

```go
type ConnectionRole string

const (
    RoleStream   ConnectionRole = "stream"
    RoleQuery    ConnectionRole = "query"
    RoleMutation ConnectionRole = "mutation"
)

func clientKey(routerName string, role ConnectionRole) string {
    return routerName + ":" + string(role)
}

// Penggunaan
manager.Register(ctx, clientKey("router-alpha", RoleStream), cfgStream)

streamClient, _ := manager.Get(clientKey("router-alpha", RoleStream))
queryClient, _  := manager.Get(clientKey("router-alpha", RoleQuery))
```

### Config yang Direkomendasikan per Role

Karena setiap koneksi punya `Config` sendiri, timeout dan queue bisa disesuaikan per peran:

```go
cfgStream := client.Config{
    Host:     "192.168.1.1",
    Port:     8728,
    Username: "admin",
    Password: "...",
    Timeout:  0,               // tidak ada timeout — stream berjalan selamanya
}

cfgQuery := client.Config{
    Host:     "192.168.1.1",
    Port:     8728,
    Username: "admin",
    Password: "...",
    Timeout:  10 * time.Second,
}

cfgMutation := client.Config{
    Host:     "192.168.1.1",
    Port:     8728,
    Username: "admin",
    Password: "...",
    Timeout:  30 * time.Second, // lebih lama — write bisa lebih lambat
}
```

Queue size per koneksi diatur saat `Connect()` dipanggil via `conn.Queue`. Saat ini nilainya dikontrol oleh konstanta `DefaultQueueSize = 100`; untuk koneksi stream bisa dinaikkan langsung di `Connect()` jika diperlukan.

### Perbandingan: Satu vs Tiga Koneksi

| Aspek | Satu koneksi | Tiga koneksi (stream/query/mutation) |
|---|---|---|
| Queue isolation | Tidak — stream bisa habiskan queue | Ya — masing-masing queue terpisah |
| Timeout config | Satu nilai untuk semua | Stream `0`, query `10s`, mutation `30s` |
| Lifecycle | Jika koneksi putus, semua terdampak | Masing-masing reconnect sendiri |
| Log MikroTik | 1 login / 1 logout | 3 login / 3 logout (mudah debug per role) |
| Perubahan kode | — | Tidak ada — cukup konvensi key |

### Alur Lifecycle dengan Tiga Koneksi

```
App Start
  ├─ Register("router-alpha:stream",   cfgStream)   → LOG: "logged in" (stream)
  ├─ Register("router-alpha:query",    cfgQuery)    → LOG: "logged in" (query)
  └─ Register("router-alpha:mutation", cfgMutation) → LOG: "logged in" (mutation)

App Berjalan
  ├─ Get("router-alpha:stream")   → ListenArgs(...)   ← streaming terus-menerus
  ├─ Get("router-alpha:query")    → Run("/ip/...")    ← read, tidak ganggu stream
  └─ Get("router-alpha:mutation") → Run("/ip/.../add") ← write dengan timeout lebih lama

App Stop
  └─ Manager.CloseAll()
        ├─ Close "router-alpha:stream"   → LOG: "logged out"
        ├─ Close "router-alpha:query"    → LOG: "logged out"
        └─ Close "router-alpha:mutation" → LOG: "logged out"
```

---

## File Terkait

| File | Tanggung Jawab |
|---|---|
| `client/client.go` | Core `Client`: connect, run, listen, reconnect, close |
| `client/manager.go` | `Manager`: kelola banyak router dalam satu map |
| `client/options.go` | `Config`: host, port, credentials, timeout |
| `client/helpers.go` | Parser nilai RouterOS: rate, byte size, bool, slash-value |
