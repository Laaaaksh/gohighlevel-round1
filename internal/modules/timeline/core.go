package timeline

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/Laaaaksh/gohighlevel-round1/internal/logger"
	postentities "github.com/Laaaaksh/gohighlevel-round1/internal/modules/post/entities"
	"github.com/Laaaaksh/gohighlevel-round1/pkg/apperror"
)

// ICore is the timeline module's business logic, free of HTTP concerns.
type ICore interface {
	GetTimeline(ctx context.Context, userID, cursor string, limit int) ([]postentities.PostResponse, string, error)
}

// Core implements ICore by composing a PostReader and a FollowReader - this
// is fan-out-on-read (§3.3): it fetches the followee list at request time,
// then asks for their posts, rather than materializing a per-user timeline
// on every post write. See the project report for the trade-off and the
// follower count at which this degrades.
type Core struct {
	posts   PostReader
	follows FollowReader
}

var _ ICore = (*Core)(nil)

func NewCore(posts PostReader, follows FollowReader) *Core {
	return &Core{posts: posts, follows: follows}
}

func (c *Core) GetTimeline(ctx context.Context, userID, cursor string, limit int) ([]postentities.PostResponse, string, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, "", userIDRequiredError()
	}
	if _, err := uuid.Parse(userID); err != nil {
		return nil, "", invalidUserIDError()
	}

	followeeIDs, err := c.follows.ListFollowees(ctx, userID)
	if err != nil {
		logger.Ctx(ctx).Error(logMsgListFolloweesFailed, logFieldError, err, logFieldUserID, userID)
		return nil, "", err
	}

	authorIDs := followeeIDs
	if includeOwnPosts {
		authorIDs = append(authorIDs, userID)
	}
	if len(authorIDs) == 0 {
		return []postentities.PostResponse{}, "", nil
	}

	return c.posts.ListByAuthors(ctx, authorIDs, cursor, limit)
}

func userIDRequiredError() *apperror.Error {
	return apperror.New(apperror.CodeBadRequest, apperror.MsgUserIDRequired).WithField(apperror.FieldUserID, apperror.MsgUserIDRequired)
}

func invalidUserIDError() *apperror.Error {
	return apperror.New(apperror.CodeBadRequest, apperror.MsgUserIDMalformed).WithField(apperror.FieldUserID, apperror.MsgUserIDMalformed)
}
