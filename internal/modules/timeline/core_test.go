// External test package: mock/mock_dependencies.go imports the post
// entities package for its domain types, so an internal (same-package)
// test importing mock could create an import cycle if timeline ever grew
// its own entities the mock needed. See go-testing-standards and
// AGENTS.md for the mocking pattern.
package timeline_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	postentities "github.com/Laaaaksh/gohighlevel-round1/internal/modules/post/entities"
	"github.com/Laaaaksh/gohighlevel-round1/internal/modules/timeline"
	"github.com/Laaaaksh/gohighlevel-round1/internal/modules/timeline/mock"
	"github.com/Laaaaksh/gohighlevel-round1/pkg/apperror"
)

const (
	testUserID           = "018f9a1e-0000-7000-8000-0000000000d1"
	testFolloweeID       = "018f9a1e-0000-7000-8000-0000000000d2"
	testInvalidUserID    = "not-a-uuid"
	testCursor           = ""
	testLimit            = 20
	errMsgRepositoryDown = "repository unavailable"
)

type CoreTestSuite struct {
	suite.Suite
	ctx         context.Context
	ctrl        *gomock.Controller
	mockPosts   *mock.MockPostReader
	mockFollows *mock.MockFollowReader
	core        *timeline.Core
}

func (s *CoreTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.mockPosts = mock.NewMockPostReader(s.ctrl)
	s.mockFollows = mock.NewMockFollowReader(s.ctrl)
	s.core = timeline.NewCore(s.mockPosts, s.mockFollows)
	s.ctx = context.Background()
}

func (s *CoreTestSuite) TearDownTest() {
	s.ctrl.Finish()
}

func (s *CoreTestSuite) TestGetTimelineRequiresUserID() {
	posts, cursor, err := s.core.GetTimeline(s.ctx, "", testCursor, testLimit)

	s.Nil(posts)
	s.Empty(cursor)
	var appErr *apperror.Error
	s.Require().ErrorAs(err, &appErr)
	s.Equal(apperror.CodeBadRequest, appErr.Code)
}

func (s *CoreTestSuite) TestGetTimelineRejectsMalformedUserID() {
	posts, cursor, err := s.core.GetTimeline(s.ctx, testInvalidUserID, testCursor, testLimit)

	s.Nil(posts)
	s.Empty(cursor)
	var appErr *apperror.Error
	s.Require().ErrorAs(err, &appErr)
	s.Equal(apperror.CodeBadRequest, appErr.Code)
}

func (s *CoreTestSuite) TestGetTimelineIncludesOwnPostsWithFollowees() {
	expectedAuthorIDs := []string{testFolloweeID, testUserID}
	expectedPosts := []postentities.PostResponse{{ID: "p1", UserID: testFolloweeID}}

	s.mockFollows.EXPECT().ListFollowees(s.ctx, testUserID).Return([]string{testFolloweeID}, nil).Times(1)
	s.mockPosts.EXPECT().ListByAuthors(s.ctx, expectedAuthorIDs, testCursor, testLimit).Return(expectedPosts, "", nil).Times(1)

	posts, nextCursor, err := s.core.GetTimeline(s.ctx, testUserID, testCursor, testLimit)

	s.NoError(err)
	s.Empty(nextCursor)
	s.Equal(expectedPosts, posts)
}

func (s *CoreTestSuite) TestGetTimelineWithNoFolloweesStillReturnsOwnPosts() {
	expectedAuthorIDs := []string{testUserID}
	expectedPosts := []postentities.PostResponse{{ID: "p1", UserID: testUserID}}

	s.mockFollows.EXPECT().ListFollowees(s.ctx, testUserID).Return([]string{}, nil).Times(1)
	s.mockPosts.EXPECT().ListByAuthors(s.ctx, expectedAuthorIDs, testCursor, testLimit).Return(expectedPosts, "", nil).Times(1)

	posts, nextCursor, err := s.core.GetTimeline(s.ctx, testUserID, testCursor, testLimit)

	s.NoError(err)
	s.Empty(nextCursor)
	s.Equal(expectedPosts, posts)
}

func (s *CoreTestSuite) TestGetTimelineFollowReaderErrorFails() {
	repoErr := errors.New(errMsgRepositoryDown)
	s.mockFollows.EXPECT().ListFollowees(s.ctx, testUserID).Return(nil, repoErr).Times(1)

	posts, cursor, err := s.core.GetTimeline(s.ctx, testUserID, testCursor, testLimit)

	s.Nil(posts)
	s.Empty(cursor)
	s.ErrorIs(err, repoErr)
}

func (s *CoreTestSuite) TestGetTimelinePostReaderErrorFails() {
	expectedAuthorIDs := []string{testFolloweeID, testUserID}
	repoErr := errors.New(errMsgRepositoryDown)

	s.mockFollows.EXPECT().ListFollowees(s.ctx, testUserID).Return([]string{testFolloweeID}, nil).Times(1)
	s.mockPosts.EXPECT().ListByAuthors(s.ctx, expectedAuthorIDs, testCursor, testLimit).Return(nil, "", repoErr).Times(1)

	posts, cursor, err := s.core.GetTimeline(s.ctx, testUserID, testCursor, testLimit)

	s.Nil(posts)
	s.Empty(cursor)
	s.ErrorIs(err, repoErr)
}

func (s *CoreTestSuite) TestGetTimelinePropagatesNextCursor() {
	expectedAuthorIDs := []string{testFolloweeID, testUserID}
	const wantCursor = "opaque-cursor-value"

	s.mockFollows.EXPECT().ListFollowees(s.ctx, testUserID).Return([]string{testFolloweeID}, nil).Times(1)
	s.mockPosts.EXPECT().ListByAuthors(s.ctx, expectedAuthorIDs, testCursor, testLimit).Return([]postentities.PostResponse{}, wantCursor, nil).Times(1)

	_, nextCursor, err := s.core.GetTimeline(s.ctx, testUserID, testCursor, testLimit)

	s.NoError(err)
	s.Equal(wantCursor, nextCursor)
}

func TestCoreTestSuite(t *testing.T) {
	suite.Run(t, new(CoreTestSuite))
}
