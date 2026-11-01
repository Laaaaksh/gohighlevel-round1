package post

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Laaaaksh/gohighlevel-round1/internal/logger"
	"github.com/Laaaaksh/gohighlevel-round1/internal/modules/post/entities"
	"github.com/Laaaaksh/gohighlevel-round1/pkg/apperror"
)

const routePosts = "/posts"

// HTTPHandler is the post module's transport layer. It decodes requests,
// calls core, and maps the result to a status code - it never touches the
// database directly.
type HTTPHandler struct {
	core ICore
}

func NewHTTPHandler(core ICore) *HTTPHandler {
	return &HTTPHandler{core: core}
}

func (h *HTTPHandler) RegisterRoutes(router gin.IRouter) {
	router.POST(routePosts, h.Create)
	router.GET(routePosts, h.List)
}

func (h *HTTPHandler) Create(c *gin.Context) {
	var req entities.CreatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, bindError(c, err))
		return
	}

	created, err := h.core.CreatePost(c.Request.Context(), req)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, created)
}

func (h *HTTPHandler) List(c *gin.Context) {
	var query entities.ListPostsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		writeError(c, bindError(c, err))
		return
	}

	posts, nextCursor, err := h.core.ListByUser(c.Request.Context(), query.UserID, query.Cursor, query.Limit)
	if err != nil {
		writeError(c, err)
		return
	}
	if nextCursor != "" {
		c.Header(entities.HeaderNextCursor, nextCursor)
	}
	c.JSON(http.StatusOK, posts)
}

// bindError keeps the validator's detail out of the client response (it
// names Go struct fields and tags) but logs it, so a rejected request is
// still diagnosable from the server side.
func bindError(c *gin.Context, err error) *apperror.Error {
	logger.Ctx(c.Request.Context()).Warn(logMsgBindRequestFailed, logFieldError, err)
	return apperror.Wrap(apperror.CodeBadRequest, apperror.MsgInvalidRequest, err)
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
