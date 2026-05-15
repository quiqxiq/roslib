// Demo inherently-streaming commands (monitor-traffic, ping, torch,
// sniffer, .../monitor). Build:
//
//	go build -tags=example ./examples/streaming
//
// Cara menjalankan — set env lalu run:
//
//	export ROSLIB_ROUTER_ADDRESS=192.168.88.1:8728
//	export ROSLIB_ROUTER_USERNAME=admin
//	export ROSLIB_ROUTER_PASSWORD=secret
//	go run -tags=example ./examples/streaming
//
// Atau edit langsung field Address/Username/Password di bawah.
//
//go:build example

package main

import (
	"context"
	"log"
	"time"

	"github.com/quiqxiq/roslib"
	"github.com/sirupsen/logrus"
)

func main() {
	logger := logrus.New()
	ctx := context.Background()

	dev, err := roslib.New(ctx, roslib.Options{
		Address:          "192.168.230.2:8728",
		Username:         "admin",
		Password:         "r00t",
		Logger:           logger,
		StrictCapability: true,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer dev.Close()

	// /interface/monitor-traffic — emit rx/tx bits per detik per interface.
	if err := dev.Path("/interface/monitor-traffic").
		With("interface", "ether1").
		Stream("nic-ether1", func(s *roslib.Sentence) {
			logger.WithFields(logrus.Fields{
				"rx-bps": s.Get("rx-bits-per-second"),
				"tx-bps": s.Get("tx-bits-per-second"),
			}).Info("nic stats")
		}); err != nil {
		log.Printf("monitor-traffic register: %v", err)
	}

	// /tool/ping — ICMP echo dengan count terbatas, lalu listener selesai.
	// if err := dev.Path("/tool/ping").
	// 	With("address", "8.8.8.8").
	// 	With("count", "5").
	// 	Stream("ping-8888", func(s *roslib.Sentence) {
	// 		logger.WithFields(logrus.Fields{
	// 			"seq":  s.Get("seq"),
	// 			"time": s.Get("time"),
	// 			"ttl":  s.Get("ttl"),
	// 		}).Info("ping")
	// 	}); err != nil {
	// 	log.Printf("ping register: %v", err)
	// }

	// /interface/ethernet/monitor — interface fisik (link rate, auto-negotiation).
	if err := dev.Path("/interface/ethernet/monitor").
		With("numbers", "ether1").
		With("once", "yes").
		Stream("eth1-link", func(s *roslib.Sentence) {
			logger.WithField("status", s.Get("status")).Info("eth1")
		}); err != nil {
		log.Printf("ethernet monitor: %v", err)
	}

	// /tool/torch — real-time traffic analyzer.
	if err := dev.Path("/tool/torch").
		With("interface", "ether1").
		With("src-address", "0.0.0.0/0").
		Stream("torch-ether1", func(s *roslib.Sentence) {
			logger.WithField("src", s.Get("src-address")).Info("torch")
		}); err != nil {
		log.Printf("torch register: %v", err)
	}

	// /system/resource/monitor — CPU, memory, uptime secara real-time.
	// Action "monitor" → ClassStreaming, pakai .Stream() langsung dari PathBuilder.
	// Interval default RouterOS = 1 detik. Override via With("interval", "2").
	if err := dev.Path("/system/resource/monitor").
		Stream("sys-resource-monitor", func(s *roslib.Sentence) {
			logger.WithFields(logrus.Fields{
				"cpu-load":    s.Get("cpu-load"),
				"free-memory": s.Get("free-memory"),
				"uptime":      s.Get("uptime"),
			}).Info("system resource")
		}); err != nil {
		log.Printf("system resource monitor register: %v", err)
	}

	// Misuse demo (strict mode): .Exec() di path streaming → error.
	if _, err := dev.Path("/interface/monitor-traffic").Print().Exec(ctx); err != nil {
		logger.WithError(err).Info("strict validator caught misuse (expected)")
	}

	time.Sleep(300 * time.Second)
	dev.UnregisterStream("nic-ether1")
	dev.UnregisterStream("sys-resource-monitor")
}
