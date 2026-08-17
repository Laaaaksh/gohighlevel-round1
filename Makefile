.PHONY: help dev web test test-short test-coverage lint fmt build mock mock-clean seed tidy docker-up docker-down

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

dev: ## Run the Go API server (cmd/api)
	go run ./cmd/api

web: ## Run the Next.js dev server
	cd web && npm run dev

seed: ## Insert sample items into MongoDB
	go run ./cmd/seed

build: ## Build the API and seed binaries into bin/
	go build -o bin/api ./cmd/api
	go build -o bin/seed ./cmd/seed

test: ## Run all tests, including e2e tests against a real MongoDB
	go test ./...

test-short: ## Run tests, skipping database-dependent e2e tests
	go test -short ./...

test-coverage: ## Run tests with a coverage report
	go test -coverprofile=coverage.profile ./...
	go tool cover -html=coverage.profile -o coverage.html
	@echo "coverage report: coverage.html"

lint: ## Run go vet (and golangci-lint if installed)
	go vet ./...
	@if command -v golangci-lint >/dev/null 2>&1; then golangci-lint run; else echo "golangci-lint not installed, skipping"; fi

fmt: ## Format Go source
	@find . -type f -name '*.go' -not -path "./web/*" -exec gofmt -w {} +

mock: ## Regenerate gomock mocks (go:generate directives)
	go generate ./...

mock-clean: ## Remove generated mocks
	find . -type d -name mock -not -path "./web/*" -exec rm -rf {} +

tidy: ## Tidy go.mod/go.sum
	go mod tidy

docker-up: ## Start the fallback MongoDB via docker compose
	docker compose -f deployment/dev/docker-compose.yml up -d

docker-down: ## Stop the fallback MongoDB
	docker compose -f deployment/dev/docker-compose.yml down
