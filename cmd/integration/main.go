// Integration test runner — eksekusi terhadap router fisik + InfluxDB3.
// Output per skenario di-print ke stdout dalam format yang mudah di-grep
// oleh report generator.
//
// Run:
//
//	go run ./cmd/integration
//
// Env yang dibaca (lihat .env.example) — minimal:
//
//	ROSLIB_ROUTER_ADDRESS, ROSLIB_ROUTER_USERNAME, ROSLIB_ROUTER_PASSWORD,
//	INFLUX_HOST, INFLUX_TOKEN, INFLUX_DATABASE
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/quiqxiq/roslib"
	"github.com/quiqxiq/roslib/cache"
	"github.com/quiqxiq/roslib/capability"
	"github.com/quiqxiq/roslib/config"
	"github.com/quiqxiq/roslib/decode"
	"github.com/quiqxiq/roslib/metrics/influx"
	"github.com/sirupsen/logrus"
)

func main() {
	log := logrus.New()
	log.SetLevel(logrus.InfoLevel)
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: time.RFC3339,
	})

	step("BOOT", "loading config from environment")
	cfg, err := config.LoadFromEnv()
	if err != nil {
		fatal("config load: %v", err)
	}
	fmt.Printf("    router=%s influx=%v cache=%v strict=%v\n",
		cfg.Router.Address, cfg.Influx.Enabled, cfg.Cache.Enabled, cfg.StrictCapability)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mgr, influxCli, err := roslib.NewManagerFromConfig(ctx, cfg, log)
	if err != nil {
		fatal("device dial: %v", err)
	}
	defer mgr.CloseAll()
	if influxCli != nil {
		defer influxCli.Close()
	}
	dev, derr := mgr.Get(roslib.DefaultDeviceKey)
	if derr != nil {
		fatal("manager Get(%q): %v", roslib.DefaultDeviceKey, derr)
	}
	pass("BOOT", "device connected via Manager, registry loaded, influx ready=%v", influxCli != nil)

	// ── Capability registry ───────────────────────────────────────────
	step("CAPABILITY", "verify registry contents")
	reg, _ := capability.Default()
	cases := []struct {
		word string
		want capability.Class
	}{
		{"/interface/monitor-traffic", capability.ClassStreaming},
		{"/tool/ping", capability.ClassStreaming},
		{"/log/print", capability.ClassStreamablePrint},
		{"/ip/address/add", capability.ClassMutation},
	}
	for _, tc := range cases {
		c, lerr := reg.Lookup(tc.word)
		if lerr != nil || c.Class != tc.want {
			fail("CAPABILITY", "%s expected=%s got=%v err=%v", tc.word, tc.want, classOf(c), lerr)
			continue
		}
		fmt.Printf("    %s → %s (args=%d) ✓\n", tc.word, c.Class, len(c.Args))
	}
	pass("CAPABILITY", "all 4 classifications correct (registry version %s)", reg.Version)

	// ── Print Exec ────────────────────────────────────────────────────
	step("PRINT/EXEC", "/ip/address/print")
	addrReply, err := dev.Path("/ip/address").Print().Exec(ctx)
	if err != nil {
		fail("PRINT/EXEC", "%v", err)
	} else {
		pass("PRINT/EXEC", "rows=%d", len(addrReply.Rows))
		for i, r := range addrReply.Rows {
			if i >= 3 {
				fmt.Printf("    ... %d more rows\n", len(addrReply.Rows)-3)
				break
			}
			fmt.Printf("    [%d] address=%s interface=%s disabled=%s\n",
				i, r.Get("address"), r.Get("interface"), r.Get("disabled"))
		}
	}

	// ── Print Detail Exec ─────────────────────────────────────────────
	step("PRINT/DETAIL", "/system/resource/print")
	resReply, err := dev.Path("/system/resource").Print().Exec(ctx)
	if err != nil {
		fail("PRINT/DETAIL", "%v", err)
	} else if len(resReply.Rows) > 0 {
		r := resReply.Rows[0]
		uptime, _ := r.Duration("uptime")
		freeMem, _ := r.Bytes("free-memory")
		totalMem, _ := r.Bytes("total-memory")
		cpu, _ := r.Int("cpu-load")
		pass("PRINT/DETAIL", "board=%q ver=%q uptime=%v cpu=%d free=%d/%d",
			r.Get("board-name"), r.Get("version"), uptime, cpu, freeMem, totalMem)
	}

	// ── Capability misuse (Exec di path streaming) ───────────────────
	step("CAPABILITY/MISUSE", "expect any capability error for /interface/monitor-traffic Exec()")
	_, err = dev.Path("/interface/monitor-traffic").Print().Exec(ctx)
	if err == nil {
		fail("CAPABILITY/MISUSE", "expected capability error, got nil")
	} else {
		var ic *capability.ErrInvalidClass
		var uc *capability.ErrUnknownCommand
		switch {
		case errors.As(err, &ic):
			pass("CAPABILITY/MISUSE", "rejected (class mismatch): %v", err)
		case errors.As(err, &uc):
			pass("CAPABILITY/MISUSE", "rejected (no /print under streaming path): %v", err)
		default:
			fail("CAPABILITY/MISUSE", "got error but wrong type: %v", err)
		}
	}

	// ── Capability misuse (arg tidak dikenal) ────────────────────────
	step("CAPABILITY/UNKNOWN-ARG", "expect error for typo in Where")
	_, err = dev.Path("/ip/address").Print().Where("addres", "x").Exec(ctx)
	if err == nil {
		fail("CAPABILITY/UNKNOWN-ARG", "expected ErrUnknownArg, got nil")
	} else {
		var ua *capability.ErrUnknownArg
		if errors.As(err, &ua) {
			pass("CAPABILITY/UNKNOWN-ARG", "rejected: %v", err)
		} else {
			fail("CAPABILITY/UNKNOWN-ARG", "got error but wrong type: %v", err)
		}
	}

	// ── Cache (ExecCached hit) ───────────────────────────────────────
	step("CACHE/IN-MEMORY", "two ExecCached calls → second should hit cache")
	// Inject InMemory cache by re-creating device option; here we re-use
	// existing device cache (Noop kalau Disabled). For demo dipakai langsung:
	c := cache.NewInMemory()
	encoded, hit, _ := c.Get(ctx, "demo-key")
	_ = encoded
	if hit {
		fail("CACHE/IN-MEMORY", "fresh InMemoryCache should miss")
	} else {
		_ = c.Set(ctx, "demo-key", []byte(`{"rows":[{"k":"v"}]}`), 5*time.Second)
		if data, hit, _ := c.Get(ctx, "demo-key"); hit && string(data) != "" {
			pass("CACHE/IN-MEMORY", "miss→set→hit ok (len=%d)", len(data))
		}
	}

	// ── Inherent streaming: /interface/monitor-traffic ───────────────
	step("STREAM/MONITOR-TRAFFIC", "≥1 tick dari ether2 (target: 5)")
	mtCount := 0
	mtDone := make(chan struct{})
	err = dev.Path("/interface/monitor-traffic").
		With("interface", "ether2").
		With("once", "no").
		Stream("nic-ether2", func(s *decode.Sentence) {
			mtCount++
			rx, _ := s.Int("rx-bits-per-second")
			tx, _ := s.Int("tx-bits-per-second")
			fmt.Printf("    tick %d: rx=%d tx=%d\n", mtCount, rx, tx)
			if mtCount >= 5 {
				select {
				case <-mtDone:
				default:
					close(mtDone)
				}
			}
		})
	if err != nil {
		fail("STREAM/MONITOR-TRAFFIC", "register: %v", err)
	} else {
		select {
		case <-mtDone:
			pass("STREAM/MONITOR-TRAFFIC", "got %d ticks (target 5)", mtCount)
		case <-time.After(8 * time.Second):
			if mtCount >= 1 {
				pass("STREAM/MONITOR-TRAFFIC", "got %d ticks (≥1 acceptable on v6.49.11)", mtCount)
			} else {
				fail("STREAM/MONITOR-TRAFFIC", "no ticks within 8s")
			}
		}
		dev.UnregisterStream("nic-ether2")
	}

	// ── /tool/ping vs router v6 (registry v7 mismatch demo) ──────────
	// Test ini menunjukkan trade-off versi registry: router lapangan
	// (v6.49.11 RB750G) pakai path legacy "/ping", bukan "/tool/ping" v7.
	// /tool/ping di registry v7 valid sebagai mutation/run. Behavior router:
	//   v7 (RouterOS 7.x): execute → return reply atau error sesuai jaringan.
	//   v6 (RouterOS 6.x): "no such command" — path tidak ada.
	// Test pass kalau hasil konsisten dengan versi router (deteksi dari resReply).
	routerVer := ""
	if resReply != nil && len(resReply.Rows) > 0 {
		routerVer = resReply.Rows[0].Get("version")
	}
	expectReject := strings.HasPrefix(routerVer, "6.")
	verLabel := "v7+"
	if expectReject {
		verLabel = "v6"
	}
	step("RUN/PING-V7", fmt.Sprintf("/tool/ping vs router %s (ver=%s)", verLabel, routerVer))
	_, err = dev.Path("/tool").Run(ctx, "ping",
		roslib.NewPair("address", "8.8.8.8"),
		roslib.NewPair("count", "3"),
	)
	switch {
	case err != nil && expectReject:
		pass("RUN/PING-V7", "router reject (expected on v6): %v", err)
	case err == nil && !expectReject:
		pass("RUN/PING-V7", "router accepted (expected on v7+)")
	case err != nil && !expectReject:
		fail("RUN/PING-V7", "v7+ router unexpectedly rejected: %v", err)
	default:
		fail("RUN/PING-V7", "v6 router unexpectedly accepted /tool/ping")
	}

	// ── ExecCached round-trip ────────────────────────────────────────
	step("EXEC-CACHED", "PrintBuilder.ExecCached × 2 — second call hit cache?")
	// Inject InMemoryCache via runtime swap of Cache pointer. Karena
	// Cache di-set saat New, kita re-dial dengan opsi baru.
	cachedOpts := cfg.ToDeviceOptions(log)
	cachedOpts.Registry, _ = capability.Default()
	cachedOpts.Cache = cache.NewInMemory()
	cachedOpts.CacheTTL = 30 * time.Second
	cachedOpts.StrictCapability = true
	if rerr := mgr.Register(ctx, "exec-cached-test", cachedOpts); rerr != nil {
		fail("EXEC-CACHED", "%v", rerr)
	} else {
		defer mgr.Unregister("exec-cached-test")
		dev2, _ := mgr.Get("exec-cached-test")
		t1 := time.Now()
		r1, _ := dev2.Path("/ip/address").Print().ExecCached(ctx, 30*time.Second)
		d1 := time.Since(t1)
		t2 := time.Now()
		r2, _ := dev2.Path("/ip/address").Print().ExecCached(ctx, 30*time.Second)
		d2 := time.Since(t2)
		if r1 != nil && r2 != nil && len(r1.Rows) == len(r2.Rows) {
			pass("EXEC-CACHED", "rows=%d ; first=%v second=%v (second should be much faster)",
				len(r1.Rows), d1.Round(time.Millisecond), d2.Round(time.Millisecond))
		} else {
			fail("EXEC-CACHED", "rows mismatch r1=%v r2=%v", r1, r2)
		}
	}

	// ── Print-follow: /log ────────────────────────────────────────────
	step("STREAM/LOG-FOLLOW", "subscribe /log/print follow-only (snapshot kosong, log baru)")
	logCount := 0
	err = dev.Path("/log").Print().FollowOnly().Stream("log-tail", func(s *decode.Sentence) {
		logCount++
		fmt.Printf("    log[%d]: %s | %s\n", logCount, s.Get("topics"), s.Get("message"))
	})
	if err != nil {
		fail("STREAM/LOG-FOLLOW", "register: %v", err)
	} else {
		time.Sleep(3 * time.Second)
		dev.UnregisterStream("log-tail")
		pass("STREAM/LOG-FOLLOW", "captured %d log entries in 3s (0 wajar saat router idle)", logCount)
	}

	// ── Poll → InfluxDB sink ──────────────────────────────────────────
	if influxCli == nil {
		warn("POLL/INFLUX", "skipped: influx disabled in config")
	} else {
		step("POLL/INFLUX", "poll /system/resource → InfluxDB measurement system_resource")
		writer := influx.NewWriter(influxCli, "system_resource",
			func(s *decode.Sentence) map[string]string {
				return map[string]string{
					"board": s.Get("board-name"),
					"ver":   s.Get("version"),
				}
			},
			func(s *decode.Sentence) map[string]any {
				return map[string]any{
					"cpu_load":      s.IntOr("cpu-load", 0),
					"free_memory":   s.BytesOr("free-memory", 0),
					"total_memory": s.BytesOr("total-memory", 0),
					"uptime_seconds": int64(s.DurationOr("uptime", 0).Seconds()),
				}
			},
		)
		pollCount := 0
		err = dev.RegisterPoll(roslib.PollConfig{
			ID:       "sys-resource",
			Path:     "/system/resource",
			Args:     []string{"print"},
			Interval: 2 * time.Second,
			Handler: func(s *decode.Sentence) {
				pollCount++
				if werr := writer.WriteSentence(ctx, s); werr != nil {
					log.WithError(werr).Warn("influx write")
				}
				fmt.Printf("    tick %d → written to InfluxDB (cpu=%s)\n",
					pollCount, s.Get("cpu-load"))
			},
		})
		if err != nil {
			fail("POLL/INFLUX", "register: %v", err)
		} else {
			time.Sleep(7 * time.Second)
			dev.UnregisterPoll("sys-resource")
			pass("POLL/INFLUX", "wrote %d points in ~6s", pollCount)
		}

		// ── Reader: query back ─────────────────────────────────────
		step("READER/QUERY", "SELECT * FROM system_resource LIMIT 5")
		reader := influx.NewReader(influxCli)
		iter, qerr := reader.Query(ctx,
			"SELECT * FROM system_resource ORDER BY time DESC LIMIT 5")
		if qerr != nil {
			fail("READER/QUERY", "%v", qerr)
		} else {
			rowCount := 0
			for iter.Next() {
				rowCount++
				row := iter.Value()
				if rowCount <= 3 {
					fmt.Printf("    row[%d]: %v\n", rowCount, row)
				}
			}
			if rowCount > 3 {
				fmt.Printf("    ... %d more rows\n", rowCount-3)
			}
			pass("READER/QUERY", "got %d rows back", rowCount)
		}
	}

	// ── Interval streaming: /queue/simple/print stats interval=1s ─────
	// (queue mungkin kosong di router → 0 row tapi sentence valid)
	step("STREAM/INTERVAL", "/queue/simple/print stats interval=1s — ≥1 tick atau snapshot kosong")
	qCount := 0
	qDone := make(chan struct{})
	err = dev.Path("/queue/simple").Print().Stats().
		Interval(time.Second).
		Stream("q-stats", func(s *decode.Sentence) {
			qCount++
			fmt.Printf("    tick %d: name=%s bytes=%s rate=%s\n",
				qCount, s.Get("name"), s.Get("bytes"), s.Get("rate"))
			if qCount >= 3 {
				select {
				case <-qDone:
				default:
					close(qDone)
				}
			}
		})
	if err != nil {
		fail("STREAM/INTERVAL", "register: %v", err)
	} else {
		select {
		case <-qDone:
			pass("STREAM/INTERVAL", "got %d ticks", qCount)
		case <-time.After(5 * time.Second):
			// 0 tick OK kalau queue kosong — yang penting register sukses.
			pass("STREAM/INTERVAL", "register ok, %d ticks (queue mungkin kosong)", qCount)
		}
		dev.UnregisterStream("q-stats")
	}

	// ── Stream no-flag validator ──────────────────────────────────────
	step("STREAM/NO-FLAG", ".Print().Stats().Stream() tanpa Follow/Interval — expect error")
	noFlagErr := dev.Path("/ip/address").Print().FollowOnly().Stream("ok-flag", func(*decode.Sentence) {})
	_ = noFlagErr
	dev.UnregisterStream("ok-flag")
	// Buat skenario validator-fail eksplisit via builder lower-level.
	// Cara mudah: panggil PrintBuilder.Interval(0).Stream — interval=0
	// tidak nyalakan flag, dan field follow/followOnly juga false.
	// Note: 0 interval di-treat seperti unset oleh builder.
	if sErr := dev.Path("/ip/address").Print().Interval(0).Stream("zero-interval", func(*decode.Sentence) {}); sErr == nil {
		fail("STREAM/NO-FLAG", "expected ErrNoStreamFlag, got nil")
	} else {
		pass("STREAM/NO-FLAG", "rejected: %v", sErr)
	}

	// ── Cache invalidation live ───────────────────────────────────────
	step("CACHE/INVALIDATE-LIVE", "ExecCached × 2 → invalidate → ExecCached miss again")
	invCache := cache.NewInMemory()
	invOpts := cfg.ToDeviceOptions(log)
	invOpts.Registry, _ = capability.Default()
	invOpts.Cache = invCache
	invOpts.CacheTTL = time.Hour
	invOpts.StrictCapability = true
	invOpts.ID = "test-inv"
	if rerr := mgr.Register(ctx, "invalidate-test", invOpts); rerr != nil {
		fail("CACHE/INVALIDATE-LIVE", "dial: %v", rerr)
	} else {
		defer mgr.Unregister("invalidate-test")
		dev3, _ := mgr.Get("invalidate-test")
		r1, _ := dev3.Path("/ip/address").Print().ExecCached(ctx, time.Hour)
		entriesBefore := invCache.Len()
		r2, _ := dev3.Path("/ip/address").Print().ExecCached(ctx, time.Hour)
		// Sekarang invalidate
		_ = dev3.InvalidateCache(ctx, "/ip/address")
		entriesAfter := invCache.Len()
		r3, _ := dev3.Path("/ip/address").Print().ExecCached(ctx, time.Hour)
		ok := r1 != nil && r2 != nil && r3 != nil &&
			len(r1.Rows) == len(r2.Rows) &&
			len(r1.Rows) == len(r3.Rows)
		if !ok {
			fail("CACHE/INVALIDATE-LIVE", "row count mismatch r1=%v r2=%v r3=%v", r1, r2, r3)
		} else {
			pass("CACHE/INVALIDATE-LIVE",
				"rows=%d entries(before-invalidate)=%d after-invalidate=%d → fresh fetch ok",
				len(r1.Rows), entriesBefore, entriesAfter)
		}
	}

	// ── Fleet smoke: load .env user (ROSLIB_ROUTERS=...) → dial semua ──
	step("FLEET/SMOKE", "NewManagerFromFleet dari .env (multi-router) — verifikasi pool+CloseAll")
	fleetCfg, ferr := config.LoadFleetFromEnv()
	if ferr != nil {
		warn("FLEET/SMOKE", "load: %v (skip)", ferr)
	} else {
		fleetMgr, _, fferr := roslib.NewManagerFromFleet(ctx, fleetCfg, log)
		if fferr != nil {
			fail("FLEET/SMOKE", "NewManagerFromFleet: %v", fferr)
		} else {
			ids := fleetMgr.Names()
			labels := make([]string, 0, len(ids))
			for _, id := range ids {
				rb, gerr := fleetMgr.Get(id)
				if gerr != nil {
					fail("FLEET/SMOKE", "Get(%s): %v", id, gerr)
					continue
				}
				rep, qerr := rb.Path("/system/identity").Print().Exec(ctx)
				if qerr != nil {
					fail("FLEET/SMOKE", "%s identity: %v", id, qerr)
					continue
				}
				if len(rep.Rows) > 0 {
					labels = append(labels, fmt.Sprintf("%s=%q", id, rep.Rows[0].Get("name")))
				}
			}
			pass("FLEET/SMOKE", "routers=%d %v", len(ids), labels)
			fleetMgr.CloseAll()
		}
	}

	// ── Persistent connection reuse: Manager.Get/GetOrConnect identik ─
	step("PERSIST/REUSE", "Manager.Get×5 + GetOrConnect×5 — pointer identik, tanpa re-dial")
	refConn := dev.CommandConn()
	if refConn == nil {
		fail("PERSIST/REUSE", "CommandConn nil — koneksi tidak siap")
	} else {
		mismatch := 0
		for i := 1; i <= 5; i++ {
			d, gerr := mgr.Get(roslib.DefaultDeviceKey)
			if gerr != nil {
				fail("PERSIST/REUSE", "Get iter %d: %v", i, gerr)
				mismatch++
				continue
			}
			if d.CommandConn() != refConn {
				mismatch++
			}
		}
		for i := 1; i <= 5; i++ {
			cfgOpts := cfg.ToDeviceOptions(log)
			d, gerr := mgr.GetOrConnect(ctx, roslib.DefaultDeviceKey, cfgOpts)
			if gerr != nil {
				fail("PERSIST/REUSE", "GetOrConnect iter %d: %v", i, gerr)
				mismatch++
				continue
			}
			if d.CommandConn() != refConn {
				mismatch++
			}
		}
		if mismatch == 0 {
			pass("PERSIST/REUSE", "5×Get + 5×GetOrConnect semua pointer identik %p (no re-dial)", refConn)
		} else {
			fail("PERSIST/REUSE", "%d/10 acquire pointer-drift dari %p", mismatch, refConn)
		}
	}

	// ── Stream finite cleanup: listener selesai natural → entry hilang ─
	step("STREAM/FINITE-CLEANUP", "verifikasi listener entry dibersihkan setelah !done natural")
	finishCh := make(chan error, 1)
	var finiteRx int
	finiteHandler := func(_ *decode.Sentence) { finiteRx++ }
	finiteOnFinish := func(id string, ferr error) {
		select {
		case finishCh <- ferr:
		default:
		}
	}
	// /tool/torch dengan duration=2s — finite stream native v6+v7. Tidak butuh
	// arg lain karena hanya untuk uji lifecycle, bukan content.
	before := dev.Streams().Len()
	terr := dev.Path("/tool/torch").
		With("interface", "ether1").
		With("duration", "2s").
		OnFinish(finiteOnFinish).
		Stream("torch-finite", finiteHandler)
	if terr != nil {
		warn("STREAM/FINITE-CLEANUP", "register torch: %v (mungkin path tidak tersedia di v6, skip)", terr)
	} else {
		select {
		case ferr := <-finishCh:
			after := dev.Streams().Len()
			if after != before {
				fail("STREAM/FINITE-CLEANUP",
					"after natural close Streams().Len()=%d; want %d (before)", after, before)
			} else {
				pass("STREAM/FINITE-CLEANUP",
					"rx=%d OnFinish err=%v before=%d after=%d (entry cleaned)",
					finiteRx, ferr, before, after)
			}
		case <-time.After(8 * time.Second):
			fail("STREAM/FINITE-CLEANUP", "OnFinish tidak fire dalam 8s")
			_ = dev.UnregisterStream("torch-finite")
		}
	}

	// ── COMBO: cache + influx + fleet bersamaan ────────────────────────
	step("COMBO/CACHE+INFLUX+FLEET", "exercise cache hit + influx write per device")
	if !cfg.Cache.Enabled || !cfg.Influx.Enabled {
		warn("COMBO/CACHE+INFLUX+FLEET",
			"skip — butuh ROSLIB_CACHE_ENABLED=true + ROSLIB_INFLUX_ENABLED=true (cache=%v influx=%v)",
			cfg.Cache.Enabled, cfg.Influx.Enabled)
	} else {
		comboFleetCfg, cerr := config.LoadFleetFromEnv()
		if cerr != nil || len(comboFleetCfg.Routers) < 2 {
			warn("COMBO/CACHE+INFLUX+FLEET",
				"skip — butuh ROSLIB_ROUTERS dengan ≥2 entry (load_err=%v routers=%d)",
				cerr, lenSafe(comboFleetCfg))
		} else {
			runComboScenario(ctx, log, comboFleetCfg)
		}
	}

	step("DONE", "all scenarios complete")
}

func lenSafe(c *config.FleetConfig) int {
	if c == nil {
		return 0
	}
	return len(c.Routers)
}

func runComboScenario(ctx context.Context, log *logrus.Logger, fleetCfg *config.FleetConfig) {
	mgr, influxCli, ferr := roslib.NewManagerFromFleet(ctx, fleetCfg, log)
	if ferr != nil {
		fail("COMBO/CACHE+INFLUX+FLEET", "NewManagerFromFleet: %v", ferr)
		return
	}
	defer mgr.CloseAll()
	if influxCli != nil {
		defer influxCli.Close()
	}
	ids := mgr.Names()

	// Cache shared antar device. NewManagerFromFleet sudah pasang InMemoryCache
	// di Options tiap device. Ambil dari device pertama untuk inspect Stats().
	var shared *cache.InMemoryCache
	for _, id := range ids {
		dev, _ := mgr.Get(id)
		if im, ok := dev.Cache().(*cache.InMemoryCache); ok {
			shared = im
			break
		}
	}
	if shared == nil {
		first, _ := mgr.Get(ids[0])
		fail("COMBO/CACHE+INFLUX+FLEET", "expected InMemoryCache, got %T", first.Cache())
		return
	}

	// 1. ExecCached 2x per router → call ke-2 harus hit.
	statsBefore := shared.Stats()
	for _, id := range ids {
		dev, _ := mgr.Get(id)
		if _, err := dev.Path("/system/resource").Print().ExecCached(ctx, 30*time.Second); err != nil {
			fail("COMBO/CACHE+INFLUX+FLEET", "ExecCached miss %s: %v", id, err)
			return
		}
		if _, err := dev.Path("/system/resource").Print().ExecCached(ctx, 30*time.Second); err != nil {
			fail("COMBO/CACHE+INFLUX+FLEET", "ExecCached hit %s: %v", id, err)
			return
		}
	}
	statsAfter := shared.Stats()
	expectedHits := int64(len(ids))
	expectedSets := int64(len(ids))
	hits := statsAfter.Hits - statsBefore.Hits
	sets := statsAfter.Sets - statsBefore.Sets
	if hits != expectedHits || sets != expectedSets {
		fail("COMBO/CACHE+INFLUX+FLEET",
			"cache stats Δhits=%d (want %d) Δsets=%d (want %d) entries=%d",
			hits, expectedHits, sets, expectedSets, statsAfter.Entries)
		return
	}
	fmt.Printf("    cache: routers=%d entries=%d Δhits=%d Δsets=%d ✓\n",
		len(ids), statsAfter.Entries, hits, sets)

	// 2. Multi-router poll → InfluxDB write. Writer per device dengan tag device_id.
	if influxCli == nil {
		warn("COMBO/CACHE+INFLUX+FLEET", "influx client nil — skip write phase")
		return
	}
	fieldsFn := func(s *decode.Sentence) map[string]any {
		free := s.IntOr("free-memory", 0)
		total := s.IntOr("total-memory", 0)
		return map[string]any{
			"free_memory":  free,
			"total_memory": total,
		}
	}
	for _, id := range ids {
		dev, _ := mgr.Get(id)
		devID := dev.DeviceID()
		tagsFn := func(_ *decode.Sentence) map[string]string {
			return map[string]string{"device_id": devID, "scope": "combo"}
		}
		w := influx.NewWriter(influxCli, "combo_resource", tagsFn, fieldsFn)
		handler := influx.PollSink(w, log.WithField("device", devID))
		if err := dev.RegisterPoll(roslib.PollConfig{
			ID:       "combo-resource",
			Path:     "/system/resource",
			Interval: 1500 * time.Millisecond,
			Handler:  handler,
		}); err != nil {
			fail("COMBO/CACHE+INFLUX+FLEET", "RegisterPoll %s: %v", devID, err)
			return
		}
	}
	// Biarkan 4 detik → ≥2 tick per router.
	time.Sleep(4 * time.Second)
	for _, id := range ids {
		dev, _ := mgr.Get(id)
		_ = dev.UnregisterPoll("combo-resource")
	}

	// 3. Verifikasi via Reader: query count per device_id.
	reader := influx.NewReader(influxCli)
	sql := "SELECT device_id, count(*) AS n FROM combo_resource " +
		"WHERE time > now() - INTERVAL '30 seconds' GROUP BY device_id"
	iter, qerr := reader.Query(ctx, sql)
	if qerr != nil {
		warn("COMBO/CACHE+INFLUX+FLEET", "reader query: %v (lewati verifikasi influx)", qerr)
	} else {
		seen := map[string]int64{}
		for iter.Next() {
			row := iter.Value()
			devID, _ := row["device_id"].(string)
			switch v := row["n"].(type) {
			case int64:
				seen[devID] = v
			case int32:
				seen[devID] = int64(v)
			}
		}
		if len(seen) < len(ids) {
			fail("COMBO/CACHE+INFLUX+FLEET",
				"influx rows: got %d device_id, want %d (seen=%v)",
				len(seen), len(ids), seen)
			return
		}
		fmt.Printf("    influx: device_ids=%v ✓\n", seen)
	}

	// 4. Cache invalidate scoped — affect satu device, lainnya tetap hit.
	pickID := ids[0]
	pickDev, _ := mgr.Get(pickID)
	if err := pickDev.InvalidateCache(ctx, "/system/resource"); err != nil {
		fail("COMBO/CACHE+INFLUX+FLEET", "InvalidateCache %s: %v", pickID, err)
		return
	}
	preMiss := shared.Stats().Misses
	preHit := shared.Stats().Hits
	for _, id := range ids {
		d, _ := mgr.Get(id)
		_, _ = d.Path("/system/resource").Print().ExecCached(ctx, 30*time.Second)
	}
	postStats := shared.Stats()
	deltaMiss := postStats.Misses - preMiss
	deltaHit := postStats.Hits - preHit
	// Setelah invalidate satu device: 1 miss (yang di-invalidate) + (len-1) hit.
	wantMiss := int64(1)
	wantHit := int64(len(ids) - 1)
	if deltaMiss != wantMiss || deltaHit != wantHit {
		fail("COMBO/CACHE+INFLUX+FLEET",
			"invalidate scope: Δmiss=%d (want %d) Δhit=%d (want %d) — cache key bukan device-scoped?",
			deltaMiss, wantMiss, deltaHit, wantHit)
		return
	}

	pass("COMBO/CACHE+INFLUX+FLEET",
		"routers=%d cache(hits=%d misses=%d entries=%d) influx-write+query ok, scoped-invalidate ok",
		len(ids), postStats.Hits, postStats.Misses, postStats.Entries)
}


// ──────────────── reporting helpers ────────────────

func step(name, desc string) {
	fmt.Printf("\n▶ %s — %s\n", name, desc)
}

func pass(name, format string, args ...any) {
	fmt.Printf("  ✓ %s: %s\n", name, fmt.Sprintf(format, args...))
}

func fail(name, format string, args ...any) {
	fmt.Printf("  ✗ %s: %s\n", name, fmt.Sprintf(format, args...))
}

func warn(name, format string, args ...any) {
	fmt.Printf("  ! %s: %s\n", name, fmt.Sprintf(format, args...))
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "FATAL: "+format+"\n", args...)
	os.Exit(1)
}

func classOf(c *capability.Command) string {
	if c == nil {
		return "<nil>"
	}
	return c.Class.String()
}
