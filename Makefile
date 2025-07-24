.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

# Raw Build
.PHONY: build
build: ## Compiles the application and creates a binary in the root directory.
	@CGO_ENABLED=0 go build -o blockwatch cmd/cli/main.go

.PHONY: run
run: build ## Builds and runs the application.
	@./blockwatch

# Container Build
.PHONY: docker-build
docker-build: ## Builds a Docker image for the application.
	@docker build -t blockwatch .

.PHONY: docker-run
docker-run: docker-build ## Builds and runs the application in a Docker container.
	@docker run --rm blockwatch

# Tests
.PHONY: mocks
mocks: ## Generates mocks for the interfaces using mockery.
	@docker run -v "$$PWD":/src -w /src vektra/mockery:3

.PHONY: unit-tests
unit-tests: ## Runs the unit tests.
	@go clean -testcache
	@go test ./...

.PHONY: coverage
coverage: ## Runs the unit tests and generates an HTML coverage report.
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@open coverage.html

# PostgreSQL
.PHONY: generate-queries
generate-queries: ## Generates type-safe Go code from SQL queries using sqlc.
	@docker run --rm -v $$PWD:/src -w /src \
		sqlc/sqlc --file internal/infra/storage/postgresql/sqlc.yaml generate

.PHONY: new-migration
new-migration: ## Creates a new database migration file.
	@read -p 'Migration name: ' MIGRATION_NAME; \
	docker run --rm -v $$PWD/internal/infra/storage/postgresql/migrations:/migrations --network host \
		migrate/migrate create -ext sql -dir /migrations -seq $$MIGRATION_NAME

.PHONY: apply-migrations
apply-migrations: ## Applies all pending database migrations.
	@docker run --rm -v $$PWD/internal/infra/storage/postgresql/migrations:/migrations --network host \
		migrate/migrate -path=/migrations/ -database $${POSTGRESQL_DSN:-postgres://blockwatch:blockwatch@localhost:5432/blockwatch?sslmode=disable} up
