package interceptors

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Laaaaksh/gohighlevel-round1/internal/logger"
)

const (
	logMsgRequestHandled = "request handled"
	logFieldMethod       = "method"
	logFieldPath         = "path"
	logFieldStatus       = "status"
	logFieldDurationMS   = "durationMs"
)

// Logging emits one static-message log line per request, with the
// per-request detail as key-value pairs.
func Logging() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		logger.Ctx(c.Request.Context()).Info(
			logMsgRequestHandled,
			logFieldMethod, c.Request.Method,
			logFieldPath, c.Request.URL.Path,
			logFieldStatus, c.Writer.Status(),
			logFieldDurationMS, time.Since(start).Milliseconds(),
		)
	}
}
