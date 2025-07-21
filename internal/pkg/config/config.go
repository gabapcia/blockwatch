// Package config provides the centralized application configuration layer.
//
// It loads and validates configuration from environment variables using the
// github.com/sethvargo/go-envconfig library, and supports advanced validation
// for engine references across use cases.
package config

import (
	"context"
	"reflect"

	"github.com/gabapcia/blockwatch/internal/pkg/config/messaging"
	"github.com/gabapcia/blockwatch/internal/pkg/config/pkg"
	"github.com/gabapcia/blockwatch/internal/pkg/config/storage"
	"github.com/gabapcia/blockwatch/internal/pkg/validator"

	"github.com/sethvargo/go-envconfig"
)

func init() {
	// Register the custom struct-level validation for Config.
	validator.RegisterStructValidation(validateConfigStruct, Config{}, &Config{})
}

// validateConfigStruct performs cross-field validation on Config.
//
// It checks that whenever a storage.Picker or messaging.Picker specifies an engine,
// the referenced engine is defined and configured in the Engines block.
func validateConfigStruct(sl validator.StructLevel) {
	var config Config

	switch v := sl.Current().Interface().(type) {
	case Config:
		config = v
	case *Config:
		if v == nil {
			return
		}

		config = *v
	default:
		return
	}

	validatePickersInStruct(sl, reflect.ValueOf(config), config.Engines)
}

// validatePickersInStruct recursively checks fields of type storage.Picker and messaging.Picker.
//
// For each picker, it verifies that the referenced engine is defined in the Engines struct.
func validatePickersInStruct(sl validator.StructLevel, structValue reflect.Value, engines Engines) {
	structType := structValue.Type()

	for i := 0; i < structValue.NumField(); i++ {
		field := structValue.Field(i)
		fieldType := structType.Field(i)

		if !field.CanInterface() {
			continue
		}

		switch field.Type() {
		case reflect.TypeOf(storage.Picker{}):
			validateStoragePicker(sl, field.Interface().(storage.Picker), engines.Storage, fieldType.Name)

		case reflect.TypeOf((*storage.Picker)(nil)):
			if !field.IsNil() {
				validateStoragePicker(sl, *field.Interface().(*storage.Picker), engines.Storage, fieldType.Name)
			}

		case reflect.TypeOf(messaging.Picker{}):
			validateMessagingPicker(sl, field.Interface().(messaging.Picker), engines.Messaging, fieldType.Name)

		case reflect.TypeOf((*messaging.Picker)(nil)):
			if !field.IsNil() {
				validateMessagingPicker(sl, *field.Interface().(*messaging.Picker), engines.Messaging, fieldType.Name)
			}

		default:
			if field.Kind() == reflect.Struct {
				validatePickersInStruct(sl, field, engines)
			}
		}
	}
}

// validateStoragePicker checks that the given storage.Picker references a configured engine.
func validateStoragePicker(sl validator.StructLevel, picker storage.Picker, engines storage.Engines, fieldName string) {
	if picker.Engine == "" {
		return
	}

	switch picker.Engine {
	case storage.EngineRedis:
		if engines.Redis == nil {
			sl.ReportError(reflect.ValueOf(picker), fieldName+".Engine", "Engine", "engine_not_configured", picker.Engine)
		}
	case storage.EnginePostgreSQL:
		if engines.PostgreSQL == nil {
			sl.ReportError(reflect.ValueOf(picker), fieldName+".Engine", "Engine", "engine_not_configured", picker.Engine)
		}
	default:
		sl.ReportError(reflect.ValueOf(picker), fieldName+".Engine", "Engine", "engine_not_registered", picker.Engine)
	}
}

// validateMessagingPicker checks that the given messaging.Picker references a configured engine.
func validateMessagingPicker(sl validator.StructLevel, picker messaging.Picker, engines messaging.Engines, fieldName string) {
	if picker.Engine == "" {
		return
	}

	switch picker.Engine {
	case messaging.EngineRedis:
		if engines.Redis == nil {
			sl.ReportError(reflect.ValueOf(picker), fieldName+".Engine", "Engine", "engine_not_configured", picker.Engine)
		}
	case messaging.EngineRabbitMQ:
		if engines.RabbitMQ == nil {
			sl.ReportError(reflect.ValueOf(picker), fieldName+".Engine", "Engine", "engine_not_configured", picker.Engine)
		}
	default:
		sl.ReportError(reflect.ValueOf(picker), fieldName+".Engine", "Engine", "engine_not_registered", picker.Engine)
	}
}

// Engines defines globally shared engine configurations for storage and messaging.
type Engines struct {
	// Storage holds backend options for database persistence (e.g., Redis, PostgreSQL).
	Storage storage.Engines `env:", prefix=STORAGE_" validate:"omitempty"`

	// Messaging holds backend options for message passing (e.g., Redis Streams, RabbitMQ).
	Messaging messaging.Engines `env:", prefix=MESSAGING_" validate:"omitempty"`
}

// Config aggregates the top-level configuration for the entire application.
type Config struct {
	ServiceName string     `env:"SERVICE_NAME, default=blockwatch" validate:"required"` // The service name for logging, metrics, etc.
	Log         pkg.Logger `env:", prefix=LOG_" validate:"required"`                    // Logging configuration for the service.

	Engines Engines `env:", prefix=ENGINES_" validate:"omitempty"` // Engines contains globally available backends (databases, brokers).

	Walletregistry WalletRegistry `env:", prefix=WALLETREGISTRY_" validate:"required"` // Walletregistry defines configuration for the wallet registry use case.
	Walletwatch    WalletWatch    `env:", prefix=WALLETWATCH_" validate:"required"`    // Walletwatch defines configuration for the wallet transaction watch use case.
	Chainstream    ChainStream    `env:", prefix=CHAINSTREAM_" validate:"required"`    // Chainstream defines configuration for the chainstream processing use case.
}

// process loads environment variables into the given configuration target.
//
// It skips automatic field initialization (DefaultNoInit = true), enabling zero-value
// struct checks and stricter validation downstream.
func process(ctx context.Context, cfg any) error {
	return envconfig.ProcessWith(ctx, &envconfig.Config{
		Target:        cfg,
		DefaultNoInit: true,
	})
}

// validate runs field and struct-level validations using the shared validator.
func validate(cfg any) error {
	return validator.Validate(cfg)
}

// Load loads and validates the entire application configuration.
//
// It reads from environment variables, applies default values, and enforces
// custom validation logic (e.g., engine dependencies).
//
// Returns:
//   - A fully populated Config instance
//   - An error if any environment value is missing or invalid
func Load(ctx context.Context) (config Config, err error) {
	if err = process(ctx, &config); err != nil {
		return
	}

	if err = validate(config); err != nil {
		return
	}

	return
}
