// Package database owns the MongoDB client lifecycle: connecting with a
// verified Ping at startup (never lazily on first query), and index
// creation, which stands in for migrations on a schemaless database.
package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/Laaaaksh/gohighlevel-round1/internal/config"
)

var (
	ErrConnectMongo = errors.New("failed to connect to mongodb")
	ErrPingMongo    = errors.New("failed to ping mongodb")
)

// Connect dials MongoDB and verifies the connection with a Ping so a
// misconfigured or unreachable database fails startup immediately instead
// of surfacing on the first request.
func Connect(ctx context.Context, cfg config.MongoConfig) (*mongo.Client, error) {
	connectCtx, cancel := context.WithTimeout(ctx, time.Duration(cfg.ConnectTimeoutSeconds)*time.Second)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(cfg.URI))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConnectMongo, err)
	}

	if err := client.Ping(connectCtx, nil); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPingMongo, err)
	}

	return client, nil
}

// Ping is used by the health endpoint to verify liveness on demand, and
// reflects a real round trip rather than a cached or hardcoded status.
func Ping(ctx context.Context, client *mongo.Client) error {
	return client.Ping(ctx, nil)
}
