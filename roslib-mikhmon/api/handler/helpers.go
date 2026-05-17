// Package handler berisi gin handler per resource. Tiap file = satu
// kelompok resource (mis. hotspot_user.go). Handler tipis: extract input
// dari context → call mikrotik.<X>.Client atau workflows.<F> → wrap
// envelope. Logic kompleks ada di workflows/ atau mikrotik/.
package handler

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/quiqxiq/roslib-mikhmon/api/dto"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik"
	"github.com/quiqxiq/roslib-mikhmon/service/devmgr"
)

// WriteOK menulis envelope success dengan status code opsional (default 200).
func WriteOK(c *gin.Context, data any, status ...int) {
	code := http.StatusOK
	if len(status) > 0 {
		code = status[0]
	}
	c.JSON(code, dto.OK(data))
}

// WriteCreated shortcut WriteOK dengan 201.
func WriteCreated(c *gin.Context, data any) {
	WriteOK(c, data, http.StatusCreated)
}

// WriteList membungkus list response + meta {count: N}.
func WriteList(c *gin.Context, items any, count int) {
	c.JSON(http.StatusOK, dto.List(items, map[string]any{"count": count}))
}

// WriteNoContent untuk DELETE / aksi tanpa body.
func WriteNoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// WriteErr map error ke status + envelope. Sentinel mikrotik.Err* ter-handle.
func WriteErr(c *gin.Context, err error) {
	code, status, msg := MapError(err)
	c.JSON(status, dto.Err(code, msg, c.Request.URL.Path))
}

// WriteValidationErr untuk c.ShouldBindJSON yang gagal. Details = err.Error()
// (gin pakai go-playground/validator yang produce human-readable error).
func WriteValidationErr(c *gin.Context, err error) {
	c.JSON(http.StatusBadRequest, dto.ErrDetails(
		"VALIDATION",
		"request validation failed",
		c.Request.URL.Path,
		err.Error(),
	))
}

// parseCacheQuery membaca query ?cache=true&ttl=60s untuk endpoint yang
// support cached variant. Default false; ttl default = defaultTTL.
func parseCacheQuery(c *gin.Context, defaultTTL time.Duration) (time.Duration, bool) {
	if c.Query("cache") != "true" && c.Query("cache") != "1" {
		return 0, false
	}
	ttl := defaultTTL
	if v := c.Query("ttl"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			ttl = d
		}
	}
	return ttl, true
}

// mustClients mengambil devmgr.ClientSet yang di-inject DeviceMiddleware.
func mustClients(c *gin.Context) *devmgr.ClientSet {
	return c.MustGet("device_clients").(*devmgr.ClientSet)
}

// MapError central error → HTTP status + code envelope. Dipanggil oleh
// WriteErr dan middleware/recovery.
func MapError(err error) (code string, status int, msg string) {
	switch {
	case err == nil:
		return "OK", http.StatusOK, ""
	case errors.Is(err, mikrotik.ErrNotFound):
		return "NOT_FOUND", http.StatusNotFound, err.Error()
	case errors.Is(err, mikrotik.ErrInvalidArgument):
		return "INVALID_ARGUMENT", http.StatusBadRequest, err.Error()
	case errors.Is(err, mikrotik.ErrAmbiguous):
		return "AMBIGUOUS", http.StatusConflict, err.Error()
	case errors.Is(err, context.DeadlineExceeded):
		return "TIMEOUT", http.StatusGatewayTimeout, "operation timed out"
	case errors.Is(err, context.Canceled):
		return "CANCELED", 499, "request canceled"
	default:
		return "INTERNAL", http.StatusInternalServerError, err.Error()
	}
}
