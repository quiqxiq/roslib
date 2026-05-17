// Package middleware berisi gin middleware untuk roslib-mikhmon HTTP server.
package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/quiqxiq/roslib-mikhmon/api/dto"
	"github.com/sirupsen/logrus"
)

// Recovery menangkap panic, log stack trace, dan kirim envelope error 500.
// Tidak pakai gin.Recovery() default karena default-nya kirim plain text;
// kita butuh format envelope yang konsisten.
func Recovery(log *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.WithFields(logrus.Fields{
					"panic":      r,
					"method":     c.Request.Method,
					"path":       c.Request.URL.Path,
					"request_id": c.GetString("request_id"),
					"stack":      string(debug.Stack()),
				}).Error("panic recovered")

				c.AbortWithStatusJSON(http.StatusInternalServerError, dto.Err(
					"INTERNAL",
					fmt.Sprintf("internal server error: %v", r),
					c.Request.URL.Path,
				))
			}
		}()
		c.Next()
	}
}
