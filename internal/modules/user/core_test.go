// External test package: mock/mock_repository.go imports user for its
// domain types, so an internal (same-package) test importing mock would
// create an import cycle. See go-testing-standards and AGENTS.md for the
// mocking pattern.
package user_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"golang.org/x/crypto/bcrypt"

	"github.com/Laaaaksh/gohighlevel-round1/internal/modules/user"
	"github.com/Laaaaksh/gohighlevel-round1/internal/modules/user/entities"
	"github.com/Laaaaksh/gohighlevel-round1/internal/modules/user/mock"
	"github.com/Laaaaksh/gohighlevel-round1/pkg/apperror"
)

const (
	testUserID           = "018f9a1e-0000-7000-8000-000000000001"
	testName             = "Ada Lovelace"
	testHandle           = "ada"
	testPassword         = "correct horse battery staple"
	errMsgRepositoryDown = "repository unavailable"
	testPadCharacter     = "a"
	// A three-byte rune, so a byte-based length check would see this name as
	// well over the limit while the binding tag's rune count sees it at
	// exactly the limit.
	testMultiBytePadCharacter = "界"
)

// testFixedNow is a fixed point far enough past any dob used below that
// age arithmetic is exact and never depends on the day the suite runs.
var testFixedNow = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

var (
	testTooLongName          = strings.Repeat(testPadCharacter, entities.MaxNameLength+1)
	testMaxLengthUnicodeName = strings.Repeat(testMultiBytePadCharacter, entities.MaxNameLength)
	testDOBExactly18         = testFixedNow.AddDate(-entities.MinAgeYears, 0, 0).Format(entities.DOBLayout)
	testDOBOneDayShyOf18     = testFixedNow.AddDate(-entities.MinAgeYears, 0, 1).Format(entities.DOBLayout)
	// A leap-day dob whose 18th "anniversary" (2022-03-01, since Go's
	// AddDate rolls Feb 29 + 18y into the following non-leap year onto
	// Mar 1) has already passed by testFixedNow - proving the leap-day
	// case does not panic and resolves as an adult.
	testDOBLeapDay18 = "2004-02-29"
)

func testFixedClock() time.Time    { return testFixedNow }
func testFixedID() (string, error) { return testUserID, nil }

// createdUserMatcher matches the *user.User passed to Repository.Create.
// Every field except PasswordHash is compared exactly; PasswordHash is
// checked against the plaintext password via bcrypt instead of equality,
// because bcrypt salts its output so two hashes of the same password are
// never byte-identical - a plain gomock.Eq would be flaky by construction,
// and go-testing-standards forbids gomock.Any() as the escape hatch.
type createdUserMatcher struct {
	wantID        string
	wantName      string
	wantHandle    string
	wantDOB       time.Time
	wantCreatedAt time.Time
	wantPassword  string
}

func (m createdUserMatcher) Matches(x any) bool {
	got, ok := x.(*user.User)
	if !ok {
		return false
	}
	if got.ID != m.wantID || got.Name != m.wantName || got.Handle != m.wantHandle {
		return false
	}
	if !got.DOB.Equal(m.wantDOB) || !got.CreatedAt.Equal(m.wantCreatedAt) {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(got.PasswordHash), []byte(m.wantPassword)) == nil
}

func (m createdUserMatcher) String() string {
	return fmt.Sprintf("matches user{id:%s name:%s handle:%s} with a valid hash of the expected password", m.wantID, m.wantName, m.wantHandle)
}

type CoreTestSuite struct {
	suite.Suite
	ctx      context.Context
	ctrl     *gomock.Controller
	mockRepo *mock.MockIRepository
	core     *user.Core
}

func (s *CoreTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.mockRepo = mock.NewMockIRepository(s.ctrl)
	s.core = user.NewCoreWithClock(s.mockRepo, testFixedClock, testFixedID)
	s.ctx = context.Background()
}

func (s *CoreTestSuite) TearDownTest() {
	s.ctrl.Finish()
}

func (s *CoreTestSuite) validRequest() entities.RegisterUserRequest {
	return entities.RegisterUserRequest{
		Name:     testName,
		Handle:   testHandle,
		DOB:      testDOBExactly18,
		Password: testPassword,
	}
}

func (s *CoreTestSuite) matcherFor(req entities.RegisterUserRequest, dob time.Time) createdUserMatcher {
	return createdUserMatcher{
		wantID:        testUserID,
		wantName:      req.Name,
		wantHandle:    req.Handle,
		wantDOB:       dob,
		wantCreatedAt: testFixedNow,
		wantPassword:  req.Password,
	}
}

func (s *CoreTestSuite) TestRegisterSuccess() {
	req := s.validRequest()
	dob, err := time.Parse(entities.DOBLayout, req.DOB)
	s.Require().NoError(err)

	s.mockRepo.EXPECT().Create(s.ctx, s.matcherFor(req, dob)).Return(nil).Times(1)

	result, registerErr := s.core.Register(s.ctx, req)

	s.NoError(registerErr)
	s.Require().NotNil(result)
	s.Equal(testUserID, result.UserID)
}

func (s *CoreTestSuite) TestRegisterNameMissingFails() {
	req := s.validRequest()
	req.Name = "   "

	result, err := s.core.Register(s.ctx, req)

	s.Nil(result)
	var appErr *apperror.Error
	s.Require().ErrorAs(err, &appErr)
	s.Equal(apperror.CodeValidationError, appErr.Code)
	s.Equal(apperror.MsgNameRequired, appErr.Fields[apperror.FieldName])
}

func (s *CoreTestSuite) TestRegisterNameTooLongFails() {
	req := s.validRequest()
	req.Name = testTooLongName

	result, err := s.core.Register(s.ctx, req)

	s.Nil(result)
	var appErr *apperror.Error
	s.Require().ErrorAs(err, &appErr)
	s.Equal(apperror.MsgNameTooLong, appErr.Fields[apperror.FieldName])
}

func (s *CoreTestSuite) TestRegisterNameAtMaxUnicodeLengthSucceeds() {
	req := s.validRequest()
	req.Name = testMaxLengthUnicodeName
	dob, err := time.Parse(entities.DOBLayout, req.DOB)
	s.Require().NoError(err)

	s.mockRepo.EXPECT().Create(s.ctx, s.matcherFor(req, dob)).Return(nil).Times(1)

	result, registerErr := s.core.Register(s.ctx, req)

	s.NoError(registerErr)
	s.NotNil(result)
}

func (s *CoreTestSuite) TestRegisterHandleMissingFails() {
	req := s.validRequest()
	req.Handle = ""

	result, err := s.core.Register(s.ctx, req)

	s.Nil(result)
	var appErr *apperror.Error
	s.Require().ErrorAs(err, &appErr)
	s.Equal(apperror.MsgHandleRequired, appErr.Fields[apperror.FieldHandle])
}

func (s *CoreTestSuite) TestRegisterDuplicateHandleFails() {
	req := s.validRequest()
	dob, err := time.Parse(entities.DOBLayout, req.DOB)
	s.Require().NoError(err)

	s.mockRepo.EXPECT().Create(s.ctx, s.matcherFor(req, dob)).Return(user.ErrHandleTaken).Times(1)

	result, registerErr := s.core.Register(s.ctx, req)

	s.Nil(result)
	var appErr *apperror.Error
	s.Require().ErrorAs(registerErr, &appErr)
	s.Equal(apperror.CodeConflict, appErr.Code)
}

func (s *CoreTestSuite) TestRegisterDOBMissingFails() {
	req := s.validRequest()
	req.DOB = ""

	result, err := s.core.Register(s.ctx, req)

	s.Nil(result)
	var appErr *apperror.Error
	s.Require().ErrorAs(err, &appErr)
	s.Equal(apperror.MsgDOBRequired, appErr.Fields[apperror.FieldDOB])
}

func (s *CoreTestSuite) TestRegisterDOBMalformedFails() {
	req := s.validRequest()
	req.DOB = "not-a-date"

	result, err := s.core.Register(s.ctx, req)

	s.Nil(result)
	var appErr *apperror.Error
	s.Require().ErrorAs(err, &appErr)
	s.Equal(apperror.MsgDOBMalformed, appErr.Fields[apperror.FieldDOB])
}

func (s *CoreTestSuite) TestRegisterUnderageByOneDayFails() {
	req := s.validRequest()
	req.DOB = testDOBOneDayShyOf18

	result, err := s.core.Register(s.ctx, req)

	s.Nil(result)
	var appErr *apperror.Error
	s.Require().ErrorAs(err, &appErr)
	s.Equal(apperror.MsgDOBTooYoung, appErr.Fields[apperror.FieldDOB])
}

func (s *CoreTestSuite) TestRegisterExactly18TodaySucceeds() {
	req := s.validRequest()
	req.DOB = testDOBExactly18
	dob, err := time.Parse(entities.DOBLayout, req.DOB)
	s.Require().NoError(err)

	s.mockRepo.EXPECT().Create(s.ctx, s.matcherFor(req, dob)).Return(nil).Times(1)

	result, registerErr := s.core.Register(s.ctx, req)

	s.NoError(registerErr)
	s.NotNil(result)
}

func (s *CoreTestSuite) TestRegisterLeapDayBirthdaySucceeds() {
	req := s.validRequest()
	req.DOB = testDOBLeapDay18
	dob, err := time.Parse(entities.DOBLayout, req.DOB)
	s.Require().NoError(err)

	s.mockRepo.EXPECT().Create(s.ctx, s.matcherFor(req, dob)).Return(nil).Times(1)

	result, registerErr := s.core.Register(s.ctx, req)

	s.NoError(registerErr)
	s.NotNil(result)
}

func (s *CoreTestSuite) TestRegisterPasswordMissingFails() {
	req := s.validRequest()
	req.Password = ""

	result, err := s.core.Register(s.ctx, req)

	s.Nil(result)
	var appErr *apperror.Error
	s.Require().ErrorAs(err, &appErr)
	s.Equal(apperror.MsgPasswordRequired, appErr.Fields[apperror.FieldPassword])
}

func (s *CoreTestSuite) TestRegisterRepositoryErrorFails() {
	req := s.validRequest()
	dob, err := time.Parse(entities.DOBLayout, req.DOB)
	s.Require().NoError(err)
	repoErr := errors.New(errMsgRepositoryDown)

	s.mockRepo.EXPECT().Create(s.ctx, s.matcherFor(req, dob)).Return(repoErr).Times(1)

	result, registerErr := s.core.Register(s.ctx, req)

	s.Nil(result)
	var appErr *apperror.Error
	s.Require().ErrorAs(registerErr, &appErr)
	s.Equal(apperror.CodeInternalError, appErr.Code)
}

func (s *CoreTestSuite) TestExistsTrueWhenRepositoryFindsUser() {
	s.mockRepo.EXPECT().ExistsByID(s.ctx, testUserID).Return(true, nil).Times(1)

	found, err := s.core.Exists(s.ctx, testUserID)

	s.NoError(err)
	s.True(found)
}

func (s *CoreTestSuite) TestExistsFalseWhenRepositoryFindsNothing() {
	s.mockRepo.EXPECT().ExistsByID(s.ctx, testUserID).Return(false, nil).Times(1)

	found, err := s.core.Exists(s.ctx, testUserID)

	s.NoError(err)
	s.False(found)
}

func (s *CoreTestSuite) TestExistsRepositoryErrorFails() {
	repoErr := errors.New(errMsgRepositoryDown)
	s.mockRepo.EXPECT().ExistsByID(s.ctx, testUserID).Return(false, repoErr).Times(1)

	found, err := s.core.Exists(s.ctx, testUserID)

	s.False(found)
	var appErr *apperror.Error
	s.Require().ErrorAs(err, &appErr)
	s.Equal(apperror.CodeInternalError, appErr.Code)
}

func TestCoreTestSuite(t *testing.T) {
	suite.Run(t, new(CoreTestSuite))
}
