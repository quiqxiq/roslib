package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/quiqxiq/roslib-mikhmon/api/dto"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik/ppp"
)

type PPPSecret struct{ PPP *ppp.Client }

func NewPPPSecret(pp *ppp.Client) *PPPSecret { return &PPPSecret{PPP: pp} }

func (h *PPPSecret) Register(g *gin.RouterGroup) {
	mk := func(c *gin.Context) *PPPSecret { return NewPPPSecret(mustClients(c).PPP) }
	s := g.Group("/ppp/secrets")
	s.GET("", func(c *gin.Context) { mk(c).List(c) })
	s.GET("/by-name/:name", func(c *gin.Context) { mk(c).GetByName(c) })
	s.POST("", func(c *gin.Context) { mk(c).Create(c) })
	s.PUT("/:id", func(c *gin.Context) { mk(c).Update(c) })
	s.PATCH("/:id/disabled", func(c *gin.Context) { mk(c).SetDisabled(c) })
	s.DELETE("/:id", func(c *gin.Context) { mk(c).Delete(c) })
}

func (h *PPPSecret) List(c *gin.Context) {
	ss, err := h.PPP.SecretList(c.Request.Context())
	if err != nil {
		WriteErr(c, err)
		return
	}
	out := dto.FromDomainPPPSecrets(ss)
	WriteList(c, out, len(out))
}

func (h *PPPSecret) GetByName(c *gin.Context) {
	s, err := h.PPP.SecretByName(c.Request.Context(), c.Param("name"))
	if err != nil {
		WriteErr(c, err)
		return
	}
	WriteOK(c, dto.FromDomainPPPSecret(s))
}

func (h *PPPSecret) Create(c *gin.Context) {
	var req dto.PPPSecretCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteValidationErr(c, err)
		return
	}
	id, err := h.PPP.SecretAdd(c.Request.Context(), req.ToArgs())
	if err != nil {
		WriteErr(c, err)
		return
	}
	WriteCreated(c, dto.IDResponse{ID: id})
}

func (h *PPPSecret) Update(c *gin.Context) {
	var req dto.PPPSecretUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteValidationErr(c, err)
		return
	}
	if err := h.PPP.SecretSet(c.Request.Context(), req.ToArgs(c.Param("id"))); err != nil {
		WriteErr(c, err)
		return
	}
	WriteNoContent(c)
}

func (h *PPPSecret) SetDisabled(c *gin.Context) {
	var req dto.SetBoolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteValidationErr(c, err)
		return
	}
	if err := h.PPP.SecretSetDisabled(c.Request.Context(), c.Param("id"), req.Value); err != nil {
		WriteErr(c, err)
		return
	}
	WriteNoContent(c)
}

func (h *PPPSecret) Delete(c *gin.Context) {
	if err := h.PPP.SecretRemove(c.Request.Context(), c.Param("id")); err != nil {
		WriteErr(c, err)
		return
	}
	WriteNoContent(c)
}
