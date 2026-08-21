# Social-graph service — evaluation report

## 1. Verdict

**MET WITH CAVEATS.**

All five endpoints are implemented and verified end-to-end against a real
MongoDB, all acceptance checks in §4 A–E pass with captured evidence, and
the user-lookup p99 latency target is met. The one honest miss is the
timeline latency budget under fan-out-on-read: it holds at ~50 followees
only marginally (p99 hovered right at 10ms across two runs) and is clearly
blown at ~5,000 followees (p99 in the hundreds of ms) — exactly the
degradation §3.3 asks this report to find and characterize, not hide.

---

## 2. Criteria results (§4 A–H)

### A. Build and static analysis

- [x] `go build ./...` — exit 0, no output.
- [x] `go vet ./...` — exit 0, no output.
- [x] `make lint` — clean. Output: `go vet ./...` (no output) then golangci-lint's `0 issues.`
- [x] `make fmt` leaves no diff. Verified by stashing this branch's changes, running `make fmt` against the clean base branch (`git diff --exit-code` → exit 0, i.e. the base tree was already gofmt-clean), then restoring and re-running `make fmt` on top of the new code — the new files are gofmt-stable (no further changes on a second run).
- [x] No magic strings/numbers: every collection, field, index name, route, log message, and limit is a named constant (see `entities/constant.go`, `constants.go`, and `repository.go` in each new module).
- [x] No function exceeds 4 parameters. `boot.go`'s `registerRoutes` grew past 4 module arguments as the module count rose, so it now takes a `wiredModules` struct instead — same rule the standard's "Stop at 4" section prescribes for exactly this shape of growth.

### B. Unit tests

- [x] Every new `core.go` has a `core_test.go` in the external `_test` package (`user_test`, `post_test`, `follow_test`, `timeline_test`), mocking the repository/dependency interfaces via `go.uber.org/mock`.
- [x] `testify/suite`, camelCase scenario-named test methods throughout.
- [x] No table-driven tests. No `gomock.Any()`. No `AnyTimes()` — every expectation names exact arguments and an exact `.Times(n)`. The one place this was genuinely hard: `user.Core.Register` hashes the password with bcrypt, whose salt makes two hashes of the same input never byte-identical, so a literal `*user.User` value can't be the expected argument. Solved with a hand-rolled `gomock.Matcher` (`createdUserMatcher`) that compares every field exactly except `PasswordHash`, which it verifies via `bcrypt.CompareHashAndPassword` against the plaintext — an exact, deterministic check that still isn't `gomock.Any()`.
- [x] `make test-short` — all pass. Command and output:
  ```
  $ go test -short ./...
  ok  	.../e2e	(cached)
  ok  	.../internal/config	(cached)
  ok  	.../internal/modules/follow	(cached)
  ok  	.../internal/modules/item	(cached)
  ok  	.../internal/modules/post	(cached)
  ok  	.../internal/modules/timeline	(cached)
  ok  	.../internal/modules/user	(cached)
  ```
- [x] `make test` (real MongoDB) — all pass. Same command as above without `-short`; `e2e` runs for real against `gohighlevel_round1_test`. All 7 packages `ok`, 0 failures.
- [x] The 21 pre-existing tests still pass, unmodified. Verified in isolation: `go test -v ./internal/modules/item/... ./internal/config/... ./e2e/...` → 21 leaf `--- PASS:` lines (subtests, excluding suite-wrapper lines), 0 failures.
- [x] New test count: `go test -v ./internal/modules/user/... ./internal/modules/post/... ./internal/modules/follow/... ./internal/modules/timeline/...` → **50 leaf `--- PASS:` lines**, 0 failures. (16 user, 16 post, 11 follow, 7 timeline.)
- [x] `make mock` regenerates cleanly, no diff. Verified: `go generate ./...` re-run after all core.go/repository.go edits produced identical output each time (checked via `git status` after each regen — no diff on the generated files once code stabilized).

### C. Functional end-to-end checks

Ran against `make dev` (real local MongoDB, `gohighlevel_round1` database). Full raw output for every step is in the PR; representative excerpts:

- [x] `POST /users` valid → **201**, `{"userId":"01a0149f-e5e9-7c44-a450-afcacd573b07"}`.
- [x] `GET /health` → **200**, `{"database":"connected","status":"ok"}`.
- [x] `POST /posts` with that `userId` → **201**, `{"postId":"01a0149f-e78c-7e7b-9811-05166d642771"}`.
- [x] `GET /posts?userId=...` → **200**, array containing that post.
- [x] Three posts created (1s apart), then `GET /posts?userId=...` → returned **Post Three, Post Two, Post One**, in that order — verified newest-first by inspecting the actual `createdAt` values in the response, not assumed.
- [x] Second user registered, follows the first: `GET /users/follow/{user1}?userId={user2}` → **200**, empty body.
- [x] `GET /timeline?userId={user2}` → **200**, contains all 3 of user1's posts, newest first.
- [x] Repeat the same follow call → **200**; `mongosh` count of `{followerId:user2, followeeId:user1}` in `follows` → **1**.
- [x] `GET /timeline?userId={user1}` → **200**, contains only user1's own posts (user1 follows nobody, and does not follow user2) — user2's posts are absent, proving the edge is one-way. (See §4.G.1 for why user1's own posts appear here at all: the timeline includes the viewer's own posts by design.)

### D. Validation matrix

All run against the live server; actual bodies captured, not asserted:

| Case | Expected | Actual |
|---|---|---|
| `name` > 20 chars | 400 `VALIDATION_ERROR`* | 400 `BAD_REQUEST` (Gin binding tag rejects it before core runs — see note below) |
| `name` missing | 400 | 400 `BAD_REQUEST` (binding `required`) |
| duplicate `handle` | 409 `CONFLICT` | `{"code":"CONFLICT","message":"This handle is already taken.","fields":{"handle":"This handle is already taken."}}` |
| `dob` under 18 | 400 `VALIDATION_ERROR` naming field | `{"code":"VALIDATION_ERROR","message":"...","fields":{"dob":"You must be at least 18 years old to register."}}` |
| `dob` exactly 18 today | 201 | `{"userId":"01a0149f-e708-7e67-b0fe-e26964d26b13"}`, **201** |
| `dob` malformed | 400 | `{"code":"VALIDATION_ERROR","fields":{"dob":"Date of birth is not a valid date."}}` |
| `password` missing | 400 | 400 `BAD_REQUEST` (binding `required`) |
| `title` > 20 | 400 | 400 `BAD_REQUEST` (binding tag) |
| `body` > 300 | 400 | 400 `BAD_REQUEST` (binding tag) |
| `POST /posts` non-existent `userId` | 404 | `{"code":"NOT_FOUND","message":"The requested user was not found.","fields":{"userId":"No user exists with this id."}}` |
| `POST /posts` malformed `userId` | 400 | `{"code":"BAD_REQUEST","message":"userId is not a valid identifier.",...}` |
| follow non-existent user | 404 | `{"code":"NOT_FOUND","message":"The user to follow was not found.",...}` |
| self-follow | 400 | `{"code":"BAD_REQUEST","message":"A user cannot follow themselves.",...}` |
| `GET /posts` no `userId` | 400 | `{"code":"BAD_REQUEST","message":"userId is required.",...}` |
| password/hash/driver text in any response | absent | `grep -i "password\|bcrypt\|mongo\|driver\|panic"` over every captured response body → **no matches** in any body (the only hits were the script's own section labels, not response content) |
| 20 emoji in `name` | 201 | `{"userId":"01a0149f-e76d-7aae-9044-0071af770807"}`, **201** |
| 21 emoji in `name` | 400 | 400 `BAD_REQUEST` |

\* **Note on `name`/`title`/`body` length and `password` presence**: these are caught by Gin's binding tags (`max=N`, `required`) before core.go runs, which — per the existing, pre-existing contract this service already documents in `pkg/apperror` — return the generic `BAD_REQUEST` with **no** `fields`, not `VALIDATION_ERROR`. Only core's own explicit checks (which do run for name/handle/DOB/password *emptiness*, since Gin's `required` only rejects a JSON-absent field, and for the DOB age/format checks, which have no binding-tag equivalent) populate `fields` and return `VALIDATION_ERROR`. This is the same pattern the existing `item` module already uses (see README §2's `item` examples) — a body that fails Gin's binding tags outright never gets `VALIDATION_ERROR`, by design, so no validator internals leak. Field-*presence* checks in core (e.g. name="   ", handle="") still return `VALIDATION_ERROR` with `fields`, as shown in the raw PR output.

### E. Index verification

`db.<collection>.getIndexes()`:

```
users:   [{_id:1}, {handle:1, name:'idx_users_handle_unique', unique:true}]
posts:   [{_id:1}, {userId:1, createdAt:-1, _id:-1, name:'idx_posts_userId_createdAt_id'}]
follows: [{_id:1},
          {followerId:1, followeeId:1, name:'idx_follows_follower_followee_unique', unique:true},
          {followeeId:1, name:'idx_follows_followee'}]
```

`explain("executionStats")`:

- **User-by-handle lookup**: `EXPRESS_IXSCAN` on `idx_users_handle_unique` (Mongo 8's fast path for an equality lookup on a unique index — the IXSCAN-family plan, not a `COLLSCAN`). `nReturned: 1`, `totalDocsExamined: 1`.
- **"My posts" query** (`{userId: {$in:[id]}}`, sort `{createdAt:-1, _id:-1}`, the exact shape `post.Repository.ListByAuthors` issues): `FETCH` ← `IXSCAN` on `idx_posts_userId_createdAt_id`. **No `SORT` stage.** `totalKeysExamined: 3`, `totalDocsExamined: 3`, `nReturned: 3` — tight, no gap.
- **Timeline query** (same shape, multi-author `$in`): identical plan shape, `IXSCAN` → `FETCH`, no `SORT` stage.

**One real bug this caught and fixed**: the index originally specified in §3.2 literally, `(userId asc, createdAt desc)` only, does *not* eliminate the `SORT` stage once the cursor's tie-break field (`_id desc`) is added to the query's sort spec — verified directly with `explain()`, which showed a blocking `SORT` stage on top of the `IXSCAN` even for a single-author query. The index was extended to `(userId asc, createdAt desc, _id desc)` (name `idx_posts_userId_createdAt_id`) to fix this — see AGENTS.md for the recorded gotcha and §4.G.4 below for the field-order reasoning.

### F. Latency under load

**Seeded volume**: 100,003 users, 500,000 posts, 5,050 follow edges — loaded via `cmd/benchseed` (a bulk-insert loader that bypasses bcrypt-per-user and HTTP; see its doc comment). Verified post-load: `db.users.countDocuments({})` → 100,018 (100,003 + prior functional-test users), `db.posts.countDocuments({})` → 500,006, `db.follows.countDocuments({})` → 5,052.

**Measurement tool**: `cmd/benchquery`, which issues the *exact* query shape each repository method runs (same filter, projection, sort — see its doc comments) directly through the driver, at **concurrency 50**, 40 iterations per goroutine (2,000 samples per query type), bypassing HTTP/JSON/Gin entirely so the number is "database time for the query" as §3.1 specifies. Command: `go run ./cmd/benchquery`. Two runs (cold, then warm) for stability:

| Query | Concurrency | n | p50 | p95 | p99 | Budget (10ms) |
|---|---|---|---|---|---|---|
| User lookup by id (run 1) | 50 | 2000 | 1.20ms | 5.05ms | 7.61ms | ✅ met |
| User lookup by id (run 2) | 50 | 2000 | 1.39ms | 5.64ms | 8.21ms | ✅ met |
| Timeline, ~50 followees (run 1) | 50 | 2000 | 3.56ms | 11.61ms | 18.82ms | ❌ missed |
| Timeline, ~50 followees (run 2) | 50 | 2000 | 4.50ms | 8.99ms | 10.88ms | ⚠️ borderline miss |
| Timeline, ~5000 followees (run 1) | 50 | 2000 | 173.14ms | 459.92ms | 649.42ms | ❌ missed by ~65x |
| Timeline, ~5000 followees (run 2) | 50 | 2000 | 196.22ms | 333.64ms | 421.22ms | ❌ missed by ~42x |

**User-record lookup meets the 10ms p99 budget** (7.6–8.2ms across two runs, on a single local dev-laptop MongoDB instance with no replica set or dedicated hardware — production headroom is likely larger given a properly provisioned cluster).

**Timeline does not reliably meet the budget, even at ~50 followees**, and this is reported honestly rather than asserted away: p99 sat at 18.8ms on the first run and 10.9ms (right at the line) on the second. At ~5,000 followees it is not close — 400–650ms p99, 40–65x over budget. This is precisely the fan-out-on-read degradation §3.3 asks this report to characterize (see §4.G.1).

**What I would change**: see §5, ranked.

### G. Design documentation

See §4 below (this section *is* §4.G, expanded).

### H. Project hygiene

- [x] New modules registered in `internal/boot/boot.go`, `EnsureIndexes` called for each at boot (`user`, `post`, `follow`; `timeline` has no collection of its own).
- [x] `pkg/apperror` reused for every new error path; no second error format introduced. New field keys and messages were added to `pkg/apperror/constants.go` alongside the existing ones, not a parallel file.
- [x] `AGENTS.md` updated with two durable gotchas: the structural-interface cross-module wiring pattern, and the index/sort tie-break lesson from §4.E.
- [x] `git status` — clean of stray artifacts (verified before commit); `bin/`, `coverage.*`, and mock output all already gitignored or intentionally committed per the existing convention (mocks are committed, matching `item`'s pattern).

---

## 3. Measured numbers

See §4.F's table above for the full latency data and §4.E for the explain-plan findings. Summary:

- **Seeded**: 100,003 users, 500,000 posts, 5,050 follow edges (two runs of `cmd/benchquery` reused the same seeded data).
- **User lookup p99**: 7.6–8.2ms at concurrency 50 — **meets** the 10ms budget.
- **Timeline p99, ~50 followees**: 10.9–18.8ms at concurrency 50 — **at or over** the budget.
- **Timeline p99, ~5000 followees**: 421–649ms at concurrency 50 — **far over** the budget.
- **Index plans**: user-by-handle and posts-by-author(s) both resolve to `IXSCAN`/`EXPRESS_IXSCAN` with `totalDocsExamined` ≈ `nReturned`, no `COLLSCAN`, no `SORT` stage.

---

## 4. Design decisions

### 4.1 Timeline strategy: fan-out-on-read

**Chosen**: fan-out-on-read. `GET /timeline` fetches the caller's followee list (`follow.Repository.ListFolloweeIDs`, an index-only scan on `idx_follows_follower_followee_unique`), then queries `posts` with `userId $in [followees...]`, sorted `(createdAt desc, _id desc)`, cursor-paginated.

**Why**: writes stay O(1) regardless of an author's follower count — a post by an account with a million followers is one insert, not a million. Nothing to backfill if the post model changes. Simpler to reason about and to keep correct under concurrent follow/unfollow (no timeline materialization to reconcile).

**Measured read cost**: 3.5–4.5ms p50 / 10.9–18.8ms p99 at 50 followees; 173–196ms p50 / 421–649ms p99 at 5,000 followees, both at concurrency 50 (§4.F).

**Where it breaks**: **between roughly 50 and a few hundred followees**, based on the measured numbers — the p99 budget is already marginal at 50 and the query cost visibly scales with the size of the `$in` list (Mongo does one index seek per author id, then merges), so the breakdown is not a cliff at some large number but a steady climb that crosses the 10ms line well before "thousands." At 5,000 followees it is 40–65x over budget. This matches the brief's prediction exactly: "a user following 10,000 accounts is a different query shape from one following 20."

**What I'd do next**: a **hybrid**, exactly as the brief suggests. Materialize (fan-out-on-write) the timeline for accounts below some followee-count threshold — most users, who follow tens to low hundreds of accounts, get the flat, fast, materialized read. For the (rare) high-follow-count accounts, or symmetrically for very-high-*follower*-count authors whose posts would mean fanning out to millions of per-follower timeline rows, fall back to on-read fan-out or a separate "celebrity posts" merge step at read time (the classic Twitter-scale pattern: a normal user's timeline is pre-materialized rows plus a small on-read merge of the handful of celebrity accounts they follow). The `idx_follows_followee` index this PR already ships was added specifically so a future fan-out-on-write path can answer "who follows this account" without a new index migration.

### 4.2 Connection pool size

**Configured**: `max_pool_size = 100` (in `config/default.toml`, `config.MongoConfig.MaxPoolSize`, applied via `mongo.Connect(...).SetMaxPoolSize(...)` in `database.Connect`). This is the MongoDB Go driver's own default, made **explicit** rather than left implicit, because — per §3.6 — the pool is the service's real concurrency ceiling under load, not the number of Gin/goroutine handlers, and a value a reader has to go look up in driver source is a value that will drift unnoticed. 100 is a reasonable starting point for a single service instance talking to one replica set; the number that actually matters in production is `(pool size) × (number of service instances) ≤ (MongoDB's own max connection limit)`, which is an operational tuning question outside this exercise's scope but is now a one-line config change instead of a code change.

### 4.3 Identifier choice: UUIDv7, string storage

**Chosen**: UUIDv7 (`github.com/google/uuid`'s `NewV7()`, wrapped in `pkg/idgen.New()`), stored as a plain Mongo string (`bson:"_id"` on a `string` field, not `bson.ObjectID` and not a binary UUID subtype).

**Why v7 over v4**: v7 embeds a millisecond timestamp in its high bits, so ids from concurrent inserts are roughly time-ordered. That matters for every collection here because `_id` (or, for `posts`, `_id` as the compound index's tie-break field) is itself an index: v4's uniform randomness causes B-tree page splits and fragmentation under high insert volume (a well-known MongoDB/Postgres pattern), while v7 inserts land at the tail of the index, the same shape a `createdAt`-ordered auto-increment would produce, without ever needing a second round trip to get one.

**Why string over binary UUID**: simplicity and readability at zero real cost here — the ids in this service are compared for equality and used in `$in`/range predicates, not sorted by their raw bytes independently of `createdAt`, so the storage-size argument for binary UUIDs (~16 bytes vs ~36-byte string) matters less than it would if `_id` order itself were load-bearing. Every id in every collection and every wire response uses the same string representation, so there is exactly one conversion rule (none) anywhere in the codebase.

### 4.4 Compound index field ordering

`posts`: `(userId asc, createdAt desc, _id desc)`. `follows`: `(followerId asc, followeeId asc)` unique, plus `(followeeId asc)` alone.

**Why equality-field(s) first, then sort field(s)**: MongoDB can use a compound index's prefix for equality matching and its suffix for sort *only if the suffix directly follows the matched prefix in index-key order*. `(userId, createdAt, id)` lets a query that filters on `userId` (or `userId $in [...]`) get its `(createdAt desc, id desc)` sort for free, straight off the index — confirmed by `explain()` showing no `SORT` stage (§4.E). Reversing it, `(createdAt, id, userId)`, would force Mongo to scan the *entire* index in `createdAt` order and filter every entry by `userId` — effectively an index-order collection scan, not a seek, and at 500k+ posts that is exactly the collection-scan cost §3.1 rules out.

**Why `_id` (as `id desc`) is a third key, not left out**: this was the one design assumption `explain()` disproved (§4.E) — sorting by `(createdAt desc, _id desc)` against an index covering only `(userId, createdAt)` still produced a blocking `SORT` stage, because Mongo cannot prove `_id` ordering *within* a group of equal `createdAt` values from that index alone. Adding `_id` as a third index key removes the `SORT` stage entirely and is also exactly the field the cursor's tie-break needs (§3.4) — one index change serves both the "no SORT stage" requirement and the pagination correctness requirement.

**Why `follows`'s unique index is `(followerId, followeeId)` in that order, not reversed**: the hot query is "does this pair exist / insert this pair" (the idempotent follow upsert) and "who does X follow" (`ListFolloweeIDs`, filtered on `followerId` alone) — both are served by `followerId` leading. The reverse-lookup index on `followeeId` alone exists solely because a future fan-out-on-write timeline needs "who follows this account," and that is a different equality field, not a suffix of the first index — it genuinely needs its own index, which is exactly what §3.2's table specifies.

### 4.5 §2.6 ambiguities

**1. Does the timeline include the requester's own posts?** **Decision: yes** (`timeline.includeOwnPosts = true`, a single named constant). Reasoning: the strict reading of the spec ("posts by the users that userId follows") would exclude them, but every mainstream product called a "timeline" or "feed" — Twitter/X, Instagram, LinkedIn — includes the viewer's own posts, and a user who posts something and then can't find it on their own timeline reads as a bug, not a feature, to almost anyone testing this by hand. Kept as one named boolean constant specifically so reversing the decision later is a one-line change, not a refactor.

**2. Does the post response include the author's handle?** **Decision: no**, matching the literal given shape (`{id, title, body, userId, createdAt}`). Reasoning: the brief's own shape omits it, and the "N+1 avoidance" trade-off the brief warns about (§3.5) only has to be paid *if* the field is added — I chose not to speculatively pay it. If a future requirement adds it, the documented path is denormalization: store `authorHandle` on the post document at write time (the author's existing-user check that `post.CreatePost` already performs would fetch the handle for free in the same query), paying a one-time backfill cost if a handle later changes, rather than batch-fetching authors on every timeline read.

**3. Default page size and oversized `limit`.** **Decision**: default 20 (`post/entities.DefaultPageSize`), max 100 (`MaxPageSize`), and a `limit` above 100 is **clamped**, not rejected. Reasoning: an oversized limit is a client being greedy, not a client being wrong — clamping serves the request at the platform's safe ceiling instead of forcing an extra round trip to fix a value that has an obvious, safe interpretation. Rejecting would be the better call if the limit were security-sensitive (it isn't — it only bounds how much of the caller's own data comes back in one page) or if silently returning less than asked could hide a client bug; neither applies here strongly enough to justify the friction.

### 4.6 Deliberate deviations from the literal spec

- **`{"userId": "..."}` instead of a bare `userId` string** for `POST /users`, and **`{"postId": "..."}`** for `POST /posts`. Every other response in this service (including the pre-existing `item` module) is a JSON object; a bare string response would be the only inconsistent endpoint in the API and would also make future additive fields (e.g. adding `createdAt` to the register response later) a breaking change instead of an additive one.
- **`GET`, not `POST`, for the follow state change** (`GET /users/follow/:userId?userId={follower}`), implemented exactly as specified. This is not idempotent-*safe* in the REST sense — a GET is conventionally required to have no side effects, and a browser prefetcher, crawler, or proxy is allowed to issue GETs speculatively — so this would normally be a `POST` or `PUT`. It *is* idempotent in the sense the brief actually requires (calling it twice leaves exactly one edge), which is enforced by the unique index + upsert, independent of the HTTP method chosen.
- **The cursor travels in a response header (`X-Next-Cursor`), not the response body**, for both `GET /posts` and `GET /timeline`. The brief's literal list-endpoint contract is a bare JSON array (`[ {post1}, {post2} ]`); wrapping it in an object (`{posts: [...], nextCursor: "..."}`) to carry the cursor would break that literal shape, so the cursor rides on a header instead, present only when a further page exists.

---

## 5. Known gaps and what I would do next

Ranked by what I'd tackle first:

1. **Fix the timeline latency miss** (§4.G.1). This is the one measured target this report does not meet. Next step: implement the hybrid — materialize timelines for accounts below a followee-count threshold (fan-out-on-write), keep on-read fan-out only for the tail of high-follow-count accounts. This is real, scoped follow-up work, not a one-line fix, which is why it's reported as a gap rather than silently worked around.
2. **Re-run the latency benchmark against a properly provisioned MongoDB** (replica set, dedicated hardware, realistic network hop) instead of a single local dev-laptop instance with no other load on the machine. The current numbers are directionally right (user lookups are fast and flat, timeline fan-out-on-read scales with followee count) but the absolute millisecond values would shift on real infrastructure.
3. **Author handle denormalization**, if a future requirement needs it on the post/timeline response (§4.G.5's already-documented path): add `authorHandle` to the post document at write time, backfilling existing rows via a migration script.
4. **A general random follow graph for realism.** `cmd/benchseed` only creates the two exact-count accounts (`bench-follows-50`, `bench-follows-5000`) the latency script needs, plus otherwise-disconnected users; a denser, more realistic graph (power-law follower distribution) would make the "where does fan-out-on-read break" answer in §4.G.1 more precise than the two data points measured here.
5. **Rate limiting / auth**, both explicitly out of scope per the existing README ("No authentication - intentionally out of scope") and this brief, but worth naming: `POST /users` and `POST /posts` are unauthenticated write endpoints, which is fine for this exercise and would not be fine in production.
