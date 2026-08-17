// Package e2e holds database-dependent integration tests. They run against
// a real MongoDB using config/test.toml's database, and skip automatically
// (via testing.Short or a failed connection) so `go test ./...` never fails
// on a machine without MongoDB running.
package e2e

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/Laaaaksh/gohighlevel-round1/internal/config"
	"github.com/Laaaaksh/gohighlevel-round1/internal/database"
	"github.com/Laaaaksh/gohighlevel-round1/internal/modules/item"
	"github.com/Laaaaksh/gohighlevel-round1/internal/modules/item/entities"
)

const (
	testEnvVar        = "APP_ENV"
	testEnvValue      = "test"
	connectTimeout    = 5 * time.Second
	testItemName        = "E2E Item"
	testItemDescription = "Created by the item e2e suite."
)

type ItemE2ETestSuite struct {
	suite.Suite
	ctx        context.Context
	client     *mongo.Client
	collection *mongo.Collection
	repo       *item.Repository
}

func (s *ItemE2ETestSuite) SetupSuite() {
	if testing.Short() {
		s.T().Skip(skipReasonShortMode)
	}

	os.Setenv(testEnvVar, testEnvValue)
	cfg, err := config.Load()
	s.Require().NoError(err)

	ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)
	defer cancel()

	client, err := database.Connect(ctx, cfg.Mongo)
	if err != nil {
		s.T().Skip(skipReasonNoDatabase)
	}

	s.ctx = context.Background()
	s.client = client
	s.collection = client.Database(cfg.Mongo.Database).Collection(entities.CollectionItems)
	s.repo = item.NewRepository(s.collection)
}

func (s *ItemE2ETestSuite) TearDownSuite() {
	if s.client == nil {
		return
	}
	s.client.Disconnect(s.ctx)
}

func (s *ItemE2ETestSuite) SetupTest() {
	_, err := s.collection.DeleteMany(s.ctx, map[string]any{})
	s.Require().NoError(err)
}

func (s *ItemE2ETestSuite) TestCreateAndGetItemRoundTrip() {
	now := time.Now().UTC()
	newItem := &item.Item{Name: testItemName, Description: testItemDescription, CreatedAt: now, UpdatedAt: now}

	err := s.repo.Create(s.ctx, newItem)
	s.Require().NoError(err)
	s.False(newItem.ID.IsZero())

	found, err := s.repo.GetByID(s.ctx, newItem.ID)
	s.Require().NoError(err)
	s.Equal(testItemName, found.Name)
	s.Equal(testItemDescription, found.Description)
}

func (s *ItemE2ETestSuite) TestDeleteItemRemovesIt() {
	now := time.Now().UTC()
	newItem := &item.Item{Name: testItemName, Description: testItemDescription, CreatedAt: now, UpdatedAt: now}
	s.Require().NoError(s.repo.Create(s.ctx, newItem))

	err := s.repo.Delete(s.ctx, newItem.ID)
	s.Require().NoError(err)

	_, err = s.repo.GetByID(s.ctx, newItem.ID)
	s.ErrorIs(err, item.ErrItemNotFound)
}

const (
	skipReasonShortMode = "skipping e2e test in short mode"
	skipReasonNoDatabase = "skipping e2e test: mongodb is not reachable"
)

func TestItemE2ETestSuite(t *testing.T) {
	suite.Run(t, new(ItemE2ETestSuite))
}
