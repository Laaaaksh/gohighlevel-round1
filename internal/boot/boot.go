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
	"github.com/Laaaaksh/gohighlevel-round1/internal/modules/follow"
	followEntities "github.com/Laaaaksh/gohighlevel-round1/internal/modules/follow/entities"
	"github.com/Laaaaksh/gohighlevel-round1/internal/modules/health"
	"github.com/Laaaaksh/gohighlevel-round1/internal/modules/item"
	itemEntities "github.com/Laaaaksh/gohighlevel-round1/internal/modules/item/entities"
	"github.com/Laaaaksh/gohighlevel-round1/internal/modules/post"
	postEntities "github.com/Laaaaksh/gohighlevel-round1/internal/modules/post/entities"
	"github.com/Laaaaksh/gohighlevel-round1/internal/modules/timeline"
	"github.com/Laaaaksh/gohighlevel-round1/internal/modules/user"
	userEntities "github.com/Laaaaksh/gohighlevel-round1/internal/modules/user/entities"
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

	itemModule := item.NewModule(db.Collection(itemEntities.CollectionItems))
	if err := itemModule.GetRepository().EnsureIndexes(ctx); err != nil {
		return nil, err
	}

	userModule := user.NewModule(db.Collection(userEntities.CollectionUsers))
	if err := userModule.GetRepository().EnsureIndexes(ctx); err != nil {
		return nil, err
	}

	// post and follow both need to validate a userId against the user
	// module. They declare their own narrow UserChecker interface (see
	// their core.go files) that *user.Core happens to satisfy structurally,
	// so neither package imports user - boot.go is the only place that
	// wires the concrete dependency across module boundaries.
	postModule := post.NewModule(db.Collection(postEntities.CollectionPosts), userModule.GetCore())
	if err := postModule.GetRepository().EnsureIndexes(ctx); err != nil {
		return nil, err
	}

	followModule := follow.NewModule(db.Collection(followEntities.CollectionFollows), userModule.GetCore())
	if err := followModule.GetRepository().EnsureIndexes(ctx); err != nil {
		return nil, err
	}

	// timeline has no collection of its own: it composes post's paginated
	// author-fetch with follow's followee-list read.
	timelineModule := timeline.NewModule(postModule.GetCoreConcrete(), followModule.GetCore())

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

	registerRoutes(router, client, wiredModules{
		item:     itemModule,
		user:     userModule,
		post:     postModule,
		follow:   followModule,
		timeline: timelineModule,
	})

	return &App{Router: router, MongoClient: client}, nil
}

// wiredModules bundles every feature module so registerRoutes stays within
// the "Stop at 4" parameter rule as the module count grows.
type wiredModules struct {
	item     item.IModule
	user     user.IModule
	post     post.IModule
	follow   follow.IModule
	timeline timeline.IModule
}

func registerRoutes(router *gin.Engine, client *mongo.Client, modules wiredModules) {
	healthModule := health.NewModule(client)
	healthModule.GetHandler().RegisterRoutes(router)

	modules.item.GetHandler().RegisterRoutes(router)
	modules.user.GetHandler().RegisterRoutes(router)
	modules.post.GetHandler().RegisterRoutes(router)
	modules.follow.GetHandler().RegisterRoutes(router)
	modules.timeline.GetHandler().RegisterRoutes(router)
}
