// Package follow is the copyable pattern's follow resource: a one-way,
// idempotent edge between two users.
package follow

//go:generate mockgen -source=repository.go -destination=mock/mock_repository.go -package=mock

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/Laaaaksh/gohighlevel-round1/internal/modules/follow/entities"
)

const (
	// indexNameFollowsByFollowerAndFollowee is unique so a second Follow
	// call for the same pair cannot create a second edge (§2.4: "Follow
	// twice creates exactly one edge") - enforced by the index, never by a
	// read-then-write check, which races under concurrency.
	indexNameFollowsByFollowerAndFollowee = "idx_follows_follower_followee_unique"
	// indexNameFollowsByFollowee is the reverse-lookup index: not used by
	// any query this module issues today, but required the moment a
	// fan-out-on-write timeline needs "who follows this account" - see the
	// project report's §3.3 discussion.
	indexNameFollowsByFollowee = "idx_follows_followee"

	indexAscending = 1

	operatorSetOnInsert = "$setOnInsert"
)

// Follow is the domain model persisted in the follows collection.
type Follow struct {
	FollowerID string    `bson:"followerId"`
	FolloweeID string    `bson:"followeeId"`
	CreatedAt  time.Time `bson:"createdAt"`
}

// IRepository defines follow data access. core.go depends only on this
// interface, never on *mongo.Collection, so it is testable without a
// database - see mock/mock_repository.go.
type IRepository interface {
	EnsureIndexes(ctx context.Context) error
	Follow(ctx context.Context, followerID, followeeID string, createdAt time.Time) error
	ListFolloweeIDs(ctx context.Context, followerID string) ([]string, error)
}

// Repository is the MongoDB implementation of IRepository.
type Repository struct {
	collection *mongo.Collection
}

var _ IRepository = (*Repository)(nil)

func NewRepository(collection *mongo.Collection) *Repository {
	return &Repository{collection: collection}
}

// EnsureIndexes creates both indexes §3.2 requires for this collection.
// Index creation is idempotent, so it is safe to call on every boot.
func (r *Repository) EnsureIndexes(ctx context.Context) error {
	_, err := r.collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: entities.FieldFollowerID, Value: indexAscending},
				{Key: entities.FieldFolloweeID, Value: indexAscending},
			},
			Options: options.Index().SetName(indexNameFollowsByFollowerAndFollowee).SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: entities.FieldFolloweeID, Value: indexAscending}},
			Options: options.Index().SetName(indexNameFollowsByFollowee),
		},
	})
	return err
}

// Follow upserts the edge: the filter is exactly the unique index's key, so
// a second call for the same pair matches the existing document and writes
// nothing new, rather than racing a duplicate-key error against a
// read-then-write check.
func (r *Repository) Follow(ctx context.Context, followerID, followeeID string, createdAt time.Time) error {
	filter := bson.M{
		entities.FieldFollowerID: followerID,
		entities.FieldFolloweeID: followeeID,
	}
	update := bson.M{
		operatorSetOnInsert: bson.M{
			entities.FieldFollowerID: followerID,
			entities.FieldFolloweeID: followeeID,
			entities.FieldCreatedAt:  createdAt,
		},
	}
	_, err := r.collection.UpdateOne(ctx, filter, update, options.UpdateOne().SetUpsert(true))
	return err
}

// ListFolloweeIDs returns every account followerID follows, unsorted - the
// caller (the timeline module) only needs the set of ids for an $in query,
// not an ordering over them. Projecting only followeeId keeps this an
// index-only scan on idx_follows_follower_followee_unique.
func (r *Repository) ListFolloweeIDs(ctx context.Context, followerID string) ([]string, error) {
	projection := options.Find().SetProjection(bson.M{
		entities.FieldFolloweeID: indexAscending,
		entities.FieldID:         0,
	})

	found, err := r.collection.Find(ctx, bson.M{entities.FieldFollowerID: followerID}, projection)
	if err != nil {
		return nil, err
	}
	defer func() { _ = found.Close(ctx) }()

	var rows []Follow
	if err := found.All(ctx, &rows); err != nil {
		return nil, err
	}

	followeeIDs := make([]string, 0, len(rows))
	for _, row := range rows {
		followeeIDs = append(followeeIDs, row.FolloweeID)
	}
	return followeeIDs, nil
}
