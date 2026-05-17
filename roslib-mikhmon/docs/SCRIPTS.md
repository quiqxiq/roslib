# Scripts Mapping

Peta dari analisis §3 (RouterOS Scripts Yang Tertanam di PHP) dan §7 (Quick Print) ke generator Go.

## §3.1 on-login Script

Generator: `scripts/onlogin`

```go
import "github.com/quiqxiq/roslib-mikhmon/scripts/onlogin"
import "github.com/quiqxiq/roslib-mikhmon/domain"

s := onlogin.Build(onlogin.Options{
    Mode:      domain.ModeNoticeRecord,
    Validity:  "30d",
    Price:     25000,
    SellPrice: 20000,
    LockMAC:   true,
})
// s siap dipakai sebagai value =on-login=<s> saat hot.ProfileAdd / hot.ProfileSet.
```

Mode yang didukung:

| Mode | Konstanta | Behavior |
|---|---|---|
| None | `domain.ModeNone` | hanya metadata `:put`, no expiry |
| Remove | `domain.ModeRemove` | metadata + expiry calc |
| Notice | `domain.ModeNotice` | metadata + expiry calc |
| RemoveRecord | `domain.ModeRemoveRecord` | + record transaksi |
| NoticeRecord | `domain.ModeNoticeRecord` | + record transaksi |

`LockMAC=true` → tambah blok `:local mac $"mac-address"; /ip hotspot user set mac-address=$mac …`.

Untuk record transaksi, hasil mengandung placeholder `<PROFILE>` dan `<MONTHYEAR>` yang harus di-substitusi pakai `onlogin.PostProcessNamePlaceholders(s, profileName, monthYear)`. Caller umumnya tahu profile name saat panggil ProfileAdd, dan month/year dari `rosfmt.CurrentMonthOwner(time.Now())`.

Output golden file: `scripts/onlogin/testdata/golden/<mode>.txt`.

## §3.2 on-event Scheduler Script

Generator: `scripts/onevent`

```go
import "github.com/quiqxiq/roslib-mikhmon/scripts/onevent"

s := onevent.Build(onevent.Options{
    ProfileName: "1day",
    Mode:        domain.ModeRemove,
})
// s dipasang sebagai value =on-event=<s> saat sys.SchedulerAdd.
```

Output adalah body kalkulasi expired loop (lihat analisis §3.2). Action di dalam loop dipilih berdasarkan Mode (`/ip hotspot user remove` vs `/ip hotspot user set limit-uptime=1s`).

## §3.1 Transaction Name Format

Generator: `scripts/transaction`

```go
import "github.com/quiqxiq/roslib-mikhmon/scripts/transaction"

name := transaction.Format(domain.TransactionRecord{
    Date: "jan/05/2025", Time: "14:32:01",
    User: "u1", Price: "5000",
    IP: "10.0.0.1", MAC: "AA:BB:CC:DD:EE:FF",
    Validity: "1d", Profile: "1day", Comment: "kasir-andi",
})
// → "jan/05/2025-|-14:32:01-|-u1-|-5000-|-10.0.0.1-|-AA:BB:CC:DD:EE:FF-|-1d-|-1day-|-kasir-andi"

rec, err := transaction.Parse(name)
```

Format: 9 field dipisah `-|-`. Lihat tabel di analisis §3.1.

## §7 Quick Print Source Format

Generator: `scripts/quickprint`

```go
import "github.com/quiqxiq/roslib-mikhmon/scripts/quickprint"

src := quickprint.Format(quickprint.Config{
    Name: "1day", Server: "all", UserMode: "vc",
    Length: 8, Prefix: "u", Charset: domain.CharsetMixed,
    Profile: "1day-prof", TimeLimit: "1d", DataLimit: 0,
    Comment: "kasir", Validity: "30d",
    Price: 5000, SellPrice: 4500, LockMAC: true,
})
// → "#1day#all#vc#8#u#mixed#1day-prof#1d#0#kasir#30d#5000_4500#Enable"

cfg, err := quickprint.Parse(src)
```

Disimpan di RouterOS sebagai `/system/script` dengan `name=<NamaPaket>` dan `comment=QuickPrintMikhmon` (gunakan `system.CommentQuickPrint`).
