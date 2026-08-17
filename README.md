# gohighlevel-round1

Ready-to-code scaffold for a live 60-minute technical interview. Everything
here is plumbing - config, DB connection, one full CRUD resource, tests,
frontend demo pages - so that on interview day you can go straight from
"here's the problem" to designing data models and writing endpoints.

**Stack:** Go 1.25 + Gin + MongoDB (official driver) · Next.js (App Router) + TypeScript.

---

## 1. Get both servers running (cold laptop)

### Prerequisites

- Go 1.25+
- Node 20+
- MongoDB running locally on port 27017. On this machine it's already
  installed and running as a brew service:
  ```
  brew services list | grep mongodb
  ```
  If it's not running: `brew services start mongodb-community`.
  **Fallback only** if the brew service is unavailable:
  `make docker-up` (see [`deployment/dev/docker-compose.yml`](deployment/dev/docker-compose.yml)).

### Backend

```bash
cp .env.example .env        # optional - loaded at startup; defaults already match a local Mongo
go mod tidy
make seed                   # inserts 3 sample items so the frontend has data
make dev                    # starts the API on :8080
```

Verify:

```bash
curl localhost:8080/health
# {"status":"ok","database":"connected"}
```

### Frontend

In a second terminal:

```bash
cd web
cp .env.example .env.local  # optional - defaults to http://localhost:8080
npm install
npm run dev                 # starts Next.js on :3000
```

Open http://localhost:3000 - it links to the SSR and CSR demo pages.

---

## 2. Endpoint reference

Errors return a consistent JSON shape: `{"code": "...", "message": "..."}`
(see [`pkg/apperror`](pkg/apperror)). A `"fields"` object is added when the
failure is attributable to a named field - the explicit checks in `core.go`
populate it, keyed by field name. Its values are always fixed messages, never
the value you submitted, because the frontend renders them straight into the
form. A body that fails Gin's binding tags outright returns the generic
`BAD_REQUEST` message with no `fields`, so no validator internals leak.

Every response below was captured from a running server against local
MongoDB.

| Method | Path              | Description        |
|--------|-------------------|---------------------|
| GET    | `/health`         | Liveness + live DB ping |
| POST   | `/api/items`      | Create an item      |
| GET    | `/api/items`      | List all items       |
| GET    | `/api/items/:id`  | Get one item by id   |
| PUT    | `/api/items/:id`  | Update an item        |
| DELETE | `/api/items/:id`  | Delete an item         |

```bash
# Create -> 201
curl -X POST localhost:8080/api/items \
  -H 'Content-Type: application/json' \
  -d '{"name":"Widget","description":"A test widget"}'
# {"id":"6a82c7e9136ba795994b0d01","name":"Widget","description":"A test widget",
#  "createdAt":"2026-08-17T08:35:53.967889Z","updatedAt":"2026-08-17T08:35:53.967889Z"}

# List -> 200, newest first (sorted by createdAt descending)
curl localhost:8080/api/items
# [{"id":"6a82c7e9136ba795994b0d02","name":"Second",...},
#  {"id":"6a82c7e9136ba795994b0d01","name":"Widget",...}]

# Get by id -> 200
curl localhost:8080/api/items/6a82c7e9136ba795994b0d02
# {"id":"6a82c7e9136ba795994b0d02","name":"Second","description":"two",...}

# Update -> 200, updatedAt advances, createdAt does not
curl -X PUT localhost:8080/api/items/6a82c7e9136ba795994b0d02 \
  -H 'Content-Type: application/json' \
  -d '{"name":"Widget v2","description":"updated"}'
# {"id":"6a82c7e9136ba795994b0d02","name":"Widget v2","description":"updated",
#  "createdAt":"2026-08-17T08:35:53.979Z","updatedAt":"2026-08-17T08:35:54.01Z"}

# Delete -> 204, empty body
curl -X DELETE localhost:8080/api/items/6a82c7e9136ba795994b0d02
```

Error cases:

```bash
# Binding rejects the body outright -> 400, no "fields"
curl -X POST localhost:8080/api/items -d '{}'
# {"code":"BAD_REQUEST","message":"The request could not be processed."}

# Passes binding (name is non-empty), fails core's explicit check -> 400
curl -X POST localhost:8080/api/items \
  -H 'Content-Type: application/json' \
  -d '{"name":"   "}'
# {"code":"VALIDATION_ERROR","message":"The request contains an invalid field.",
#  "fields":{"name":"Name is required."}}

# Malformed id -> 400
curl localhost:8080/api/items/not-an-id
# {"code":"BAD_REQUEST","message":"The provided id is not valid.",
#  "fields":{"id":"This id is not in the expected format."}}

# Well-formed id, no such item -> 404
curl localhost:8080/api/items/000000000000000000000000
# {"code":"NOT_FOUND","message":"The requested item was not found.",
#  "fields":{"id":"No item exists with this id."}}
```

---

## 3. How to add a new resource

Copy [`internal/modules/item/`](internal/modules/item) - it is the
intentionally domain-neutral, complete pattern. For a resource called
`widget`:

1. `cp -r internal/modules/item internal/modules/widget` and rename the
   package (`item` → `widget`) throughout.
2. **`entities/`** - rewrite `request.go`, `response.go`, `constant.go` for
   your fields, collection name, and DB field names.
3. **`repository.go`** - rewrite the `Item`/`Widget` struct's bson tags and
   any query filters. Keep the `IRepository` interface, the
   `var _ IRepository = (*Repository)(nil)` check, and `EnsureIndexes`.

   `EnsureIndexes` ships with exactly one index - `createdAt` descending,
   which is the sort `List` issues. Add an index when you add the query that
   needs it, not before. For a `GetByName` you introduce:

   ```go
   // entities/constant.go - the field name, once
   FieldName = "name"

   // repository.go - the index name and direction as constants
   const (
       indexNameWidgetsByName = "idx_widgets_name"
       indexAscending         = 1
   )

   // repository.go - inside EnsureIndexes, swap CreateOne for CreateMany
   _, err := r.collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
       {
           Keys:    bson.D{{Key: entities.FieldCreatedAt, Value: indexDescending}},
           Options: options.Index().SetName(indexNameWidgetsByCreatedAt),
       },
       {
           Keys:    bson.D{{Key: entities.FieldName, Value: indexAscending}},
           Options: options.Index().SetName(indexNameWidgetsByName),
       },
   })
   ```

   Add `.SetUnique(true)` to the options if the field must be unique - a
   duplicate insert then returns a driver error you map to
   `apperror.CodeConflict` (409).
4. **`core.go`** - rewrite the business logic against `IRepository`. This is
   the only file that should contain domain rules; it has no `*gin.Context`.
5. **`server.go`** - rewrite routes/handlers. This is the only file (besides
   `init.go`) that imports Gin.
6. **`init.go`** - update the module wiring (usually just renames).
7. Regenerate the mock: `make mock` (runs the `//go:generate` directive in
   `repository.go`).
8. Register the module in [`internal/boot/boot.go`](internal/boot/boot.go),
   next to `item`.
9. Copy `core_test.go` and rewrite the scenarios: `testify/suite`, one test
   method per scenario (no table tests), no `gomock.Any()`/`AnyTimes()` -
   see section 5.
10. If you want an e2e test, copy [`e2e/item_e2e_test.go`](e2e/item_e2e_test.go).

Frontend equivalent: copy [`web/lib/api/items.ts`](web/lib/api/items.ts) and
[`web/types/item.ts`](web/types/item.ts), add the new routes to
[`web/lib/constants.ts`](web/lib/constants.ts), and reuse
[`ItemList`](web/components/items/ItemList.tsx) /
[`ItemForm`](web/components/items/ItemForm.tsx) as templates.

---

## 4. Project layout

```
cmd/api/main.go              API entry point
cmd/seed/main.go              seed script
internal/
  boot/                       wires config, MongoDB, modules, routes
  config/                     typed config from config/*.toml + env vars
  constants/                  repo-wide constants, contextkeys/
  database/                   Mongo client connect + Ping + Disconnect
  interceptors/                request id, recovery, logging, CORS middleware
  logger/                     structured logging (log/slog) + ctx helper
  modules/
    health/                   GET /health
    item/                     the copyable CRUD pattern (see section 3)
pkg/apperror/                  typed error codes, HTTP mapping, wire format
config/                        default.toml, dev.toml, test.toml
e2e/                            MongoDB-dependent integration tests
deployment/dev/                 docker-compose fallback for MongoDB
web/                             Next.js frontend (App Router)
```

---

## 5. Testing

```bash
make test          # everything, including e2e tests against real MongoDB
make test-short     # unit tests only - safe with no MongoDB running
make test-coverage   # writes coverage.html
```

Standards used throughout (see `go-testing-standards`): `testify/suite`,
one test method per scenario (no table-driven tests), camelCase test names,
`go.uber.org/mock` mocks with exact argument/`.Times()` assertions (no
`gomock.Any()`/`AnyTimes()`).

---

## 6. Makefile targets

Run `make help` for the full, current list.

---

## 7. Frontend demo pages

- **`/ssr-demo`** - a server component (no `"use client"`) that fetches
  `/api/items` on the server before the HTML is sent. Confirm with
  `curl localhost:3000/ssr-demo | grep 'item-row'` - the rendered
  `<li class="item-row"><strong>First Sample Item</strong>` markup is in the
  raw response, not only in the hydrated DOM. (Run `make seed` first, or the
  list is legitimately empty.)
- **`/csr-demo`** - a client component (`"use client"`) that fetches in the
  browser with a visible loading state, and includes the item-creation
  form (full write path from the UI).

---

## Notes

- No authentication - intentionally out of scope for the interview.
- No CI pipeline or production deployment config - out of scope.
- `deployment/dev/docker-compose.yml` is a MongoDB fallback only. The
  primary path is the native local MongoDB service.
