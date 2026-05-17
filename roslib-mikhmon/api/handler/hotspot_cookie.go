package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/quiqxiq/roslib-mikhmon/api/dto"
	"github.com/quiqxiq/roslib-mikhmon/domain"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik/hotspot"
)

type HotspotCookie struct{ Hot *hotspot.Client }

func NewHotspotCookie(hot *hotspot.Client) *HotspotCookie { return &HotspotCookie{Hot: hot} }

func (h *HotspotCookie) Register(g *gin.RouterGroup) {
	mk := func(c *gin.Context) *HotspotCookie { return NewHotspotCookie(mustClients(c).Hot) }
	cookies := g.Group("/hotspot/cookies")
	cookies.GET("", func(c *gin.Context) { mk(c).List(c) })
	cookies.GET("/count", func(c *gin.Context) { mk(c).Count(c) })
	cookies.DELETE("/:id", func(c *gin.Context) { mk(c).Delete(c) })
}

func (h *HotspotCookie) List(c *gin.Context) {
	ctx := c.Request.Context()
	var cs []domain.HotspotCookie
	var err error
	if user := c.Query("user"); user != "" {
		cs, err = h.Hot.CookieByUser(ctx, user)
	} else {
		cs, err = h.Hot.CookieList(ctx)
	}
	if err != nil {
		WriteErr(c, err)
		return
	}
	out := dto.FromDomainCookies(cs)
	WriteList(c, out, len(out))
}

func (h *HotspotCookie) Count(c *gin.Context) {
	n, err := h.Hot.CookieCount(c.Request.Context())
	if err != nil {
		WriteErr(c, err)
		return
	}
	WriteOK(c, dto.CountResponse{Count: n})
}

func (h *HotspotCookie) Delete(c *gin.Context) {
	if err := h.Hot.CookieRemove(c.Request.Context(), c.Param("id")); err != nil {
		WriteErr(c, err)
		return
	}
	WriteNoContent(c)
}
