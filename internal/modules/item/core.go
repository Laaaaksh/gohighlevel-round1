package item

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/Laaaaksh/gohighlevel-round1/internal/logger"
	"github.com/Laaaaksh/gohighlevel-round1/internal/modules/item/entities"
	"github.com/Laaaaksh/gohighlevel-round1/pkg/apperror"
)

// ICore is the item module's business logic, free of HTTP concerns - no
// *gin.Context, no status codes. That is what makes it testable without a
// server or a database (see core_test.go, which mocks IRepository).
type ICore interface {
	CreateItem(ctx context.Context, req entities.CreateItemRequest) (*entities.ItemResponse, error)
	GetItem(ctx context.Context, id string) (*entities.ItemResponse, error)
	ListItems(ctx context.Context) ([]entities.ItemResponse, error)
	UpdateItem(ctx context.Context, id string, req entities.UpdateItemRequest) (*entities.ItemResponse, error)
	DeleteItem(ctx context.Context, id string) error
}

// Core implements ICore against an IRepository. now is injected so tests can
// fix the clock instead of racing time.Now() against gomock's exact-argument
// matching.
type Core struct {
	repo IRepository
	now  func() time.Time
}

var _ ICore = (*Core)(nil)

func NewCore(repo IRepository) *Core {
	return &Core{repo: repo, now: time.Now}
}

// NewCoreWithClock is like NewCore but lets a caller fix the clock - used by
// tests so gomock's exact-argument expectations aren't racing time.Now().
func NewCoreWithClock(repo IRepository, now func() time.Time) *Core {
	return &Core{repo: repo, now: now}
}

func (c *Core) CreateItem(ctx context.Context, req entities.CreateItemRequest) (*entities.ItemResponse, error) {
	if err := validateItemFields(req.Name, req.Description); err != nil {
		return nil, err
	}

	createdAt := c.now().UTC()
	newItem := &Item{
		Name:        req.Name,
		Description: req.Description,
		CreatedAt:   createdAt,
		UpdatedAt:   createdAt,
	}

	if err := c.repo.Create(ctx, newItem); err != nil {
		logger.Ctx(ctx).Error(logMsgCreateItemFailed, logFieldError, err)
		return nil, apperror.Wrap(apperror.CodeInternalError, apperror.MsgInternalError, err)
	}

	logger.Ctx(ctx).Info(logMsgItemCreated, logFieldItemID, newItem.ID.Hex())
	return toResponse(newItem), nil
}

func (c *Core) GetItem(ctx context.Context, id string) (*entities.ItemResponse, error) {
	objectID, err := parseObjectID(id)
	if err != nil {
		return nil, invalidIDError()
	}

	found, err := c.repo.GetByID(ctx, objectID)
	if errors.Is(err, ErrItemNotFound) {
		return nil, notFoundError()
	}
	if err != nil {
		logger.Ctx(ctx).Error(logMsgGetItemFailed, logFieldError, err, logFieldItemID, id)
		return nil, apperror.Wrap(apperror.CodeInternalError, apperror.MsgInternalError, err)
	}
	return toResponse(found), nil
}

func (c *Core) ListItems(ctx context.Context) ([]entities.ItemResponse, error) {
	items, err := c.repo.List(ctx)
	if err != nil {
		logger.Ctx(ctx).Error(logMsgListItemsFailed, logFieldError, err)
		return nil, apperror.Wrap(apperror.CodeInternalError, apperror.MsgInternalError, err)
	}

	responses := make([]entities.ItemResponse, 0, len(items))
	for i := range items {
		responses = append(responses, *toResponse(&items[i]))
	}
	return responses, nil
}

func (c *Core) UpdateItem(ctx context.Context, id string, req entities.UpdateItemRequest) (*entities.ItemResponse, error) {
	if err := validateItemFields(req.Name, req.Description); err != nil {
		return nil, err
	}

	objectID, err := parseObjectID(id)
	if err != nil {
		return nil, invalidIDError()
	}

	updated, err := c.repo.Update(ctx, objectID, &Item{
		Name:        req.Name,
		Description: req.Description,
		UpdatedAt:   c.now().UTC(),
	})
	if errors.Is(err, ErrItemNotFound) {
		return nil, notFoundError()
	}
	if err != nil {
		logger.Ctx(ctx).Error(logMsgUpdateItemFailed, logFieldError, err, logFieldItemID, id)
		return nil, apperror.Wrap(apperror.CodeInternalError, apperror.MsgInternalError, err)
	}

	logger.Ctx(ctx).Info(logMsgItemUpdated, logFieldItemID, id)
	return toResponse(updated), nil
}

func (c *Core) DeleteItem(ctx context.Context, id string) error {
	objectID, err := parseObjectID(id)
	if err != nil {
		return invalidIDError()
	}

	if err := c.repo.Delete(ctx, objectID); err != nil {
		if errors.Is(err, ErrItemNotFound) {
			return notFoundError()
		}
		logger.Ctx(ctx).Error(logMsgDeleteItemFailed, logFieldError, err, logFieldItemID, id)
		return apperror.Wrap(apperror.CodeInternalError, apperror.MsgInternalError, err)
	}

	logger.Ctx(ctx).Info(logMsgItemDeleted, logFieldItemID, id)
	return nil
}

func parseObjectID(id string) (bson.ObjectID, error) {
	return bson.ObjectIDFromHex(id)
}

// validateItemFields is the explicit write-path check that sits behind Gin's
// binding tags. It reads the limits from entities' constants, so the tag
// literals in request.go are a first pass and these constants are the
// authority - a mismatch cannot let a bad value through to the database.
func validateItemFields(name, description string) *apperror.Error {
	if strings.TrimSpace(name) == "" {
		return validationError(apperror.FieldName, apperror.MsgNameRequired)
	}
	if utf8.RuneCountInString(name) > entities.MaxNameLength {
		return validationError(apperror.FieldName, apperror.MsgNameTooLong)
	}
	if utf8.RuneCountInString(description) > entities.MaxDescriptionLength {
		return validationError(apperror.FieldDescription, apperror.MsgDescriptionTooLong)
	}
	return nil
}

func validationError(field, detail string) *apperror.Error {
	return apperror.New(apperror.CodeValidationError, apperror.MsgValidationFailed).WithField(field, detail)
}

func invalidIDError() *apperror.Error {
	return apperror.New(apperror.CodeBadRequest, apperror.MsgInvalidID).
		WithField(apperror.FieldID, apperror.MsgIDMalformed)
}

func notFoundError() *apperror.Error {
	return apperror.New(apperror.CodeNotFound, apperror.MsgItemNotFound).
		WithField(apperror.FieldID, apperror.MsgIDUnknown)
}

func toResponse(item *Item) *entities.ItemResponse {
	return &entities.ItemResponse{
		ID:          item.ID.Hex(),
		Name:        item.Name,
		Description: item.Description,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
	}
}
