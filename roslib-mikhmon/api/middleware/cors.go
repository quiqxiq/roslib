package middleware

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORS membangun gin-contrib/cors config dengan origins yang di-allow.
// Default permissive (origin "*"); production override via env
// CORS_ALLOWED_ORIGINS.
func CORS(origins []string) gin.HandlerFunc {
	cfg := cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", HeaderRequestID},
		ExposeHeaders:    []string{HeaderRequestID},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}
	if len(origins) == 1 && origins[0] == "*" {
		cfg.AllowAllOrigins = true
	} else {
		cfg.AllowOrigins = origins
	}
	return cors.New(cfg)
}
