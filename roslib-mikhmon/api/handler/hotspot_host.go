package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/quiqxiq/roslib-mikhmon/api/dto"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik/hotspot"
)

type HotspotHost struct{ Hot *hotspot.Client }

func NewHotspotHost(hot *hotspot.Client) *HotspotHost { return &HotspotHost{Hot: hot} }

func (h *HotspotHost) Register(g *gin.RouterGroup) {
	mk := func(c *gin.Context) *HotspotHost { return NewHotspotHost(mustClients(c).Hot) }
	hosts := g.Group("/hotspot/hosts")
	hosts.GET("", func(c *gin.Context) { mk(c).List(c) })
	hosts.DELETE("/:id", func(c *gin.Context) { mk(c).Delete(c) })
}

func (h *HotspotHost) List(c *gin.Context) {
	hs, err := h.Hot.HostList(c.Request.Context())
	if err != nil {
		WriteErr(c, err)
		return
	}
	out := dto.FromDomainHosts(hs)
	WriteList(c, out, len(out))
}

func (h *HotspotHost) Delete(c *gin.Context) {
	if err := h.Hot.HostRemove(c.Request.Context(), c.Param("id")); err != nil {
		WriteErr(c, err)
		return
	}
	WriteNoContent(c)
}
