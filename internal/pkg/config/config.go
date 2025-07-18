package config

import (
	"context"

	"github.com/gabapcia/blockwatch/internal/pkg/config/messaging"
	"github.com/gabapcia/blockwatch/internal/pkg/config/pkg"
	"github.com/gabapcia/blockwatch/internal/pkg/config/storage"
	"github.com/gabapcia/blockwatch/internal/pkg/validator"

	"github.com/kelseyhightower/envconfig"
)

// Engines defines globally shared engine configurations for storage and messaging backends.
type Engines struct {
	// Storage holds global storage backend configurations (e.g., Redis, PostgreSQL).
	Storage storage.Engines

	// Messaging holds global messaging backend configurations (e.g., Redis Streams, RabbitMQ).
	Messaging messaging.Engines
}

// Config represents the full application configuration, including logging, telemetry,
// global engine definitions, and per-use-case settings.
type Config struct {
	Log       pkg.Logger    // Log defines the logging configuration for the application.
	Telemetry pkg.Telemetry // Telemetry defines the telemetry and service identity configuration.

	Engines Engines // Engines holds the globally defined storage and messaging engine configurations.

	Walletregistry WalletRegistry // Walletregistry contains the configuration for the wallet registry use case.
	Walletwatch    WalletWatch    // Walletwatch contains the configuration for the wallet transaction watcher use case.
	Chainstream    ChainStream    // Chainstream contains the configuration for the chainstream use case.
}

// Load reads the application configuration from environment variables,
// applies defaults, and performs validation.
//
// This function uses `envconfig` to populate the Config struct and `validator`
// to ensure all fields meet their validation constraints.
//
// Parameters:
//   - ctx: context for cancellation (not used in this implementation but reserved for future use).
//
// Returns:
//   - A fully populated and validated Config struct.
//   - An error if parsing or validation fails.
func Load(ctx context.Context) (Config, error) {
	var config Config
	if err := envconfig.Process("", &config); err != nil {
		return Config{}, err
	}

	if err := validator.Validate(config); err != nil {
		return Config{}, err
	}

	return config, nil
}
