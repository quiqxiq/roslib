package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/quiqxiq/roslib-mikhmon/api/dto"
	"github.com/quiqxiq/roslib-mikhmon/domain"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik/hotspot"
	"github.com/quiqxiq/roslib-mikhmon/workflows"
)

type HotspotActive struct {
	Hot *hotspot.Client
	WF  *workflows.Clients
}

func NewHotspotActive(hot *hotspot.Client, wf *workflows.Clients) *HotspotActive {
	return &HotspotActive{Hot: hot, WF: wf}
}

func (h *HotspotActive) Register(g *gin.RouterGroup) {
	mk := func(c *gin.Context) *HotspotActive {
		cs := mustClients(c)
		return NewHotspotActive(cs.Hot, cs.WF)
	}
	a := g.Group("/hotspot/active")
	a.GET("", func(c *gin.Context) { mk(c).List(c) })
	a.GET("/count", func(c *gin.Context) { mk(c).Count(c) })
	a.GET("/:id", func(c *gin.Context) { mk(c).Get(c) })
	a.DELETE("/:id", func(c *gin.Context) { mk(c).Kick(c) })
}

func (h *HotspotActive) List(c *gin.Context) {
	ctx := c.Request.Context()
	var as []domain.HotspotActive
	var err error
	if server := c.Query("server"); server != "" {
		as, err = h.Hot.ActiveByServer(ctx, server)
	} else {
		as, err = h.Hot.ActiveList(ctx)
	}
	if err != nil {
		WriteErr(c, err)
		return
	}
	out := dto.FromDomainActives(as)
	WriteList(c, out, len(out))
}

func (h *HotspotActive) Count(c *gin.Context) {
	ctx := c.Request.Context()
	var n int
	var err error
	if server := c.Query("server"); server != "" {
		n, err = h.Hot.ActiveCountByServer(ctx, server)
	} else {
		n, err = h.Hot.ActiveCount(ctx)
	}
	if err != nil {
		WriteErr(c, err)
		return
	}
	WriteOK(c, gin.H{"count": n})
}

func (h *HotspotActive) Get(c *gin.Context) {
	a, err := h.Hot.ActiveByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		WriteErr(c, err)
		return
	}
	WriteOK(c, dto.FromDomainActive(a))
}

func (h *HotspotActive) Kick(c *gin.Context) {
	if err := workflows.KickActive(c.Request.Context(), h.WF, c.Param("id")); err != nil {
		WriteErr(c, err)
		return
	}
	WriteNoContent(c)
}
