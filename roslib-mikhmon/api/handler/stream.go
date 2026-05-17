package handler

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/quiqxiq/roslib"
	"github.com/quiqxiq/roslib-mikhmon/api/dto"
	"github.com/quiqxiq/roslib-mikhmon/api/sse"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik/hotspot"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik/network"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik/ppp"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik/syslog"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik/system"
)

// Stream meng-handle semua endpoint SSE. Pakai sse.Hub untuk broker
// shared (1 stream router → N SSE client fan-out).
type Stream struct {
	Hub  *sse.Hub
	Hot  *hotspot.Client
	Sys  *system.Client
	Net  *network.Client
	PPP  *ppp.Client
	Logs *syslog.Client
}

func NewStream(hub *sse.Hub, hot *hotspot.Client, sys *system.Client, net *network.Client, pp *ppp.Client, log *syslog.Client) *Stream {
	return &Stream{Hub: hub, Hot: hot, Sys: sys, Net: net, PPP: pp, Logs: log}
}

func (s *Stream) Register(g *gin.RouterGroup) {
	// Stream handler menyimpan Hub (global), tapi clients diambil per-request
	// agar setiap device mendapat stream-nya sendiri.
	mk := func(c *gin.Context) *Stream {
		cs := mustClients(c)
		return NewStream(s.Hub, cs.Hot, cs.Sys, cs.Net, cs.PPP, cs.Log)
	}
	g.GET("/stream/hotspot/active", func(c *gin.Context) { mk(c).HotspotActive(c) })
	g.GET("/stream/ppp/active", func(c *gin.Context) { mk(c).PPPActive(c) })
	g.GET("/stream/log", func(c *gin.Context) { mk(c).Log(c) })
	g.GET("/stream/system/resource", func(c *gin.Context) { mk(c).SystemResource(c) })
	g.GET("/stream/network/interfaces/:name/traffic", func(c *gin.Context) { mk(c).InterfaceTraffic(c) })
	g.GET("/stream/network/interfaces/stats", func(c *gin.Context) { mk(c).InterfaceStats(c) })
	g.GET("/stream/network/queues/stats", func(c *gin.Context) { mk(c).QueueStats(c) })
}

// HotspotActive — /stream/hotspot/active (?mode=follow-only opsional).
func (s *Stream) HotspotActive(c *gin.Context) {
	mode := c.Query("mode")
	base := sse.TopicHotspotActive
	if mode == "follow-only" {
		base = sse.TopicHotspotActiveFollowOnly
	}
	topic := deviceTopic(c, base)
	broker := s.Hub.GetOrCreate(topic,
		func(b *sse.Broker) error {
			handler := func(sen *roslib.Sentence) {
				b.Publish(sse.Event{Type: "change", Data: sentenceToActiveMap(sen)})
			}
			if mode == "follow-only" {
				return s.Hot.ActiveStreamFollowOnly(topic, handler)
			}
			return s.Hot.ActiveStream(topic, handler)
		},
		func() { s.Hot.StopActiveStream(topic) },
	)
	sse.Stream(c, broker)
}

// PPPActive — /stream/ppp/active.
func (s *Stream) PPPActive(c *gin.Context) {
	topic := deviceTopic(c, sse.TopicPPPActive)
	broker := s.Hub.GetOrCreate(topic,
		func(b *sse.Broker) error {
			return s.PPP.ActiveStream(topic, func(sen *roslib.Sentence) {
				b.Publish(sse.Event{Type: "change", Data: sentenceToPPPActiveMap(sen)})
			})
		},
		func() { s.PPP.StopActiveStream(topic) },
	)
	sse.Stream(c, broker)
}

// Log — /stream/log?topics=hotspot,info.
func (s *Stream) Log(c *gin.Context) {
	topics := c.Query("topics")
	topic := deviceTopic(c, sse.TopicLog(topics))
	broker := s.Hub.GetOrCreate(topic,
		func(b *sse.Broker) error {
			return s.Logs.LogStream(topic, topics, func(sen *roslib.Sentence) {
				b.Publish(sse.Event{Type: "log", Data: sentenceToLogMap(sen)})
			})
		},
		func() { s.Logs.StopLogStream(topic) },
	)
	sse.Stream(c, broker)
}

// SystemResource — /stream/system/resource?interval=1s.
func (s *Stream) SystemResource(c *gin.Context) {
	interval := parseInterval(c, 2*time.Second)
	topic := deviceTopic(c, sse.TopicResource(interval.String()))
	broker := s.Hub.GetOrCreate(topic,
		func(b *sse.Broker) error {
			return s.Sys.MonitorResource(topic, interval, func(sen *roslib.Sentence) {
				if sen.Word() != "!re" {
					return
				}
				b.Publish(sse.Event{Type: "resource", Data: sentenceToResourceMap(sen)})
			})
		},
		func() { s.Sys.StopMonitor(topic) },
	)
	sse.Stream(c, broker)
}

// InterfaceTraffic — /stream/network/interfaces/:name/traffic (inherent stream).
func (s *Stream) InterfaceTraffic(c *gin.Context) {
	iface := c.Param("name")
	topic := deviceTopic(c, sse.TopicInterfaceTraffic(iface))
	broker := s.Hub.GetOrCreate(topic,
		func(b *sse.Broker) error {
			return s.Net.InterfaceTrafficStream(topic, iface, func(sen *roslib.Sentence) {
				b.Publish(sse.Event{Type: "traffic", Data: sentenceToTrafficMap(sen)})
			})
		},
		func() { s.Net.StopStream(topic) },
	)
	sse.Stream(c, broker)
}

// InterfaceStats — /stream/network/interfaces/stats?interval=2s.
func (s *Stream) InterfaceStats(c *gin.Context) {
	interval := parseInterval(c, 2*time.Second)
	topic := deviceTopic(c, sse.TopicInterfaceStats(interval.String()))
	broker := s.Hub.GetOrCreate(topic,
		func(b *sse.Broker) error {
			return s.Net.InterfaceStatsStream(topic, interval, func(sen *roslib.Sentence) {
				if sen.Word() != "!re" {
					return
				}
				b.Publish(sse.Event{Type: "stats", Data: sentenceToInterfaceMap(sen)})
			})
		},
		func() { s.Net.StopStream(topic) },
	)
	sse.Stream(c, broker)
}

// QueueStats — /stream/network/queues/stats?interval=1s.
func (s *Stream) QueueStats(c *gin.Context) {
	interval := parseInterval(c, 1*time.Second)
	topic := deviceTopic(c, sse.TopicQueueStats(interval.String()))
	broker := s.Hub.GetOrCreate(topic,
		func(b *sse.Broker) error {
			return s.Net.QueueStatsStream(topic, interval, func(sen *roslib.Sentence) {
				if sen.Word() != "!re" {
					return
				}
				b.Publish(sse.Event{Type: "stats", Data: sentenceToQueueMap(sen)})
			})
		},
		func() { s.Net.StopStream(topic) },
	)
	sse.Stream(c, broker)
}

// ── sentence → DTO map helpers ────────────────────────────────────────
// Tidak pakai dto.From* karena Sentence punya field tambahan dari follow
// (mis. ".dead" marker untuk row yang dihapus). Map langsung lebih
// faithful ke event-source semantic.

func sentenceToActiveMap(s *roslib.Sentence) any {
	return gin.H{
		"id":          s.Get(".id"),
		"user":        s.Get("user"),
		"address":     s.Get("address"),
		"mac_address": s.Get("mac-address"),
		"server":      s.Get("server"),
		"login_by":    s.Get("login-by"),
		"uptime":      s.Get("uptime"),
		"bytes_in":    s.IntOr("bytes-in", 0),
		"bytes_out":   s.IntOr("bytes-out", 0),
		"dead":        s.Get(".dead") == "true",
	}
}

func sentenceToPPPActiveMap(s *roslib.Sentence) any {
	return gin.H{
		"id":        s.Get(".id"),
		"name":      s.Get("name"),
		"service":   s.Get("service"),
		"caller_id": s.Get("caller-id"),
		"address":   s.Get("address"),
		"uptime":    s.Get("uptime"),
		"dead":      s.Get(".dead") == "true",
	}
}

func sentenceToLogMap(s *roslib.Sentence) any {
	return gin.H{
		"id":      s.Get(".id"),
		"time":    s.Get("time"),
		"topics":  s.Get("topics"),
		"message": s.Get("message"),
	}
}

func sentenceToResourceMap(s *roslib.Sentence) any {
	return dto.SystemResourceResponse{
		Uptime:           s.Get("uptime"),
		Version:          s.Get("version"),
		BoardName:        s.Get("board-name"),
		CPULoad:          int(s.IntOr("cpu-load", 0)),
		FreeMemory:       s.IntOr("free-memory", 0),
		TotalMemory:      s.IntOr("total-memory", 0),
		FreeHDDSpace:     s.IntOr("free-hdd-space", 0),
		TotalHDDSpace:    s.IntOr("total-hdd-space", 0),
		ArchitectureName: s.Get("architecture-name"),
	}
}

func sentenceToTrafficMap(s *roslib.Sentence) any {
	return dto.TrafficResponse{
		Name:            s.Get("name"),
		RxBitsPerSec:    s.IntOr("rx-bits-per-second", 0),
		TxBitsPerSec:    s.IntOr("tx-bits-per-second", 0),
		RxPacketsPerSec: s.IntOr("rx-packets-per-second", 0),
		TxPacketsPerSec: s.IntOr("tx-packets-per-second", 0),
	}
}

func sentenceToInterfaceMap(s *roslib.Sentence) any {
	return gin.H{
		"id":         s.Get(".id"),
		"name":       s.Get("name"),
		"type":       s.Get("type"),
		"rx_byte":    s.IntOr("rx-byte", 0),
		"tx_byte":    s.IntOr("tx-byte", 0),
		"rx_packet":  s.IntOr("rx-packet", 0),
		"tx_packet":  s.IntOr("tx-packet", 0),
		"running":    s.BoolOr("running", false),
		"disabled":   s.BoolOr("disabled", false),
	}
}

func sentenceToQueueMap(s *roslib.Sentence) any {
	return gin.H{
		"id":       s.Get(".id"),
		"name":     s.Get("name"),
		"target":   s.Get("target"),
		"bytes":    s.Get("bytes"),
		"packets":  s.Get("packets"),
		"rate":     s.Get("rate"),
		"max_limit": s.Get("max-limit"),
	}
}

func parseInterval(c *gin.Context, def time.Duration) time.Duration {
	v := c.Query("interval")
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		return def
	}
	return d
}

// deviceTopic scopes a base topic to the current device so that the global
// SSE Hub does not mix streams from different routers.
func deviceTopic(c *gin.Context, base string) string {
	id := c.Param("device_id")
	if id == "" {
		return base
	}
	return id + ":" + base
}
