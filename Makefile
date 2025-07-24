.PHONY: help
help:
	@awk 'BEGIN {FS = ":.*?## "} \
	/^# / {printf "\n\033[1;33m%s\033[0m\n", substr($$0, 3)} \
	/^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-28s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Raw Build
.PHONY: build-dev
build-dev: ## Compiles the application in dev mode and creates a binary in the root directory.
	@CGO_ENABLED=0 go build -o blockwatch cmd/cli/main.go

.PHONY: build-prod
build-prod: ## Compiles the application in prod mode and creates a binary in the root directory.
	@CGO_ENABLED=0 go build -o blockwatch -ldflags "-s -w" cmd/cli/main.go

.PHONY: run-dev
run-dev: build ## Builds and runs the application in dev mode.
	@./blockwatch

.PHONY: run-prod
run-prod: build ## Builds and runs the application in prod mode.
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
