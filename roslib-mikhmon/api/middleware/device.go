package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/quiqxiq/roslib-mikhmon/api/dto"
	"github.com/quiqxiq/roslib-mikhmon/service/devmgr"
)

// DeviceMiddleware membaca :device_id dari path, mencari ClientSet dari
// DeviceManager, lalu menyimpannya ke context untuk diakses handler.
func DeviceMiddleware(mgr *devmgr.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		slug := c.Param("device_id")
		cs, err := mgr.Get(slug)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusNotFound,
				dto.Err("DEVICE_NOT_FOUND", "device not found or not connected", slug))
			return
		}
		// Import api package akan menyebabkan import cycle — gunakan key langsung
		c.Set("device_clients", cs)
		c.Next()
	}
}
