// Package post is the copyable pattern's post resource: creating a post and
// listing posts by author, cursor-paginated. ListByAuthors is shared by both
// "my posts" (a single author) and the timeline module (many authors), so
// the pagination and index-usage story is identical for both endpoints.
package post

//go:generate mockgen -source=repository.go -destination=mock/mock_repository.go -package=mock

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/Laaaaksh/gohighlevel-round1/internal/modules/post/entities"
)

// indexNamePostsByUserAndCreatedAt serves two queries with one index:
// "my posts" (§2.3) and the timeline's per-author fan-out (§3.3). Field
// order: equality field (userId) first, then the sort fields
// (createdAt desc, _id desc) - that order lets Mongo both filter by author
// and return the sort already ordered, so it never adds a blocking
// in-memory SORT stage. Reversing the leading field (createdAt, userId)
// would force a full collection-order scan filtered by userId instead of
// an index seek.
//
// _id is part of the index, not just the query's sort, for a reason
// verified by explain(): a sort on (createdAt desc, _id desc) against an
// index that only covers (userId, createdAt) still costs a blocking SORT
// stage, because the index alone cannot prove _id order within an
// equal-createdAt group. Adding _id as a third key removes that stage
// entirely - required for the "no SORT stage" acceptance check - and it is
// also exactly the field the composite cursor (§3.4) ties on, so the same
// three keys serve the filter, the sort, and the pagination boundary.
const (
	indexNamePostsByUserAndCreatedAt = "idx_posts_userId_createdAt_id"
	indexAscending                   = 1
	indexDescending                  = -1
)

// operatorIn/operatorLT/operatorOr are Mongo query operators used to build
// the cursor's continuation filter - named so the raw operator string
// appears exactly once.
const (
	operatorIn = "$in"
	operatorLT = "$lt"
	operatorOr = "$or"
)

// Post is the domain model persisted in the posts collection. The id is a
// UUIDv7 string (see pkg/idgen), stamped by core before Create is called.
type Post struct {
	ID        string    `bson:"_id"`
	Title     string    `bson:"title"`
	Body      string    `bson:"body"`
	UserID    string    `bson:"userId"`
	CreatedAt time.Time `bson:"createdAt"`
}

// IRepository defines post data access. core.go depends only on this
// interface, never on *mongo.Collection, so it is testable without a
// database - see mock/mock_repository.go.
type IRepository interface {
	EnsureIndexes(ctx context.Context) error
	Create(ctx context.Context, post *Post) error
	// ListByAuthors returns up to fetchLimit posts by any of authorIDs,
	// newest first, continuing after cursor. It fetches one extra row past
	// the caller's page size so core can tell whether a further page
	// exists without a separate count query - see core.go.
	ListByAuthors(ctx context.Context, authorIDs []string, cursor entities.Cursor, fetchLimit int) ([]Post, error)
}

// Repository is the MongoDB implementation of IRepository.
type Repository struct {
	collection *mongo.Collection
}

var _ IRepository = (*Repository)(nil)

func NewRepository(collection *mongo.Collection) *Repository {
	return &Repository{collection: collection}
}

// EnsureIndexes creates the one compound index this module's queries need.
// Index creation is idempotent, so it is safe to call on every boot.
func (r *Repository) EnsureIndexes(ctx context.Context) error {
	_, err := r.collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: entities.FieldUserID, Value: indexAscending},
			{Key: entities.FieldCreatedAt, Value: indexDescending},
			{Key: entities.FieldID, Value: indexDescending},
		},
		Options: options.Index().SetName(indexNamePostsByUserAndCreatedAt),
	})
	return err
}

func (r *Repository) Create(ctx context.Context, post *Post) error {
	_, err := r.collection.InsertOne(ctx, post)
	return err
}

// ListByAuthors filters on userId $in authorIDs and sorts by
// (createdAt desc, _id desc) - the tie-break on _id is what makes the
// order total when two posts share a millisecond timestamp, so cursor
// pagination never skips or repeats a row. The tie-break branch of the $or
// only ever matches rows at one exact createdAt value, so it costs nothing
// extra even though _id itself is not part of the compound index.
func (r *Repository) ListByAuthors(ctx context.Context, authorIDs []string, cursor entities.Cursor, fetchLimit int) ([]Post, error) {
	filter := bson.M{entities.FieldUserID: bson.M{operatorIn: authorIDs}}
	if !cursor.CreatedAt.IsZero() {
		filter[operatorOr] = bson.A{
			bson.M{entities.FieldCreatedAt: bson.M{operatorLT: cursor.CreatedAt}},
			bson.M{
				entities.FieldCreatedAt: cursor.CreatedAt,
				entities.FieldID:        bson.M{operatorLT: cursor.ID},
			},
		}
	}

	opts := options.Find().
		SetSort(bson.D{
			{Key: entities.FieldCreatedAt, Value: indexDescending},
			{Key: entities.FieldID, Value: indexDescending},
		}).
		SetLimit(int64(fetchLimit))

	found, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer func() { _ = found.Close(ctx) }()

	posts := make([]Post, 0, fetchLimit)
	if err := found.All(ctx, &posts); err != nil {
		return nil, err
	}
	return posts, nil
}
