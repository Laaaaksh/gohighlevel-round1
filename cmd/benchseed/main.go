// Command benchseed is a one-off, non-HTTP loader for the §4.F scale
// checks: it writes directly through the domain repositories with bulk
// inserts, bypassing bcrypt-per-user and the HTTP layer entirely, because
// hashing 100k real passwords or round-tripping 600k HTTP requests would
// turn a data-volume setup step into the slowest part of the exercise.
// It is not part of `make seed` and is never called from boot.go.
package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"golang.org/x/crypto/bcrypt"

	"github.com/Laaaaksh/gohighlevel-round1/internal/config"
	"github.com/Laaaaksh/gohighlevel-round1/internal/database"
	"github.com/Laaaaksh/gohighlevel-round1/internal/logger"
	followEntities "github.com/Laaaaksh/gohighlevel-round1/internal/modules/follow/entities"
	postEntities "github.com/Laaaaksh/gohighlevel-round1/internal/modules/post/entities"
	userEntities "github.com/Laaaaksh/gohighlevel-round1/internal/modules/user/entities"
	"github.com/Laaaaksh/gohighlevel-round1/pkg/idgen"
)

const (
	userCount   = 100_000
	postCount   = 500_000
	batchSize   = 2000
	loadTimeout = 20 * time.Minute

	lowFanoutFolloweeCount  = 50
	highFanoutFolloweeCount = 5000

	// BenchUserLowFanoutHandle/BenchUserHighFanoutHandle are the two
	// well-known accounts the §4.F latency script queries: one following
	// ~50 accounts, one following ~5000 - see docs/social-graph-evals.md.
	benchUserLowFanoutHandle  = "bench-follows-50"
	benchUserHighFanoutHandle = "bench-follows-5000"
	benchUserLookupHandle     = "bench-lookup-target"

	logMsgConfigLoadFailed = "failed to load config"
	logMsgConnectFailed    = "failed to connect to mongodb"
	logMsgDisconnectFailed = "mongo disconnect failed"
	logMsgSeedComplete     = "benchseed complete"
	logMsgInsertFailed     = "bulk insert failed"

	logFieldError = "error"
	logFieldStage = "stage"
	logFieldCount = "count"
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

	ctx, cancel := context.WithTimeout(context.Background(), loadTimeout)
	defer cancel()

	client, err := database.Connect(ctx, cfg.Mongo)
	if err != nil {
		log.Error(logMsgConnectFailed, logFieldError, err)
		return err
	}
	defer func() {
		if err := database.Disconnect(client); err != nil {
			log.Error(logMsgDisconnectFailed, logFieldError, err)
		}
	}()

	db := client.Database(cfg.Mongo.Database)

	userIDs, err := seedUsers(ctx, log, db)
	if err != nil {
		return err
	}

	if err := seedPosts(ctx, log, db, userIDs); err != nil {
		return err
	}

	if err := seedFollowGraph(ctx, log, db, userIDs); err != nil {
		return err
	}

	log.Info(logMsgSeedComplete, "users", userCount, "posts", postCount)
	return nil
}

// seedUsers inserts userCount users and returns every id, for posts and
// follows to reference. All synthetic users share one pre-computed bcrypt
// hash: hashing is deliberately slow (§3.1), and 100k distinct hashes would
// make loading the benchmark dataset slower than the benchmark itself -
// these accounts are never used to test the register/login path.
func seedUsers(ctx context.Context, log *slog.Logger, db *mongo.Database) ([]string, error) {
	sharedHash, err := bcrypt.GenerateFromPassword([]byte("benchseed-password"), bcrypt.MinCost)
	if err != nil {
		return nil, err
	}

	collection := db.Collection(userEntities.CollectionUsers)
	dob := time.Date(1990, time.January, 1, 0, 0, 0, 0, time.UTC)
	now := time.Now().UTC()

	userIDs := make([]string, 0, userCount+3)
	batch := make([]any, 0, batchSize)

	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		_, err := collection.InsertMany(ctx, batch)
		batch = batch[:0]
		return err
	}

	for i := 0; i < userCount; i++ {
		id, err := idgen.New()
		if err != nil {
			return nil, err
		}
		userIDs = append(userIDs, id)
		batch = append(batch, bson.M{
			userEntities.FieldID:           id,
			userEntities.FieldName:         "Bench User",
			userEntities.FieldHandle:       "bench_" + id,
			userEntities.FieldDOB:          dob,
			userEntities.FieldPasswordHash: string(sharedHash),
			userEntities.FieldCreatedAt:    now,
		})
		if len(batch) >= batchSize {
			if err := flush(); err != nil {
				log.Error(logMsgInsertFailed, logFieldStage, "users", logFieldError, err)
				return nil, err
			}
		}
	}

	// The three well-known accounts §4.F's latency script targets by handle.
	for _, handle := range []string{benchUserLowFanoutHandle, benchUserHighFanoutHandle, benchUserLookupHandle} {
		id, err := idgen.New()
		if err != nil {
			return nil, err
		}
		userIDs = append(userIDs, id)
		batch = append(batch, bson.M{
			userEntities.FieldID:           id,
			userEntities.FieldName:         "Bench User",
			userEntities.FieldHandle:       handle,
			userEntities.FieldDOB:          dob,
			userEntities.FieldPasswordHash: string(sharedHash),
			userEntities.FieldCreatedAt:    now,
		})
	}
	if err := flush(); err != nil {
		log.Error(logMsgInsertFailed, logFieldStage, "users", logFieldError, err)
		return nil, err
	}

	log.Info("users seeded", logFieldCount, len(userIDs))
	return userIDs, nil
}

// seedPosts distributes postCount posts round-robin across every seeded
// user, spread over the last 30 days so createdAt values are not all
// identical (which would make the cursor's tie-break the common case
// instead of the rare one it is designed for).
func seedPosts(ctx context.Context, log *slog.Logger, db *mongo.Database, userIDs []string) error {
	collection := db.Collection(postEntities.CollectionPosts)
	base := time.Now().UTC().Add(-30 * 24 * time.Hour)

	batch := make([]any, 0, batchSize)
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		_, err := collection.InsertMany(ctx, batch)
		batch = batch[:0]
		return err
	}

	for i := 0; i < postCount; i++ {
		id, err := idgen.New()
		if err != nil {
			return err
		}
		author := userIDs[i%len(userIDs)]
		createdAt := base.Add(time.Duration(i) * time.Second)
		batch = append(batch, bson.M{
			postEntities.FieldID:        id,
			postEntities.FieldTitle:     "Bench post",
			postEntities.FieldBody:      "Body of a synthetic benchmark post.",
			postEntities.FieldUserID:    author,
			postEntities.FieldCreatedAt: createdAt,
		})
		if len(batch) >= batchSize {
			if err := flush(); err != nil {
				log.Error(logMsgInsertFailed, logFieldStage, "posts", logFieldError, err)
				return err
			}
		}
	}
	if err := flush(); err != nil {
		log.Error(logMsgInsertFailed, logFieldStage, "posts", logFieldError, err)
		return err
	}

	log.Info("posts seeded", logFieldCount, postCount)
	return nil
}

// seedFollowGraph gives the two well-known bench accounts exactly the
// followee counts §4.F measures against (~50 and ~5000), each following
// distinct users from the pool.
func seedFollowGraph(ctx context.Context, log *slog.Logger, db *mongo.Database, userIDs []string) error {
	collection := db.Collection(followEntities.CollectionFollows)
	now := time.Now().UTC()

	// seedUsers appends the three well-known accounts last, in order:
	// low-fanout, high-fanout, lookup-target.
	lowFanoutUser := userIDs[len(userIDs)-3]
	highFanoutUser := userIDs[len(userIDs)-2]

	edges := make([]any, 0, lowFanoutFolloweeCount+highFanoutFolloweeCount)
	for i := 0; i < lowFanoutFolloweeCount; i++ {
		edges = append(edges, followEdge(lowFanoutUser, userIDs[i], now))
	}
	for i := 0; i < highFanoutFolloweeCount; i++ {
		edges = append(edges, followEdge(highFanoutUser, userIDs[i], now))
	}

	if _, err := collection.InsertMany(ctx, edges); err != nil {
		log.Error(logMsgInsertFailed, logFieldStage, "follows", logFieldError, err)
		return err
	}

	log.Info("follow graph seeded", logFieldCount, len(edges))
	return nil
}

func followEdge(followerID, followeeID string, createdAt time.Time) bson.M {
	return bson.M{
		followEntities.FieldFollowerID: followerID,
		followEntities.FieldFolloweeID: followeeID,
		followEntities.FieldCreatedAt:  createdAt,
	}
}
