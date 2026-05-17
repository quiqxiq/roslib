// Package metrics mengumpulkan data RouterOS ke InfluxDB3 menggunakan stream
// native RouterOS (interval= dan follow) — bukan Go-side ticker.
// Satu set stream per device; lifecycle dikelola via StartDevice/StopDevice
// yang dipanggil dari devmgr hooks.
package metrics

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/InfluxCommunity/influxdb3-go/v2/influxdb3"
	roslibinflux "github.com/quiqxiq/roslib/metrics/influx"
	"github.com/quiqxiq/roslib-mikhmon/service/devmgr"
	"github.com/quiqxiq/roslib-mikhmon/store/model"
	"github.com/sirupsen/logrus"

	"github.com/quiqxiq/roslib"
)

// Service mengumpulkan metrics dari semua device yang terhubung ke InfluxDB3.
type Service struct {
	cli    *influxdb3.Client
	devMgr *devmgr.Manager
	log    *logrus.Logger

	mu      sync.Mutex
	cancels map[string]context.CancelFunc
	rootCtx context.Context
}

func New(cli *influxdb3.Client, mgr *devmgr.Manager, log *logrus.Logger) *Service {
	return &Service{
		cli:     cli,
		devMgr:  mgr,
		log:     log,
		cancels: make(map[string]context.CancelFunc),
	}
}

// Start memulai collection untuk semua device yang sudah terhubung.
func (s *Service) Start(ctx context.Context) {
	s.rootCtx = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	for slug, cs := range s.devMgr.ListActive() {
		s.startLocked(slug, cs)
	}
}

// StartDevice memulai collection untuk device baru. Idempotent.
func (s *Service) StartDevice(d model.MikrotikDevice) {
	cs, err := s.devMgr.Get(d.Slug)
	if err != nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.cancels[d.Slug]; exists {
		return
	}
	s.startLocked(d.Slug, cs)
}

// StopDevice menghentikan collection untuk device yang dihapus.
func (s *Service) StopDevice(slug string) {
	s.mu.Lock()
	cancel, ok := s.cancels[slug]
	delete(s.cancels, slug)
	s.mu.Unlock()
	if ok {
		cancel()
	}
}

func (s *Service) startLocked(slug string, cs *devmgr.ClientSet) {
	ctx, cancel := context.WithCancel(s.rootCtx)
	s.cancels[slug] = cancel
	log := s.log.WithField("device", slug)

	// ── system_resource — system resource print interval=5s ──────────────
	resW := roslibinflux.NewWriter(s.cli, "system_resource",
		fixedTags(slug),
		func(sen *roslib.Sentence) map[string]any {
			return map[string]any{
				"cpu_load":     sen.IntOr("cpu-load", 0),
				"free_memory":  sen.IntOr("free-memory", 0),
				"total_memory": sen.IntOr("total-memory", 0),
				"free_hdd":     sen.IntOr("free-hdd-space", 0),
			}
		},
	)
	resID := "metrics:" + slug + ":resource"
	if err := cs.Sys.MonitorResource(resID, 5*time.Second, roslibinflux.PollSink(resW, log)); err != nil {
		log.WithError(err).Warn("metrics: MonitorResource failed")
	}

	// ── interface_stats — interface print stats interval=5s ──────────────
	ifaceW := roslibinflux.NewWriter(s.cli, "interface_stats",
		func(sen *roslib.Sentence) map[string]string {
			return map[string]string{"device": slug, "iface": sen.Get("name")}
		},
		func(sen *roslib.Sentence) map[string]any {
			return map[string]any{
				"rx_byte":   sen.IntOr("rx-byte", 0),
				"tx_byte":   sen.IntOr("tx-byte", 0),
				"rx_packet": sen.IntOr("rx-packet", 0),
				"tx_packet": sen.IntOr("tx-packet", 0),
			}
		},
	)
	ifaceID := "metrics:" + slug + ":iface-stats"
	ifaceSink := roslibinflux.StreamSink(ifaceW, log)
	if err := cs.Net.InterfaceStatsStream(ifaceID, 5*time.Second, func(sen *roslib.Sentence) {
		if sen.Word() == "!re" {
			ifaceSink(sen)
		}
	}); err != nil {
		log.WithError(err).Warn("metrics: InterfaceStatsStream failed")
	}

	// ── hotspot_user_bytes — ip/hotspot/user/print bytes interval=10s ────
	ubW := roslibinflux.NewWriter(s.cli, "hotspot_user_bytes",
		func(sen *roslib.Sentence) map[string]string {
			return map[string]string{"device": slug, "user": sen.Get("name")}
		},
		func(sen *roslib.Sentence) map[string]any {
			return map[string]any{
				"bytes_in":  sen.IntOr("bytes-in", 0),
				"bytes_out": sen.IntOr("bytes-out", 0),
			}
		},
	)
	ubID := "metrics:" + slug + ":user-bytes"
	ubSink := roslibinflux.StreamSink(ubW, log)
	if err := cs.Hot.UserBytesStream(ubID, 10*time.Second, func(sen *roslib.Sentence) {
		if sen.Word() == "!re" {
			ubSink(sen)
		}
	}); err != nil {
		log.WithError(err).Warn("metrics: UserBytesStream failed")
	}

	// ── hotspot_user_packets — ip/hotspot/user/print packets interval=10s ─
	upW := roslibinflux.NewWriter(s.cli, "hotspot_user_packets",
		func(sen *roslib.Sentence) map[string]string {
			return map[string]string{"device": slug, "user": sen.Get("name")}
		},
		func(sen *roslib.Sentence) map[string]any {
			return map[string]any{
				"packets_in":  sen.IntOr("packets-in", 0),
				"packets_out": sen.IntOr("packets-out", 0),
			}
		},
	)
	upID := "metrics:" + slug + ":user-packets"
	upSink := roslibinflux.StreamSink(upW, log)
	if err := cs.Hot.UserPacketsStream(upID, 10*time.Second, func(sen *roslib.Sentence) {
		if sen.Word() == "!re" {
			upSink(sen)
		}
	}); err != nil {
		log.WithError(err).Warn("metrics: UserPacketsStream failed")
	}

	// ── hotspot_active — ip/hotspot/active/print follow ──────────────────
	// Simpan setiap event connect/disconnect dengan field dead=true/false.
	haW := roslibinflux.NewWriter(s.cli, "hotspot_active",
		func(sen *roslib.Sentence) map[string]string {
			return map[string]string{
				"device": slug,
				"user":   sen.Get("user"),
				"server": sen.Get("server"),
			}
		},
		func(sen *roslib.Sentence) map[string]any {
			return map[string]any{
				"bytes_in":  sen.IntOr("bytes-in", 0),
				"bytes_out": sen.IntOr("bytes-out", 0),
				"dead":      sen.Get(".dead") == "true",
			}
		},
	)
	haID := "metrics:" + slug + ":hotspot-active"
	haSink := roslibinflux.StreamSink(haW, log)
	if err := cs.Hot.ActiveStream(haID, func(sen *roslib.Sentence) {
		if sen.Word() == "!re" {
			haSink(sen)
		}
	}); err != nil {
		log.WithError(err).Warn("metrics: hotspot ActiveStream failed")
	}

	// ── ppp_active — ppp/active/print follow ─────────────────────────────
	pppW := roslibinflux.NewWriter(s.cli, "ppp_active",
		func(sen *roslib.Sentence) map[string]string {
			return map[string]string{
				"device":  slug,
				"name":    sen.Get("name"),
				"service": sen.Get("service"),
			}
		},
		func(sen *roslib.Sentence) map[string]any {
			return map[string]any{
				"dead": sen.Get(".dead") == "true",
			}
		},
	)
	pppID := "metrics:" + slug + ":ppp-active"
	pppSink := roslibinflux.StreamSink(pppW, log)
	if err := cs.PPP.ActiveStream(pppID, func(sen *roslib.Sentence) {
		if sen.Word() == "!re" {
			pppSink(sen)
		}
	}); err != nil {
		log.WithError(err).Warn("metrics: ppp ActiveStream failed")
	}

	// ── queue_stats — queue/simple/print stats interval=10s ──────────────
	// Field "bytes" dari RouterOS berformat "in/out" → di-parse jadi dua field.
	qW := roslibinflux.NewWriter(s.cli, "queue_stats",
		func(sen *roslib.Sentence) map[string]string {
			return map[string]string{"device": slug, "queue": sen.Get("name")}
		},
		func(sen *roslib.Sentence) map[string]any {
			bin, bout := splitInOut(sen.Get("bytes"))
			pin, pout := splitInOut(sen.Get("packets"))
			return map[string]any{
				"bytes_in":    bin,
				"bytes_out":   bout,
				"packets_in":  pin,
				"packets_out": pout,
			}
		},
	)
	qID := "metrics:" + slug + ":queue-stats"
	qSink := roslibinflux.StreamSink(qW, log)
	if err := cs.Net.QueueStatsStream(qID, 10*time.Second, func(sen *roslib.Sentence) {
		if sen.Word() == "!re" {
			qSink(sen)
		}
	}); err != nil {
		log.WithError(err).Warn("metrics: QueueStatsStream failed")
	}

	// Cleanup streams saat device dihapus atau server shutdown
	go func() {
		<-ctx.Done()
		cs.Sys.StopMonitor(resID)
		cs.Net.StopStream(ifaceID)
		cs.Hot.StopUserStream(ubID)
		cs.Hot.StopUserStream(upID)
		cs.Hot.StopActiveStream(haID)
		cs.PPP.StopActiveStream(pppID)
		cs.Net.StopStream(qID)
	}()

	log.Info("metrics: streams started")
}

// ── helpers ──────────────────────────────────────────────────────────────────

// fixedTags mengembalikan TagFn dengan satu tag device=slug.
func fixedTags(slug string) roslibinflux.TagFn {
	return func(*roslib.Sentence) map[string]string {
		return map[string]string{"device": slug}
	}
}

// splitInOut mem-parse field format RouterOS "in/out" (mis. "1024/2048") menjadi
// dua int64. Mengembalikan 0/0 jika format tidak sesuai.
func splitInOut(s string) (int64, int64) {
	parts := strings.SplitN(s, "/", 2)
	if len(parts) != 2 {
		return 0, 0
	}
	in, _ := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
	out, _ := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
	return in, out
}
