# Analisis Lengkap mikhmonv3 — RouterOS Commands & Scripts (Revisi v2)

> Repo: [laksa19/mikhmonv3](https://github.com/laksa19/mikhmonv3)  
> Metode: Full scan semua 96 file PHP, routing index.php, dan string PHP-embedded scripts  
> Total command unik: **48 API path** + **12 PPP command (file hilang)** + **14 inline script command**

---

## ⚠️ Temuan Kritis: File Yang HILANG Dari Repo

Saat `index.php` di-routing, ada **12 file yang di-`include` tapi tidak ada** di repo ini. Artinya **fitur PPP belum di-upload** developer atau sengaja dihapus dari versi publik ini:

| File yang di-include | Status | Fitur |
|---|---|---|
| `ppp/pppsecrets.php` | ❌ MISSING | List PPP secrets |
| `ppp/addsecret.php` | ❌ MISSING | Tambah PPP secret |
| `ppp/secretbyname.php` | ❌ MISSING | Edit PPP secret |
| `process/psecret.php` | ❌ MISSING | Enable/disable/remove secret |
| `ppp/pppprofile.php` | ❌ MISSING | List PPP profiles |
| `ppp/addpppprofile.php` | ❌ MISSING | Tambah PPP profile |
| `ppp/profilebyname.php` | ❌ MISSING | Edit PPP profile |
| `process/removepprofile.php` | ❌ MISSING | Hapus PPP profile |
| `ppp/pppactive.php` | ❌ MISSING | List PPP active connections |
| `hotspot/binding.php` | ❌ MISSING | Halaman binding (berbeda dari ipbinding.php) |
| `report/export.php` | ❌ MISSING | Export report |
| `process/makebinding.php` | ❌ MISSING | Proses pembuatan binding dari MAC |

**Implikasi:** Command PPP di bawah ini **dapat disimpulkan** dari routing dan variabel yang ada di `index.php` & `menu.php`, meski file-nya tidak ada.

---

## 1. Semua RouterOS API Commands

### 1.1 System Info & Control

| Command | Parameter | File | Tujuan |
|---|---|---|---|
| `/system/identity/print` | — | `index.php` | Nama router |
| `/system/resource/print` | — | `dashboard/home.php`, `aload.php` | CPU, RAM, uptime, version ROS |
| `/system/routerboard/print` | — | `dashboard/home.php`, `aload.php` | Model board, firmware version |
| `/system/clock/print` | — | `dashboard/home.php`, `report/selling.php`, `report/print.php` | Waktu, timezone router |
| `/system/reboot` | — | `process/reboot.php` | **Reboot router** (via `write()`) |
| `/system/shutdown` | — | `process/shutdown.php` | **Matikan router** (via `write()`) |

### 1.2 System Logging

| Command | Parameter | File | Tujuan |
|---|---|---|---|
| `/system/logging/print` | `?prefix="->"` | `dashboard/home.php`, `aload.php` | Cek apakah logging hotspot aktif |
| `/system/logging/add` | `action=disk`, `prefix="->"`, `topics=hotspot,info,debug` | `dashboard/home.php`, `aload.php` | Auto-aktifkan logging hotspot ke disk |

### 1.3 System Script (Inti Sistem Laporan & QuickPrint)

| Command | Parameter | File | Tujuan |
|---|---|---|---|
| `/system/script/print` | `?source=<tanggal>` | `dashboard/home.php`, `report/selling.php`, `report/print.php` | Laporan hari ini |
| `/system/script/print` | `?owner=<bulanTahun>` | `dashboard/home.php`, `report/selling.php`, `livereport.php`, `userlog.php` | Laporan bulan ini |
| `/system/script/print` | `?comment=mikhmon` | `report/selling.php`, `include/userlog.php` | Semua script transaksi |
| `/system/script/print` | `?comment=QuickPrintMikhmon` | `hotspot/listquickprint.php`, `quickprint.php` | Daftar konfigurasi Quick Print |
| `/system/script/print` | `?.id=<id>` | `hotspot/listquickprint.php` | Quick Print spesifik |
| `/system/script/print` | `?name=<username>` | `process/removehotspotuser.php` | Script transaksi per user |
| `/system/script/add` | `name`, `source`, `comment`, `owner` | On-login script (inject saat profile dibuat) | Catat transaksi |
| `/system/script/add` | `name`, `source`, `comment=QuickPrintMikhmon` | `hotspot/listquickprint.php` | Simpan config Quick Print |
| `/system/script/set` | `.id`, `name`, `source`, `comment` | `hotspot/listquickprint.php` | Update Quick Print |
| `/system/script/remove` | `.id` | `process/removehotspotuser.php`, `removereport.php`, `listquickprint.php`, `report/print.php` | Hapus script / transaksi |

**Parameter khusus (via `write()` low-level):**
- `=.proplist=.id` — ambil hanya field `.id`, efisien untuk bulk delete
- `?=source=<value>` — filter alternatif (syntax lama, ada di `userlog.php`)
- `?=owner=<value>` — filter alternatif (syntax lama)
- `?=comment=mikhmon` — filter alternatif (syntax lama)

### 1.4 System Scheduler

| Command | Parameter | File | Tujuan |
|---|---|---|---|
| `/system/scheduler/print` | — | `system/scheduler.php` | List semua scheduler |
| `/system/scheduler/print` | `count-only=""` | `system/scheduler.php` | Hitung scheduler |
| `/system/scheduler/print` | `?name=<username>` | `hotspot/userbyname.php`, `userprofile.php`, `resethotspotuser.php`, `removehotspotuser.php` | Cari scheduler milik user |
| `/system/scheduler/print` | `?name=<profname>` | `hotspot/userprofilebyname.php` | Cari monitor scheduler profil |
| `/system/scheduler/add` | `name`, `start-time`, `interval`, `on-event`, `disabled=no`, `comment="Monitor Profile <nama>"` | `hotspot/adduserprofile.php`, `userprofilebyname.php` | Buat background expiry monitor |
| `/system/scheduler/set` | `.id`, `name`, `interval`, `on-event`, `disabled` | `adduserprofile.php`, `userprofilebyname.php`, `process/pscheduler.php` | Update / enable / disable |
| `/system/scheduler/remove` | `.id` | Banyak file `process/` | Hapus scheduler |
| `/sys/sch/print` | `?name=<user>` | `status/status.php` (alias path pendek) | Cek expiry lewat next-run |

### 1.5 Log

| Command | Parameter | File | Tujuan |
|---|---|---|---|
| `/log/print` | `?topics=hotspot,info,debug` | `hotspot/log.php`, `dashboard/home.php`, `aload.php` | Baca log hotspot real-time |

### 1.6 IP Hotspot — User

| Command | Parameter | File | Tujuan |
|---|---|---|---|
| `/ip/hotspot/user/print` | — | `hotspot/users.php`, `exportusers.php` | List semua user |
| `/ip/hotspot/user/print` | `count-only=""` | Banyak file | Hitung user |
| `/ip/hotspot/user/print` | `?name=<nama>` | `userbyname.php`, `status/status.php`, `quickuser.php` | User by nama |
| `/ip/hotspot/user/print` | `?.id=<id>` | `process/removehotspotuser.php`, `removeuseractive.php`, `resethotspotuser.php`, `userbyname.php` | User by `.id` |
| `/ip/hotspot/user/print` | `?profile=<profil>` | `hotspot/users.php` | Filter by profil |
| `/ip/hotspot/user/print` | `?comment=<comment>` | `hotspot/users.php`, `removehotspotuserbycomment.php`, `removeexpiredhotspotuser.php` | Filter by comment |
| `/ip/hotspot/user/add` | `server`, `name`, `password`, `profile`, `disabled=no`, `limit-uptime`, `limit-bytes-total`, `comment` | `adduser.php`, `generateuser.php`, `quickuser.php` | Tambah user baru |
| `/ip/hotspot/user/set` | `.id`, `server`, `name`, `password`, `profile`, `disabled`, `limit-uptime`, `limit-bytes-total`, `comment` | `userbyname.php` | Edit data user (full update) |
| `/ip/hotspot/user/set` | `.id`, `disabled=yes/no` | `process/disablehotspotuser.php`, `enablehotspotuser.php` | Enable/disable user |
| `/ip/hotspot/user/set` | `.id`, `limit-uptime=0`, `comment=""` | `process/resethotspotuser.php` | Reset uptime & comment |
| `/ip/hotspot/user/set` | `.id`, `comment=<expiry>` | On-login script (inline) | Set expiry date |
| `/ip/hotspot/user/set` | `.id`, `mac-address=<mac>` | On-login script (inline, opsional lock) | Lock user ke MAC |
| `/ip/hotspot/user/remove` | `.id` | `process/removehotspotuser.php`, `removehotspotuserbycomment.php`, `removeexpiredhotspotuser.php` | Hapus user |
| `/ip/hotspot/user/reset-counters` | `.id` | `process/resethotspotuser.php` | Reset counter bytes/uptime |

### 1.7 IP Hotspot — User Profile

| Command | Parameter | File | Tujuan |
|---|---|---|---|
| `/ip/hotspot/user/profile/print` | — | Banyak file | List semua profil |
| `/ip/hotspot/user/profile/print` | `?name=<nama>` | `adduserprofile.php`, `generateuser.php`, `listquickprint.php`, `getvalidprice.php` | Profil by nama |
| `/ip/hotspot/user/profile/print` | `?.id=<id>` | `hotspot/userprofilebyname.php` | Profil by ID |
| `/ip/hotspot/user/profile/add` | `name`, `address-pool`, `rate-limit`, `shared-users`, `status-autorefresh=1m`, `on-login`, `parent-queue` | `hotspot/adduserprofile.php` | Buat profil baru + inject on-login |
| `/ip/hotspot/user/profile/set` | `.id`, semua field profil + `on-login` baru | `hotspot/userprofilebyname.php` | Update profil |
| `/ip/hotspot/user/profile/remove` | `.id` | `process/removeuserprofile.php` | Hapus profil |

### 1.8 IP Hotspot — Active, Host, Cookie, Print

| Command | Parameter | File | Tujuan |
|---|---|---|---|
| `/ip/hotspot/print` | — | `adduser.php`, `generateuser.php`, `listquickprint.php` | List server hotspot |
| `/ip/hotspot/active/print` | — | `hotspot/hotspotactive.php` | List sesi aktif |
| `/ip/hotspot/active/print` | `count-only=""` | `dashboard/home.php`, `aload.php` | Hitung sesi aktif |
| `/ip/hotspot/active/print` | `?server=<nama>` | `hotspot/hotspotactive.php` | Filter sesi by server |
| `/ip/hotspot/active/print` | `count-only=""`, `?server=<nama>` | `hotspot/hotspotactive.php` | Hitung sesi per server |
| `/ip/hotspot/active/remove` | `.id` | `process/removeuseractive.php`, on-login bgservice | Kick sesi aktif |
| `/ip/hotspot/host/print` | — | `hotspot/hosts.php` | List hosts |
| `/ip/hotspot/host/print` | dengan filter search | `hotspot/hosts.php` | Filter hosts |
| `/ip/hotspot/host/remove` | `.id` | `process/removehost.php` | Hapus host |
| `/ip/hotspot/cookie/print` | — | `hotspot/cookies.php` | List cookies |
| `/ip/hotspot/cookie/print` | `count-only=""` | `hotspot/cookies.php` | Hitung cookies |
| `/ip/hotspot/cookie/print` | `?user=<username>` | `process/removeuseractive.php` | Cookie by user |
| `/ip/hotspot/cookie/remove` | `.id` | `process/removecookie.php`, `removeuseractive.php` | Hapus cookie |

### 1.9 IP Hotspot — IP Binding

| Command | Parameter | File | Tujuan |
|---|---|---|---|
| `/ip/hotspot/ip-binding/print` | — | `hotspot/ipbinding.php` | List IP binding |
| `/ip/hotspot/ip-binding/print` | `count-only=""` | `hotspot/ipbinding.php` | Hitung IP binding |
| `/ip/hotspot/ip-binding/set` | `.id`, `type=regular` / `type=bypassed` | `process/pipbinding.php` | Ubah tipe binding |
| `/ip/hotspot/ip-binding/set` | `.id`, `disabled=no` / `disabled=yes` | `process/pipbinding.php` | Enable/disable binding |
| `/ip/hotspot/ip-binding/remove` | `.id` | `process/pipbinding.php` | Hapus binding |

### 1.10 Queue, Pool, ARP, DHCP

| Command | Parameter | File | Tujuan |
|---|---|---|---|
| `/queue/simple/print` | `?dynamic=false` | `adduserprofile.php`, `userprofilebyname.php` | List queue static (parent-queue) |
| `/queue/simple/print` | `?name=<nama>` | `process/pipbinding.php` | Cari queue by nama |
| `/queue/simple/remove` | `.id` | `process/pipbinding.php` | Hapus queue (saat hapus binding) |
| `/ip/pool/print` | — | `adduserprofile.php`, `userprofilebyname.php` | List IP pool |
| `/ip/arp/print` | `?mac-address=<mac>` | `process/pipbinding.php` | Cari ARP entry by MAC |
| `/ip/arp/remove` | `.id` | `process/pipbinding.php` | Hapus ARP entry |
| `/ip/dhcp-server/lease/print` | — | `dhcp/dhcpleases.php` | List DHCP lease |
| `/ip/dhcp-server/lease/print` | `count-only=""` | `dhcp/dhcpleases.php` | Hitung lease |
| `/ip/dhcp-server/lease/print` | `?mac-address=<mac>` | `process/pipbinding.php` | Cari lease by MAC |
| `/ip/dhcp-server/lease/remove` | `.id` | `process/pipbinding.php` | Hapus lease (cleanup saat hapus binding) |

### 1.11 Interface & Traffic

| Command | Parameter | File | Tujuan |
|---|---|---|---|
| `/interface/print` | — | `dashboard/home.php`, `traffic/trafficmonitor.php` | List semua interface |
| `/interface/monitor-traffic` | `interface=<nama>`, `once=""` | `traffic/traffic.php` | Snapshot TX/RX bps real-time |

### 1.12 PPP (File Hilang — Disimpulkan dari routing)

Command ini **hampir pasti digunakan** di file yang hilang, berdasarkan nama variabel dan struktur `index.php`:

| Command (Inferensi) | Basis Inferensi |
|---|---|
| `/ppp/secret/print` | `$ppp == "secrets"` → `pppsecrets.php` |
| `/ppp/secret/add` | `$ppp == "addsecret"` → `addsecret.php` |
| `/ppp/secret/set` | `$secretbyname != ""` → `secretbyname.php` |
| `/ppp/secret/enable` / `/ppp/secret/set disabled=no` | `$enablesecr`, `psecret.php` |
| `/ppp/secret/remove` | `$removesecr`, `psecret.php` |
| `/ppp/profile/print` | `$ppp == "profiles"` → `pppprofile.php` |
| `/ppp/profile/add` | `$ppp == "add-profile"` → `addpppprofile.php` |
| `/ppp/profile/set` | `$ppp == "edit-profile"` → `profilebyname.php` |
| `/ppp/profile/remove` | `$removepprofile`, `removepprofile.php` |
| `/ppp/active/print` | `$ppp == "active"` → `pppactive.php` |
| `/ppp/active/remove` | `.id` | `process/removepactive.php` ← **file INI ADA** |

> `/ppp/active/remove` terkonfirmasi di `process/removepactive.php` yang **memang ada** di repo.

### 1.13 Login & Auth (Low-Level)

| Command | Penggunaan |
|---|---|
| `/login` | `lib/routeros_api.class.php` — otentikasi awal koneksi |
| `=name=<user>` | Parameter nama saat login |
| `=password=<pass>` | Password plaintext (ROS ≥ v6.43) |
| `=response=00<md5>` | MD5 challenge-response (ROS < v6.43) |

---

## 2. Parameter Khusus API (QueryFlags & Modifiers)

| Prefix | Arti | Contoh | Keterangan |
|---|---|---|---|
| `?key=val` | Filter (query) | `?name=john` | Hasil hanya row yang match |
| `?.id=val` | Filter by internal ID | `?.id=*1A` | Cari by `.id` spesifik |
| `?dynamic=false` | Filter boolean | Queue non-dynamic only | |
| `=.id=val` | Set target by ID | `=.id=*1A` | Digunakan di `set`/`remove` |
| `=.proplist=.id` | Projection — ambil field tertentu saja | Hanya return `.id` | Efisien untuk bulk delete |
| `count-only=""` | Hanya hitung, tidak return data | `count-only=""` | Lebih ringan dari print penuh |
| `once=""` | Snapshot sekali (stop setelah 1 sample) | `interface/monitor-traffic` | Tidak streaming |
| `?=source=val` | Filter syntax alternatif (lama) | `userlog.php` | Setara `?source=val` |

---

## 3. RouterOS Scripts Yang Tertanam di PHP

### 3.1 Script `on-login` — Kalkulasi & Set Expiry Date

Disuntik ke field `on-login` setiap User Profile. Berjalan **otomatis tiap user login** ke hotspot.

```routeros
# Metadata untuk dibaca PHP (parse expmode, price, validity, sprice, lock)
:put (",<expmode>,<price>,<validity>,<sprice>,,<lock>,");

{
  # Ambil comment user (cek apakah sudah ada expiry)
  :local comment [/ip hotspot user get [/ip hotspot user find where name="$user"] comment];
  :local ucode [:pic $comment 0 2];

  # Proses hanya jika kode "vc"(voucher), "up"(user/pass), atau kosong
  :if ($ucode = "vc" or $ucode = "up" or $comment = "") do={

    :local date  [/system clock get date];
    :local year  [:pick $date 7 11];
    :local month [:pick $date 0 3];

    # Trik kalkulasi expiry: buat scheduler sementara dengan interval=validity
    # lalu ambil next-run-nya = waktu expiry
    /sys sch add name="$user" disable=no start-date=$date interval="<validity>";
    :delay 5s;

    :local exp    [/sys sch get [/sys sch find where name="$user"] next-run];
    :local getxp  [len $exp];

    # Format tergantung panjang string next-run
    :if ($getxp = 15) do={
      # Format: "jan/02/2025 15:04:05" → perlu tambah tahun
      :local d [:pic $exp 0 6];
      :local t [:pic $exp 7 16];
      :local s ("/");
      :local exp ("$d$s$year $t");
      /ip hotspot user set comment="$exp" [find where name="$user"];
    };
    :if ($getxp = 8) do={
      # Format: "15:04:05" → hari ini + waktu
      /ip hotspot user set comment="$date $exp" [find where name="$user"];
    };
    :if ($getxp > 15) do={
      # Format sudah lengkap
      /ip hotspot user set comment="$exp" [find where name="$user"];
    };

    :delay 5s;
    /sys sch remove [find where name="$user"];
  }
}
```

**Variasi Lock MAC (jika "Lock User" = Enable):**
```routeros
# Ditambahkan di akhir on-login setelah blok expiry
:local mac $"mac-address";
/ip hotspot user set mac-address=$mac [find where name=$user]
```

**Variasi Record Transaksi (mode `remc` / `ntfc`):**
```routeros
# Script pencatatan → disimpan sebagai /system/script
:local mac  $"mac-address";
:local time [/system clock get time];

/system script add \
  name="<date>-|-<time>-|-<user>-|-<price>-|-<ip>-|-<mac>-|-<validity>-|-<profile>-|-<comment>" \
  owner="<bulan><tahun>" \
  source="<date>" \
  comment="mikhmon"
```

**Konvensi field pada nama script transaksi:**

| Index | Nilai | Contoh |
|---|---|---|
| `[0]` | Tanggal | `jan/05/2025` |
| `[1]` | Waktu | `14:32:01` |
| `[2]` | Username | `user001` |
| `[3]` | Harga | `5000` |
| `[4]` | IP address | `192.168.88.10` |
| `[5]` | MAC address | `AA:BB:CC:DD:EE:FF` |
| `[6]` | Validity | `1d` |
| `[7]` | Profile | `paket-1hari` |
| `[8]` | Comment | `kasir-andi` |

**Versi on-login untuk profil tanpa expiry (mode `0`):**
```routeros
# Hanya metadata price, tidak ada logika expiry
:put (",," . <price> . ",,,noexp," . <lock> . ",")
```

---

### 3.2 Script `on-event` Scheduler — Background Expiry Monitor

Disuntik ke `on-event` scheduler yang berjalan **tiap 2–5 menit** (interval random `00:02:XX`). Satu scheduler per profil.

```routeros
# Fungsi konversi tanggal MikroTik ke integer YYYYMMDD
:local dateint do={
  :local montharray ("jan","feb","mar","apr","may","jun","jul","aug","sep","oct","nov","dec");
  :local days  [:pick $d 4 6];
  :local month [:pick $d 0 3];
  :local year  [:pick $d 7 11];
  :local monthint ([:find $montharray $month]);
  :local month ($monthint + 1);
  :if ([len $month] = 1) do={
    :local zero ("0");
    :return [:tonum ("$year$zero$month$days")];
  } else={
    :return [:tonum ("$year$month$days")];
  }
};

# Fungsi konversi waktu HH:MM:SS ke total menit
:local timeint do={
  :local hours   [:pick $t 0 2];
  :local minutes [:pick $t 3 5];
  :return ($hours * 60 + $minutes);
};

# Waktu sekarang
:local date    [/system clock get date];
:local time    [/system clock get time];
:local today   [$dateint d=$date];
:local curtime [$timeint t=$time];

# Loop semua user dengan profil ini
:foreach i in [/ip hotspot user find where profile="<PROFILE_NAME>"] do={
  :local comment [/ip hotspot user get $i comment];
  :local name    [/ip hotspot user get $i name];
  :local gettime [:pic $comment 12 20];

  # Cek apakah comment berisi format date dd/mm/yyyy
  :if ([:pic $comment 3] = "/" and [:pic $comment 6] = "/") do={
    :local expd [$dateint d=$comment];
    :local expt [$timeint t=$gettime];

    # Kondisi expired (3 kasus):
    :if (
      ($expd < $today and $expt < $curtime) or   # hari lewat & jam lewat
      ($expd < $today and $expt > $curtime) or   # hari lewat & jam belum
      ($expd = $today and $expt < $curtime)       # hari sama tapi jam lewat
    ) do={
      # Aksi sesuai mode expired:
      # mode "rem"/"remc"  → /ip hotspot user remove $i
      # mode "ntf"/"ntfc"  → /ip hotspot user set limit-uptime=1s $i
      [/ip hotspot user <MODE> $i];
      [/ip hotspot active remove [find where user=$name]];
    };
  };
};
```

---

### 3.3 Script Inline di `on-login` — Command Ringkas

Command RouterOS yang digunakan langsung dalam string on-login (shorthand/inline):

| Command Inline | Tujuan |
|---|---|
| `/ip hotspot user find where name="$user"` | Cari ID user saat ini |
| `/ip hotspot user get $i comment` | Baca comment user |
| `/ip hotspot user get $i name` | Baca nama user |
| `/ip hotspot user set comment="$exp" [find where name="$user"]` | Tulis expiry ke comment |
| `/ip hotspot user set mac-address=$mac [find where name=$user]` | Lock ke MAC |
| `/ip hotspot user remove $i` | Hapus user expired (mode rem) |
| `/ip hotspot user set limit-uptime=1s $i` | Notice user expired (mode ntf) |
| `/ip hotspot active remove [find where user=$name]` | Kick sesi aktif |
| `/sys sch add name="$user" ...` | Scheduler temp kalkulasi expiry |
| `/sys sch get [/sys sch find where name="$user"] next-run` | Baca next-run (=expiry) |
| `/sys sch remove [find where name="$user"]` | Hapus scheduler temp |
| `/system clock get date` | Tanggal saat ini |
| `/system clock get time` | Waktu saat ini |
| `/system script add name=... owner=... source=... comment=...` | Simpan transaksi |

---

## 4. Alur Proses Lengkap

### 4.1 Delete User — Cascade Cleanup

Saat user dihapus (`removehotspotuser.php`), dilakukan **4 operasi berurutan**:
```
1. /ip/hotspot/user/print  (?.id=$uid)       → dapat nama user
2. /system/script/print   (?name=$name)      → cari script transaksi user
3. /system/scheduler/print (?name=$name)     → cari scheduler expiry user
4. /system/script/remove   (.id=$scr)        → hapus script
5. /system/scheduler/remove (.id=$sch)       → hapus scheduler
6. /ip/hotspot/user/remove  (.id=$uid)       → hapus user
```

**Bulk delete** (`removehotspotusers`) menggunakan separator `~` di URL, loop tiap ID.

### 4.2 Remove IP Binding — Full Cascade Cleanup

Saat IP binding dihapus (`process/pipbinding.php`), cleanup **5 resource** sekaligus:
```
1. /ip/hotspot/ip-binding/remove (.id)     → hapus binding
2. /queue/simple/print   (?name=$mac)      → cari queue
3. /queue/simple/remove  (.id)             → hapus queue
4. /system/scheduler/print (?name=$mac)   → cari scheduler
5. /system/scheduler/remove (.id)         → hapus scheduler
6. /ip/arp/print   (?mac-address=$mac)    → cari ARP entry
7. /ip/arp/remove  (.id)                  → hapus ARP
8. /ip/dhcp-server/lease/print (?mac-address=$mac) → cari lease
9. /ip/dhcp-server/lease/remove (.id)    → hapus lease
```

### 4.3 Delete User Profile — Cascade

```
1. /system/scheduler/print (?name=$profname)  → cari monitor scheduler
2. /ip/hotspot/user/profile/remove (.id)      → hapus profil
3. /system/scheduler/remove (.id)             → hapus monitor scheduler
```

### 4.4 Kick User Active — Cascade

```
1. /ip/hotspot/active/print  (?.id=$uid)     → dapat nama user
2. /ip/hotspot/cookie/print  (?user=$name)   → cari cookie
3. /ip/hotspot/cookie/remove (.id)           → hapus cookie
4. /ip/hotspot/active/remove (.id)           → kick sesi
```

---

## 5. Mode Expired Profile

| Mode | Kode | Aksi saat expired | Record transaksi |
|---|---|---|---|
| None | `0` | Tidak ada expiry | Tidak |
| Remove | `rem` | `/ip hotspot user remove` | Tidak |
| Notice | `ntf` | `/ip hotspot user set limit-uptime=1s` | Tidak |
| Remove & Record | `remc` | `/ip hotspot user remove` | **Ya** |
| Notice & Record | `ntfc` | `/ip hotspot user set limit-uptime=1s` | **Ya** |

---

## 6. RouterOS API Class — Detail Teknis

File: `lib/routeros_api.class.php` (v1.6 oleh Denis Basta)

| Property | Default | Keterangan |
|---|---|---|
| `port` | `8728` | Port API biasa |
| `ssl` | `false` | Bisa diubah ke `true` → port `8729` |
| `timeout` | `3` detik | Timeout koneksi & baca |
| `attempts` | `5` kali | Jumlah retry connect |
| `delay` | `3` detik | Jeda antar retry |

**Dua metode login yang didukung:**
- **Post-v6.43:** Kirim password plaintext, langsung `!done`
- **Pre-v6.43:** Challenge-response MD5 (`/login` → dapat `ret` token → MD5 hash)

**Metode utama:**
- `connect($ip, $login, $password)` — koneksi + login
- `comm($path, $arr)` — shorthand: write + read sekaligus
- `write($command, $param2)` — kirim satu word (false = lanjut, true = end sentence)
- `read($parse)` — baca response, parse jika `true`
- `disconnect()` — tutup socket

**Enkripsi password config:**
Mikhmon menyimpan password dengan fungsi `encrypt()`/`decrypt()` custom berbasis XOR + base64. **Bukan enkripsi yang kuat** — hanya obfuskasi.

---

## 7. Quick Print System

Konfigurasi template cetak voucher disimpan sebagai `/system/script` di router (comment `QuickPrintMikhmon`).

**Format field `source`:**
```
#<nama_paket>#<server>#<usermode>#<panjang>#<prefix>#<charset>#<profile>#<timelimit>#<datalimit>#<comment>#<validity>#<price>_<sprice>#<lock>
```

| Index | Field | Nilai Valid |
|---|---|---|
| `[1]` | Nama paket | string |
| `[2]` | Server hotspot | nama server / "all" |
| `[3]` | User mode | `up` (user/pass), `vc` (voucher code) |
| `[4]` | Panjang karakter | angka |
| `[5]` | Prefix username | string |
| `[6]` | Charset | `lower`, `upper`, `mixed`, `number`, dll |
| `[7]` | Profile | nama profil |
| `[8]` | Time limit | format RouterOS `1h`, `1d`, dll |
| `[9]` | Data limit | bytes |
| `[10]` | Comment default | string |
| `[11]` | Validity | format RouterOS |
| `[12]` | `price_sprice` | `5000_4500` |
| `[13]` | Lock user | `Enable`/`Disable` |

---

## 8. Ringkasan: Apa Yang Hilang di Analisis Pertama

| Temuan Baru | Penjelasan |
|---|---|
| **12 file missing (PPP)** | Routing ada tapi file tidak ada di repo — kemungkinan fitur belum diupload |
| **`/ip/hotspot/ip-binding/set` untuk enable/disable** | Selain set `type`, juga set `disabled=yes/no` |
| **Filter `?.id=<val>`** | Selain `?name=`, ada filter by `.id` langsung |
| **`=.proplist=.id`** | Proyeksi field — hanya fetch `.id` untuk efisiensi bulk delete |
| **`?=source=`/`?=owner=`** | Syntax filter alternatif (versi lama) di `userlog.php` |
| **Bulk delete via `~` separator** | Multi-ID delete: `uid1~uid2~uid3` di URL |
| **Cascade delete binding** | Hapus 9 resource sekaligus saat remove IP binding |
| **Login dual-mode (pre/post v6.43)** | Detail login mechanism di `routeros_api.class.php` |
| **Enkripsi password XOR custom** | Password config bukan AES, hanya XOR+base64 |
| **`/ip/hotspot/active/print` dengan filter gabungan** | `count-only` + `?server` bersamaan |
| **`/system/script/print` dengan `?comment=mikhmon`** | Filter ke semua transaksi (view all report) |
| **`/ppp/active/remove`** | Satu-satunya PPP command yang file-nya ADA |
