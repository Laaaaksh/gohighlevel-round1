// Package interceptors holds Gin middleware: request id propagation, panic
// recovery, request logging, and CORS. Only these files and server.go files
// know about Gin.
package interceptors

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Laaaaksh/gohighlevel-round1/internal/constants"
	"github.com/Laaaaksh/gohighlevel-round1/internal/constants/contextkeys"
)

// RequestID reuses an inbound X-Request-ID header if present, otherwise
// generates one, and echoes it back so client and server logs correlate.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader(constants.HeaderRequestID)
		if requestID == "" {
			requestID = uuid.NewString()
		}

		ctx := context.WithValue(c.Request.Context(), contextkeys.RequestIDKey, requestID)
		c.Request = c.Request.WithContext(ctx)
		c.Header(constants.HeaderRequestID, requestID)
		c.Next()
	}
}
