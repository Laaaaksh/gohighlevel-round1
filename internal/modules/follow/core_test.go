// External test package: mock/mock_repository.go and mock/mock_core.go
// import follow for its domain types, so an internal (same-package) test
// importing mock would create an import cycle. See go-testing-standards
// and AGENTS.md for the mocking pattern.
package follow_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/Laaaaksh/gohighlevel-round1/internal/modules/follow"
	"github.com/Laaaaksh/gohighlevel-round1/internal/modules/follow/mock"
	"github.com/Laaaaksh/gohighlevel-round1/pkg/apperror"
)

const (
	testFollowerID       = "018f9a1e-0000-7000-8000-0000000000f1"
	testFolloweeID       = "018f9a1e-0000-7000-8000-0000000000f2"
	testInvalidUserID    = "not-a-uuid"
	errMsgRepositoryDown = "repository unavailable"
)

var testFixedNow = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

func testFixedClock() time.Time { return testFixedNow }

type CoreTestSuite struct {
	suite.Suite
	ctx      context.Context
	ctrl     *gomock.Controller
	mockRepo *mock.MockIRepository
	mockUser *mock.MockUserChecker
	core     *follow.Core
}

func (s *CoreTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.mockRepo = mock.NewMockIRepository(s.ctrl)
	s.mockUser = mock.NewMockUserChecker(s.ctrl)
	s.core = follow.NewCoreWithClock(s.mockRepo, s.mockUser, testFixedClock)
	s.ctx = context.Background()
}

func (s *CoreTestSuite) TearDownTest() {
	s.ctrl.Finish()
}

func (s *CoreTestSuite) TestFollowSuccess() {
	s.mockUser.EXPECT().Exists(s.ctx, testFolloweeID).Return(true, nil).Times(1)
	s.mockUser.EXPECT().Exists(s.ctx, testFollowerID).Return(true, nil).Times(1)
	s.mockRepo.EXPECT().Follow(s.ctx, testFollowerID, testFolloweeID, testFixedNow).Return(nil).Times(1)

	err := s.core.Follow(s.ctx, testFollowerID, testFolloweeID)

	s.NoError(err)
}

func (s *CoreTestSuite) TestFollowIsIdempotentOnSecondCall() {
	s.mockUser.EXPECT().Exists(s.ctx, testFolloweeID).Return(true, nil).Times(2)
	s.mockUser.EXPECT().Exists(s.ctx, testFollowerID).Return(true, nil).Times(2)
	s.mockRepo.EXPECT().Follow(s.ctx, testFollowerID, testFolloweeID, testFixedNow).Return(nil).Times(2)

	s.NoError(s.core.Follow(s.ctx, testFollowerID, testFolloweeID))
	s.NoError(s.core.Follow(s.ctx, testFollowerID, testFolloweeID))
}

func (s *CoreTestSuite) TestFollowMissingFollowerIDFails() {
	err := s.core.Follow(s.ctx, "", testFolloweeID)

	var appErr *apperror.Error
	s.Require().ErrorAs(err, &appErr)
	s.Equal(apperror.CodeBadRequest, appErr.Code)
	s.Equal(apperror.MsgUserIDRequired, appErr.Fields[apperror.FieldUserID])
}

func (s *CoreTestSuite) TestFollowMalformedFollowerIDFails() {
	err := s.core.Follow(s.ctx, testInvalidUserID, testFolloweeID)

	var appErr *apperror.Error
	s.Require().ErrorAs(err, &appErr)
	s.Equal(apperror.CodeBadRequest, appErr.Code)
}

func (s *CoreTestSuite) TestFollowMalformedFolloweeIDFails() {
	err := s.core.Follow(s.ctx, testFollowerID, testInvalidUserID)

	var appErr *apperror.Error
	s.Require().ErrorAs(err, &appErr)
	s.Equal(apperror.CodeBadRequest, appErr.Code)
}

func (s *CoreTestSuite) TestFollowSelfFollowFails() {
	err := s.core.Follow(s.ctx, testFollowerID, testFollowerID)

	var appErr *apperror.Error
	s.Require().ErrorAs(err, &appErr)
	s.Equal(apperror.CodeBadRequest, appErr.Code)
	s.Equal(apperror.MsgSelfFollowNotAllowed, appErr.Fields[apperror.FieldUserID])
}

func (s *CoreTestSuite) TestFollowNonExistentFolloweeFails() {
	s.mockUser.EXPECT().Exists(s.ctx, testFolloweeID).Return(false, nil).Times(1)

	err := s.core.Follow(s.ctx, testFollowerID, testFolloweeID)

	var appErr *apperror.Error
	s.Require().ErrorAs(err, &appErr)
	s.Equal(apperror.CodeNotFound, appErr.Code)
	s.Equal(apperror.MsgFolloweeNotFound, appErr.Message)
}

func (s *CoreTestSuite) TestFollowNonExistentFollowerFails() {
	s.mockUser.EXPECT().Exists(s.ctx, testFolloweeID).Return(true, nil).Times(1)
	s.mockUser.EXPECT().Exists(s.ctx, testFollowerID).Return(false, nil).Times(1)

	err := s.core.Follow(s.ctx, testFollowerID, testFolloweeID)

	var appErr *apperror.Error
	s.Require().ErrorAs(err, &appErr)
	s.Equal(apperror.CodeNotFound, appErr.Code)
	s.Equal(apperror.MsgFollowerNotFound, appErr.Message)
}

func (s *CoreTestSuite) TestFollowRepositoryErrorFails() {
	repoErr := errors.New(errMsgRepositoryDown)
	s.mockUser.EXPECT().Exists(s.ctx, testFolloweeID).Return(true, nil).Times(1)
	s.mockUser.EXPECT().Exists(s.ctx, testFollowerID).Return(true, nil).Times(1)
	s.mockRepo.EXPECT().Follow(s.ctx, testFollowerID, testFolloweeID, testFixedNow).Return(repoErr).Times(1)

	err := s.core.Follow(s.ctx, testFollowerID, testFolloweeID)

	var appErr *apperror.Error
	s.Require().ErrorAs(err, &appErr)
	s.Equal(apperror.CodeInternalError, appErr.Code)
}

func (s *CoreTestSuite) TestListFolloweesSuccess() {
	expected := []string{testFolloweeID}
	s.mockRepo.EXPECT().ListFolloweeIDs(s.ctx, testFollowerID).Return(expected, nil).Times(1)

	followees, err := s.core.ListFollowees(s.ctx, testFollowerID)

	s.NoError(err)
	s.Equal(expected, followees)
}

func (s *CoreTestSuite) TestListFolloweesRepositoryErrorFails() {
	repoErr := errors.New(errMsgRepositoryDown)
	s.mockRepo.EXPECT().ListFolloweeIDs(s.ctx, testFollowerID).Return(nil, repoErr).Times(1)

	followees, err := s.core.ListFollowees(s.ctx, testFollowerID)

	s.Nil(followees)
	var appErr *apperror.Error
	s.Require().ErrorAs(err, &appErr)
	s.Equal(apperror.CodeInternalError, appErr.Code)
}

func TestCoreTestSuite(t *testing.T) {
	suite.Run(t, new(CoreTestSuite))
}
