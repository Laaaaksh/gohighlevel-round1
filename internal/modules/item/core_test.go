// External test package: mock/mock_repository.go imports item for its
// domain types, so an internal (same-package) test importing mock would
// create an import cycle. See go-testing-standards for the mocking pattern.
package item_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.uber.org/mock/gomock"

	"github.com/Laaaaksh/gohighlevel-round1/internal/modules/item"
	"github.com/Laaaaksh/gohighlevel-round1/internal/modules/item/entities"
	"github.com/Laaaaksh/gohighlevel-round1/internal/modules/item/mock"
	"github.com/Laaaaksh/gohighlevel-round1/pkg/apperror"
)

const (
	testItemName         = "Test Item"
	testItemDescription  = "A description used across core tests."
	testItemIDHex        = "507f1f77bcf86cd799439011"
	testInvalidIDHex     = "not-a-valid-object-id"
	testMissingIDHex     = "5f5f5f5f5f5f5f5f5f5f5f5f"
	errMsgRepositoryDown = "repository unavailable"
)

var testFixedTime = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

type CoreTestSuite struct {
	suite.Suite
	ctx      context.Context
	ctrl     *gomock.Controller
	mockRepo *mock.MockIRepository
	core     *item.Core
}

func (s *CoreTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.mockRepo = mock.NewMockIRepository(s.ctrl)
	s.core = item.NewCoreWithClock(s.mockRepo, func() time.Time { return testFixedTime })
	s.ctx = context.Background()
}

func (s *CoreTestSuite) TearDownTest() {
	s.ctrl.Finish()
}

func (s *CoreTestSuite) TestCreateItemSuccess() {
	expected := &item.Item{Name: testItemName, Description: testItemDescription, CreatedAt: testFixedTime, UpdatedAt: testFixedTime}
	testID, err := bson.ObjectIDFromHex(testItemIDHex)
	s.Require().NoError(err)

	s.mockRepo.EXPECT().
		Create(s.ctx, expected).
		DoAndReturn(func(_ context.Context, created *item.Item) error {
			created.ID = testID
			return nil
		}).
		Times(1)

	result, err := s.core.CreateItem(s.ctx, entities.CreateItemRequest{Name: testItemName, Description: testItemDescription})

	s.NoError(err)
	s.Require().NotNil(result)
	s.Equal(testItemIDHex, result.ID)
	s.Equal(testItemName, result.Name)
	s.Equal(testItemDescription, result.Description)
	s.Equal(testFixedTime, result.CreatedAt)
	s.Equal(testFixedTime, result.UpdatedAt)
}

func (s *CoreTestSuite) TestCreateItemRepositoryError() {
	expected := &item.Item{Name: testItemName, Description: testItemDescription, CreatedAt: testFixedTime, UpdatedAt: testFixedTime}
	repoErr := errors.New(errMsgRepositoryDown)

	s.mockRepo.EXPECT().
		Create(s.ctx, expected).
		Return(repoErr).
		Times(1)

	result, err := s.core.CreateItem(s.ctx, entities.CreateItemRequest{Name: testItemName, Description: testItemDescription})

	s.Error(err)
	s.Nil(result)
	appErr, ok := err.(*apperror.Error)
	s.Require().True(ok)
	s.Equal(apperror.CodeInternalError, appErr.Code)
}

func (s *CoreTestSuite) TestGetItemSuccess() {
	testID, err := bson.ObjectIDFromHex(testItemIDHex)
	s.Require().NoError(err)
	stored := &item.Item{ID: testID, Name: testItemName, Description: testItemDescription, CreatedAt: testFixedTime, UpdatedAt: testFixedTime}

	s.mockRepo.EXPECT().GetByID(s.ctx, testID).Return(stored, nil).Times(1)

	result, err := s.core.GetItem(s.ctx, testItemIDHex)

	s.NoError(err)
	s.Require().NotNil(result)
	s.Equal(testItemIDHex, result.ID)
	s.Equal(testItemName, result.Name)
}

func (s *CoreTestSuite) TestGetItemInvalidID() {
	result, err := s.core.GetItem(s.ctx, testInvalidIDHex)

	s.Error(err)
	s.Nil(result)
	appErr, ok := err.(*apperror.Error)
	s.Require().True(ok)
	s.Equal(apperror.CodeBadRequest, appErr.Code)
}

func (s *CoreTestSuite) TestGetItemNotFound() {
	testID, err := bson.ObjectIDFromHex(testMissingIDHex)
	s.Require().NoError(err)

	s.mockRepo.EXPECT().GetByID(s.ctx, testID).Return(nil, item.ErrItemNotFound).Times(1)

	result, err := s.core.GetItem(s.ctx, testMissingIDHex)

	s.Error(err)
	s.Nil(result)
	appErr, ok := err.(*apperror.Error)
	s.Require().True(ok)
	s.Equal(apperror.CodeNotFound, appErr.Code)
}

func (s *CoreTestSuite) TestListItemsSuccess() {
	testID, err := bson.ObjectIDFromHex(testItemIDHex)
	s.Require().NoError(err)
	stored := []item.Item{{ID: testID, Name: testItemName, Description: testItemDescription, CreatedAt: testFixedTime, UpdatedAt: testFixedTime}}

	s.mockRepo.EXPECT().List(s.ctx).Return(stored, nil).Times(1)

	result, err := s.core.ListItems(s.ctx)

	s.NoError(err)
	s.Require().Len(result, 1)
	s.Equal(testItemIDHex, result[0].ID)
}

func (s *CoreTestSuite) TestUpdateItemSuccess() {
	testID, err := bson.ObjectIDFromHex(testItemIDHex)
	s.Require().NoError(err)
	updatedName := "Updated Name"
	expectedArg := &item.Item{Name: updatedName, Description: testItemDescription}
	updated := &item.Item{ID: testID, Name: updatedName, Description: testItemDescription, CreatedAt: testFixedTime, UpdatedAt: testFixedTime}

	s.mockRepo.EXPECT().Update(s.ctx, testID, expectedArg).Return(updated, nil).Times(1)

	result, err := s.core.UpdateItem(s.ctx, testItemIDHex, entities.UpdateItemRequest{Name: updatedName, Description: testItemDescription})

	s.NoError(err)
	s.Require().NotNil(result)
	s.Equal(updatedName, result.Name)
}

func (s *CoreTestSuite) TestUpdateItemNotFound() {
	testID, err := bson.ObjectIDFromHex(testMissingIDHex)
	s.Require().NoError(err)
	expectedArg := &item.Item{Name: testItemName, Description: testItemDescription}

	s.mockRepo.EXPECT().Update(s.ctx, testID, expectedArg).Return(nil, item.ErrItemNotFound).Times(1)

	result, err := s.core.UpdateItem(s.ctx, testMissingIDHex, entities.UpdateItemRequest{Name: testItemName, Description: testItemDescription})

	s.Error(err)
	s.Nil(result)
	appErr, ok := err.(*apperror.Error)
	s.Require().True(ok)
	s.Equal(apperror.CodeNotFound, appErr.Code)
}

func (s *CoreTestSuite) TestDeleteItemSuccess() {
	testID, err := bson.ObjectIDFromHex(testItemIDHex)
	s.Require().NoError(err)

	s.mockRepo.EXPECT().Delete(s.ctx, testID).Return(nil).Times(1)

	err = s.core.DeleteItem(s.ctx, testItemIDHex)

	s.NoError(err)
}

func (s *CoreTestSuite) TestDeleteItemNotFound() {
	testID, err := bson.ObjectIDFromHex(testMissingIDHex)
	s.Require().NoError(err)

	s.mockRepo.EXPECT().Delete(s.ctx, testID).Return(item.ErrItemNotFound).Times(1)

	err = s.core.DeleteItem(s.ctx, testMissingIDHex)

	s.Error(err)
	appErr, ok := err.(*apperror.Error)
	s.Require().True(ok)
	s.Equal(apperror.CodeNotFound, appErr.Code)
}

func (s *CoreTestSuite) TestDeleteItemInvalidID() {
	err := s.core.DeleteItem(s.ctx, testInvalidIDHex)

	s.Error(err)
	appErr, ok := err.(*apperror.Error)
	s.Require().True(ok)
	s.Equal(apperror.CodeBadRequest, appErr.Code)
}

func TestCoreTestSuite(t *testing.T) {
	suite.Run(t, new(CoreTestSuite))
}
