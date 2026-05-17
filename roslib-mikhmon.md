# Analisis: Rebuild Mikhmon dengan roslib (Go)

> Perbandingan mendalam antara pendekatan PHP lama (mikhmonv3) vs
> reimplementasi Go menggunakan `github.com/quiqxiq/roslib`

---

## Ringkasan Eksekutif

**Feasibility: SANGAT TINGGI** — hampir semua fitur mikhmon bisa diport ke Go
dengan roslib, bahkan dengan kualitas arsitektur yang jauh lebih baik.
Tapi ada **satu masalah kritis** dan **dua trade-off desain** yang harus
diputuskan sebelum mulai coding.

---

## 1. Pemetaan Fitur: PHP → Go + roslib

### 1.1 Snapshot Query (List, Filter, Count)

**PHP (mikhmon):**
```php
$API->comm("/ip/hotspot/user/print", ["?name" => "$name"]);
$API->comm("/ip/hotspot/user/print", ["count-only" => ""]);
```

**Go (roslib):**
```go
// Filter by name
reply, err := dev.Path("/ip/hotspot/user").Print().
    Where("name", name).Exec(ctx)

// Count-only
reply, err := dev.Path("/ip/hotspot/user").Print().
    Count().Exec(ctx)

// Filter by profile
reply, err := dev.Path("/ip/hotspot/user").Print().
    Where("profile", profileName).Exec(ctx)

// Filter by .id
reply, err := dev.Path("/ip/hotspot/user").Print().
    WherePair(roslib.Where(".id", id)).Exec(ctx)

// Cache (tidak ada di PHP, bonus dari roslib)
reply, err := dev.Path("/system/resource").Print().
    ExecCached(ctx, 10*time.Second)
```

**Verdict: ✅ 100% coverage, API lebih bersih, concurrent-safe**

---

### 1.2 Mutation: Add, Set, Remove, Enable, Disable

**PHP:**
```php
$API->comm("/ip/hotspot/user/add", [
    "server" => $server, "name" => $name, "password" => $pass,
    "profile" => $profile, "limit-uptime" => $timelimit,
    "limit-bytes-total" => $datalimit, "comment" => $comment,
]);
$API->comm("/ip/hotspot/user/set", [".id" => $id, "disabled" => "yes"]);
$API->comm("/ip/hotspot/user/remove", [".id" => $id]);
```

**Go:**
```go
// Add user
_, err := dev.Path("/ip/hotspot/user").Add(ctx,
    roslib.NewPair("server", server),
    roslib.NewPair("name", name),
    roslib.NewPair("password", password),
    roslib.NewPair("profile", profile),
    roslib.NewPair("limit-uptime", timeLimit),
    roslib.NewPair("limit-bytes-total", dataLimit),
    roslib.NewPair("comment", comment),
)

// Disable user
_, err = dev.Path("/ip/hotspot/user").Set(ctx, id,
    roslib.NewPair("disabled", "yes"),
)

// Remove user
_, err = dev.Path("/ip/hotspot/user").Remove(ctx, id)

// Reset counters (tidak ada helper khusus, pakai Run)
_, err = dev.Path("/ip/hotspot/user").Run(ctx, "reset-counters",
    roslib.NewPair("numbers", id),
)
```

**Verdict: ✅ 100% coverage**

---

### 1.3 Scheduler (System Scheduler CRUD)

**PHP:**
```php
$API->comm("/system/scheduler/add", [
    "name" => $name, "start-time" => $time,
    "interval" => $interval, "on-event" => $script,
    "disabled" => "no", "comment" => "Monitor Profile $name"
]);
$API->comm("/system/scheduler/set", [".id" => $id, "disabled" => "yes"]);
$API->comm("/system/scheduler/remove", [".id" => $id]);
```

**Go:**
```go
_, err := dev.Path("/system/scheduler").Add(ctx,
    roslib.NewPair("name", name),
    roslib.NewPair("start-time", startTime),
    roslib.NewPair("interval", interval),
    roslib.NewPair("on-event", onEventScript),
    roslib.NewPair("disabled", "no"),
    roslib.NewPair("comment", "Monitor Profile "+name),
)

_, err = dev.Path("/system/scheduler").Set(ctx, id,
    roslib.NewPair("disabled", "yes"),
)

_, err = dev.Path("/system/scheduler").Remove(ctx, id)
```

**Verdict: ✅ 100% coverage**

---

### 1.4 Real-time Traffic Monitor

**PHP (mikhmon):** AJAX polling tiap 3 detik ke PHP endpoint yang konek-putus-konek ke router.

**Go (roslib):**
```go
// Inherent streaming — koneksi persisten, tidak polling
err := dev.Path("/interface/monitor-traffic").
    With("interface", ifaceName).
    Stream("traffic-"+ifaceName, func(s *roslib.Sentence) {
        rx := s.IntOr("rx-bits-per-second", 0)
        tx := s.IntOr("tx-bits-per-second", 0)
        // push ke WebSocket atau SSE ke frontend
        hub.Broadcast(TrafficEvent{Interface: ifaceName, RxBps: rx, TxBps: tx})
    })
```

**Verdict: ✅ JAUH LEBIH BAIK — persistent connection vs reconnect tiap 3 detik**

---

### 1.5 Hotspot Log Real-time

**PHP:** Page reload manual, atau AJAX polling.

**Go:**
```go
err := dev.Path("/log").Print().FollowOnly().
    Stream("hotspot-log", func(s *roslib.Sentence) {
        if strings.Contains(s.Get("topics"), "hotspot") {
            hub.Broadcast(LogEvent{
                Time:    s.Get("time"),
                Message: s.Get("message"),
            })
        }
    })
```

**Verdict: ✅ Real-time push, tidak perlu polling**

---

### 1.6 Hotspot Active — Real-time

**PHP:** AJAX reload tiap X detik.

**Go:**
```go
err := dev.Path("/ip/hotspot/active").Print().Follow().
    Stream("hs-active", func(s *roslib.Sentence) {
        hub.Broadcast(ActiveSessionEvent{
            User:    s.Get("user"),
            Address: s.Get("address"),
            Uptime:  s.Get("uptime"),
            Server:  s.Get("server"),
        })
    })
```

**Verdict: ✅ Jauh lebih baik**

---

### 1.7 System Resource Dashboard

**PHP:** AJAX reload tiap X detik, reconnect ke router tiap kali.

**Go:**
```go
err := dev.RegisterPoll(roslib.PollConfig{
    ID:       "sys-resource",
    Path:     "/system/resource",
    Args:     []string{"print"},
    Interval: 5 * time.Second,
    Handler: func(s *roslib.Sentence) {
        hub.Broadcast(ResourceEvent{
            CPULoad:    s.IntOr("cpu-load", 0),
            FreeMemory: s.BytesOr("free-memory", 0),
            Uptime:     s.Get("uptime"),
            Version:    s.Get("version"),
        })
    },
})
```

**Verdict: ✅ Satu koneksi, efficient polling**

---

### 1.8 IP Binding — Enable/Disable/Remove + Cascade Cleanup

**PHP:** Rangkaian 9 API call: hapus binding → cari queue → hapus queue → cari ARP → hapus ARP → cari DHCP lease → hapus lease → cari scheduler → hapus scheduler.

**Go:**
```go
func (svc *BindingService) Remove(ctx context.Context, dev *roslib.Device, bindingID, mac string) error {
    // 1. Hapus binding
    if _, err := dev.Path("/ip/hotspot/ip-binding").Remove(ctx, bindingID); err != nil {
        return err
    }
    // 2. Queue cleanup (if exists)
    if q, _ := dev.Path("/queue/simple").Print().Where("name", mac).Exec(ctx); len(q.Rows) > 0 {
        dev.Path("/queue/simple").Remove(ctx, q.Rows[0].Get(".id"))
    }
    // 3. ARP cleanup
    if arp, _ := dev.Path("/ip/arp").Print().Where("mac-address", mac).Exec(ctx); len(arp.Rows) > 0 {
        dev.Path("/ip/arp").Remove(ctx, arp.Rows[0].Get(".id"))
    }
    // 4. DHCP lease cleanup
    if lease, _ := dev.Path("/ip/dhcp-server/lease").Print().Where("mac-address", mac).Exec(ctx); len(lease.Rows) > 0 {
        dev.Path("/ip/dhcp-server/lease").Remove(ctx, lease.Rows[0].Get(".id"))
    }
    // 5. Scheduler cleanup
    if sch, _ := dev.Path("/system/scheduler").Print().Where("name", mac).Exec(ctx); len(sch.Rows) > 0 {
        dev.Path("/system/scheduler").Remove(ctx, sch.Rows[0].Get(".id"))
    }
    return nil
}
```

Semua 9 call bisa dijalankan **concurrent** karena roslib thread-safe:
```go
var wg sync.WaitGroup
wg.Add(4)
go func() { defer wg.Done(); dev.Path("/queue/simple").Remove(ctx, queueID) }()
go func() { defer wg.Done(); dev.Path("/ip/arp").Remove(ctx, arpID) }()
go func() { defer wg.Done(); dev.Path("/ip/dhcp-server/lease").Remove(ctx, leaseID) }()
go func() { defer wg.Done(); dev.Path("/system/scheduler").Remove(ctx, schID) }()
wg.Wait()
```

**Verdict: ✅ Bisa parallel — cleanup LEBIH CEPAT dari PHP**

---

### 1.9 Multi-Router (Multi-Session)

**PHP:** Mikhmon simpan kredensial di PHP session, reconnect tiap request.

**Go (roslib):**
```go
// Fleet dari env atau config
fleet, influxCli, err := roslib.NewFleet(ctx, fleetCfg, logger)

// Akses per session/router ID
dev := fleet["router-ghaibnet"]
reply, _ := dev.Path("/ip/hotspot/user").Print().Exec(ctx)

// Semua router persistent, tidak ada reconnect per-request
```

**Verdict: ✅ Jauh lebih baik — 2 persistent connections per router, bukan reconnect tiap request**

---

## 2. MASALAH KRITIS: Capability Registry vs RouterOS v6

Ini adalah **rintangan terbesar** yang wajib ditangani sebelum coding dimulai.

### Masalah

roslib mem-embed registry **RouterOS 7.20.8** (541 endpoint).
Mikhmon ditargetkan ke router seperti **RB750G RouterOS 6.49.11**.

Dengan `StrictCapability=true` (default), ini akan **GAGAL**:

```go
// Error karena /sys/sch tidak ada di registry v7
dev.Path("/sys/sch").Print().Exec(ctx)
// → capability: unknown command word /sys/sch/print

// /ip/hotspot/user/reset-counters mungkin ada di v7 tapi berbeda di v6
dev.Path("/ip/hotspot/user").Run(ctx, "reset-counters", ...)
```

### Solusi

**Opsi A — `StrictCapability=false` (paling cepat):**
```go
dev, _ := roslib.New(ctx, roslib.Options{
    Address:          "192.168.88.1:8728",
    Username:         "admin",
    Password:         "secret",
    Logger:           logger,
    StrictCapability: false, // ← router yang validasi, bukan library
})
```
Semua command dikirim langsung ke router. Library hanya log-warn kalau ada yang tidak dikenal.
Cocok untuk v6 yang path-nya tidak selalu sama dengan v7.

**Opsi B — Custom registry v6:**
```go
// Generate JSON registry dari router v6 yang ada
// (perlu tool scraping RouterOS /console/inspect atau manual mapping)
reg, _ := capability.Load(capability.LoadOptions{Path: "routeros_6.49.json"})
dev, _ := roslib.New(ctx, roslib.Options{
    ...,
    Registry:         reg,
    StrictCapability: true,
})
```
Lebih proper tapi butuh kerja ekstra untuk generate JSON registry v6.

**Rekomendasi: Opsi A untuk development, target Opsi B untuk production.**

---

## 3. TRADE-OFF DESAIN KRITIS: Expiry System

Ini adalah **keputusan arsitektur paling penting** dalam rebuild ini.

### Sistem Lama (Mikhmon PHP)

Mikhmon menyuntikkan RouterOS script ke field `on-login` profil dan `on-event` scheduler. Script ini berjalan **di dalam router** — server PHP tidak perlu aktif. Router yang handle expiry secara mandiri.

```routeros
# on-login (di dalam router, dieksekusi oleh RouterOS saat user login)
/sys sch add name="$user" interval="30d";
:delay 5s;
:local exp [/sys sch get [find name="$user"] next-run];
/ip hotspot user set comment="$exp" [find name="$user"];
/sys sch remove [find name="$user"];
```

### Opsi A: Tetap Pakai RouterOS Scripts (Keep Lama, Ganti Server)

Go hanya berperan sebagai **generator** script dan **web server** — logika expiry tetap jalan di router.

```go
// Go generate script RouterOS, lalu inject via API
onLoginScript := generateOnLoginScript(profile.ExpiryMode, profile.Validity, profile.Price)

_, err := dev.Path("/ip/hotspot/user/profile").Add(ctx,
    roslib.NewPair("name", profile.Name),
    roslib.NewPair("rate-limit", profile.RateLimit),
    roslib.NewPair("on-login", onLoginScript), // ← inject script ke router
    roslib.NewPair("shared-users", profile.SharedUsers),
)
```

**Keuntungan:**
- Router tetap mandiri — expiry tetap bekerja meski Go service mati
- Tidak perlu redesign logika utama mikhmon
- Migrasi dari mikhmon PHP ke mikhmon Go lebih smooth
- Compatible dengan v6

**Kelemahan:**
- Tetap bergantung pada RouterOS scripting yang "hacky" (scheduler temp, delay 5s)
- Susah debug kalau ada bug di script yang sudah ter-inject
- Tidak bisa unit test logika expiry tanpa router nyata

---

### Opsi B: Server-side Expiry (Full Go, No Router Scripts)

Semua logika expiry pindah ke Go. Router tidak perlu script sama sekali.

```go
// 1. Listen login events dari router secara real-time
err := dev.Path("/ip/hotspot/active").Print().Follow().
    Stream("hs-login-tracker", func(s *roslib.Sentence) {
        user := s.Get("user")
        if s.Word() == "!re" { // user baru login
            svc.HandleLogin(ctx, dev, user)
        }
    })

// 2. HandleLogin: set expiry waktu login + update comment di router
func (svc *ExpiryService) HandleLogin(ctx context.Context, dev *roslib.Device, username string) {
    profile := svc.GetProfileForUser(username)
    if profile.ExpiryMode == "0" { return }

    expiry := time.Now().Add(profile.Validity)
    comment := expiry.Format("Jan/02/2006 15:04:05") // format mikhmon

    // Set comment di router (sama persis dengan yang dilakukan on-login script)
    userReply, _ := dev.Path("/ip/hotspot/user").Print().Where("name", username).Exec(ctx)
    if len(userReply.Rows) > 0 {
        userID := userReply.Rows[0].Get(".id")
        dev.Path("/ip/hotspot/user").Set(ctx, userID,
            roslib.NewPair("comment", comment))
    }
}

// 3. Background goroutine check expiry (gantikan RouterOS scheduler)
func (svc *ExpiryService) RunExpiryChecker(ctx context.Context, dev *roslib.Device) {
    ticker := time.NewTicker(2 * time.Minute)
    defer ticker.Stop()
    for {
        select {
        case <-ctx.Done(): return
        case <-ticker.C:
            svc.checkAndExpireUsers(ctx, dev)
        }
    }
}

// 4. Catat transaksi di DB (bukan di /system/script)
func (svc *TransactionService) RecordSale(tx Transaction) error {
    return svc.db.Create(&tx).Error // PostgreSQL/SQLite via GORM
}
```

**Keuntungan:**
- Arsitektur bersih — tidak ada script tersembunyi di dalam router
- Bisa unit test penuh (mock `dev`)
- Logika expiry terpusat, mudah debug
- Bisa tambah fitur: notifikasi WhatsApp/Telegram saat expiry, dll
- Laporan di DB = query SQL, bukan parse `/system/script`

**Kelemahan:**
- **Jika Go service mati → expiry tidak jalan** (user expired tetap bisa connect)
- Harus selalu running — tidak cocok untuk deployment sederhana
- Butuh persistent storage (minimal SQLite)
- Lebih banyak pekerjaan implementasi

---

### Rekomendasi: **Opsi Hybrid**

```
┌──────────────────────────────────────────────────┐
│  Go Service (always-running)                     │
│  ┌────────────────┐  ┌────────────────────────┐  │
│  │  Web/REST API  │  │  Background Services   │  │
│  │  (Gin/Echo)    │  │  - Expiry checker      │  │
│  │                │  │  - Login tracker       │  │
│  │                │  │  - Traffic stream      │  │
│  └────────────────┘  └────────────────────────┘  │
│                                                  │
│  roslib Device (2 persistent conn per router)    │
└──────────────────────────────────────────────────┘
         │                          │
         ▼                          ▼
┌─────────────────┐      ┌──────────────────────┐
│  RouterOS API   │      │  SQLite / PostgreSQL  │
│  (port 8728)    │      │  - Transaksi          │
│                 │      │  - Config profil      │
│  on-login:      │      │  - Cache expiry       │
│  HAPUS SCRIPT   │      └──────────────────────┘
│  (serahkan ke   │
│  Go service)    │
└─────────────────┘
```

Untuk failsafe: **tetap inject on-login script minimal** (hanya lock MAC dan set comment format sederhana), tapi backup checker ada di Go service. Kalau service mati, router masih handle expiry. Kalau service hidup, Go yang handle lebih akurat.

---

## 4. Trade-off: Penyimpanan Laporan Transaksi

### Sistem Lama
Transaksi disimpan di `/system/script` router:
```
name: "jan/05/2025-|-14:32:01-|-user001-|-5000-|-192.168.88.10-|-AA:BB:...-|-1d-|-paket-1hari-|-kasir"
owner: "jan2025"
comment: "mikhmon"
```

**Masalah:**
- RouterOS bukan database — tidak ada indexing, aggregation, SQL
- Filter hanya by `?source`, `?owner`, `?comment` — tidak bisa filter multi-field
- Lambat kalau transaksi ribuan (PHP iterate semua)
- Tidak ada backup yang proper
- Storage terbatas (flash ROM router)

### Rekomendasi Go: SQLite atau PostgreSQL

```go
type Transaction struct {
    ID        uint      `gorm:"primarykey"`
    CreatedAt time.Time
    Date      string    // "jan/05/2025"
    Time      string    // "14:32:01"
    Username  string    `gorm:"index"`
    Price     int
    IP        string
    MAC       string
    Validity  string
    Profile   string    `gorm:"index"`
    Comment   string
    RouterID  string    `gorm:"index"` // multi-router support
}

// Query jauh lebih powerful
db.Where("profile = ? AND created_at >= ?", "paket-1hari", startOfMonth).Find(&txns)
db.Where("comment LIKE ?", "%kasir-andi%").Find(&txns)
```

**Untuk backward compat:** Buat adapter yang bisa baca dan tulis ke KEDUANYA
(`/system/script` untuk compat + DB untuk performa).

---

## 5. Apa Yang Tidak Bisa Dilakukan roslib (Gap)

### 5.1 Script Generator (Tetap Perlu Manual)

roslib tidak punya helper untuk **generate** string RouterOS script. Perlu ditulis sendiri:

```go
func GenerateOnLoginScript(expMode, validity, price, sprice, lock string) string {
    // String builder untuk menghasilkan script RouterOS
    // yang sama dengan yang di-hardcode di PHP mikhmon
    return fmt.Sprintf(`
:put (",%s,%s,%s,%s,,%s,");
{ ... }`, expMode, price, validity, sprice, lock)
}

func GenerateBgServiceScript(profileName, expMode string) string {
    // Generate on-event scheduler script
    return `...`
}
```

Ini sepenuhnya Go code biasa, tidak butuh library tambahan.

### 5.2 Voucher Printing / Template HTML

roslib tidak handle UI/HTML. Template voucher perlu diport ke Go HTML template:

```go
tmpl := template.Must(template.ParseFiles("templates/voucher.html"))
tmpl.Execute(w, VoucherData{
    Username: user, Password: pass,
    Profile: profile, Validity: validity,
    Price: price, SSID: ssid,
})
```

### 5.3 `=.proplist=.id` Projection

roslib fluent API tidak punya `.PropList("field1","field2")` yang menghasilkan
`=.proplist=field1,field2`. Perlu pakai `Run()` atau request ke upstream untuk tambah
method `.Proplist()` ke builder. Workaround:

```go
// Workaround: ambil semua, extract .id di Go
reply, _ := dev.Path("/system/script").Print().Where("source", date).Exec(ctx)
for _, row := range reply.Rows {
    dev.Path("/system/script").Remove(ctx, row.Get(".id"))
}
```

Atau pakai concurrent bulk delete dengan goroutine — tetap lebih cepat dari PHP yang sequential.

### 5.4 `on-login` / `on-event` Introspection

roslib tidak punya method khusus untuk extract/parse RouterOS script yang sudah
ter-inject di profil. Harus parse manual string dari `reply.Rows[0].Get("on-login")`.

---

## 6. Kelebihan Nyata Go + roslib vs PHP

| Aspek | PHP (mikhmonv3) | Go + roslib |
|---|---|---|
| Koneksi per request | Reconnect tiap request | 2 persistent conn per router |
| Concurrent ops | Sequential per page | Ratusan ops paralel |
| Traffic monitor | AJAX poll tiap 3s + reconnect | Persistent stream |
| Log real-time | AJAX poll | Push via WebSocket/SSE |
| Expiry check | RouterOS scheduler | Goroutine (bisa keduanya) |
| Multi-router | PHP session per router | Fleet + shared pool |
| Report query | Iterate `/system/script` | SQL query |
| Error handling | `if false return` | `if err != nil` + typed errors |
| Unit testing | Hampir tidak mungkin | Bisa mock `Executor` interface |
| Memory (dashboard) | Reconnect+query tiap reload | Cache + push update |
| Startup | Tiap request dari nol | Service, startup sekali |

---

## 7. Risiko dan Mitigasi

### Risiko 1: roslib registry v7 vs router v6

**Mitigasi:** `StrictCapability: false` — production-ready, tinggal set satu opsi.

### Risiko 2: on-login script injection masih butuh string building

**Mitigasi:** Buat package `scriptgen` terpisah dengan unit test penuh.
Script RouterOS yang di-generate bisa di-test outputnya tanpa router.

### Risiko 3: Roslib adalah library baru, belum battle-tested di scale besar

**Mitigasi:** Roslib wrap `go-routeros/v3` yang sudah mature. Core connection
logic bukan dari roslib sendiri. Kalau ada bug di roslib layer, bisa bypass
dan akses `dev.CommandConn()` langsung.

### Risiko 4: Expiry failsafe kalau Go service mati

**Mitigasi:** Hybrid approach — tetap inject minimal on-login script ke router
sebagai backup. Go service jadi primary checker, router jadi fallback.

### Risiko 5: PPP features (file hilang di mikhmon)

**Risiko rendah** — PPP commands standard, semua tersedia:
```go
dev.Path("/ppp/secret").Print().Exec(ctx)    // list secrets
dev.Path("/ppp/secret").Add(ctx, ...)        // add secret
dev.Path("/ppp/profile").Print().Exec(ctx)   // list profiles
dev.Path("/ppp/active").Print().Exec(ctx)    // list active
dev.Path("/ppp/active").Remove(ctx, id)      // kick PPP session
```

---

## 8. Arsitektur yang Direkomendasikan

```
mikhmon-go/
├── cmd/
│   └── server/main.go          ← entry point, setup & start
├── internal/
│   ├── api/                    ← HTTP handlers (Gin/Echo)
│   │   ├── hotspot.go
│   │   ├── profile.go
│   │   ├── report.go
│   │   └── system.go
│   ├── service/                ← business logic
│   │   ├── user.go             ← add/remove/edit user
│   │   ├── profile.go          ← create profile + inject on-login
│   │   ├── expiry.go           ← expiry checker goroutine
│   │   ├── transaction.go      ← catat & baca laporan
│   │   └── binding.go          ← cascade cleanup
│   ├── scriptgen/              ← RouterOS script generator
│   │   ├── onlogin.go          ← generate on-login script
│   │   ├── bgservice.go        ← generate on-event bgservice
│   │   └── scriptgen_test.go   ← unit test output script
│   ├── stream/                 ← real-time handlers
│   │   ├── traffic.go          ← monitor-traffic → WebSocket
│   │   ├── log.go              ← hotspot log → SSE
│   │   └── active.go           ← hotspot active → WebSocket
│   └── store/                  ← data persistence
│       ├── transaction.go      ← GORM model + queries
│       └── migration.go
├── config/                     ← config loader (env + YAML)
└── web/                        ← frontend (Vue/React atau htmx)
```

---

## 9. Contoh Konkret: Rebuild `adduserprofile.php`

**PHP (mikhmon) — 120+ baris, string concatenation:**
```php
$onlogin = ':put (",\'.$expmode.\',\'.$price.\',...");{ ... /sys sch add name="$user" ... }';
$API->comm("/ip/hotspot/user/profile/add", [
    "name" => $name, "rate-limit" => $ratelimit,
    "on-login" => $onlogin, ...
]);
```

**Go (mikhmon-go) — typed, testable:**
```go
// scriptgen/onlogin.go
func OnLogin(cfg OnLoginConfig) string {
    var b strings.Builder
    fmt.Fprintf(&b, `:put (",%s,%s,%s,%s,,%s,");`, 
        cfg.ExpiryMode, cfg.Price, cfg.Validity, cfg.SalePrice, cfg.Lock)
    // ... rest of script generation
    return b.String()
}

// service/profile.go
func (svc *ProfileService) Create(ctx context.Context, dev *roslib.Device, req CreateProfileRequest) error {
    onLogin := scriptgen.OnLogin(scriptgen.OnLoginConfig{
        ExpiryMode: req.ExpiryMode,
        Price:      req.Price,
        Validity:   req.Validity,
        Lock:       req.LockUser,
    })

    bgService := scriptgen.BgService(scriptgen.BgServiceConfig{
        ProfileName: req.Name,
        ExpiryMode:  req.ExpiryMode,
    })

    interval := fmt.Sprintf("00:0%d:%02d", rand.Intn(3)+2, rand.Intn(60))
    startTime := fmt.Sprintf("00:0%d:%02d", rand.Intn(3)+1, rand.Intn(60))

    // Inject on-login ke profil
    if _, err := dev.Path("/ip/hotspot/user/profile").Add(ctx,
        roslib.NewPair("name", req.Name),
        roslib.NewPair("rate-limit", req.RateLimit),
        roslib.NewPair("shared-users", req.SharedUsers),
        roslib.NewPair("address-pool", req.AddressPool),
        roslib.NewPair("status-autorefresh", "1m"),
        roslib.NewPair("on-login", onLogin),
        roslib.NewPair("parent-queue", req.ParentQueue),
    ); err != nil {
        return fmt.Errorf("add profile: %w", err)
    }

    // Buat background monitor scheduler
    if req.ExpiryMode != "0" {
        if _, err := dev.Path("/system/scheduler").Add(ctx,
            roslib.NewPair("name", req.Name),
            roslib.NewPair("start-time", startTime),
            roslib.NewPair("interval", interval),
            roslib.NewPair("on-event", bgService),
            roslib.NewPair("disabled", "no"),
            roslib.NewPair("comment", "Monitor Profile "+req.Name),
        ); err != nil {
            return fmt.Errorf("add scheduler: %w", err)
        }
    }

    return nil
}
```

---

## 10. Kesimpulan

| Pertanyaan | Jawaban |
|---|---|
| Bisa pakai roslib untuk rebuild mikhmon? | **Ya, feasible** |
| Semua command mikhmon bisa diport? | **Ya, 100%** |
| Ada yang lebih bagus dari PHP? | **Hampir semuanya lebih bagus** |
| Ada yang tidak bisa dilakukan? | **Tidak ada — hanya butuh workaround kecil** |
| Masalah terbesar? | **StrictCapability harus false untuk v6** |
| Keputusan desain terpenting? | **Opsi Hybrid untuk expiry system** |
| Estimasi effort? | **2–4x lebih lama dari PHP, tapi maintainable jangka panjang** |
| Recommended stack? | Go + roslib + Gin + GORM/SQLite + WebSocket + Vue/htmx |