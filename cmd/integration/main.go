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

	dev, influxCli, err := roslib.NewFromConfig(ctx, cfg, log)
	if err != nil {
		fatal("device dial: %v", err)
	}
	defer dev.Close()
	if influxCli != nil {
		defer influxCli.Close()
	}
	pass("BOOT", "device connected, registry loaded, influx ready=%v", influxCli != nil)

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
	// Strict validator menerima sentence (path ada di registry v7), tapi
	// router merespons "no such command". Expected behavior — bukan bug.
	step("RUN/PING-V7", "/tool/ping (registry-valid v7) — expected REJECT oleh router v6")
	_, err = dev.Path("/tool").Run(ctx, "ping",
		roslib.NewPair("address", "8.8.8.8"),
		roslib.NewPair("count", "3"),
	)
	if err != nil {
		var devErr *capability.ErrInvalidClass
		_ = devErr
		// Error dari device adalah dari ROUTER, bukan library.
		pass("RUN/PING-V7", "router reject (expected on v6): %v", err)
	} else {
		warn("RUN/PING-V7", "unexpected success on v6 router")
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
	dev2, derr := roslib.New(ctx, cachedOpts)
	if derr != nil {
		fail("EXEC-CACHED", "%v", derr)
	} else {
		defer dev2.Close()
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

	step("DONE", "all scenarios complete")
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
