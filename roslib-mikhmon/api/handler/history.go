package handler

import (
	"fmt"
	"net/http"
	"time"

	influxdb3 "github.com/InfluxCommunity/influxdb3-go/v2/influxdb3"
	"github.com/gin-gonic/gin"
	"github.com/quiqxiq/roslib-mikhmon/api/dto"
	roslibinflux "github.com/quiqxiq/roslib/metrics/influx"
)

// History menyediakan endpoint query data historis dari InfluxDB3.
// Reader nil artinya InfluxDB tidak dikonfigurasi — semua endpoint kembalikan 503.
type History struct {
	Reader *roslibinflux.Reader
}

func NewHistory(r *roslibinflux.Reader) *History {
	return &History{Reader: r}
}

func (h *History) Register(g *gin.RouterGroup) {
	r := g.Group("/history")
	r.GET("/resource", h.Resource)
	r.GET("/interfaces", h.Interfaces)
	r.GET("/hotspot/users", h.HotspotUsers)
	r.GET("/hotspot/active", h.HotspotActive)
	r.GET("/ppp/active", h.PPPActive)
	r.GET("/queues", h.Queues)
}

// Resource — historis CPU load, memory, free HDD.
func (h *History) Resource(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	p, ok := parseHistoryParams(c)
	if !ok {
		return
	}
	sql := fmt.Sprintf(
		`SELECT DATE_BIN(INTERVAL '%s', time) AS time,
		        AVG(cpu_load) AS cpu_load,
		        AVG(free_memory) AS free_memory,
		        AVG(total_memory) AS total_memory,
		        AVG(free_hdd) AS free_hdd
		 FROM system_resource
		 WHERE device = $device AND time >= $from AND time < $to
		 GROUP BY 1 ORDER BY 1`,
		p.interval,
	)
	h.execQuery(c, sql, p)
}

// Interfaces — delta rx/tx bytes per interface per interval.
func (h *History) Interfaces(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	p, ok := parseHistoryParams(c)
	if !ok {
		return
	}
	sql := fmt.Sprintf(
		`SELECT DATE_BIN(INTERVAL '%s', time) AS time, iface,
		        MAX(rx_byte) - MIN(rx_byte) AS rx_delta,
		        MAX(tx_byte) - MIN(tx_byte) AS tx_delta,
		        MAX(rx_packet) - MIN(rx_packet) AS rx_packet_delta,
		        MAX(tx_packet) - MIN(tx_packet) AS tx_packet_delta
		 FROM interface_stats
		 WHERE device = $device AND time >= $from AND time < $to
		 GROUP BY 1, iface ORDER BY 1`,
		p.interval,
	)
	h.execQuery(c, sql, p)
}

// HotspotUsers — delta bytes/packets per user per interval.
func (h *History) HotspotUsers(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	p, ok := parseHistoryParams(c)
	if !ok {
		return
	}
	sql := fmt.Sprintf(
		`SELECT DATE_BIN(INTERVAL '%s', time) AS time, user,
		        MAX(bytes_in) - MIN(bytes_in) AS bytes_in_delta,
		        MAX(bytes_out) - MIN(bytes_out) AS bytes_out_delta
		 FROM hotspot_user_bytes
		 WHERE device = $device AND time >= $from AND time < $to
		 GROUP BY 1, user ORDER BY 1`,
		p.interval,
	)
	h.execQuery(c, sql, p)
}

// HotspotActive — events koneksi/diskoneksi hotspot session.
func (h *History) HotspotActive(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	p, ok := parseHistoryParams(c)
	if !ok {
		return
	}
	sql := `SELECT time, user, server, bytes_in, bytes_out, dead
	        FROM hotspot_active
	        WHERE device = $device AND time >= $from AND time < $to
	        ORDER BY time`
	h.execQuery(c, sql, p)
}

// PPPActive — events koneksi/diskoneksi PPP session.
func (h *History) PPPActive(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	p, ok := parseHistoryParams(c)
	if !ok {
		return
	}
	sql := `SELECT time, name, service, dead
	        FROM ppp_active
	        WHERE device = $device AND time >= $from AND time < $to
	        ORDER BY time`
	h.execQuery(c, sql, p)
}

// Queues — delta bytes/packets per queue per interval.
func (h *History) Queues(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	p, ok := parseHistoryParams(c)
	if !ok {
		return
	}
	sql := fmt.Sprintf(
		`SELECT DATE_BIN(INTERVAL '%s', time) AS time, queue,
		        MAX(bytes_in) - MIN(bytes_in) AS bytes_in_delta,
		        MAX(bytes_out) - MIN(bytes_out) AS bytes_out_delta
		 FROM queue_stats
		 WHERE device = $device AND time >= $from AND time < $to
		 GROUP BY 1, queue ORDER BY 1`,
		p.interval,
	)
	h.execQuery(c, sql, p)
}

// ── helpers ──────────────────────────────────────────────────────────────────

type historyParams struct {
	device   string
	from     time.Time
	to       time.Time
	interval string // mis. "5m", "1h"
}

func (h *History) ready(c *gin.Context) bool {
	if h.Reader == nil {
		c.JSON(http.StatusServiceUnavailable,
			dto.Err("INFLUX_DISABLED", "historical data not configured", ""))
		return false
	}
	return true
}

func parseHistoryParams(c *gin.Context) (historyParams, bool) {
	device := c.Param("device_id")
	if device == "" {
		c.JSON(http.StatusBadRequest, dto.Err("MISSING_DEVICE", "device_id required", ""))
		return historyParams{}, false
	}

	fromStr := c.DefaultQuery("from", time.Now().Add(-1*time.Hour).UTC().Format(time.RFC3339))
	toStr := c.DefaultQuery("to", time.Now().UTC().Format(time.RFC3339))
	interval := c.DefaultQuery("interval", "1m")

	from, err := time.Parse(time.RFC3339, fromStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Err("INVALID_FROM", "from must be RFC3339", fromStr))
		return historyParams{}, false
	}
	to, err := time.Parse(time.RFC3339, toStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Err("INVALID_TO", "to must be RFC3339", toStr))
		return historyParams{}, false
	}
	if !to.After(from) {
		c.JSON(http.StatusBadRequest, dto.Err("INVALID_RANGE", "to must be after from", ""))
		return historyParams{}, false
	}

	return historyParams{device: device, from: from, to: to, interval: interval}, true
}

func (h *History) execQuery(c *gin.Context, sql string, p historyParams) {
	params := influxdb3.QueryParameters{
		"device": p.device,
		"from":   p.from,
		"to":     p.to,
	}
	iter, err := h.Reader.QueryWithParameters(c.Request.Context(), sql, params)
	if err != nil {
		WriteErr(c, err)
		return
	}

	rows := make([]any, 0)
	for iter.Next() {
		rows = append(rows, iter.Value())
	}
	WriteList(c, rows, len(rows))
}
