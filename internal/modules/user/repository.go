// Package user is the copyable pattern's user resource: registration and
// the narrow, password-free lookups other modules (post, follow) need to
// validate that a userId they were handed actually exists.
package user

//go:generate mockgen -source=repository.go -destination=mock/mock_repository.go -package=mock

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/Laaaaksh/gohighlevel-round1/internal/modules/user/entities"
)

var ErrHandleTaken = errors.New("handle already taken")

// indexNameUsersByHandle enforces registration uniqueness (§3.2) and serves
// a future lookup-by-handle query with the same index - one index, two jobs.
const (
	indexNameUsersByHandle = "idx_users_handle_unique"
	indexAscending         = 1
	projectionInclude      = 1
)

// User is the domain model persisted in the users collection. The id is a
// UUIDv7 string (see pkg/idgen), stamped by core before Create is called -
// the repository never generates or mutates it.
type User struct {
	ID           string    `bson:"_id"`
	Name         string    `bson:"name"`
	Handle       string    `bson:"handle"`
	DOB          time.Time `bson:"dob"`
	PasswordHash string    `bson:"passwordHash"`
	CreatedAt    time.Time `bson:"createdAt"`
}

// idOnly decodes the narrowest possible projection for an existence check -
// never the password hash, even on an internal read path.
type idOnly struct {
	ID string `bson:"_id"`
}

// IRepository defines user data access. core.go depends only on this
// interface, never on *mongo.Collection, so it is testable without a
// database - see mock/mock_repository.go.
type IRepository interface {
	EnsureIndexes(ctx context.Context) error
	Create(ctx context.Context, u *User) error
	ExistsByID(ctx context.Context, id string) (bool, error)
}

// Repository is the MongoDB implementation of IRepository.
type Repository struct {
	collection *mongo.Collection
}

var _ IRepository = (*Repository)(nil)

func NewRepository(collection *mongo.Collection) *Repository {
	return &Repository{collection: collection}
}

// EnsureIndexes creates the unique index on handle. It is the only index
// this module needs today: Create and FindSummaryByID both key on it or on
// _id, which Mongo indexes on its own. Add an index when you add the query
// that needs it, not before.
func (r *Repository) EnsureIndexes(ctx context.Context) error {
	_, err := r.collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: entities.FieldHandle, Value: indexAscending}},
		Options: options.Index().SetName(indexNameUsersByHandle).SetUnique(true),
	})
	return err
}

// Create inserts a user. A duplicate handle surfaces as a driver duplicate
// key error, translated here to ErrHandleTaken so core never needs to know
// what a Mongo write-error code means - see the uniqueness note in the
// project report: this relies on the unique index above, never a
// read-then-write check, which would race under concurrency.
func (r *Repository) Create(ctx context.Context, u *User) error {
	_, err := r.collection.InsertOne(ctx, u)
	if mongo.IsDuplicateKeyError(err) {
		return ErrHandleTaken
	}
	return err
}

// ExistsByID is the narrow lookup other modules (post, follow) use to
// validate a userId. It projects only _id, so it is served entirely from
// the index without ever touching the rest of the document - the shape
// that keeps this under the service's 10ms lookup budget at 10M users.
func (r *Repository) ExistsByID(ctx context.Context, id string) (bool, error) {
	projection := options.FindOne().SetProjection(bson.M{
		entities.FieldID: projectionInclude,
	})

	var found idOnly
	err := r.collection.FindOne(ctx, bson.M{entities.FieldID: id}, projection).Decode(&found)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
