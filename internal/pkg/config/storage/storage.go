package storage

// Supported storage engine identifiers used for selection and validation.
const (
	// EngineRedis represents Redis as a storage backend.
	EngineRedis = "REDIS"

	// EnginePostgreSQL represents PostgreSQL as a storage backend.
	EnginePostgreSQL = "POSTGRESQL"
)

// Engines contains global/shared storage configurations that can be reused
// by multiple use cases.
//
// Each field is optional and only populated if explicitly configured.
type Engines struct {
	Redis      *Redis      `env:", prefix=REDIS_" validate:"omitempty"`      // Global Redis configuration
	PostgreSQL *PostgreSQL `env:", prefix=POSTGRESQL_" validate:"omitempty"` // Global PostgreSQL configuration
}

// InlineConfig defines an inline storage configuration for a specific use case.
//
// Only one engine must be configured per instance. This struct is mutually exclusive
// with Engine-based selection in Picker.
type InlineConfig struct {
	Redis      *Redis      `env:", prefix=REDIS_" validate:"omitempty,required_alone"`      // Inline Redis configuration
	PostgreSQL *PostgreSQL `env:", prefix=POSTGRESQL_" validate:"omitempty,required_alone"` // Inline PostgreSQL configuration
}

// Picker allows a use case to select a storage configuration either by referring to a
// globally defined engine (via the Engine field), or by providing an inline configuration.
//
// If Engine is set, it must match one of the global engines defined in Engines.
// If Config is set, a new connection will be created based on the provided configuration.
//
// These two fields are mutually exclusive and validated accordingly.
type Picker struct {
	// Engine indicates which globally defined storage engine to use.
	// Accepted values: "REDIS", "POSTGRESQL".
	Engine string `env:"ENGINE" validate:"required_without=InlineConfig,excluded_with=InlineConfig,omitempty,oneof=REDIS POSTGRESQL"`

	// InlineConfig provides an inline configuration for use-case-specific connection setup.
	InlineConfig `validate:"required_without=Engine,excluded_with=Engine"`
}
