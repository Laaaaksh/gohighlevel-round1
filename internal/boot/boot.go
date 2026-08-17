// Package boot wires the service together: connect to MongoDB, ensure
// indexes, construct every module, and register routes. main.go should
// contain nothing but calling Boot and running/shutting down the server.
package boot

import (
	"context"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/Laaaaksh/gohighlevel-round1/internal/config"
	"github.com/Laaaaksh/gohighlevel-round1/internal/database"
	"github.com/Laaaaksh/gohighlevel-round1/internal/interceptors"
	"github.com/Laaaaksh/gohighlevel-round1/internal/modules/health"
	"github.com/Laaaaksh/gohighlevel-round1/internal/modules/item"
	"github.com/Laaaaksh/gohighlevel-round1/internal/modules/item/entities"
)

// App bundles what main.go needs to serve requests and shut down cleanly.
type App struct {
	Router      *gin.Engine
	MongoClient *mongo.Client
}

// Boot connects to MongoDB (fail-fast on an unreachable database), ensures
// indexes, wires every module, and returns a ready-to-serve router.
func Boot(ctx context.Context, cfg config.Config) (app *App, err error) {
	client, err := database.Connect(ctx, cfg.Mongo)
	if err != nil {
		return nil, err
	}

	// Once Connect succeeds the pool and topology goroutines are live, so no
	// failure below may drop the client without closing it. A defer on the
	// named error covers boot steps added here later, not just the ones
	// present today.
	defer func() {
		if err != nil {
			_ = database.Disconnect(client)
		}
	}()

	db := client.Database(cfg.Mongo.Database)
	itemModule := item.NewModule(db.Collection(entities.CollectionItems))
	if err := itemModule.GetRepository().EnsureIndexes(ctx); err != nil {
		return nil, err
	}

	router := gin.New()
	// Logging is registered outside Recovery on purpose: a panic unwinds
	// straight through the inner interceptors, so if Recovery wrapped Logging
	// the request that panicked would never be logged.
	router.Use(
		interceptors.RequestID(),
		interceptors.Logging(),
		interceptors.Recovery(),
		interceptors.CORS(cfg.Server.AllowedOrigin),
	)

	registerRoutes(router, client, itemModule)

	return &App{Router: router, MongoClient: client}, nil
}

func registerRoutes(router *gin.Engine, client *mongo.Client, itemModule item.IModule) {
	healthModule := health.NewModule(client)
	healthModule.GetHandler().RegisterRoutes(router)

	itemModule.GetHandler().RegisterRoutes(router)
}
