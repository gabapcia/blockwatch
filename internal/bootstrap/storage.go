package bootstrap

import (
	"context"
	"reflect"
	"strings"

	"github.com/gabapcia/blockwatch/internal/infra/storage/postgresql"
	"github.com/gabapcia/blockwatch/internal/infra/storage/redis"
	"github.com/gabapcia/blockwatch/internal/pkg/config/storage"
	"github.com/gabapcia/blockwatch/internal/pkg/logger"
)

// storageFactory defines the constructor signature for supported storage backends.
//
// Each registered factory is responsible for creating a new instance of the
// storage backend using the provided configuration.
type storageFactory func(ctx context.Context, config any) (any, error)

// storageFactories maps each supported storage engine to its corresponding factory function.
//
// Keys must match the uppercase engine identifiers used in configuration,
// such as "REDIS" or "POSTGRESQL".
var storageFactories = map[string]storageFactory{}

// init registers the available storage engines and their factory constructors.
//
// To support a new storage backend, add its factory function here.
//
// Example:
//
//	storageFactories["MONGODB"] = func(ctx context.Context, cfg any) (any, error) {
//		mongoCfg := cfg.(storage.MongoDB)
//		return mongodb.NewClient(ctx, mongoCfg.URI)
//	}
func init() {
	storageFactories["REDIS"] = func(ctx context.Context, cfg any) (any, error) {
		redisCfg := cfg.(storage.Redis)
		return redis.NewClient(ctx, redisCfg.Address, redisCfg.Username, redisCfg.Password, redisCfg.DB)
	}

	storageFactories["POSTGRESQL"] = func(ctx context.Context, cfg any) (any, error) {
		pgCfg := cfg.(storage.PostgreSQL)
		return postgresql.New(ctx, pgCfg.DSN)
	}
}

// buildDefaultStorages instantiates default storage engines from the global configuration.
//
// It reflects over the fields of the Engines struct, identifies which backends are configured,
// and uses the corresponding factory to create shared instances.
//
// Parameters:
//   - ctx: request-scoped context for cancellation and logging.
//   - enginesConfig: the global Engines struct populated from configuration.
//
// Returns:
//   - A map of engine names to their initialized instances.
//
// Panics (via logger.Fatal) if a configured engine has no registered factory or fails during instantiation.
func buildDefaultStorages(ctx context.Context, enginesConfig storage.Engines) map[string]any {
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
			logger.Fatal(ctx, "No factory registered for storage engine", "engine", engineName)
		}

		instance, err := constructor(ctx, fieldVal.Elem().Interface())
		if err != nil {
			logger.Fatal(ctx, "Failed to initialize storage engine", "engine", engineName, "error", err)
		}

		defaultInstances[engineName] = instance
	}

	return defaultInstances
}

// resolveStorage selects and returns a storage instance for a use case based on a Picker.
//
// It supports two selection mechanisms:
//   - If Picker.Engine is set, it looks up the corresponding default instance.
//   - If InlineConfig is provided, it dynamically creates a new instance using the factory.
//
// Parameters:
//   - ctx: request-scoped context for cancellation and logging.
//   - picker: configuration for selecting or creating a storage engine.
//   - defaults: map of shared default instances, usually created by buildDefaultStorages.
//
// Returns:
//   - The resolved storage instance, casted to the generic type S.
//   - A boolean indicating whether the cast succeeded (always true unless Fatal is bypassed).
//
// Panics (via logger.Fatal) if selection fails, the factory is not registered,
// the creation fails, or the type cast is invalid.
func resolveStorage[S any](ctx context.Context, picker storage.Picker, defaults map[string]any) (engine S, ok bool) {
	engineKey := strings.ToUpper(picker.Engine)

	if engineKey != "" {
		instance, found := defaults[engineKey]
		if !found {
			logger.Fatal(ctx, "No default instance found for selected engine", "engine", engineKey)
		}

		engine, ok = instance.(S)
		if !ok {
			logger.Fatal(ctx, "Failed to cast default storage instance to requested type", "engine", engineKey)
		}

		return
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
			logger.Fatal(ctx, "No factory registered for inline-configured storage engine", "engine", engineName)
		}

		instance, err := constructor(ctx, inlineField.Elem().Interface())
		if err != nil {
			logger.Fatal(ctx, "Failed to create inline storage instance", "engine", engineName, "error", err)
		}

		engine, ok = instance.(S)
		if !ok {
			logger.Fatal(ctx, "Failed to cast inline storage instance to requested type", "engine", engineName)
		}

		return
	}

	logger.Fatal(ctx, "No valid storage engine configuration provided")
	return
}
