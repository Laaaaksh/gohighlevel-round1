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

All write endpoints validate their body and return a consistent JSON error
shape: `{"code": "...", "message": "...", "fields": {...}}` (see
[`pkg/apperror`](pkg/apperror)).

| Method | Path              | Description        |
|--------|-------------------|---------------------|
| GET    | `/health`         | Liveness + live DB ping |
| POST   | `/api/items`      | Create an item      |
| GET    | `/api/items`      | List all items       |
| GET    | `/api/items/:id`  | Get one item by id   |
| PUT    | `/api/items/:id`  | Update an item        |
| DELETE | `/api/items/:id`  | Delete an item         |

```bash
# Create
curl -X POST localhost:8080/api/items \
  -H 'Content-Type: application/json' \
  -d '{"name":"Widget","description":"A test widget"}'

# List
curl localhost:8080/api/items

# Get by id
curl localhost:8080/api/items/<id>

# Update
curl -X PUT localhost:8080/api/items/<id> \
  -H 'Content-Type: application/json' \
  -d '{"name":"Widget v2","description":"updated"}'

# Delete
curl -X DELETE localhost:8080/api/items/<id>

# Error cases
curl -X POST localhost:8080/api/items -d '{}'             # 400 - name required
curl localhost:8080/api/items/not-an-id                    # 400 - invalid id
curl localhost:8080/api/items/000000000000000000000000     # 404 - not found
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
  database/                   Mongo client connect + Ping
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
  `curl localhost:3000/ssr-demo | grep "Sample Item"` - the data is in the
  raw response, not only in the hydrated DOM.
- **`/csr-demo`** - a client component (`"use client"`) that fetches in the
  browser with a visible loading state, and includes the item-creation
  form (full write path from the UI).

---

## Notes

- No authentication - intentionally out of scope for the interview.
- No CI pipeline or production deployment config - out of scope.
- `deployment/dev/docker-compose.yml` is a MongoDB fallback only. The
  primary path is the native local MongoDB service.
