package interceptors

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

const (
	corsMethodGet     = "GET"
	corsMethodPost    = "POST"
	corsMethodPut     = "PUT"
	corsMethodDelete  = "DELETE"
	corsMethodOptions = "OPTIONS"

	corsHeaderContentType = "Content-Type"
	corsHeaderRequestID   = "X-Request-ID"

	corsMaxAge = 12 * time.Hour
)

// CORS allows the given origin (the Next.js dev server by default) to call
// the API's write and read endpoints with a browser fetch.
func CORS(allowedOrigin string) gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins:     []string{allowedOrigin},
		AllowMethods:     []string{corsMethodGet, corsMethodPost, corsMethodPut, corsMethodDelete, corsMethodOptions},
		AllowHeaders:     []string{corsHeaderContentType, corsHeaderRequestID},
		AllowCredentials: true,
		MaxAge:           corsMaxAge,
	})
}
