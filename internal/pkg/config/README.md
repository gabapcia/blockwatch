# Config Package

The `config` package provides a comprehensive configuration management system for the Blockwatch application. It handles environment variable loading, validation, and provides a structured approach to configuring various application components including storage backends, messaging systems, and blockchain network connections.

## Overview

This package implements a hierarchical configuration system that supports:

- **Environment variable-based configuration** using the `envconfig` library
- **Comprehensive validation** with custom struct-level validators
- **Global and inline configuration patterns** for storage and messaging backends
- **Type-safe configuration structs** for all application components
- **Flexible engine selection** allowing both shared and use-case-specific configurations

## Architecture

### Core Components

#### Main Configuration (`config.go`)
- `Config`: Root configuration struct containing all application settings
- `Engines`: Global shared configurations for storage and messaging backends
- `Load()`: Main entry point for loading and validating configuration
- Custom validation logic for cross-field dependencies

#### Use Case Configurations
- `ChainStream`: Configuration for blockchain data streaming
- `WalletRegistry`: Configuration for wallet registration management
- `WalletWatch`: Configuration for wallet transaction monitoring

#### Backend Configurations
- `storage/`: Storage backend configurations (Redis, PostgreSQL)
- `messaging/`: Messaging backend configurations (Redis Streams, RabbitMQ)
- `blockchain/`: Blockchain network configurations (Ethereum)
- `pkg/`: Common configuration types (logging, telemetry, transport, resilience)

## Configuration Structure

```go
type Config struct {
    ServiceName string     `env:"SERVICE_NAME, default=blockwatch" validate:"required"`
    Log         pkg.Logger `env:", prefix=LOG_" validate:"required"`

    Engines Engines `env:", prefix=ENGINES_" validate:"omitempty"`

    Walletregistry WalletRegistry `env:", prefix=WALLETREGISTRY_" validate:"required"`
    Walletwatch    WalletWatch    `env:", prefix=WALLETWATCH_" validate:"required"`
    Chainstream    ChainStream    `env:", prefix=CHAINSTREAM_" validate:"required"`
}
```

## Backend Selection Patterns

The package implements two patterns for backend selection:

### 1. Global Engine Reference
Use a globally defined engine configuration:

```bash
# Define global Redis storage engine
ENGINES_STORAGE_REDIS_ADDRESS=localhost:6379
ENGINES_STORAGE_REDIS_PASSWORD=secret

# Reference it in a use case
WALLETREGISTRY_WALLET_STORAGE_ENGINE=REDIS
```

### 2. Inline Configuration
Define backend configuration directly for a specific use case:

```bash
# Inline Redis configuration for wallet registry
WALLETREGISTRY_WALLET_STORAGE_REDIS_ADDRESS=localhost:6379
WALLETREGISTRY_WALLET_STORAGE_REDIS_PASSWORD=secret
```

## Supported Backends

### Storage Backends
- **Redis**: In-memory data structure store
- **PostgreSQL**: Relational database

### Messaging Backends
- **Redis Streams**: Redis-based message streaming
- **RabbitMQ**: Message broker

### Blockchain Networks
- **Ethereum**: Ethereum network via JSON-RPC

## Environment Variables

### Core Configuration

```bash
# Service
SERVICE_NAME=blockwatch           # Service identifier

# Logging
LOG_LEVEL=INFO                    # DEBUG, INFO, WARN, ERROR, PANIC, FATAL
```

### Global Engines

```bash
# Storage engines
ENGINES_STORAGE_REDIS_ADDRESS=localhost:6379
ENGINES_STORAGE_REDIS_USERNAME=user
ENGINES_STORAGE_REDIS_PASSWORD=pass
ENGINES_STORAGE_REDIS_DB=0

ENGINES_STORAGE_POSTGRESQL_DSN=postgres://user:pass@localhost/db

# Messaging engines
ENGINES_MESSAGING_REDIS_ADDRESS=localhost:6379
ENGINES_MESSAGING_REDIS_USERNAME=user
ENGINES_MESSAGING_REDIS_PASSWORD=pass
ENGINES_MESSAGING_REDIS_DB=0

ENGINES_MESSAGING_RABBITMQ_URI=amqp://user:pass@localhost:5672/
```

### Use Case Configurations

#### Wallet Registry
```bash
# Using global engine
WALLETREGISTRY_WALLET_STORAGE_ENGINE=REDIS

# Or using inline configuration
WALLETREGISTRY_WALLET_STORAGE_REDIS_ADDRESS=localhost:6379
```

#### Wallet Watch
```bash
# Processing timeout
WALLETWATCH_MAX_PROCESSING_TIME=5m

# Storage configuration
WALLETWATCH_WALLET_STORAGE_ENGINE=POSTGRESQL
WALLETWATCH_IDEMPOTENCY_GUARD_ENGINE=REDIS

# Messaging configuration
WALLETWATCH_TRANSACTION_NOTIFIER_ENGINE=RABBITMQ
WALLETWATCH_TRANSACTION_NOTIFIER_RABBITMQ_EXCHANGE=transactions
WALLETWATCH_TRANSACTION_NOTIFIER_RABBITMQ_ROUTING_KEY=wallet.events
```

#### Chain Stream
```bash
# Blockchain networks
CHAINSTREAM_NETWORKS_ETHEREUM_PROVIDER_ENDPOINT=https://eth.example.com
CHAINSTREAM_NETWORKS_ETHEREUM_PROVIDER_TIMEOUT=30s

# Storage and messaging
CHAINSTREAM_CHECKPOINT_STORAGE_ENGINE=REDIS
CHAINSTREAM_DISPATCH_FAILURE_NOTIFIER_ENGINE=REDIS
CHAINSTREAM_DISPATCH_FAILURE_NOTIFIER_REDIS_STREAM=failures

# Retry configuration
CHAINSTREAM_RETRY_ATTEMPTS=3
CHAINSTREAM_RETRY_DELAY=1s
CHAINSTREAM_RETRY_MAX_DELAY=30s
```

## Usage

### Loading Configuration

```go
package main

import (
    "context"
    "log"
    
    "github.com/gabapcia/blockwatch/internal/pkg/config"
)

func main() {
    ctx := context.Background()
    
    cfg, err := config.Load(ctx)
    if err != nil {
        log.Fatalf("Failed to load configuration: %v", err)
    }
    
    // Use configuration
    fmt.Printf("Service: %s\n", cfg.ServiceName)
    fmt.Printf("Log Level: %s\n", cfg.Log.Level)
}
```

### Accessing Backend Configurations

```go
// Check if global Redis storage is configured
if cfg.Engines.Storage.Redis != nil {
    fmt.Printf("Global Redis: %s\n", cfg.Engines.Storage.Redis.Address)
}

// Access use case configuration
walletStorage := cfg.Walletregistry.WalletStorage
if walletStorage.Engine != "" {
    fmt.Printf("Using global engine: %s\n", walletStorage.Engine)
} else if walletStorage.InlineConfig.Redis != nil {
    fmt.Printf("Using inline Redis: %s\n", walletStorage.InlineConfig.Redis.Address)
}
```

## Validation

The package includes comprehensive validation:

### Field-Level Validation
- Required fields validation
- Enum value validation (e.g., log levels, engine names)
- Format validation (e.g., duration strings)

### Cross-Field Validation
- Mutual exclusivity between `Engine` and `InlineConfig`
- Engine reference validation (ensures referenced engines are configured)
- Backend-specific publisher configuration validation

### Custom Validation Rules
- `required_alone`: Only one field in a group can be set
- `engine_not_configured`: Referenced engine is not defined
- `engine_not_registered`: Unknown engine name

## Error Handling

Configuration errors are returned with descriptive messages:

```go
cfg, err := config.Load(ctx)
if err != nil {
    // Handle validation errors
    if strings.Contains(err.Error(), "engine_not_configured") {
        log.Fatal("Referenced storage engine is not configured")
    }
    log.Fatalf("Configuration error: %v", err)
}
```

## Testing

The package includes comprehensive test coverage:

- Unit tests for all configuration structs
- Validation logic testing
- Environment variable processing tests
- Cross-field validation tests

Run tests:
```bash
go test ./internal/pkg/config/...
```

## Extending the Configuration

### Adding a New Storage Backend

1. Add engine constant in `storage/storage.go`:
```go
const EngineNewBackend = "NEW_BACKEND"
```

2. Add configuration struct:
```go
type NewBackend struct {
    Address string `env:"ADDRESS" validate:"required"`
    // ... other fields
}
```

3. Update `Engines` and `InlineConfig` structs
4. Add validation case in `validateStoragePicker`

### Adding a New Use Case

1. Create configuration struct:
```go
type NewUseCase struct {
    Storage storage.Picker `env:", prefix=STORAGE_" validate:"required"`
    // ... other fields
}
```

2. Add to main `Config` struct
3. Update environment variable prefix

## Dependencies

- `github.com/sethvargo/go-envconfig`: Environment variable processing
- `github.com/gabapcia/blockwatch/internal/pkg/validator`: Custom validation logic

## Best Practices

1. **Use global engines** for shared backend configurations
2. **Use inline configurations** for use-case-specific settings
3. **Always validate** configuration after loading
4. **Provide sensible defaults** where appropriate
5. **Document environment variables** in deployment guides
6. **Test configuration** with various environment setups
