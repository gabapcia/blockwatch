package bootstrap

import (
	"context"
	"fmt"
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
// To support a new storage engine, add its factory directly to this map.
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

// buildDefaultStorages instantiates default storage engines from the global configuration.
//
// It reflects over the fields of the Engines struct, identifies which backends are configured,
// and uses the corresponding factory to create shared instances.
//
// Parameters:
//   - ctx: request-scoped context for cancellation.
//   - enginesConfig: the global Engines struct populated from configuration.
//
// Returns:
//   - A map of engine names to their initialized instances.
//   - An error if any engine has no registered factory or fails during instantiation.
func buildDefaultStorages(ctx context.Context, enginesConfig storage.Engines) (map[string]any, error) {
	defaultInstances := make(map[string]any)

	structVal := reflect.ValueOf(enginesConfig)
	structType := reflect.TypeOf(enginesConfig)

	for i := 0; i < structVal.NumField(); i++ {
		fieldVal := structVal.Field(i)
		if fieldVal.Kind() != reflect.Ptr || fieldVal.IsNil() {
			continue
		}

		engineName := strings.ToUpper(structType.Field(i).Name)

		constructor, exists := storageFactories[engineName]
		if !exists {
			return nil, fmt.Errorf("no factory registered for storage engine %q", engineName)
		}

		instance, err := constructor(ctx, fieldVal.Elem().Interface())
		if err != nil {
			return nil, fmt.Errorf("failed to initialize storage engine %q: %w", engineName, err)
		}

		defaultInstances[engineName] = instance
	}

	return defaultInstances, nil
}

// resolveStorage selects and returns a storage instance for a use case based on a Picker.
//
// It supports two selection mechanisms:
//   - If Picker.Engine is set, it looks up the corresponding default instance.
//   - If InlineConfig is provided, it dynamically creates a new instance using the factory.
//
// Parameters:
//   - ctx: request-scoped context for cancellation.
//   - picker: configuration for selecting or creating a storage engine.
//   - defaults: map of shared default instances, usually created by buildDefaultStorages.
//
// Returns:
//   - The resolved storage instance, casted to the generic type S.
//   - An error if selection fails, the factory is not registered, the creation fails,
//     or the type cast is invalid.
func resolveStorage[S any](ctx context.Context, picker storage.Picker, defaults map[string]any) (S, error) {
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

		engine, ok := instance.(S)
		if !ok {
			return zero, fmt.Errorf("inline instance for engine %q has unexpected type", engineName)
		}

		return engine, nil
	}

	return zero, fmt.Errorf("no valid storage engine configuration provided")
}
