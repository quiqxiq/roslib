package middleware

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/gin-gonic/gin"
)

// HeaderRequestID adalah header standar untuk request id (in/out).
const HeaderRequestID = "X-Request-ID"

// RequestID memastikan setiap request punya ID. Kalau client kirim
// X-Request-ID, dipakai apa adanya; kalau tidak, generate 16-byte hex.
// ID di-set ke gin context (key "request_id") + di-echo ke response header.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(HeaderRequestID)
		if id == "" {
			id = newRequestID()
		}
		c.Set("request_id", id)
		c.Writer.Header().Set(HeaderRequestID, id)
		c.Next()
	}
}

func newRequestID() string {
	b := make([]byte, 8) // 16 hex chars, cukup untuk korelasi request log
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
