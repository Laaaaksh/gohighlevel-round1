package interceptors

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Laaaaksh/gohighlevel-round1/internal/logger"
	"github.com/Laaaaksh/gohighlevel-round1/pkg/apperror"
)

const (
	logMsgPanicRecovered = "recovered from panic"
	logFieldPanic        = "panic"
)

// Recovery turns a panic into a 500 JSON response instead of a dropped
// connection, and logs the recovered value before responding.
func Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		logger.Ctx(c.Request.Context()).Error(logMsgPanicRecovered, logFieldPanic, recovered)
		appErr := apperror.New(apperror.CodeInternalError, apperror.MsgInternalError)
		c.AbortWithStatusJSON(http.StatusInternalServerError, appErr.Response())
	})
}
