// Package item is the copyable pattern for a new resource: entities/ for
// DTOs and constants, repository.go for data access behind an interface,
// core.go for business logic, server.go for HTTP transport, init.go to
// wire it all together. See the README's "add a new resource" section.
package item

//go:generate mockgen -source=repository.go -destination=mock/mock_repository.go -package=mock

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/Laaaaksh/gohighlevel-round1/internal/modules/item/entities"
)

var (
	ErrItemNotFound         = errors.New("item not found")
	ErrUnexpectedInsertedID = errors.New("inserted id is not an object id")
)

const (
	indexNameItemsByName = "idx_items_name"
	indexAscending       = 1
	operatorSet          = "$set"
)

// Item is the domain model, persisted as-is in the items collection.
type Item struct {
	ID          bson.ObjectID `bson:"_id,omitempty"`
	Name        string        `bson:"name"`
	Description string        `bson:"description"`
	CreatedAt   time.Time     `bson:"createdAt"`
	UpdatedAt   time.Time     `bson:"updatedAt"`
}

// IRepository defines item data access. core.go depends only on this
// interface, never on *mongo.Collection, so it is testable without a
// database - see mock/mock_repository.go.
type IRepository interface {
	EnsureIndexes(ctx context.Context) error
	Create(ctx context.Context, item *Item) error
	GetByID(ctx context.Context, id bson.ObjectID) (*Item, error)
	List(ctx context.Context) ([]Item, error)
	Update(ctx context.Context, id bson.ObjectID, item *Item) (*Item, error)
	Delete(ctx context.Context, id bson.ObjectID) error
}

// Repository is the MongoDB implementation of IRepository.
type Repository struct {
	collection *mongo.Collection
}

var _ IRepository = (*Repository)(nil)

func NewRepository(collection *mongo.Collection) *Repository {
	return &Repository{collection: collection}
}

// EnsureIndexes creates the indexes this repository's queries rely on.
// CreateOne is idempotent, so it is safe to call on every boot.
func (r *Repository) EnsureIndexes(ctx context.Context) error {
	_, err := r.collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: entities.FieldName, Value: indexAscending}},
		Options: options.Index().SetName(indexNameItemsByName),
	})
	return err
}

func (r *Repository) Create(ctx context.Context, item *Item) error {
	result, err := r.collection.InsertOne(ctx, item)
	if err != nil {
		return err
	}
	insertedID, ok := result.InsertedID.(bson.ObjectID)
	if !ok {
		return ErrUnexpectedInsertedID
	}
	item.ID = insertedID
	return nil
}

func (r *Repository) GetByID(ctx context.Context, id bson.ObjectID) (*Item, error) {
	var found Item
	err := r.collection.FindOne(ctx, bson.M{entities.FieldID: id}).Decode(&found)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrItemNotFound
	}
	if err != nil {
		return nil, err
	}
	return &found, nil
}

func (r *Repository) List(ctx context.Context) ([]Item, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	items := make([]Item, 0)
	if err := cursor.All(ctx, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *Repository) Update(ctx context.Context, id bson.ObjectID, item *Item) (*Item, error) {
	// UpdatedAt is stamped by core (the clock owner), exactly as on create -
	// the repository writes the timestamp it was handed, never its own.
	update := bson.M{
		operatorSet: bson.M{
			entities.FieldName:        item.Name,
			entities.FieldDescription: item.Description,
			entities.FieldUpdatedAt:   item.UpdatedAt,
		},
	}
	result := r.collection.FindOneAndUpdate(
		ctx,
		bson.M{entities.FieldID: id},
		update,
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	)

	var updated Item
	err := result.Decode(&updated)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrItemNotFound
	}
	if err != nil {
		return nil, err
	}
	return &updated, nil
}

func (r *Repository) Delete(ctx context.Context, id bson.ObjectID) error {
	result, err := r.collection.DeleteOne(ctx, bson.M{entities.FieldID: id})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return ErrItemNotFound
	}
	return nil
}
