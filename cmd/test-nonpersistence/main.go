// cmd/test-nonpersistence — membuktikan bahwa memanggil device.New()
// langsung (tanpa Manager) setiap kali butuh koneksi akan selalu
// membuat sesi baru ke MikroTik.
//
// Ini adalah pola LAMA yang bermasalah. Test ini berfungsi sebagai
// baseline pembanding terhadap cmd/test-persistence.
//
// Indikator utama:
//   - Setiap New() menambah 1 entry "logged in" di /log MikroTik.
//   - Pointer *routeros.Client selalu berbeda di setiap panggilan.
//   - N iterasi = N login di MikroTik.
//
// Run:
//
//	ROSLIB_ROUTER_ADDRESS=192.168.88.1:8728 \
//	ROSLIB_ROUTER_USERNAME=admin \
//	ROSLIB_ROUTER_PASSWORD=secret \
//	go run ./cmd/test-nonpersistence
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

const iterations = 5 // jumlah kali New() dipanggil ulang

func main() {
	_ = godotenv.Load(".env")

	log := logrus.New()
	log.SetLevel(logrus.WarnLevel)
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
		StrictCapability: false,
	}

	// ── Header ───────────────────────────────────────────────────────
	title("TEST: NON-PERSISTENT CONNECTION (New() langsung, tanpa Manager)")
	fmt.Printf("  Setiap iterasi memanggil device.New() → dial baru → login baru.\n")
	fmt.Printf("  Dijalankan %d iterasi — diharapkan ada %d login di /log.\n", iterations, iterations)
	fmt.Println()

	// ── Koneksi awal hanya untuk baca login count baseline ────────────
	step("0", "Buka koneksi sementara untuk membaca baseline /log")
	baseline, err := device.New(ctx, opts)
	if err != nil {
		fatal("dial baseline: %v", err)
	}
	loginsBefore := countLogins(ctx, baseline)
	fmt.Printf("     login entries di /log sebelum test: %d\n", loginsBefore)
	baselinePtr := connPtr(baseline)
	fmt.Printf("     pointer baseline: 0x%x\n", baselinePtr)
	_ = baseline.Close() // tutup baseline, ini sendiri sudah +1 login

	// Hitung ulang setelah baseline ditutup agar angka bersih
	time.Sleep(300 * time.Millisecond)

	// ── STEP 1: Simulasi pola "New() setiap request" ──────────────────
	step("1", fmt.Sprintf("Simulasi %d 'request' — masing-masing panggil New()", iterations))

	prevPtr := uintptr(0)
	loginsDuringTest := 0

	for i := 1; i <= iterations; i++ {
		fmt.Printf("\n  [iterasi %d/%d]\n", i, iterations)

		// Pola lama: setiap kali butuh device, New() dipanggil
		dev, nerr := device.New(ctx, opts)
		if nerr != nil {
			fail("New() iterasi %d: %v", i, nerr)
			continue
		}

		curPtr := connPtr(dev)
		isNew := curPtr != prevPtr
		fmt.Printf("     New() pointer: 0x%x\n", curPtr)
		fmt.Printf("     Koneksi BARU: %v\n", isNew)
		if !isNew && prevPtr != 0 {
			warn("Pointer SAMA — mungkin Go reuse alamat memori (masih buat dial baru)")
		}
		prevPtr = curPtr

		// Eksekusi 1 command — lalu BUANG device-nya (pola lama yang salah)
		reply, rerr := dev.Path("/system/resource").Print().Exec(ctx)
		if rerr != nil || len(reply.Rows) == 0 {
			fail("command iterasi %d: %v", i, rerr)
		} else {
			r := reply.Rows[0]
			fmt.Printf("     command OK: cpu=%s%% free-mem=%s\n",
				r.Get("cpu-load"), r.Get("free-memory"))
		}

		// Hitung login yang masuk selama iterasi ini
		logins := countLogins(ctx, dev)
		loginsThisIter := logins - loginsBefore - loginsDuringTest
		loginsDuringTest += loginsThisIter
		fmt.Printf("     login entries terdeteksi saat ini: +%d (total delta: %d)\n",
			loginsThisIter, loginsDuringTest)

		_ = dev.Close() // dev dibuang setelah satu pakai → logout
		time.Sleep(300 * time.Millisecond) // beri router waktu tulis log
	}

	// ── STEP 2: Baca total login delta ────────────────────────────────
	step("2", "Verifikasi total login yang terjadi selama test")

	// Buka koneksi baru untuk baca log akhir
	reader, rerr := device.New(ctx, opts)
	if rerr != nil {
		fatal("dial reader: %v", rerr)
	}
	defer reader.Close()

	time.Sleep(500 * time.Millisecond)
	loginsAfter := countLogins(ctx, reader)

	// +1 untuk baseline sendiri, +1 untuk reader ini
	// Delta murni dari iterasi adalah loginsAfter - loginsBefore - 2
	deltaRaw := loginsAfter - loginsBefore
	fmt.Printf("     login entries sebelum test: %d\n", loginsBefore)
	fmt.Printf("     login entries sesudah test:  %d\n", loginsAfter)
	fmt.Printf("     total delta (termasuk baseline+reader): %d\n", deltaRaw)
	fmt.Printf("     login murni dari %d iterasi: %d\n", iterations, deltaRaw-2)

	if deltaRaw >= iterations {
		warn("Setiap iterasi memang membuat login baru — ini membuktikan TIDAK PERSISTEN")
		fmt.Printf("     (%d iterasi → setidaknya %d login tambahan)\n", iterations, iterations)
	} else {
		note("Delta lebih kecil dari iterasi — mungkin log /log sudah ter-scroll atau\n"+
			"     router memiliki buffer terbatas. Lihat pointer check di atas untuk konfirmasi.")
	}

	// ── STEP 3: Pointer uniqueness summary ────────────────────────────
	step("3", "Kesimpulan pointer — setiap New() menghasilkan instance terpisah")
	fmt.Printf("     Setiap New() mengalokasi struct baru di heap → pointer selalu berbeda.\n")
	fmt.Printf("     (Go bisa reuse alamat setelah GC, tapi sesi TCP tetap unik.)\n")
	fmt.Printf("     Periksa log MikroTik: System > Log > filter 'logged in' untuk konfirmasi.\n")

	// ── Ringkasan ─────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("══════════════════════════════════════════════════")
	fmt.Println("  KESIMPULAN: Koneksi TIDAK PERSISTEN (pola lama)")
	fmt.Printf("  • %d iterasi New() = %d+ login di MikroTik\n", iterations, iterations)
	fmt.Printf("  • Setiap command membuka + menutup sesi baru\n")
	fmt.Printf("  • Overhead: TCP handshake + /login per request\n")
	fmt.Printf("  • Solusi: gunakan device.NewManager() — lihat cmd/test-persistence\n")
	fmt.Println("══════════════════════════════════════════════════")
}

// countLogins menghitung entri "logged in" di /log MikroTik.
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

func warn(format string, args ...any) {
	fmt.Printf("  ⚠ "+format+"\n", args...)
}

func note(format string, args ...any) {
	fmt.Printf("  ℹ "+format+"\n", args...)
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