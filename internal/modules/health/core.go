// Package health backs GET /health. It reports a real Ping result, never a
// hardcoded "connected" string.
package health

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/Laaaaksh/gohighlevel-round1/internal/database"
	"github.com/Laaaaksh/gohighlevel-round1/internal/logger"
)

const (
	logMsgPingFailed = "mongodb ping failed"
	logFieldError    = "error"
)

// pingTimeout bounds the liveness check. Without it the driver keeps trying
// to select a server until its own server selection timeout (30s by
// default) expires, so /health would hang instead of answering 503.
const pingTimeout = 2 * time.Second

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
	pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()

	if err := database.Ping(pingCtx, c.mongoClient); err != nil {
		// Check returns only a Status, so this is the one place the cause
		// survives - without it a 503 is indistinguishable from any other.
		logger.Ctx(ctx).Error(logMsgPingFailed, logFieldError, err)
		return Status{Status: StatusUnavailable, Database: DatabaseDown}
	}
	return Status{Status: StatusOK, Database: DatabaseConnected}
}
