package messaging

import (
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"

	messagingconfig "github.com/gabapcia/blockwatch/internal/pkg/config/messaging"
)

// defaults holds the default shared messaging client instances,
// indexed by engine name (e.g., "REDIS", "RABBITMQ").
//
// These instances are created once via Init and reused during resolution.
var defaults map[string]any

// openedConnections tracks messaging clients that implement io.Closer.
//
// This allows graceful shutdown of all connections via Close.
var openedConnections []io.Closer

// typeOf returns the reflect.Type representation of a generic interface type T.
//
// This is useful for working with reflection-based registries or adapter mappings,
// where the type of an interface is required as a key.
//
// Example:
//
//	typeOf[io.Closer]() // returns reflect.Type of the io.Closer interface
func typeOf[T any]() reflect.Type {
	return reflect.TypeOf((*T)(nil)).Elem()
}

// Init initializes default messaging connections from the provided global configuration.
//
// It uses reflection to detect which engines are configured in the Engines struct,
// creates their connections using the registered messagingFactories, and stores
// the resulting instances in a global map for later resolution.
//
// If any connection implements io.Closer, it is also tracked for later cleanup via Close().
//
// Parameters:
//   - ctx: request-scoped context for cancellation.
//   - engines: the global messaging engines configuration.
//
// Returns:
//   - An error if a factory is missing or connection creation fails.
func Init(ctx context.Context, engines messagingconfig.Engines) error {
	defaults = make(map[string]any)
	openedConnections = make([]io.Closer, 0)

	engineValues := reflect.ValueOf(engines)
	engineTypes := reflect.TypeOf(engines)

	for i := 0; i < engineValues.NumField(); i++ {
		configValue := engineValues.Field(i)
		if configValue.Kind() != reflect.Ptr || configValue.IsNil() {
			continue
		}

		engineName := strings.ToUpper(engineTypes.Field(i).Name)
		factory, ok := messagingFactories[engineName]
		if !ok {
			return fmt.Errorf("no messaging factory registered for engine %q", engineName)
		}

		conn, err := factory.BuildConnection(ctx, configValue.Elem().Interface())
		if err != nil {
			return fmt.Errorf("failed to initialize messaging engine %q: %w", engineName, err)
		}

		defaults[engineName] = conn

		if closer, ok := conn.(io.Closer); ok {
			openedConnections = append(openedConnections, closer)
		}
	}

	return nil
}

// Close shuts down all messaging connections that were opened by Init.
//
// This function iterates over all instances that implement io.Closer and
// invokes Close on each. If any Close call returns an error, they are joined
// and returned as a single error value.
//
// Returns:
//   - A combined error if any of the connections failed to close.
func Close() error {
	errorList := make([]error, 0)
	for _, conn := range openedConnections {
		errorList = append(errorList, conn.Close())
	}

	return errors.Join(errorList...)
}
