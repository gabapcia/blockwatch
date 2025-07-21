# Raw Build
.PHONY: build
build:
	@CGO_ENABLED=0 go build -o blockwatch cmd/cli/main.go

.PHONY: run
run: build
	@./blockwatch

# Container Build
.PHONY: docker-build
docker-build:
	@docker build -t blockwatch .

.PHONY: docker-run
docker-run: docker-build
	@docker run --rm blockwatch

# Tests
.PHONY: mocks
mocks:
	@docker run -v "$$PWD":/src -w /src vektra/mockery:3

.PHONY: unit-tests
unit-tests:
	@go clean -testcache
	@go test ./...

.PHONY: coverage
coverage:
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@open coverage.html

# PostgreSQL
.PHONY: generate-queries
generate-queries:
	@docker run --rm -v $$PWD:/src -w /src \
		sqlc/sqlc --file internal/infra/storage/postgresql/sqlc.yaml generate

.PHONY: new-migration
new-migration:
	@read -p 'Migration name: ' MIGRATION_NAME; \
	docker run --rm -v $$PWD/internal/infra/storage/postgresql/migrations:/migrations --network host \
		migrate/migrate create -ext sql -dir /migrations -seq $$MIGRATION_NAME

.PHONY: apply-migrations
apply-migrations:
	@docker run --rm -v $$PWD/internal/infra/storage/postgresql/migrations:/migrations --network host \
		migrate/migrate -path=/migrations/ -database $${POSTGRESQL_DSN:-postgres://blockwatch:blockwatch@localhost:5432/blockwatch?sslmode=disable} up
