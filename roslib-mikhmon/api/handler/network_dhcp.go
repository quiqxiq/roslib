package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/quiqxiq/roslib-mikhmon/api/dto"
	"github.com/quiqxiq/roslib-mikhmon/domain"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik/network"
)

type NetworkDHCP struct{ Net *network.Client }

func NewNetworkDHCP(net *network.Client) *NetworkDHCP { return &NetworkDHCP{Net: net} }

func (h *NetworkDHCP) Register(g *gin.RouterGroup) {
	mk := func(c *gin.Context) *NetworkDHCP { return NewNetworkDHCP(mustClients(c).Net) }
	d := g.Group("/network/dhcp-leases")
	d.GET("", func(c *gin.Context) { mk(c).List(c) })
	d.GET("/count", func(c *gin.Context) { mk(c).Count(c) })
	d.DELETE("/:id", func(c *gin.Context) { mk(c).Delete(c) })
}

func (h *NetworkDHCP) List(c *gin.Context) {
	ctx := c.Request.Context()
	var ls []domain.DHCPLease
	var err error
	if mac := c.Query("mac"); mac != "" {
		ls, err = h.Net.DHCPLeaseByMAC(ctx, mac)
	} else {
		ls, err = h.Net.DHCPLeaseList(ctx)
	}
	if err != nil {
		WriteErr(c, err)
		return
	}
	out := dto.FromDomainLeases(ls)
	WriteList(c, out, len(out))
}

func (h *NetworkDHCP) Count(c *gin.Context) {
	n, err := h.Net.DHCPLeaseCount(c.Request.Context())
	if err != nil {
		WriteErr(c, err)
		return
	}
	WriteOK(c, dto.CountResponse{Count: n})
}

func (h *NetworkDHCP) Delete(c *gin.Context) {
	if err := h.Net.DHCPLeaseRemove(c.Request.Context(), c.Param("id")); err != nil {
		WriteErr(c, err)
		return
	}
	WriteNoContent(c)
}
