package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/quiqxiq/roslib-mikhmon/api/dto"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik/network"
)

type NetworkInterface struct{ Net *network.Client }

func NewNetworkInterface(net *network.Client) *NetworkInterface {
	return &NetworkInterface{Net: net}
}

func (h *NetworkInterface) Register(g *gin.RouterGroup) {
	mk := func(c *gin.Context) *NetworkInterface { return NewNetworkInterface(mustClients(c).Net) }
	g.GET("/network/interfaces", func(c *gin.Context) { mk(c).List(c) })
}

func (h *NetworkInterface) List(c *gin.Context) {
	is, err := h.Net.InterfaceList(c.Request.Context())
	if err != nil {
		WriteErr(c, err)
		return
	}
	out := dto.FromDomainInterfaces(is)
	WriteList(c, out, len(out))
}
