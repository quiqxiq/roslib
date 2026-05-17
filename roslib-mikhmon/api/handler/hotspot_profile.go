package handler

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/quiqxiq/roslib-mikhmon/api/dto"
	"github.com/quiqxiq/roslib-mikhmon/domain"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik/hotspot"
	"github.com/quiqxiq/roslib-mikhmon/workflows"
)

type HotspotProfile struct {
	Hot *hotspot.Client
	WF  *workflows.Clients
}

func NewHotspotProfile(hot *hotspot.Client, wf *workflows.Clients) *HotspotProfile {
	return &HotspotProfile{Hot: hot, WF: wf}
}

func (h *HotspotProfile) Register(g *gin.RouterGroup) {
	mk := func(c *gin.Context) *HotspotProfile {
		cs := mustClients(c)
		return NewHotspotProfile(cs.Hot, cs.WF)
	}
	p := g.Group("/hotspot/profiles")
	p.GET("", func(c *gin.Context) { mk(c).List(c) })
	p.GET("/by-name/:name", func(c *gin.Context) { mk(c).GetByName(c) })
	p.GET("/:id", func(c *gin.Context) { mk(c).Get(c) })
	p.POST("", func(c *gin.Context) { mk(c).Create(c) })
	p.PUT("/:id", func(c *gin.Context) { mk(c).Update(c) })
	p.DELETE("/:id", func(c *gin.Context) { mk(c).Delete(c) })
}

func (h *HotspotProfile) List(c *gin.Context) {
	ctx := c.Request.Context()
	ttl, useCache := parseCacheQuery(c, 60*time.Second)
	var ps []domain.HotspotProfile
	var err error
	if useCache {
		ps, err = h.Hot.ProfileListCached(ctx, ttl)
	} else {
		ps, err = h.Hot.ProfileList(ctx)
	}
	if err != nil {
		WriteErr(c, err)
		return
	}
	out := dto.FromDomainProfiles(ps)
	WriteList(c, out, len(out))
}

func (h *HotspotProfile) Get(c *gin.Context) {
	p, err := h.Hot.ProfileByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		WriteErr(c, err)
		return
	}
	WriteOK(c, dto.FromDomainProfile(p))
}

func (h *HotspotProfile) GetByName(c *gin.Context) {
	p, err := h.Hot.ProfileByName(c.Request.Context(), c.Param("name"))
	if err != nil {
		WriteErr(c, err)
		return
	}
	WriteOK(c, dto.FromDomainProfile(p))
}

func (h *HotspotProfile) Create(c *gin.Context) {
	var req dto.HotspotProfileCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteValidationErr(c, err)
		return
	}
	id, err := h.Hot.ProfileAdd(c.Request.Context(), req.ToArgs())
	if err != nil {
		WriteErr(c, err)
		return
	}
	WriteCreated(c, gin.H{"id": id})
}

func (h *HotspotProfile) Update(c *gin.Context) {
	var req dto.HotspotProfileUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteValidationErr(c, err)
		return
	}
	if err := h.Hot.ProfileSet(c.Request.Context(), req.ToArgs(c.Param("id"))); err != nil {
		WriteErr(c, err)
		return
	}
	WriteNoContent(c)
}

func (h *HotspotProfile) Delete(c *gin.Context) {
	var req dto.HotspotProfileDeleteRequest
	_ = c.ShouldBindJSON(&req) // body opsional — name dipakai cascade scheduler cleanup
	if err := workflows.DeleteProfile(c.Request.Context(), h.WF, c.Param("id"), req.Name); err != nil {
		WriteErr(c, err)
		return
	}
	WriteNoContent(c)
}
