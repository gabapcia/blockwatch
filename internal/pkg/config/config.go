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
	// Register the custom struct-level validator for the Config struct.
	// This ensures validation logic beyond simple field tags.
	validator.RegisterStructValidation(validateConfigStruct, Config{}, &Config{})
}

// validateConfigStruct performs cross-field validation for the Config struct.
// It ensures that when storage.Picker or messaging.Picker fields have an Engine specified,
// the corresponding engine is not nil in the Engines struct.
func validateConfigStruct(sl validator.StructLevel) {
	var config Config

	// Normalize the struct: support both Config and *Config types.
	switch v := sl.Current().Interface().(type) {
	case Config:
		config = v
	case *Config:
		if v == nil {
			return // nil pointer, nothing to validate
		}

		config = *v
	default:
		return // unsupported type
	}

	// Validate all storage.Picker and messaging.Picker fields in the config
	validatePickersInStruct(sl, reflect.ValueOf(config), config.Engines)
}

// validatePickersInStruct recursively validates all storage.Picker and messaging.Picker fields
// in a struct to ensure their Engine references point to non-nil engines in the Engines struct.
func validatePickersInStruct(sl validator.StructLevel, structValue reflect.Value, engines Engines) {
	structType := structValue.Type()

	for i := 0; i < structValue.NumField(); i++ {
		field := structValue.Field(i)
		fieldType := structType.Field(i)

		// Skip unexported fields
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
			// If it's a struct, recursively check its fields
			if field.Kind() == reflect.Struct {
				validatePickersInStruct(sl, field, engines)
			}
		}
	}
}

// validateStoragePicker validates a storage.Picker to ensure that if Engine is specified,
// the corresponding engine is not nil in the storage.Engines struct.
func validateStoragePicker(sl validator.StructLevel, picker storage.Picker, engines storage.Engines, fieldName string) {
	if picker.Engine == "" {
		return // No engine specified, nothing to validate
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

// validateMessagingPicker validates a messaging.Picker to ensure that if Engine is specified,
// the corresponding engine is not nil in the messaging.Engines struct.
func validateMessagingPicker(sl validator.StructLevel, picker messaging.Picker, engines messaging.Engines, fieldName string) {
	if picker.Engine == "" {
		return // No engine specified, nothing to validate
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

// Engines defines globally shared engine configurations for storage and messaging backends.
type Engines struct {
	// Storage holds global storage backend configurations (e.g., Redis, PostgreSQL).
	Storage storage.Engines `env:", prefix=STORAGE_" validate:"omitempty"`

	// Messaging holds global messaging backend configurations (e.g., Redis Streams, RabbitMQ).
	Messaging messaging.Engines `env:", prefix=MESSAGING_" validate:"omitempty"`
}

// Config represents the full application configuration, including logging, telemetry,
// global engine definitions, and per-use-case settings.
type Config struct {
	Log       pkg.Logger    `env:", prefix=LOG_" validate:"required"`       // Log defines the logging configuration for the application.
	Telemetry pkg.Telemetry `env:", prefix=TELEMETRY_" validate:"required"` // Telemetry defines the telemetry and service identity configuration.

	Engines Engines `env:", prefix=ENGINES_" validate:"omitempty"` // Engines holds the globally defined storage and messaging engine configurations.

	Walletregistry WalletRegistry `env:", prefix=WALLETREGISTRY_" validate:"required"` // Walletregistry contains the configuration for the wallet registry use case.
	Walletwatch    WalletWatch    `env:", prefix=WALLETWATCH_" validate:"required"`    // Walletwatch contains the configuration for the wallet transaction watcher use case.
	Chainstream    ChainStream    `env:", prefix=CHAINSTREAM_" validate:"required"`    // Chainstream contains the configuration for the chainstream use case.
}

// process loads environment variables into the provided config struct using envconfig.
//
// This internal helper enables reusability for custom bootstrapping.
func process(ctx context.Context, cfg any) error {
	return envconfig.ProcessWith(ctx, &envconfig.Config{
		Target:        cfg,
		DefaultNoInit: true,
	})
}

// validate runs validation logic on the provided config struct.
//
// It ensures all validation tags are satisfied using the application's shared validator.
func validate(cfg any) error {
	return validator.Validate(cfg)
}

// Load reads the application configuration from environment variables,
// applies defaults, and validates all fields.
//
// This is the main entry point for loading configuration at application startup.
//
// Parameters:
//   - ctx: context for cancellation.
//
// Returns:
//   - A fully populated and validated Config struct.
//   - An error if processing or validation fails.
func Load(ctx context.Context) (config Config, err error) {
	if err = process(ctx, &config); err != nil {
		return
	}

	if err = validate(config); err != nil {
		return
	}

	return
}
