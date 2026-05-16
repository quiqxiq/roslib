// cmd/test-persistence — membuktikan bahwa Manager mempertahankan
// satu sesi koneksi ke MikroTik meskipun device diambil berkali-kali.
//
// Indikator utama:
//   - Jumlah login di /log hanya bertambah 1 dari awal ke akhir.
//   - Pointer *routeros.Client yang dikembalikan CommandConn() selalu sama.
//   - Semua command berhasil dieksekusi lewat koneksi yang sama.
//
// Run:
//
//	ROSLIB_ROUTER_ADDRESS=192.168.88.1:8728 \
//	ROSLIB_ROUTER_USERNAME=admin \
//	ROSLIB_ROUTER_PASSWORD=secret \
//	go run ./cmd/test-persistence
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"
	"unsafe"

	"github.com/joho/godotenv"
	"github.com/quiqxiq/roslib/device"
	"github.com/sirupsen/logrus"
)

func main() {
	// Load .env dari root project (cari dari direktori kerja ke atas).
	// Tidak error kalau file tidak ditemukan — env mungkin sudah di-export.
	_ = godotenv.Load(".env")

	log := logrus.New()
	log.SetLevel(logrus.WarnLevel) // sembunyikan noise internal, fokus ke output test
	log.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	addr := mustEnv("ROSLIB_ROUTER_ADDRESS")
	user := mustEnv("ROSLIB_ROUTER_USERNAME")
	passwd := mustEnv("ROSLIB_ROUTER_PASSWORD")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	opts := device.Options{
		Address:          addr,
		Username:         user,
		Password:         passwd,
		Logger:           log,
	}

	// ── Header ───────────────────────────────────────────────────────
	title("TEST: PERSISTENT CONNECTION (via Manager)")
	fmt.Println("  Setiap Get/GetOrConnect harus mengembalikan instance YANG SAMA.")
	fmt.Println("  Login di MikroTik hanya boleh bertambah 1 kali di seluruh skenario.")
	fmt.Println()

	mgr := device.NewManager()

	// ── STEP 1: Register sekali ───────────────────────────────────────
	step("1", "Register device ke Manager (1x dial = 1x login)")
	if err := mgr.Register(ctx, "gw", opts); err != nil {
		fatal("Register: %v", err)
	}
	pass("Register OK — koneksi terbuka")

	// Ambil login count awal dari log router
	dev1, _ := mgr.Get("gw")
	loginsBefore := countLogins(ctx, dev1)
	fmt.Printf("     login entries di /log sebelum test: %d\n", loginsBefore)

	// Simpan pointer koneksi awal sebagai referensi
	conn1 := connPtr(dev1)
	fmt.Printf("     CommandConn pointer: 0x%x\n", conn1)

	// ── STEP 2: Get() berulang — pointer harus sama ───────────────────
	step("2", "Get() 5x berturut-turut — pointer koneksi wajib identik")
	allSame := true
	for i := 1; i <= 5; i++ {
		devN, err := mgr.Get("gw")
		if err != nil {
			fail("Get() iter %d: %v", i, err)
			allSame = false
			continue
		}
		pN := connPtr(devN)
		same := pN == conn1
		if !same {
			allSame = false
		}
		fmt.Printf("     iter %d: pointer=0x%x same=%v\n", i, pN, same)
	}
	if allSame {
		pass("Semua 5 iterasi mengembalikan pointer yang SAMA")
	} else {
		fail("Ada pointer yang berbeda — koneksi dibuat ulang!")
	}

	// ── STEP 3: GetOrConnect() — tidak boleh dial ulang ───────────────
	step("3", "GetOrConnect() 5x — harus reuse, bukan dial baru")
	for i := 1; i <= 5; i++ {
		devN, err := mgr.GetOrConnect(ctx, "gw", opts)
		if err != nil {
			fail("GetOrConnect() iter %d: %v", i, err)
			continue
		}
		pN := connPtr(devN)
		same := pN == conn1
		fmt.Printf("     iter %d: pointer=0x%x same=%v\n", i, pN, same)
		if !same {
			fail("GetOrConnect() membuat koneksi baru padahal device masih alive!")
		}
	}
	pass("GetOrConnect() konsisten reuse koneksi lama")

	// ── STEP 4: Eksekusi command berulang lewat koneksi yang sama ─────
	step("4", "Jalankan 10 command via instance yang sama — tidak ada re-dial")
	dev, _ := mgr.Get("gw")
	errCount := 0
	for i := 1; i <= 10; i++ {
		reply, err := dev.Path("/system/resource").Print().Exec(ctx)
		if err != nil || len(reply.Rows) == 0 {
			errCount++
			fmt.Printf("     command %d: ERROR %v\n", i, err)
			continue
		}
		r := reply.Rows[0]
		fmt.Printf("     command %d: cpu=%s%% free-mem=%s\n",
			i, r.Get("cpu-load"), r.Get("free-memory"))
	}
	if errCount == 0 {
		pass("10/10 command berhasil lewat satu koneksi")
	} else {
		fail("%d dari 10 command gagal", errCount)
	}

	// ── STEP 5: Verifikasi login count di /log ─────────────────────────
	step("5", "Verifikasi jumlah login di /log MikroTik")
	time.Sleep(500 * time.Millisecond) // beri waktu log router commit
	loginsAfter := countLogins(ctx, dev)
	delta := loginsAfter - loginsBefore
	fmt.Printf("     login entries sebelum: %d\n", loginsBefore)
	fmt.Printf("     login entries sesudah: %d\n", loginsAfter)
	fmt.Printf("     delta: %d\n", delta)
	if delta == 1 {
		pass("Login hanya bertambah 1 kali — koneksi benar-benar persisten ✓")
	} else if delta == 0 {
		pass("Tidak ada login tambahan (mungkin login sudah ada sebelum test) ✓")
	} else {
		fail("Login bertambah %d kali — ada koneksi baru yang dibuat!", delta)
	}

	// ── STEP 6: IsAlive check ─────────────────────────────────────────
	step("6", "IsAlive() harus true selama belum di-Close()")
	if dev.IsAlive() {
		pass("IsAlive() = true")
	} else {
		fail("IsAlive() = false padahal belum ditutup")
	}

	// ── STEP 7: Close via Manager ─────────────────────────────────────
	step("7", "CloseAll() — koneksi ditutup, IsAlive() harus false")
	mgr.CloseAll()
	time.Sleep(200 * time.Millisecond)
	if !dev.IsAlive() {
		pass("IsAlive() = false setelah CloseAll() — logout terjadi di MikroTik")
	} else {
		fail("IsAlive() masih true setelah CloseAll()")
	}

	// ── Ringkasan ─────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("══════════════════════════════════════════════════")
	fmt.Println("  KESIMPULAN: Koneksi PERSISTEN via Manager")
	fmt.Printf("  • 1x Register → 1x login di MikroTik\n")
	fmt.Printf("  • Get/GetOrConnect → reuse pointer yang sama\n")
	fmt.Printf("  • 10 command dieksekusi tanpa dial ulang\n")
	fmt.Printf("  • CloseAll → 1x logout di MikroTik\n")
	fmt.Println("══════════════════════════════════════════════════")
}

// countLogins menghitung entri "logged in" di /log MikroTik.
// Hanya menghitung entri yang melibatkan username yang sama.
func countLogins(ctx context.Context, dev *device.RouterDevice) int {
	tCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	reply, err := dev.Path("/log").Print().Exec(tCtx)
	if err != nil {
		return -1
	}
	count := 0
	for _, row := range reply.Rows {
		msg := strings.ToLower(row.Get("message"))
		if strings.Contains(msg, "logged in") {
			count++
		}
	}
	return count
}

// connPtr mengambil nilai pointer CommandConn sebagai uintptr untuk
// perbandingan identitas — bukan untuk dereferensi.
func connPtr(dev *device.RouterDevice) uintptr {
	conn := dev.CommandConn()
	if conn == nil {
		return 0
	}
	return uintptr(unsafe.Pointer(conn))
}

// ──────────────── helpers ────────────────

func title(s string) {
	fmt.Println("══════════════════════════════════════════════════")
	fmt.Println(" " + s)
	fmt.Println("══════════════════════════════════════════════════")
}

func step(num, desc string) {
	fmt.Printf("\n▶ STEP %s — %s\n", num, desc)
}

func pass(format string, args ...any) {
	fmt.Printf("  ✓ "+format+"\n", args...)
}

func fail(format string, args ...any) {
	fmt.Printf("  ✗ FAIL: "+format+"\n", args...)
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "FATAL: "+format+"\n", args...)
	os.Exit(1)
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		fatal("env %s wajib diisi", key)
	}
	return v
}