// Package database owns the MongoDB client lifecycle and nothing else:
// connecting with a verified Ping at startup (never lazily on first query),
// pinging on demand for /health, and disconnecting. Index creation lives with
// the queries it serves, in each module's repository.go EnsureIndexes.
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

// disconnectTimeout bounds Disconnect's own context. Short: by the time it
// runs the process is on its way out.
const disconnectTimeout = 5 * time.Second

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
		// mongo.Connect already started the connection pool and topology
		// monitoring goroutines, so they must be released before failing.
		_ = Disconnect(client)
		return nil, fmt.Errorf("%w: %v", ErrPingMongo, err)
	}

	return client, nil
}

// Disconnect closes the client on a context of its own rather than the
// caller's. Every caller reaches here on a shutdown or failure path where its
// context may already be cancelled or past its deadline, and Client.Disconnect
// spends that context on the endSessions command it sends before closing the
// pool - a dead context skips it and leaves sessions to expire server-side.
func Disconnect(client *mongo.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), disconnectTimeout)
	defer cancel()

	return client.Disconnect(ctx)
}

// Ping is used by the health endpoint to verify liveness on demand, and
// reflects a real round trip rather than a cached or hardcoded status.
func Ping(ctx context.Context, client *mongo.Client) error {
	return client.Ping(ctx, nil)
}
