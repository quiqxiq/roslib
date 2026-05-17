package api

import (
	"github.com/gin-gonic/gin"
	"github.com/quiqxiq/roslib-mikhmon/api/handler"
	"github.com/quiqxiq/roslib-mikhmon/api/middleware"
)

// RegisterRoutes memasang semua handler ke router group "/api/v1".
// Semua endpoint RouterOS di-prefix dengan /devices/:device_id/ agar
// mendukung multi-router. DeviceMiddleware menyelesaikan :device_id →
// ClientSet dan menyimpannya di gin.Context.
func RegisterRoutes(g *gin.RouterGroup, deps *Deps) {
	// Device CRUD — tidak perlu device middleware
	handler.NewDevices(deps.DeviceStore, deps.DevMgr).Register(g)

	// Semua endpoint RouterOS di bawah device scope
	dev := g.Group("/devices/:device_id")
	dev.Use(middleware.DeviceMiddleware(deps.DevMgr))

	handler.NewSystemInfo(nil).Register(dev)
	handler.NewSystemControl(nil).Register(dev)
	handler.NewSystemLogging(nil).Register(dev)
	handler.NewSystemScript(nil).Register(dev)
	handler.NewSystemScheduler(nil).Register(dev)
	handler.NewLog(nil).Register(dev)

	handler.NewHotspotUser(nil, nil).Register(dev)
	handler.NewHotspotProfile(nil, nil).Register(dev)
	handler.NewHotspotServer(nil).Register(dev)
	handler.NewHotspotActive(nil, nil).Register(dev)
	handler.NewHotspotHost(nil).Register(dev)
	handler.NewHotspotCookie(nil).Register(dev)
	handler.NewHotspotBinding(nil, nil).Register(dev)

	handler.NewNetworkInterface(nil).Register(dev)
	handler.NewNetworkPool(nil).Register(dev)
	handler.NewNetworkARP(nil).Register(dev)
	handler.NewNetworkDHCP(nil).Register(dev)
	handler.NewNetworkQueue(nil).Register(dev)

	handler.NewPPPSecret(nil).Register(dev)
	handler.NewPPPProfile(nil).Register(dev)
	handler.NewPPPActive(nil).Register(dev)

	handler.NewStream(deps.Hub, nil, nil, nil, nil, nil).Register(dev)

	if deps.InfluxReader != nil {
		handler.NewHistory(deps.InfluxReader).Register(dev)
	}
}
