package follow

//go:generate mockgen -source=core.go -destination=mock/mock_core.go -package=mock

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/Laaaaksh/gohighlevel-round1/internal/logger"
	"github.com/Laaaaksh/gohighlevel-round1/pkg/apperror"
)

// UserChecker is the narrow read this module needs from the user module -
// declared here, not imported from there; boot.go wires a *user.Core into
// it because the method sets happen to match (Go interfaces are
// structural), the same seam post.UserChecker uses.
type UserChecker interface {
	Exists(ctx context.Context, userID string) (bool, error)
}

// ICore is the follow module's business logic, free of HTTP concerns.
// ListFollowees is the narrow read the timeline module depends on.
type ICore interface {
	Follow(ctx context.Context, followerID, followeeID string) error
	ListFollowees(ctx context.Context, followerID string) ([]string, error)
}

// Core implements ICore against an IRepository and a UserChecker. now is
// injected so tests can fix the clock instead of racing time.Now() against
// gomock's exact-argument matching.
type Core struct {
	repo  IRepository
	users UserChecker
	now   func() time.Time
}

var _ ICore = (*Core)(nil)

func NewCore(repo IRepository, users UserChecker) *Core {
	return &Core{repo: repo, users: users, now: time.Now}
}

// NewCoreWithClock is like NewCore but lets a caller fix the clock - see
// the mock-import-cycle note in AGENTS.md for why core_test.go lives in an
// external package and needs this seam.
func NewCoreWithClock(repo IRepository, users UserChecker, now func() time.Time) *Core {
	return &Core{repo: repo, users: users, now: now}
}

// Follow makes followerID follow followeeID (one-way). It is idempotent:
// following twice is a 200 and leaves exactly one edge, enforced by
// Repository.Follow's upsert against the unique index, never a
// read-then-write check.
func (c *Core) Follow(ctx context.Context, followerID, followeeID string) error {
	if err := validateFollowIDs(followerID, followeeID); err != nil {
		return err
	}

	followeeExists, err := c.users.Exists(ctx, followeeID)
	if err != nil {
		return err
	}
	if !followeeExists {
		return followeeNotFoundError()
	}

	followerExists, err := c.users.Exists(ctx, followerID)
	if err != nil {
		return err
	}
	if !followerExists {
		return followerNotFoundError()
	}

	if err := c.repo.Follow(ctx, followerID, followeeID, c.now().UTC()); err != nil {
		logger.Ctx(ctx).Error(logMsgFollowFailed, logFieldError, err, logFieldFollowerID, followerID, logFieldFolloweeID, followeeID)
		return apperror.Wrap(apperror.CodeInternalError, apperror.MsgInternalError, err)
	}

	logger.Ctx(ctx).Info(logMsgFollowCreated, logFieldFollowerID, followerID, logFieldFolloweeID, followeeID)
	return nil
}

// ListFollowees is a read-path-only lookup for the timeline module - see
// repository.go's ListFolloweeIDs for why it stays index-only.
func (c *Core) ListFollowees(ctx context.Context, followerID string) ([]string, error) {
	followeeIDs, err := c.repo.ListFolloweeIDs(ctx, followerID)
	if err != nil {
		logger.Ctx(ctx).Error(logMsgListFolloweesFailed, logFieldError, err, logFieldFollowerID, followerID)
		return nil, apperror.Wrap(apperror.CodeInternalError, apperror.MsgInternalError, err)
	}
	return followeeIDs, nil
}

func validateFollowIDs(followerID, followeeID string) *apperror.Error {
	if followerID == "" {
		return apperror.New(apperror.CodeBadRequest, apperror.MsgUserIDRequired).WithField(apperror.FieldUserID, apperror.MsgUserIDRequired)
	}
	if _, err := uuid.Parse(followerID); err != nil {
		return invalidUserIDError()
	}
	if _, err := uuid.Parse(followeeID); err != nil {
		return invalidUserIDError()
	}
	if followerID == followeeID {
		return apperror.New(apperror.CodeBadRequest, apperror.MsgSelfFollowNotAllowed).WithField(apperror.FieldUserID, apperror.MsgSelfFollowNotAllowed)
	}
	return nil
}

func invalidUserIDError() *apperror.Error {
	return apperror.New(apperror.CodeBadRequest, apperror.MsgUserIDMalformed).WithField(apperror.FieldUserID, apperror.MsgUserIDMalformed)
}

func followeeNotFoundError() *apperror.Error {
	return apperror.New(apperror.CodeNotFound, apperror.MsgFolloweeNotFound).WithField(apperror.FieldUserID, apperror.MsgUserIDUnknown)
}

func followerNotFoundError() *apperror.Error {
	return apperror.New(apperror.CodeNotFound, apperror.MsgFollowerNotFound).WithField(apperror.FieldUserID, apperror.MsgUserIDUnknown)
}
