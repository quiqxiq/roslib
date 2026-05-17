package sse

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// KeepaliveInterval untuk SSE comment heartbeat. Cegah nginx/reverse-proxy
// timeout pada koneksi long-lived idle.
const KeepaliveInterval = 15 * time.Second

// Stream bridge broker → SSE protocol di gin response writer. Subscribe
// dengan clientID dari request_id middleware (atau fallback ke remote addr).
// Loop sampai client disconnect (ctx done) atau channel close.
//
// SSE protocol per event:
//
//	event: <type>
//	id: <id>           (optional)
//	data: <json>
//	<blank line>
//
// Keepalive di-emit tiap 15s sebagai SSE comment:
//
//	: keepalive
//	<blank line>
func Stream(c *gin.Context, broker *Broker) {
	clientID := c.GetString("request_id")
	if clientID == "" {
		clientID = c.ClientIP() + ":" + fmt.Sprint(time.Now().UnixNano())
	}

	ch, err := broker.Subscribe(clientID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"code": "STREAM_START", "message": err.Error()},
		})
		return
	}
	defer broker.Unsubscribe(clientID)

	w := c.Writer
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable nginx buffering
	w.WriteHeader(http.StatusOK)
	w.Flush()

	keepalive := time.NewTicker(KeepaliveInterval)
	defer keepalive.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case ev, ok := <-ch:
			if !ok {
				return
			}
			writeEvent(w, ev)
		case <-keepalive.C:
			fmt.Fprintf(w, ": keepalive\n\n")
			w.Flush()
		}
	}
}

func writeEvent(w gin.ResponseWriter, ev Event) {
	payload, err := json.Marshal(ev.Data)
	if err != nil {
		// Marshal failure jarang terjadi untuk DTO sederhana; tetap kirim
		// payload error supaya client tahu ada masalah.
		payload = []byte(`{"error":"marshal failed"}`)
	}
	if ev.Type != "" {
		fmt.Fprintf(w, "event: %s\n", ev.Type)
	}
	if ev.ID != "" {
		fmt.Fprintf(w, "id: %s\n", ev.ID)
	}
	fmt.Fprintf(w, "data: %s\n\n", payload)
	w.Flush()
}
