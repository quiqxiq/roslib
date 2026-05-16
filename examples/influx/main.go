// Demo full flow: load config dari env, dial router, register poll dengan
// InfluxDB3 sink, dan baca balik via Reader. Build:
//
//	go build -tags=example ./examples/influx
//
// Env yang dibutuhkan minimal:
//
//	ROSLIB_ROUTER_ADDRESS=192.168.88.1:8728
//	ROSLIB_ROUTER_USERNAME=admin
//	ROSLIB_ROUTER_PASSWORD=secret
//	ROSLIB_INFLUX_ENABLED=true
//	INFLUX_HOST=https://us-east-1-1.aws.cloud2.influxdata.com
//	INFLUX_TOKEN=<token>
//	INFLUX_DATABASE=mikrotik
//
//go:build example

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/quiqxiq/roslib"
	"github.com/quiqxiq/roslib/config"
	"github.com/quiqxiq/roslib/decode"
	"github.com/quiqxiq/roslib/metrics/influx"
	"github.com/sirupsen/logrus"
)

func main() {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	cfg, err := config.LoadFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	mgr, ifc, err := roslib.NewManagerFromConfig(ctx, cfg, logger)
	if err != nil {
		log.Fatal(err)
	}
	defer mgr.CloseAll()
	if ifc != nil {
		defer ifc.Close()
	}
	dev, gerr := mgr.Get(roslib.DefaultDeviceKey)
	if gerr != nil {
		log.Fatal(gerr)
	}

	// ── Writer: hasil poll /system/resource → measurement "system_resource"
	writer := influx.NewWriter(ifc, "system_resource",
		// Tags
		func(s *decode.Sentence) map[string]string {
			return map[string]string{
				"board": s.Get("board-name"),
				"ver":   s.Get("version"),
			}
		},
		// Fields
		func(s *decode.Sentence) map[string]any {
			return map[string]any{
				"cpu_load":      s.IntOr("cpu-load", 0),
				"free_memory":   s.BytesOr("free-memory", 0),
				"total_memory":  s.BytesOr("total-memory", 0),
				"uptime_seconds": int64(s.DurationOr("uptime", 0).Seconds()),
			}
		},
	)

	if err := dev.RegisterPoll(roslib.PollConfig{
		ID:       "sys-resource",
		Path:     "/system/resource",
		Args:     []string{"print"},
		Interval: 5 * time.Second,
		Handler:  influx.PollSink(writer, logger.WithField("sink", "influx")),
	}); err != nil {
		log.Fatal(err)
	}

	// ── Stream listener: log realtime sink ke measurement "router_log"
	logWriter := influx.NewWriter(ifc, "router_log",
		func(s *decode.Sentence) map[string]string {
			return map[string]string{"topics": s.Get("topics")}
		},
		func(s *decode.Sentence) map[string]any {
			return map[string]any{"msg": s.Get("message")}
		},
	)
	_ = dev.Path("/log").Print().FollowOnly().Stream("log-stream",
		influx.StreamSink(logWriter, logger.WithField("sink", "influx-log")),
	)

	// ── Reader: SQL query terhadap measurement.
	reader := influx.NewReader(ifc)
	iter, err := reader.Query(ctx,
		"SELECT time, cpu_load, free_memory FROM system_resource ORDER BY time DESC LIMIT 5")
	if err != nil {
		log.Printf("query: %v", err)
	} else {
		for iter.Next() {
			row := iter.Value()
			fmt.Printf("row: %v\n", row)
		}
	}

	time.Sleep(30 * time.Second)
}
