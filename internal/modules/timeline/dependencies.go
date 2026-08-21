// Package timeline composes the post and follow modules into GET
// /timeline: it has no collection and no repository.go of its own - its
// only data access is through the two narrow interfaces below, satisfied
// structurally by *post.Core and *follow.Core (see init.go).
package timeline

//go:generate mockgen -source=dependencies.go -destination=mock/mock_dependencies.go -package=mock

import (
	"context"

	postentities "github.com/Laaaaksh/gohighlevel-round1/internal/modules/post/entities"
)

// PostReader is the read this module needs from the post module: the same
// paginated, multi-author fetch "my posts" uses, so the timeline and "my
// posts" share one pagination/index-usage story - see post/core.go's
// fetchPage.
type PostReader interface {
	ListByAuthors(ctx context.Context, authorIDs []string, cursor string, limit int) ([]postentities.PostResponse, string, error)
}

// FollowReader is the read this module needs from the follow module: the
// set of accounts a user follows, for the fan-out-on-read query - see the
// project report's §3.3 discussion of where this degrades.
type FollowReader interface {
	ListFollowees(ctx context.Context, followerID string) ([]string, error)
}
