// Demo inherently-streaming commands (monitor-traffic, ping, torch,
// sniffer, .../monitor). Tidak benar-benar konek ke router — semata
// memastikan API kompilasi. Build:
//
//	go build -tags=example ./examples/streaming
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
		Address:          "192.168.88.1:8728",
		Username:         "admin",
		Password:         "secret",
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
	if err := dev.Path("/tool/ping").
		With("address", "8.8.8.8").
		With("count", "5").
		Stream("ping-8888", func(s *roslib.Sentence) {
			logger.WithFields(logrus.Fields{
				"seq":  s.Get("seq"),
				"time": s.Get("time"),
				"ttl":  s.Get("ttl"),
			}).Info("ping")
		}); err != nil {
		log.Printf("ping register: %v", err)
	}

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

	// Misuse demo (strict mode): .Exec() di path streaming → error.
	if _, err := dev.Path("/interface/monitor-traffic").Print().Exec(ctx); err != nil {
		logger.WithError(err).Info("strict validator caught misuse (expected)")
	}

	time.Sleep(30 * time.Second)
	dev.UnregisterStream("nic-ether1")
}
