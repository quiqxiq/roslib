package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/quiqxiq/roslib-mikhmon/api/dto"
	"github.com/quiqxiq/roslib-mikhmon/domain"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik/system"
)

type SystemScheduler struct{ Sys *system.Client }

func NewSystemScheduler(sys *system.Client) *SystemScheduler { return &SystemScheduler{Sys: sys} }

func (h *SystemScheduler) Register(g *gin.RouterGroup) {
	mk := func(c *gin.Context) *SystemScheduler { return NewSystemScheduler(mustClients(c).Sys) }
	s := g.Group("/system/schedulers")
	s.GET("", func(c *gin.Context) { mk(c).List(c) })
	s.GET("/count", func(c *gin.Context) { mk(c).Count(c) })
	s.POST("", func(c *gin.Context) { mk(c).Create(c) })
	s.PUT("/:id", func(c *gin.Context) { mk(c).Update(c) })
	s.DELETE("/:id", func(c *gin.Context) { mk(c).Delete(c) })
}

func (h *SystemScheduler) List(c *gin.Context) {
	ctx := c.Request.Context()
	var ss []domain.Scheduler
	var err error
	if name := c.Query("name"); name != "" {
		ss, err = h.Sys.SchedulerByName(ctx, name)
	} else {
		ss, err = h.Sys.SchedulerList(ctx)
	}
	if err != nil {
		WriteErr(c, err)
		return
	}
	out := dto.FromDomainSchedulers(ss)
	WriteList(c, out, len(out))
}

func (h *SystemScheduler) Count(c *gin.Context) {
	n, err := h.Sys.SchedulerCount(c.Request.Context())
	if err != nil {
		WriteErr(c, err)
		return
	}
	WriteOK(c, dto.CountResponse{Count: n})
}

func (h *SystemScheduler) Create(c *gin.Context) {
	var req dto.SchedulerCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteValidationErr(c, err)
		return
	}
	id, err := h.Sys.SchedulerAdd(c.Request.Context(), req.ToArgs())
	if err != nil {
		WriteErr(c, err)
		return
	}
	WriteCreated(c, dto.IDResponse{ID: id})
}

func (h *SystemScheduler) Update(c *gin.Context) {
	var req dto.SchedulerUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteValidationErr(c, err)
		return
	}
	if err := h.Sys.SchedulerSet(c.Request.Context(), req.ToArgs(c.Param("id"))); err != nil {
		WriteErr(c, err)
		return
	}
	WriteNoContent(c)
}

func (h *SystemScheduler) Delete(c *gin.Context) {
	if err := h.Sys.SchedulerRemove(c.Request.Context(), c.Param("id")); err != nil {
		WriteErr(c, err)
		return
	}
	WriteNoContent(c)
}
