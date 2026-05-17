package handler

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/quiqxiq/roslib-mikhmon/api/dto"
	"github.com/quiqxiq/roslib-mikhmon/domain"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik/network"
)

type NetworkPool struct{ Net *network.Client }

func NewNetworkPool(net *network.Client) *NetworkPool { return &NetworkPool{Net: net} }

func (h *NetworkPool) Register(g *gin.RouterGroup) {
	mk := func(c *gin.Context) *NetworkPool { return NewNetworkPool(mustClients(c).Net) }
	g.GET("/network/pools", func(c *gin.Context) { mk(c).List(c) })
}

func (h *NetworkPool) List(c *gin.Context) {
	ctx := c.Request.Context()
	ttl, useCache := parseCacheQuery(c, 5*time.Minute)
	var ps []domain.IPPool
	var err error
	if useCache {
		ps, err = h.Net.PoolListCached(ctx, ttl)
	} else {
		ps, err = h.Net.IPPoolList(ctx)
	}
	if err != nil {
		WriteErr(c, err)
		return
	}
	out := dto.FromDomainPools(ps)
	WriteList(c, out, len(out))
}
