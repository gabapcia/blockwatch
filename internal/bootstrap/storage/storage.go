package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"

	storageconfig "github.com/gabapcia/blockwatch/internal/pkg/config/storage"
)

// defaults holds the default initialized storage instances keyed by engine name.
var defaults map[string]any

// openedConnections tracks all io.Closer instances created during Init for proper cleanup in Close.
var openedConnections []io.Closer

// Init instantiates all configured storage engines defined in the global configuration.
//
// It uses reflection to iterate over the fields of the storage.Engines struct,
// detects which backends are enabled (non-nil), and invokes their corresponding factory
// functions defined in storageFactories.
//
// If a created instance implements io.Closer, it is tracked internally for cleanup.
//
// Parameters:
//   - ctx: request-scoped context for cancellation.
//   - enginesConfig: the global Engines struct populated from configuration.
//
// Returns:
//   - nil on success.
//   - An error if any engine is not supported or fails during initialization.
func Init(ctx context.Context, enginesConfig storageconfig.Engines) error {
	defaults = make(map[string]any)
	openedConnections = make([]io.Closer, 0)

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
			return fmt.Errorf("no factory registered for storage engine %q", engineName)
		}

		instance, err := constructor(ctx, fieldVal.Elem().Interface())
		if err != nil {
			return fmt.Errorf("failed to initialize storage engine %q: %w", engineName, err)
		}

		if closer, ok := instance.(io.Closer); ok {
			openedConnections = append(openedConnections, closer)
		}

		defaults[engineName] = instance
	}

	return nil
}

// Close releases all opened storage connections that implement io.Closer.
//
// This should be called via defer after Init to ensure all resources are properly released.
//
// Returns:
//   - An aggregated error if any connection fails to close, or nil if all succeeded.
func Close() error {
	errorList := make([]error, 0)
	for _, conn := range openedConnections {
		errorList = append(errorList, conn.Close())
	}

	return errors.Join(errorList...)
}
