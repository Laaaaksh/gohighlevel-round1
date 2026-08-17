package health

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	routeHealth = "/health"

	responseFieldStatus   = "status"
	responseFieldDatabase = "database"
)

type HTTPHandler struct {
	core ICore
}

func NewHTTPHandler(core ICore) *HTTPHandler {
	return &HTTPHandler{core: core}
}

func (h *HTTPHandler) RegisterRoutes(router gin.IRouter) {
	router.GET(routeHealth, h.Check)
}

func (h *HTTPHandler) Check(c *gin.Context) {
	status := h.core.Check(c.Request.Context())

	httpStatus := http.StatusOK
	if status.Status != StatusOK {
		httpStatus = http.StatusServiceUnavailable
	}

	c.JSON(httpStatus, gin.H{
		responseFieldStatus:   status.Status,
		responseFieldDatabase: status.Database,
	})
}
