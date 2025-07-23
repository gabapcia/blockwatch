package main

import (
	"context"

	"github.com/gabapcia/blockwatch/internal/bootstrap"
	"github.com/gabapcia/blockwatch/internal/pkg/config"
	"github.com/gabapcia/blockwatch/internal/pkg/logger"
	"github.com/gabapcia/blockwatch/internal/pkg/telemetry"
)

func main() {
	// Create a root context for the application lifecycle
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Load environment variables and structured configuration
	cfg, err := config.Load(ctx)
	if err != nil {
		panic(err)
	}

	// Initialize telemetry (tracing, metrics, etc) with the service name
	telemetryShutdownFunc, err := telemetry.Init(ctx, cfg.ServiceName)
	if err != nil {
		panic(err)
	}
	defer telemetryShutdownFunc(context.Background())

	// Initialize structured logger with the desired log level
	if err := logger.Init(cfg.ServiceName, cfg.Log.Level); err != nil {
		panic(err)
	}

	// Initialize core services (messaging, storage, domain services, etc)
	bootstrap, err := bootstrap.New(ctx, cfg)
	if err != nil {
		logger.Fatal(ctx, "bootstrap initialization failed", "error", err)
	}
	defer bootstrap.Close()

	// Start CLI command handler
	if err := bootstrap.CLI(ctx); err != nil {
		logger.Fatal(ctx, "cli execution failed", "error", err)
	}
}
