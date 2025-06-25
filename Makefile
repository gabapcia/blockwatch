# Raw Build
.PHONY: build
build:
	@go build -o blockwatch cmd/cli/main.go

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
	@command -v mockery >/dev/null 2>&1 || { \
		echo "Error: mockery not found. Please install it from https://github.com/vektra/mockery"; \
		exit 1; \
	}
	@mockery

.PHONY: unit-tests
unit-tests:
	@go clean -testcache
	@go test ./...
