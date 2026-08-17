// Command seed inserts a handful of sample items so the frontend has
// something to display without a manual curl first. Tiny and obvious - a
// convenience, not a fixture framework.
package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/Laaaaksh/gohighlevel-round1/internal/config"
	"github.com/Laaaaksh/gohighlevel-round1/internal/database"
	"github.com/Laaaaksh/gohighlevel-round1/internal/logger"
	"github.com/Laaaaksh/gohighlevel-round1/internal/modules/item"
	"github.com/Laaaaksh/gohighlevel-round1/internal/modules/item/entities"
)

const seedTimeout = 10 * time.Second

const (
	msgConfigLoadFailed = "failed to load config"
	msgConnectFailed    = "failed to connect to mongodb"
	msgSeedItemFailed   = "failed to seed item"
	msgSeedItemCreated  = "seeded item"
	msgSeedComplete     = "seed complete"
	msgDisconnectFailed = "mongo disconnect failed"

	logFieldError = "error"
	logFieldName  = "name"
	logFieldID    = "id"
	logFieldCount = "count"
)

type seedItem struct {
	Name        string
	Description string
}

var seedItems = []seedItem{
	{Name: "First Sample Item", Description: "A starter item created by the seed script."},
	{Name: "Second Sample Item", Description: "Another starter item so the list view has more than one row."},
	{Name: "Third Sample Item", Description: "A third starter item for sorting and pagination checks."},
}

func main() {
	log := logger.New()

	cfg, err := config.Load()
	if err != nil {
		log.Error(msgConfigLoadFailed, logFieldError, err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), seedTimeout)
	defer cancel()

	client, err := database.Connect(ctx, cfg.Mongo)
	if err != nil {
		log.Error(msgConnectFailed, logFieldError, err)
		os.Exit(1)
	}
	defer func() {
		if err := client.Disconnect(ctx); err != nil {
			log.Error(msgDisconnectFailed, logFieldError, err)
		}
	}()

	collection := client.Database(cfg.Mongo.Database).Collection(entities.CollectionItems)
	repo := item.NewRepository(collection)

	for _, seed := range seedItems {
		if err := insertSeedItem(ctx, repo, seed, log); err != nil {
			os.Exit(1)
		}
	}

	log.Info(msgSeedComplete, logFieldCount, len(seedItems))
}

func insertSeedItem(ctx context.Context, repo *item.Repository, seed seedItem, log *slog.Logger) error {
	now := time.Now().UTC()
	newItem := &item.Item{Name: seed.Name, Description: seed.Description, CreatedAt: now, UpdatedAt: now}

	if err := repo.Create(ctx, newItem); err != nil {
		log.Error(msgSeedItemFailed, logFieldError, err, logFieldName, seed.Name)
		return err
	}

	log.Info(msgSeedItemCreated, logFieldID, newItem.ID.Hex(), logFieldName, newItem.Name)
	return nil
}
