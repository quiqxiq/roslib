package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/quiqxiq/roslib-mikhmon/api/dto"
	"github.com/quiqxiq/roslib-mikhmon/domain"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik/system"
)

type SystemScript struct{ Sys *system.Client }

func NewSystemScript(sys *system.Client) *SystemScript { return &SystemScript{Sys: sys} }

func (h *SystemScript) Register(g *gin.RouterGroup) {
	mk := func(c *gin.Context) *SystemScript { return NewSystemScript(mustClients(c).Sys) }
	s := g.Group("/system/scripts")
	s.GET("", func(c *gin.Context) { mk(c).List(c) })
	s.GET("/:id", func(c *gin.Context) { mk(c).Get(c) })
	s.POST("", func(c *gin.Context) { mk(c).Create(c) })
	s.PUT("/:id", func(c *gin.Context) { mk(c).Update(c) })
	s.DELETE("/:id", func(c *gin.Context) { mk(c).Delete(c) })
}

func (h *SystemScript) List(c *gin.Context) {
	ctx := c.Request.Context()
	var ss []domain.Script
	var err error
	switch {
	case c.Query("name") != "":
		ss, err = h.Sys.ScriptByName(ctx, c.Query("name"))
	case c.Query("comment") != "":
		ss, err = h.Sys.ScriptByComment(ctx, c.Query("comment"))
	case c.Query("owner") != "":
		ss, err = h.Sys.ScriptByOwner(ctx, c.Query("owner"))
	case c.Query("source") != "":
		ss, err = h.Sys.ScriptBySource(ctx, c.Query("source"))
	default:
		ss, err = h.Sys.ScriptList(ctx)
	}
	if err != nil {
		WriteErr(c, err)
		return
	}
	out := dto.FromDomainScripts(ss)
	WriteList(c, out, len(out))
}

func (h *SystemScript) Get(c *gin.Context) {
	s, err := h.Sys.ScriptByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		WriteErr(c, err)
		return
	}
	WriteOK(c, dto.FromDomainScript(s))
}

func (h *SystemScript) Create(c *gin.Context) {
	var req dto.ScriptCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteValidationErr(c, err)
		return
	}
	id, err := h.Sys.ScriptAdd(c.Request.Context(), req.ToArgs())
	if err != nil {
		WriteErr(c, err)
		return
	}
	WriteCreated(c, gin.H{"id": id})
}

func (h *SystemScript) Update(c *gin.Context) {
	var req dto.ScriptUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteValidationErr(c, err)
		return
	}
	if err := h.Sys.ScriptSet(c.Request.Context(), req.ToArgs(c.Param("id"))); err != nil {
		WriteErr(c, err)
		return
	}
	WriteNoContent(c)
}

func (h *SystemScript) Delete(c *gin.Context) {
	if err := h.Sys.ScriptRemove(c.Request.Context(), c.Param("id")); err != nil {
		WriteErr(c, err)
		return
	}
	WriteNoContent(c)
}
