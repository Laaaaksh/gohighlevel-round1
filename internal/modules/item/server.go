package item

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Laaaaksh/gohighlevel-round1/internal/logger"
	"github.com/Laaaaksh/gohighlevel-round1/internal/modules/item/entities"
	"github.com/Laaaaksh/gohighlevel-round1/pkg/apperror"
)

const (
	routeItems    = "/api/items"
	routeItemByID = "/api/items/:id"
	paramID       = "id"
)

// HTTPHandler is the item module's transport layer. It decodes requests,
// calls core, and maps the result to a status code - it never touches the
// database directly.
type HTTPHandler struct {
	core ICore
}

func NewHTTPHandler(core ICore) *HTTPHandler {
	return &HTTPHandler{core: core}
}

func (h *HTTPHandler) RegisterRoutes(router gin.IRouter) {
	router.POST(routeItems, h.Create)
	router.GET(routeItems, h.List)
	router.GET(routeItemByID, h.Get)
	router.PUT(routeItemByID, h.Update)
	router.DELETE(routeItemByID, h.Delete)
}

func (h *HTTPHandler) Create(c *gin.Context) {
	var req entities.CreateItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, bindError(c, err))
		return
	}

	created, err := h.core.CreateItem(c.Request.Context(), req)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, created)
}

func (h *HTTPHandler) List(c *gin.Context) {
	items, err := h.core.ListItems(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *HTTPHandler) Get(c *gin.Context) {
	found, err := h.core.GetItem(c.Request.Context(), c.Param(paramID))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, found)
}

func (h *HTTPHandler) Update(c *gin.Context) {
	var req entities.UpdateItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, bindError(c, err))
		return
	}

	updated, err := h.core.UpdateItem(c.Request.Context(), c.Param(paramID), req)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, updated)
}

func (h *HTTPHandler) Delete(c *gin.Context) {
	if err := h.core.DeleteItem(c.Request.Context(), c.Param(paramID)); err != nil {
		writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// bindError keeps the validator's detail out of the client response (it
// names Go struct fields and tags) but logs it, so a rejected request is
// still diagnosable from the server side.
func bindError(c *gin.Context, err error) *apperror.Error {
	ctx := c.Request.Context()
	logger.Ctx(ctx).Warn(logMsgBindRequestFailed, logFieldError, err)
	return apperror.Wrap(apperror.CodeBadRequest, apperror.MsgInvalidRequest, err)
}

// writeError maps a core error to its HTTP response. errors.As, not a bare
// type assertion, so a deliberate 404 or 400 still maps correctly once a
// caller wraps it with %w. A non-*apperror.Error (a bug, not an expected
// failure) still returns a safe, generic body.
func writeError(c *gin.Context, err error) {
	var appErr *apperror.Error
	if !errors.As(err, &appErr) {
		appErr = apperror.Wrap(apperror.CodeInternalError, apperror.MsgInternalError, err)
	}
	c.JSON(appErr.Code.HTTPStatus(), appErr.Response())
}
