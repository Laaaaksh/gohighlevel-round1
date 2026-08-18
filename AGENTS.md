# AGENTS.md

Interview scaffold: Go 1.25 + Gin + MongoDB backend, Next.js (App Router)
frontend. See [README.md](README.md) for setup, endpoints, and the
"add a new resource" walkthrough - this file only covers non-obvious gotchas
the code doesn't already make clear.

## Non-obvious gotchas

- **Mock import cycle:** `internal/modules/item/core_test.go` lives in
  package `item_test` (external), not `item`. `mock/mock_repository.go`
  imports `item` for its domain types, so a same-package test importing
  `mock` creates an import cycle. `core.go` exposes
  `NewCoreWithClock(repo, now)` specifically so the external test package
  can inject a fixed clock without needing access to `Core`'s unexported
  fields. Follow this pattern for any new module's mocked tests.
- **`web/go.mod` is a boundary stub, not a real module.** Without it, the
  root Go module's `./...` pattern descends into `web/node_modules` (some
  npm packages bundle Go source) and `go build`/`go vet`/`go test ./...`
  pick up unrelated packages. Do not delete it or try to make it a "real"
  module.
- **`internal/config.Load()` adds `../config` and `../../config` as
  fallback search paths** so it resolves correctly whether the caller runs
  from the repo root (`cmd/api`, `cmd/seed`) or from `e2e/` (where `go test`
  sets the working directory to the package directory).
- **`APP_ENV=test`** (set automatically by `e2e/item_e2e_test.go`'s
  `SetupSuite`) points MongoDB at a separate `gohighlevel_round1_test`
  database via `config/test.toml`, so e2e tests never touch the dev
  database seeded by `make seed`.
- **`.env` lookup stops at the module root, unlike the `config/` search
  paths above.** `loadDotEnv` tries `./.env`, then walks up to the nearest
  directory containing `go.mod` and tries there. It deliberately does not
  climb past that: a bare `../.env` would resolve to the repo's *parent*
  when run from the repo root, silently picking up an unrelated file that
  changes `PORT` or `MONGO_URI`. The `config/` paths cannot escape the same
  way because `config/` only exists inside the repo.
- **`next dev` writes `web/AGENTS.md` and `web/CLAUDE.md` on startup.**
  Next.js 16 generates these agent-rule files itself; they are gitignored so
  they do not end up in a commit. Set `agentRules: false` in
  `web/next.config.ts` to stop it.
- **Cross-module dependencies use structural interfaces, not imports.**
  `post` and `follow` both need to validate a userId; each declares its own
  narrow interface for it (e.g. `Exists(ctx, userID string) (bool, error)`)
  in its own `core.go`, and `boot.go` passes the same `*user.Core` to both.
  Neither module imports `user` - Go satisfies the interface structurally.
  `timeline` extends the same pattern one step further: it has no
  collection or repository.go of its own, only a `dependencies.go`
  declaring the two interfaces it needs from `post` and `follow`. Follow
  this pattern - a shared interface file plus boot.go wiring - for any
  future module that needs another module's read path.
- **A cursor-pagination sort needs its tie-break field in the index, not
  just in the query's sort spec.** `posts`' compound index is
  `(userId, createdAt desc, _id desc)`, not just `(userId, createdAt desc)`:
  a sort on `(createdAt desc, _id desc)` against an index covering only
  `(userId, createdAt)` still costs a blocking in-memory `SORT` stage,
  because the index alone can't prove `_id` order within an
  equal-`createdAt` group. `explain("executionStats")` is the way to catch
  this - it will not show up as a correctness bug, only as an unnecessary
  `SORT` stage in the plan.

## Maintaining this file

Keep this file scoped to durable, non-obvious project knowledge - not a
restatement of what the README or code already show. Update it when a
gotcha like the ones above is discovered or resolved; delete entries that
no longer apply.
