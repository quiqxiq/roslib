package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/quiqxiq/roslib-mikhmon/api/dto"
	"github.com/quiqxiq/roslib-mikhmon/api/sse"
)

// NewServer membangun gin engine dengan middleware default + register
// semua route via RegisterRoutes. Kembalikan http.Handler agar caller
// (cmd/server) bisa bungkus ke http.Server custom (timeout, dll).
func NewServer(deps *Deps) http.Handler {
	if deps.Hub == nil {
		deps.Hub = sse.NewHub()
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, dto.Err("NOT_FOUND", "route not found", c.Request.URL.Path))
	})
	r.NoMethod(func(c *gin.Context) {
		c.JSON(http.StatusMethodNotAllowed, dto.Err("METHOD_NOT_ALLOWED", "method not allowed", c.Request.URL.Path))
	})

	r.HandleMethodNotAllowed = true

	// Health probe — minimal endpoint untuk liveness check.
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, dto.OK(map[string]string{"status": "ok"}))
	})

	v1 := r.Group("/api/v1")
	RegisterRoutes(v1, deps)

	return r
}
