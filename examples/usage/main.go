// Example usage cermin Section 6 di planing.md. Tidak benar-benar
// menyentuh router — semata untuk memastikan public API ergonomis dan
// kompilasi bersih. Build dengan: `go build -tags=example ./examples/usage`.
//
//go:build example

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/quiqxiq/roslib"
	"github.com/quiqxiq/roslib/config"
	"github.com/sirupsen/logrus"
)

func main() {
	log := logrus.New()
	log.SetLevel(logrus.InfoLevel)
	log.SetFormatter(&logrus.JSONFormatter{})

	ctx := context.Background()

	// Option A: literal Options (manual).
	device, err := roslib.New(ctx, roslib.Options{
		Address:          "192.168.88.1:8728",
		Username:         "admin",
		Password:         "secret",
		Logger:           log,
		ListenQueueSize:  100,
		StrictCapability: true,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer device.Close()

	// Option B: NewFromConfig — load semua dari env. Comment-out di-atas
	// dan pakai blok ini kalau pengin loader env.
	if false {
		cfg, cerr := config.LoadFromEnv()
		if cerr != nil {
			log.Fatal(cerr)
		}
		dev, _, ferr := roslib.NewFromConfig(ctx, cfg, log)
		if ferr != nil {
			log.Fatal(ferr)
		}
		defer dev.Close()
	}

	// Query / mutation → connCommand (concurrent via tag demux).
	go func() {
		reply, _ := device.Path("/ip/address").Print().Exec(ctx)
		fmt.Println("address rows:", len(reply.Rows))
	}()
	go func() {
		reply, _ := device.Path("/ip/route").Print().Detail().Exec(ctx)
		fmt.Println("route rows:", len(reply.Rows))
	}()

	_, _ = device.Path("/ip/address").Add(ctx,
		roslib.NewPair("address", "10.0.0.1/24"),
		roslib.NewPair("interface", "ether1"),
	)

	// Stream print-follow → connStream.
	_ = device.Path("/log").Print().FollowOnly().Stream("log-tail", func(s *roslib.Sentence) {
		log.WithField("msg", s.Get("message")).Info("router log")
	})
	_ = device.Path("/ip/hotspot/active").Print().Follow().Stream("hotspot-active", func(s *roslib.Sentence) {
		log.WithFields(logrus.Fields{"user": s.Get("user"), "uptime": s.Get("uptime")}).Info("hotspot")
	})

	// Stream inherent (tanpa Print) — monitor-traffic, ping, torch, dst.
	_ = device.Path("/interface/monitor-traffic").
		With("interface", "ether1").
		Stream("nic-1", func(s *roslib.Sentence) {
			log.WithFields(logrus.Fields{
				"rx-bps": s.Get("rx-bits-per-second"),
				"tx-bps": s.Get("tx-bits-per-second"),
			}).Info("nic")
		})

	_ = device.Path("/tool/ping").
		With("address", "8.8.8.8").
		With("count", "5").
		Stream("ping-google", func(s *roslib.Sentence) {
			log.WithField("time", s.Get("time")).Info("ping reply")
		})

	// Poll → di-batch oleh interval-group.
	_ = device.RegisterPoll(roslib.PollConfig{
		ID:       "system-resource",
		Path:     "/system/resource",
		Args:     []string{"print"},
		Interval: 5 * time.Second,
		Handler:  func(s *roslib.Sentence) { _ = s.Get("uptime") },
	})
	_ = device.RegisterPoll(roslib.PollConfig{
		ID:       "interface-stats",
		Path:     "/interface",
		Args:     []string{"print", "stats"},
		Interval: 5 * time.Second,
		Handler:  func(s *roslib.Sentence) { _ = s.Get("rx-byte") },
	})

	// ExecCached — query dengan cache otomatis (TTL 30s).
	_, _ = device.Path("/ip/address").Print().ExecCached(ctx, 30*time.Second)

	// Filtered query.
	_, _ = device.Path("/ip/firewall/filter").
		Print().
		Where("chain", "input").
		WherePair(roslib.WhereNot("disabled", "true")).
		Exec(ctx)

	device.UnregisterPoll("system-resource")

	time.Sleep(1 * time.Second)
}
