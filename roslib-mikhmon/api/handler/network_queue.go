package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/quiqxiq/roslib-mikhmon/api/dto"
	"github.com/quiqxiq/roslib-mikhmon/domain"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik/network"
)

type NetworkQueue struct{ Net *network.Client }

func NewNetworkQueue(net *network.Client) *NetworkQueue { return &NetworkQueue{Net: net} }

func (h *NetworkQueue) Register(g *gin.RouterGroup) {
	mk := func(c *gin.Context) *NetworkQueue { return NewNetworkQueue(mustClients(c).Net) }
	q := g.Group("/network/queues")
	q.GET("", func(c *gin.Context) { mk(c).List(c) })
	q.DELETE("/:id", func(c *gin.Context) { mk(c).Delete(c) })
}

func (h *NetworkQueue) List(c *gin.Context) {
	ctx := c.Request.Context()
	var qs []domain.QueueSimple
	var err error
	switch {
	case c.Query("name") != "":
		qs, err = h.Net.QueueSimpleByName(ctx, c.Query("name"))
	case c.Query("dynamic") == "false":
		qs, err = h.Net.QueueSimpleStatic(ctx)
	default:
		qs, err = h.Net.QueueSimpleList(ctx)
	}
	if err != nil {
		WriteErr(c, err)
		return
	}
	out := dto.FromDomainQueues(qs)
	WriteList(c, out, len(out))
}

func (h *NetworkQueue) Delete(c *gin.Context) {
	if err := h.Net.QueueSimpleRemove(c.Request.Context(), c.Param("id")); err != nil {
		WriteErr(c, err)
		return
	}
	WriteNoContent(c)
}
