// Package health backs GET /health. It reports a real Ping result, never a
// hardcoded "connected" string.
package health

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/Laaaaksh/gohighlevel-round1/internal/database"
)

// Status values are typed constants, never inline strings, so a status
// comparison anywhere in the codebase cannot silently mismatch on a typo.
const (
	StatusOK          = "ok"
	StatusUnavailable = "unavailable"
	DatabaseConnected = "connected"
	DatabaseDown      = "disconnected"
)

// Status is the result of a liveness check.
type Status struct {
	Status   string
	Database string
}

type ICore interface {
	Check(ctx context.Context) Status
}

type Core struct {
	mongoClient *mongo.Client
}

var _ ICore = (*Core)(nil)

func NewCore(mongoClient *mongo.Client) *Core {
	return &Core{mongoClient: mongoClient}
}

func (c *Core) Check(ctx context.Context) Status {
	if err := database.Ping(ctx, c.mongoClient); err != nil {
		return Status{Status: StatusUnavailable, Database: DatabaseDown}
	}
	return Status{Status: StatusOK, Database: DatabaseConnected}
}
