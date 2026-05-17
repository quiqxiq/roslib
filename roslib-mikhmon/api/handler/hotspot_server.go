package handler

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/quiqxiq/roslib-mikhmon/api/dto"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik/hotspot"
)

type HotspotServer struct{ Hot *hotspot.Client }

func NewHotspotServer(hot *hotspot.Client) *HotspotServer { return &HotspotServer{Hot: hot} }

func (h *HotspotServer) Register(g *gin.RouterGroup) {
	mk := func(c *gin.Context) *HotspotServer { return NewHotspotServer(mustClients(c).Hot) }
	g.GET("/hotspot/servers", func(c *gin.Context) { mk(c).List(c) })
}

func (h *HotspotServer) List(c *gin.Context) {
	ttl, useCache := parseCacheQuery(c, 5*time.Minute)
	var ss []hotspot.HotspotServer
	var err error
	if useCache {
		ss, err = h.Hot.ServerListCached(c.Request.Context(), ttl)
	} else {
		ss, err = h.Hot.ServerList(c.Request.Context())
	}
	if err != nil {
		WriteErr(c, err)
		return
	}
	out := dto.FromHotspotServers(ss)
	WriteList(c, out, len(out))
}
