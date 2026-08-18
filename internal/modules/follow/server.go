package follow

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Laaaaksh/gohighlevel-round1/pkg/apperror"
)

const (
	// routeFollowUser matches the brief's literal contract: GET, not POST,
	// for a state change. Not idempotent-safe in the REST sense (a GET is
	// supposed to be side-effect-free) - implemented as specified anyway,
	// with the idempotency the brief does require handled by the unique
	// index in repository.go.
	routeFollowUser = "/users/follow/:userId"

	// paramFolloweeID and queryFollowerID are both literally named "userId"
	// by the brief - the path param is the user being followed, the query
	// param is the follower. Internal variables are named followeeID/
	// followerID throughout so the two can never be confused with each
	// other.
	paramFolloweeID = "userId"
	queryFollowerID = "userId"
)

// HTTPHandler is the follow module's transport layer. It decodes the
// request, calls core, and maps the result to a status code - it never
// touches the database directly.
type HTTPHandler struct {
	core ICore
}

func NewHTTPHandler(core ICore) *HTTPHandler {
	return &HTTPHandler{core: core}
}

func (h *HTTPHandler) RegisterRoutes(router gin.IRouter) {
	router.GET(routeFollowUser, h.Follow)
}

func (h *HTTPHandler) Follow(c *gin.Context) {
	followeeID := c.Param(paramFolloweeID)
	followerID := c.Query(queryFollowerID)

	if err := h.core.Follow(c.Request.Context(), followerID, followeeID); err != nil {
		writeError(c, err)
		return
	}
	c.Status(http.StatusOK)
}

// writeError maps a core error to its HTTP response. errors.As, not a bare
// type assertion, so a deliberate 404/400 still maps correctly once a
// caller wraps it with %w.
func writeError(c *gin.Context, err error) {
	var appErr *apperror.Error
	if !errors.As(err, &appErr) {
		appErr = apperror.Wrap(apperror.CodeInternalError, apperror.MsgInternalError, err)
	}
	c.JSON(appErr.Code.HTTPStatus(), appErr.Response())
}
