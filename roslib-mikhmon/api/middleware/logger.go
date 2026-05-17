package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Logger mencatat satu baris per request: method, path, status, latency,
// client IP, request_id. Pakai logrus instance dari Deps supaya format
// konsisten dengan komponen lain (router supervisor, dll).
func Logger(log *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		latency := time.Since(start)

		entry := log.WithFields(logrus.Fields{
			"method":     c.Request.Method,
			"path":       c.Request.URL.Path,
			"status":     c.Writer.Status(),
			"latency":    latency.String(),
			"ip":         c.ClientIP(),
			"request_id": c.GetString("request_id"),
			"size":       c.Writer.Size(),
		})
		switch {
		case c.Writer.Status() >= 500:
			entry.Error("http")
		case c.Writer.Status() >= 400:
			entry.Warn("http")
		default:
			entry.Info("http")
		}
	}
}
