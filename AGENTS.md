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

## Maintaining this file

Keep this file scoped to durable, non-obvious project knowledge - not a
restatement of what the README or code already show. Update it when a
gotcha like the ones above is discovered or resolved; delete entries that
no longer apply.
