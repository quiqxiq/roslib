package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik/system"
)

// SystemControl menyediakan endpoint kontrol router (reboot/shutdown).
// PERINGATAN: endpoint destruktif. Caller (UI) wajib confirm dialog
// dan auth layer luar (reverse proxy) untuk lock down.
type SystemControl struct{ Sys *system.Client }

func NewSystemControl(sys *system.Client) *SystemControl { return &SystemControl{Sys: sys} }

func (h *SystemControl) Register(g *gin.RouterGroup) {
	mk := func(c *gin.Context) *SystemControl { return NewSystemControl(mustClients(c).Sys) }
	g.POST("/system/reboot", func(c *gin.Context) { mk(c).Reboot(c) })
	g.POST("/system/shutdown", func(c *gin.Context) { mk(c).Shutdown(c) })
}

func (h *SystemControl) Reboot(c *gin.Context) {
	if err := h.Sys.Reboot(c.Request.Context()); err != nil {
		WriteErr(c, err)
		return
	}
	WriteOK(c, gin.H{"action": "reboot", "status": "issued"})
}

func (h *SystemControl) Shutdown(c *gin.Context) {
	if err := h.Sys.Shutdown(c.Request.Context()); err != nil {
		WriteErr(c, err)
		return
	}
	WriteOK(c, gin.H{"action": "shutdown", "status": "issued"})
}
