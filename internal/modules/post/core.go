package post

//go:generate mockgen -source=core.go -destination=mock/mock_core.go -package=mock

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/Laaaaksh/gohighlevel-round1/internal/logger"
	"github.com/Laaaaksh/gohighlevel-round1/internal/modules/post/entities"
	"github.com/Laaaaksh/gohighlevel-round1/pkg/apperror"
	"github.com/Laaaaksh/gohighlevel-round1/pkg/idgen"
)

// UserChecker is the narrow read this module needs from the user module -
// declared here, not imported from there, so post never depends on user's
// package; boot.go wires a *user.Core into it because the method sets
// happen to match (Go interfaces are structural).
type UserChecker interface {
	Exists(ctx context.Context, userID string) (bool, error)
}

// ICore is the post module's business logic, free of HTTP concerns.
type ICore interface {
	CreatePost(ctx context.Context, req entities.CreatePostRequest) (*entities.CreatePostResponse, error)
	ListByUser(ctx context.Context, userID, cursor string, limit int) ([]entities.PostResponse, string, error)
}

// Core implements ICore against an IRepository and a UserChecker. now and
// newID are injected so tests can fix the clock/id instead of racing
// time.Now()/uuid generation against gomock's exact-argument matching.
type Core struct {
	repo  IRepository
	users UserChecker
	now   func() time.Time
	newID func() (string, error)
}

var _ ICore = (*Core)(nil)

func NewCore(repo IRepository, users UserChecker) *Core {
	return &Core{repo: repo, users: users, now: time.Now, newID: idgen.New}
}

// NewCoreWithClock is like NewCore but lets a caller fix the clock and id
// generator - see the mock-import-cycle note in AGENTS.md for why
// core_test.go lives in an external package and needs this seam.
func NewCoreWithClock(repo IRepository, users UserChecker, now func() time.Time, newID func() (string, error)) *Core {
	return &Core{repo: repo, users: users, now: now, newID: newID}
}

func (c *Core) CreatePost(ctx context.Context, req entities.CreatePostRequest) (*entities.CreatePostResponse, error) {
	if err := validatePostFields(req.Title, req.Body); err != nil {
		return nil, err
	}
	if _, err := uuid.Parse(req.UserID); err != nil {
		return nil, invalidUserIDError()
	}

	exists, err := c.users.Exists(ctx, req.UserID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, userNotFoundError()
	}

	postID, err := c.newID()
	if err != nil {
		logger.Ctx(ctx).Error(logMsgGeneratePostIDFailed, logFieldError, err)
		return nil, apperror.Wrap(apperror.CodeInternalError, apperror.MsgInternalError, err)
	}

	newPost := &Post{
		ID:        postID,
		Title:     req.Title,
		Body:      req.Body,
		UserID:    req.UserID,
		CreatedAt: c.now().UTC(),
	}
	if err := c.repo.Create(ctx, newPost); err != nil {
		logger.Ctx(ctx).Error(logMsgCreatePostFailed, logFieldError, err, logFieldUserID, req.UserID)
		return nil, apperror.Wrap(apperror.CodeInternalError, apperror.MsgInternalError, err)
	}

	logger.Ctx(ctx).Info(logMsgPostCreated, logFieldPostID, postID, logFieldUserID, req.UserID)
	return &entities.CreatePostResponse{PostID: postID}, nil
}

func (c *Core) ListByUser(ctx context.Context, userID, cursorParam string, limit int) ([]entities.PostResponse, string, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, "", userIDRequiredError()
	}
	if _, err := uuid.Parse(userID); err != nil {
		return nil, "", invalidUserIDError()
	}

	cursor, err := entities.Decode(cursorParam)
	if err != nil {
		return nil, "", cursorMalformedError()
	}

	posts, nextCursor, err := c.fetchPage(ctx, []string{userID}, cursor, limit)
	if err != nil {
		logger.Ctx(ctx).Error(logMsgListPostsFailed, logFieldError, err, logFieldUserID, userID)
		return nil, "", apperror.Wrap(apperror.CodeInternalError, apperror.MsgInternalError, err)
	}
	return posts, nextCursor, nil
}

// fetchPage is shared by ListByUser and the timeline module (which embeds a
// *Core as its post reader): it clamps the page size, asks the repository
// for one row more than requested, and uses that extra row both to build
// the next cursor and to know whether one exists - no separate count query.
func (c *Core) fetchPage(ctx context.Context, authorIDs []string, cursor entities.Cursor, limit int) ([]entities.PostResponse, string, error) {
	pageSize := clampLimit(limit)

	rows, err := c.repo.ListByAuthors(ctx, authorIDs, cursor, pageSize+1)
	if err != nil {
		return nil, "", err
	}

	hasMore := len(rows) > pageSize
	if hasMore {
		rows = rows[:pageSize]
	}

	responses := make([]entities.PostResponse, 0, len(rows))
	for i := range rows {
		responses = append(responses, toResponse(&rows[i]))
	}

	var nextCursor string
	if hasMore {
		last := rows[len(rows)-1]
		nextCursor = entities.Encode(entities.Cursor{CreatedAt: last.CreatedAt, ID: last.ID})
	}
	return responses, nextCursor, nil
}

// ListByAuthors lets the timeline module reuse this core's paginated fetch
// for many authors at once, instead of duplicating the clamp/hasMore/cursor
// logic above.
func (c *Core) ListByAuthors(ctx context.Context, authorIDs []string, cursor string, limit int) ([]entities.PostResponse, string, error) {
	decoded, err := entities.Decode(cursor)
	if err != nil {
		return nil, "", cursorMalformedError()
	}
	return c.fetchPage(ctx, authorIDs, decoded, limit)
}

// clampLimit applies the default/maximum page size (§2.6.3 decision: an
// oversized limit is clamped, not rejected - a client asking for too much
// is a usability nicety to cap, not an error worth a round trip to fix).
func clampLimit(limit int) int {
	if limit <= 0 {
		return entities.DefaultPageSize
	}
	if limit > entities.MaxPageSize {
		return entities.MaxPageSize
	}
	return limit
}

func validatePostFields(title, body string) *apperror.Error {
	if strings.TrimSpace(title) == "" {
		return validationError(apperror.FieldTitle, apperror.MsgTitleRequired)
	}
	if utf8.RuneCountInString(title) > entities.MaxTitleLength {
		return validationError(apperror.FieldTitle, apperror.MsgTitleTooLong)
	}
	if strings.TrimSpace(body) == "" {
		return validationError(apperror.FieldBody, apperror.MsgBodyRequired)
	}
	if utf8.RuneCountInString(body) > entities.MaxBodyLength {
		return validationError(apperror.FieldBody, apperror.MsgBodyTooLong)
	}
	return nil
}

func validationError(field, detail string) *apperror.Error {
	return apperror.New(apperror.CodeValidationError, apperror.MsgValidationFailed).WithField(field, detail)
}

func invalidUserIDError() *apperror.Error {
	return apperror.New(apperror.CodeBadRequest, apperror.MsgUserIDMalformed).WithField(apperror.FieldUserID, apperror.MsgUserIDMalformed)
}

func userIDRequiredError() *apperror.Error {
	return apperror.New(apperror.CodeBadRequest, apperror.MsgUserIDRequired).WithField(apperror.FieldUserID, apperror.MsgUserIDRequired)
}

func userNotFoundError() *apperror.Error {
	return apperror.New(apperror.CodeNotFound, apperror.MsgUserNotFound).WithField(apperror.FieldUserID, apperror.MsgUserIDUnknown)
}

func cursorMalformedError() *apperror.Error {
	return apperror.New(apperror.CodeBadRequest, apperror.MsgCursorMalformed).WithField(apperror.FieldCursor, apperror.MsgCursorMalformed)
}

func toResponse(p *Post) entities.PostResponse {
	return entities.PostResponse{
		ID:        p.ID,
		Title:     p.Title,
		Body:      p.Body,
		UserID:    p.UserID,
		CreatedAt: p.CreatedAt,
	}
}
