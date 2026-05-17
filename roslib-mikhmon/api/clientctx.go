package api

import (
	"github.com/gin-gonic/gin"
	"github.com/quiqxiq/roslib-mikhmon/service/devmgr"
)

const ctxKeyClients = "device_clients"

// SetClients menyimpan ClientSet ke gin.Context untuk diakses handler.
func SetClients(c *gin.Context, cs *devmgr.ClientSet) {
	c.Set(ctxKeyClients, cs)
}

// MustClients mengambil ClientSet dari gin.Context. Panic jika tidak ada
// (berarti DeviceMiddleware tidak dipasang di route tersebut).
func MustClients(c *gin.Context) *devmgr.ClientSet {
	return c.MustGet(ctxKeyClients).(*devmgr.ClientSet)
}
