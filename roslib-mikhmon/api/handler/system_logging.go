package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/quiqxiq/roslib-mikhmon/api/dto"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik/system"
)

type SystemLogging struct{ Sys *system.Client }

func NewSystemLogging(sys *system.Client) *SystemLogging { return &SystemLogging{Sys: sys} }

func (h *SystemLogging) Register(g *gin.RouterGroup) {
	mk := func(c *gin.Context) *SystemLogging { return NewSystemLogging(mustClients(c).Sys) }
	g.GET("/system/logging/by-prefix/:prefix", func(c *gin.Context) { mk(c).ByPrefix(c) })
	g.POST("/system/logging/hotspot-disk", func(c *gin.Context) { mk(c).AddHotspotDisk(c) })
}

func (h *SystemLogging) ByPrefix(c *gin.Context) {
	n, err := h.Sys.LoggingByPrefix(c.Request.Context(), c.Param("prefix"))
	if err != nil {
		WriteErr(c, err)
		return
	}
	WriteOK(c, gin.H{"count": n})
}

func (h *SystemLogging) AddHotspotDisk(c *gin.Context) {
	var req dto.LoggingAddHotspotDiskRequest
	_ = c.ShouldBindJSON(&req) // body opsional, default prefix "->"
	if err := h.Sys.LoggingAddHotspotDisk(c.Request.Context(), req.Prefix); err != nil {
		WriteErr(c, err)
		return
	}
	WriteNoContent(c)
}
