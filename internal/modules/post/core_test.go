// External test package: mock/mock_repository.go and mock/mock_core.go
// import post for its domain types, so an internal (same-package) test
// importing mock would create an import cycle. See go-testing-standards
// and AGENTS.md for the mocking pattern.
package post_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/Laaaaksh/gohighlevel-round1/internal/modules/post"
	"github.com/Laaaaksh/gohighlevel-round1/internal/modules/post/entities"
	"github.com/Laaaaksh/gohighlevel-round1/internal/modules/post/mock"
	"github.com/Laaaaksh/gohighlevel-round1/pkg/apperror"
)

const (
	testPostID           = "018f9a1e-0000-7000-8000-0000000000aa"
	testUserID           = "018f9a1e-0000-7000-8000-0000000000bb"
	testOtherUserID      = "018f9a1e-0000-7000-8000-0000000000cc"
	testInvalidUserID    = "not-a-uuid"
	testTitle            = "Hello world"
	testBody             = "This is the body of the post."
	errMsgRepositoryDown = "repository unavailable"
	testPadCharacter     = "a"
)

var testFixedNow = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

var (
	testTooLongTitle = strings.Repeat(testPadCharacter, entities.MaxTitleLength+1)
	testTooLongBody  = strings.Repeat(testPadCharacter, entities.MaxBodyLength+1)
)

func testFixedClock() time.Time    { return testFixedNow }
func testFixedID() (string, error) { return testPostID, nil }

func fixedPost(id, userID string, createdAt time.Time) post.Post {
	return post.Post{ID: id, Title: testTitle, Body: testBody, UserID: userID, CreatedAt: createdAt}
}

type CoreTestSuite struct {
	suite.Suite
	ctx      context.Context
	ctrl     *gomock.Controller
	mockRepo *mock.MockIRepository
	mockUser *mock.MockUserChecker
	core     *post.Core
}

func (s *CoreTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.mockRepo = mock.NewMockIRepository(s.ctrl)
	s.mockUser = mock.NewMockUserChecker(s.ctrl)
	s.core = post.NewCoreWithClock(s.mockRepo, s.mockUser, testFixedClock, testFixedID)
	s.ctx = context.Background()
}

func (s *CoreTestSuite) TearDownTest() {
	s.ctrl.Finish()
}

func (s *CoreTestSuite) validRequest() entities.CreatePostRequest {
	return entities.CreatePostRequest{Title: testTitle, Body: testBody, UserID: testUserID}
}

func (s *CoreTestSuite) TestCreatePostSuccess() {
	req := s.validRequest()
	expected := &post.Post{ID: testPostID, Title: testTitle, Body: testBody, UserID: testUserID, CreatedAt: testFixedNow}

	s.mockUser.EXPECT().Exists(s.ctx, testUserID).Return(true, nil).Times(1)
	s.mockRepo.EXPECT().Create(s.ctx, expected).Return(nil).Times(1)

	result, err := s.core.CreatePost(s.ctx, req)

	s.NoError(err)
	s.Require().NotNil(result)
	s.Equal(testPostID, result.PostID)
}

func (s *CoreTestSuite) TestCreatePostTitleMissingFails() {
	req := s.validRequest()
	req.Title = "   "

	result, err := s.core.CreatePost(s.ctx, req)

	s.Nil(result)
	var appErr *apperror.Error
	s.Require().ErrorAs(err, &appErr)
	s.Equal(apperror.MsgTitleRequired, appErr.Fields[apperror.FieldTitle])
}

func (s *CoreTestSuite) TestCreatePostTitleTooLongFails() {
	req := s.validRequest()
	req.Title = testTooLongTitle

	result, err := s.core.CreatePost(s.ctx, req)

	s.Nil(result)
	var appErr *apperror.Error
	s.Require().ErrorAs(err, &appErr)
	s.Equal(apperror.MsgTitleTooLong, appErr.Fields[apperror.FieldTitle])
}

func (s *CoreTestSuite) TestCreatePostBodyMissingFails() {
	req := s.validRequest()
	req.Body = ""

	result, err := s.core.CreatePost(s.ctx, req)

	s.Nil(result)
	var appErr *apperror.Error
	s.Require().ErrorAs(err, &appErr)
	s.Equal(apperror.MsgBodyRequired, appErr.Fields[apperror.FieldBody])
}

func (s *CoreTestSuite) TestCreatePostBodyTooLongFails() {
	req := s.validRequest()
	req.Body = testTooLongBody

	result, err := s.core.CreatePost(s.ctx, req)

	s.Nil(result)
	var appErr *apperror.Error
	s.Require().ErrorAs(err, &appErr)
	s.Equal(apperror.MsgBodyTooLong, appErr.Fields[apperror.FieldBody])
}

func (s *CoreTestSuite) TestCreatePostMalformedUserIDFails() {
	req := s.validRequest()
	req.UserID = testInvalidUserID

	result, err := s.core.CreatePost(s.ctx, req)

	s.Nil(result)
	var appErr *apperror.Error
	s.Require().ErrorAs(err, &appErr)
	s.Equal(apperror.CodeBadRequest, appErr.Code)
}

func (s *CoreTestSuite) TestCreatePostNonExistentUserFails() {
	req := s.validRequest()

	s.mockUser.EXPECT().Exists(s.ctx, testUserID).Return(false, nil).Times(1)

	result, err := s.core.CreatePost(s.ctx, req)

	s.Nil(result)
	var appErr *apperror.Error
	s.Require().ErrorAs(err, &appErr)
	s.Equal(apperror.CodeNotFound, appErr.Code)
	s.Equal(apperror.MsgUserIDUnknown, appErr.Fields[apperror.FieldUserID])
}

func (s *CoreTestSuite) TestCreatePostRepositoryErrorFails() {
	req := s.validRequest()
	expected := &post.Post{ID: testPostID, Title: testTitle, Body: testBody, UserID: testUserID, CreatedAt: testFixedNow}
	repoErr := errors.New(errMsgRepositoryDown)

	s.mockUser.EXPECT().Exists(s.ctx, testUserID).Return(true, nil).Times(1)
	s.mockRepo.EXPECT().Create(s.ctx, expected).Return(repoErr).Times(1)

	result, err := s.core.CreatePost(s.ctx, req)

	s.Nil(result)
	var appErr *apperror.Error
	s.Require().ErrorAs(err, &appErr)
	s.Equal(apperror.CodeInternalError, appErr.Code)
}

func (s *CoreTestSuite) TestListByUserRequiresUserID() {
	posts, cursor, err := s.core.ListByUser(s.ctx, "", "", 0)

	s.Nil(posts)
	s.Empty(cursor)
	var appErr *apperror.Error
	s.Require().ErrorAs(err, &appErr)
	s.Equal(apperror.CodeBadRequest, appErr.Code)
	s.Equal(apperror.MsgUserIDRequired, appErr.Fields[apperror.FieldUserID])
}

func (s *CoreTestSuite) TestListByUserRejectsMalformedUserID() {
	posts, cursor, err := s.core.ListByUser(s.ctx, testInvalidUserID, "", 0)

	s.Nil(posts)
	s.Empty(cursor)
	var appErr *apperror.Error
	s.Require().ErrorAs(err, &appErr)
	s.Equal(apperror.CodeBadRequest, appErr.Code)
}

func (s *CoreTestSuite) TestListByUserFirstPageUsesDefaultLimitAndReturnsNoNextCursor() {
	rows := []post.Post{
		fixedPost("p3", testUserID, testFixedNow),
		fixedPost("p2", testUserID, testFixedNow.Add(-time.Minute)),
	}

	s.mockRepo.EXPECT().
		ListByAuthors(s.ctx, []string{testUserID}, entities.Cursor{}, entities.DefaultPageSize+1).
		Return(rows, nil).
		Times(1)

	posts, nextCursor, err := s.core.ListByUser(s.ctx, testUserID, "", 0)

	s.NoError(err)
	s.Empty(nextCursor)
	s.Require().Len(posts, 2)
	s.Equal("p3", posts[0].ID)
	s.Equal("p2", posts[1].ID)
}

func (s *CoreTestSuite) TestListByUserReturnsNextCursorWhenMoreRowsExist() {
	rows := make([]post.Post, 0, entities.DefaultPageSize+1)
	for i := 0; i < entities.DefaultPageSize+1; i++ {
		rows = append(rows, fixedPost(testPostID, testUserID, testFixedNow.Add(-time.Duration(i)*time.Minute)))
	}
	lastKept := rows[entities.DefaultPageSize-1]

	s.mockRepo.EXPECT().
		ListByAuthors(s.ctx, []string{testUserID}, entities.Cursor{}, entities.DefaultPageSize+1).
		Return(rows, nil).
		Times(1)

	posts, nextCursor, err := s.core.ListByUser(s.ctx, testUserID, "", 0)

	s.NoError(err)
	s.Require().Len(posts, entities.DefaultPageSize)
	s.NotEmpty(nextCursor)
	s.Equal(entities.Encode(entities.Cursor{CreatedAt: lastKept.CreatedAt, ID: lastKept.ID}), nextCursor)
}

func (s *CoreTestSuite) TestListByUserClampsLimitAboveMax() {
	s.mockRepo.EXPECT().
		ListByAuthors(s.ctx, []string{testUserID}, entities.Cursor{}, entities.MaxPageSize+1).
		Return([]post.Post{}, nil).
		Times(1)

	posts, _, err := s.core.ListByUser(s.ctx, testUserID, "", entities.MaxPageSize+1000)

	s.NoError(err)
	s.Empty(posts)
}

func (s *CoreTestSuite) TestListByUserRejectsMalformedCursor() {
	posts, cursor, err := s.core.ListByUser(s.ctx, testUserID, "not-a-real-cursor!!", 0)

	s.Nil(posts)
	s.Empty(cursor)
	var appErr *apperror.Error
	s.Require().ErrorAs(err, &appErr)
	s.Equal(apperror.CodeBadRequest, appErr.Code)
	s.Equal(apperror.MsgCursorMalformed, appErr.Fields[apperror.FieldCursor])
}

func (s *CoreTestSuite) TestListByUserRepositoryErrorFails() {
	repoErr := errors.New(errMsgRepositoryDown)

	s.mockRepo.EXPECT().
		ListByAuthors(s.ctx, []string{testUserID}, entities.Cursor{}, entities.DefaultPageSize+1).
		Return(nil, repoErr).
		Times(1)

	posts, cursor, err := s.core.ListByUser(s.ctx, testUserID, "", 0)

	s.Nil(posts)
	s.Empty(cursor)
	var appErr *apperror.Error
	s.Require().ErrorAs(err, &appErr)
	s.Equal(apperror.CodeInternalError, appErr.Code)
}

func (s *CoreTestSuite) TestListByAuthorsUsedByTimelineFetchesMultipleAuthors() {
	authorIDs := []string{testUserID, testOtherUserID}
	rows := []post.Post{fixedPost(testPostID, testOtherUserID, testFixedNow)}

	s.mockRepo.EXPECT().
		ListByAuthors(s.ctx, authorIDs, entities.Cursor{}, entities.DefaultPageSize+1).
		Return(rows, nil).
		Times(1)

	posts, nextCursor, err := s.core.ListByAuthors(s.ctx, authorIDs, "", 0)

	s.NoError(err)
	s.Empty(nextCursor)
	s.Require().Len(posts, 1)
	s.Equal(testOtherUserID, posts[0].UserID)
}

func TestCoreTestSuite(t *testing.T) {
	suite.Run(t, new(CoreTestSuite))
}
