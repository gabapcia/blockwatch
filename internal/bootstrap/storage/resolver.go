package storage

import (
	"context"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/gabapcia/blockwatch/internal/infra/storage/postgresql"
	"github.com/gabapcia/blockwatch/internal/infra/storage/redis"
	"github.com/gabapcia/blockwatch/internal/pkg/config/storage"
)

// storageFactory defines the constructor signature for supported storage backends.
//
// Each factory is responsible for creating a new instance of the storage backend
// using the provided configuration (e.g., DSN or connection options).
type storageFactory func(ctx context.Context, config any) (any, error)

// storageFactories maps supported storage engine names to their corresponding factory functions.
//
// Keys must be uppercase engine identifiers (e.g., "REDIS", "POSTGRESQL").
//
// To support a new engine:
//  1. Define the connection struct in config/storage.
//  2. Implement the corresponding client constructor.
//  3. Add a new entry here with the appropriate conversion and instantiation logic.
//
// Example:
//
//	"MONGODB": func(ctx context.Context, cfg any) (any, error) {
//		mongoCfg := cfg.(storage.MongoDB)
//		return mongodb.New(ctx, mongoCfg.URI)
//	},
var storageFactories = map[string]storageFactory{
	"REDIS": func(ctx context.Context, cfg any) (any, error) {
		redisCfg := cfg.(storage.Redis)
		return redis.New(ctx, redisCfg.Address, redisCfg.Username, redisCfg.Password, redisCfg.DB)
	},

	"POSTGRESQL": func(ctx context.Context, cfg any) (any, error) {
		pgCfg := cfg.(storage.PostgreSQL)
		return postgresql.New(ctx, pgCfg.DSN)
	},
}

// Resolve selects and returns a storage instance adapted to the desired interface.
//
// It supports two resolution mechanisms:
//   - If Picker.Engine is set, it returns the corresponding default instance from the map.
//   - If InlineConfig is provided, it dynamically instantiates a new storage engine.
//
// Parameters:
//   - ctx: request-scoped context for cancellation.
//   - picker: configuration object for selecting or creating the storage engine.
//
// Returns:
//   - An instance of type S representing the resolved storage backend.
//   - An error if the engine is unsupported, fails during construction,
//     or the result does not match the expected type.
func Resolve[S any](ctx context.Context, picker storage.Picker) (S, error) {
	var (
		zero      S
		engineKey = strings.ToUpper(picker.Engine)
	)

	if engineKey != "" {
		instance, found := defaults[engineKey]
		if !found {
			return zero, fmt.Errorf("no default instance found for selected engine %q", engineKey)
		}

		engine, ok := instance.(S)
		if !ok {
			return zero, fmt.Errorf("default instance for engine %q has unexpected type", engineKey)
		}

		return engine, nil
	}

	inlineStruct := reflect.ValueOf(picker.InlineConfig)
	inlineType := reflect.TypeOf(picker.InlineConfig)

	for i := 0; i < inlineStruct.NumField(); i++ {
		inlineField := inlineStruct.Field(i)
		if inlineField.Kind() != reflect.Ptr || inlineField.IsNil() {
			continue
		}

		engineName := strings.ToUpper(inlineType.Field(i).Name)

		constructor, exists := storageFactories[engineName]
		if !exists {
			return zero, fmt.Errorf("no factory registered for inline-configured engine %q", engineName)
		}

		instance, err := constructor(ctx, inlineField.Elem().Interface())
		if err != nil {
			return zero, fmt.Errorf("failed to create inline instance for engine %q: %w", engineName, err)
		}

		if closer, ok := instance.(io.Closer); ok {
			openedConnections = append(openedConnections, closer)
		}

		engine, ok := instance.(S)
		if !ok {
			return zero, fmt.Errorf("inline instance for engine %q has unexpected type", engineName)
		}

		return engine, nil
	}

	return zero, fmt.Errorf("no valid storage engine configuration provided")
}
