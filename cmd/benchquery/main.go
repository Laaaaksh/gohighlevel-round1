// Command benchquery measures the §4.F/§3.1 latency numbers directly
// against the driver, bypassing HTTP: the brief's 10ms budget is
// "database time for the query", not end-to-end request time, so this
// talks to MongoDB the same way core.go does (same filter, same
// projection, same sort) without a server, router, or JSON encoding in
// the loop.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/Laaaaksh/gohighlevel-round1/internal/config"
	"github.com/Laaaaksh/gohighlevel-round1/internal/database"
	"github.com/Laaaaksh/gohighlevel-round1/internal/logger"
	followEntities "github.com/Laaaaksh/gohighlevel-round1/internal/modules/follow/entities"
	postEntities "github.com/Laaaaksh/gohighlevel-round1/internal/modules/post/entities"
	userEntities "github.com/Laaaaksh/gohighlevel-round1/internal/modules/user/entities"
)

const (
	concurrency            = 50
	iterationsPerGoroutine = 40
	pageLimit              = 21 // DefaultPageSize + 1, matching post.Core's fetchPage

	benchUserLowFanoutHandle  = "bench-follows-50"
	benchUserHighFanoutHandle = "bench-follows-5000"
	benchUserLookupHandle     = "bench-lookup-target"

	queryTimeout = 5 * time.Second
	runTimeout   = 10 * time.Minute

	logMsgConfigLoadFailed = "failed to load config"
	logMsgConnectFailed    = "failed to connect to mongodb"
	logFieldError          = "error"

	projectionInclude = 1
	operatorIn        = "$in"
)

func main() {
	if err := run(logger.New()); err != nil {
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		log.Error(logMsgConfigLoadFailed, logFieldError, err)
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), runTimeout)
	defer cancel()

	client, err := database.Connect(ctx, cfg.Mongo)
	if err != nil {
		log.Error(logMsgConnectFailed, logFieldError, err)
		return err
	}
	defer func() { _ = database.Disconnect(client) }()

	db := client.Database(cfg.Mongo.Database)
	users := db.Collection(userEntities.CollectionUsers)
	posts := db.Collection(postEntities.CollectionPosts)
	follows := db.Collection(followEntities.CollectionFollows)

	lookupUserID, err := userIDByHandle(ctx, users, benchUserLookupHandle)
	if err != nil {
		return fmt.Errorf("resolve lookup user: %w", err)
	}
	lowFanoutUserID, err := userIDByHandle(ctx, users, benchUserLowFanoutHandle)
	if err != nil {
		return fmt.Errorf("resolve low-fanout user: %w", err)
	}
	highFanoutUserID, err := userIDByHandle(ctx, users, benchUserHighFanoutHandle)
	if err != nil {
		return fmt.Errorf("resolve high-fanout user: %w", err)
	}

	lowFanoutAuthors, err := followeeIDs(ctx, follows, lowFanoutUserID)
	if err != nil {
		return fmt.Errorf("load low-fanout followees: %w", err)
	}
	highFanoutAuthors, err := followeeIDs(ctx, follows, highFanoutUserID)
	if err != nil {
		return fmt.Errorf("load high-fanout followees: %w", err)
	}
	fmt.Printf("low-fanout user follows %d accounts, high-fanout user follows %d accounts\n", len(lowFanoutAuthors), len(highFanoutAuthors))

	report("user lookup by id (ExistsByID shape)", runConcurrent(func() error {
		return userExistsByID(ctx, users, lookupUserID)
	}))

	report("timeline query, ~50 followees", runConcurrent(func() error {
		return postsByAuthors(ctx, posts, lowFanoutAuthors)
	}))

	report("timeline query, ~5000 followees", runConcurrent(func() error {
		return postsByAuthors(ctx, posts, highFanoutAuthors)
	}))

	return nil
}

func userIDByHandle(ctx context.Context, users *mongo.Collection, handle string) (string, error) {
	var found struct {
		ID string `bson:"_id"`
	}
	err := users.FindOne(ctx, bson.M{userEntities.FieldHandle: handle}).Decode(&found)
	return found.ID, err
}

func followeeIDs(ctx context.Context, follows *mongo.Collection, followerID string) ([]string, error) {
	projection := options.Find().SetProjection(bson.M{followEntities.FieldFolloweeID: projectionInclude, followEntities.FieldID: 0})
	cursor, err := follows.Find(ctx, bson.M{followEntities.FieldFollowerID: followerID}, projection)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var rows []struct {
		FolloweeID string `bson:"followeeId"`
	}
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.FolloweeID)
	}
	return ids, nil
}

// userExistsByID mirrors user.Repository.ExistsByID's exact query shape:
// filter on _id, project only _id, so the timed work is identical to the
// service's real read path.
func userExistsByID(ctx context.Context, users *mongo.Collection, id string) error {
	queryCtx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	projection := options.FindOne().SetProjection(bson.M{userEntities.FieldID: projectionInclude})
	var found struct {
		ID string `bson:"_id"`
	}
	err := users.FindOne(queryCtx, bson.M{userEntities.FieldID: id}, projection).Decode(&found)
	if err == mongo.ErrNoDocuments {
		return nil
	}
	return err
}

// postsByAuthors mirrors post.Repository.ListByAuthors's first-page query
// shape exactly: $in filter, (createdAt desc, _id desc) sort, page-sized
// limit.
func postsByAuthors(ctx context.Context, posts *mongo.Collection, authorIDs []string) error {
	queryCtx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	opts := options.Find().
		SetSort(bson.D{{Key: postEntities.FieldCreatedAt, Value: -1}, {Key: postEntities.FieldID, Value: -1}}).
		SetLimit(pageLimit)

	cursor, err := posts.Find(queryCtx, bson.M{postEntities.FieldUserID: bson.M{operatorIn: authorIDs}}, opts)
	if err != nil {
		return err
	}
	defer func() { _ = cursor.Close(queryCtx) }()

	var rows []bson.M
	return cursor.All(queryCtx, &rows)
}

// runConcurrent fires concurrency goroutines, each running query
// iterationsPerGoroutine times back to back, and returns every observed
// latency so the caller can compute percentiles over the full sample.
func runConcurrent(query func() error) []time.Duration {
	var (
		wg        sync.WaitGroup
		mu        sync.Mutex
		durations = make([]time.Duration, 0, concurrency*iterationsPerGoroutine)
	)

	for g := 0; g < concurrency; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			local := make([]time.Duration, 0, iterationsPerGoroutine)
			for i := 0; i < iterationsPerGoroutine; i++ {
				start := time.Now()
				if err := query(); err != nil {
					continue
				}
				local = append(local, time.Since(start))
			}
			mu.Lock()
			durations = append(durations, local...)
			mu.Unlock()
		}()
	}
	wg.Wait()
	return durations
}

func report(label string, durations []time.Duration) {
	if len(durations) == 0 {
		fmt.Printf("%s: no successful samples\n", label)
		return
	}
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })

	p50 := percentile(durations, 0.50)
	p95 := durations[percentileIndex(len(durations), 0.95)]
	p99 := durations[percentileIndex(len(durations), 0.99)]

	fmt.Printf(
		"%s: n=%d concurrency=%d p50=%s p95=%s p99=%s\n",
		label, len(durations), concurrency, p50, p95, p99,
	)
}

func percentile(sorted []time.Duration, p float64) time.Duration {
	return sorted[percentileIndex(len(sorted), p)]
}

func percentileIndex(n int, p float64) int {
	idx := int(float64(n) * p)
	if idx >= n {
		idx = n - 1
	}
	return idx
}
