package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/quiqxiq/roslib-mikhmon/api/dto"
	"github.com/quiqxiq/roslib-mikhmon/domain"
	"github.com/quiqxiq/roslib-mikhmon/mikrotik/syslog"
)

type Log struct{ Log *syslog.Client }

func NewLog(log *syslog.Client) *Log { return &Log{Log: log} }

func (h *Log) Register(g *gin.RouterGroup) {
	mk := func(c *gin.Context) *Log { return NewLog(mustClients(c).Log) }
	g.GET("/log", func(c *gin.Context) { mk(c).List(c) })
}

func (h *Log) List(c *gin.Context) {
	ctx := c.Request.Context()
	var ls []domain.LogEntry
	var err error
	if topics := c.Query("topics"); topics != "" {
		ls, err = h.Log.LogByTopics(ctx, topics)
	} else {
		ls, err = h.Log.LogList(ctx)
	}
	if err != nil {
		WriteErr(c, err)
		return
	}
	out := dto.FromDomainLogEntries(ls)
	WriteList(c, out, len(out))
}
