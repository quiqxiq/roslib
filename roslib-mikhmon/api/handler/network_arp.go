package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/quiqxiq/roslib-mikhmon/api/dto"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik/network"
)

type NetworkARP struct{ Net *network.Client }

func NewNetworkARP(net *network.Client) *NetworkARP { return &NetworkARP{Net: net} }

func (h *NetworkARP) Register(g *gin.RouterGroup) {
	mk := func(c *gin.Context) *NetworkARP { return NewNetworkARP(mustClients(c).Net) }
	a := g.Group("/network/arp")
	a.GET("", func(c *gin.Context) { mk(c).List(c) })
	a.DELETE("/:id", func(c *gin.Context) { mk(c).Delete(c) })
}

// List ?mac=<mac> wajib — RouterOS /ip/arp/print di mikrotik sub-client
// hanya expose ARPByMAC. Caller pasti tahu MAC yang dicari.
func (h *NetworkARP) List(c *gin.Context) {
	mac := c.Query("mac")
	if mac == "" {
		WriteErr(c, mikrotik.ErrInvalidArgument)
		return
	}
	as, err := h.Net.ARPByMAC(c.Request.Context(), mac)
	if err != nil {
		WriteErr(c, err)
		return
	}
	out := dto.FromDomainARPs(as)
	WriteList(c, out, len(out))
}

func (h *NetworkARP) Delete(c *gin.Context) {
	if err := h.Net.ARPRemove(c.Request.Context(), c.Param("id")); err != nil {
		WriteErr(c, err)
		return
	}
	WriteNoContent(c)
}
