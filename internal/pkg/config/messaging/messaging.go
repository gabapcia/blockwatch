package messaging

// Supported messaging engine identifiers used for selection and validation.
const (
	// EngineRedis represents Redis Streams as a messaging backend.
	EngineRedis = "REDIS"

	// EngineRabbitMQ represents RabbitMQ as a messaging backend.
	EngineRabbitMQ = "RABBITMQ"
)

// Engines contains global/shared messaging engine configurations.
type Engines struct {
	Redis    Redis    `validate:"omitempty"` // Redis holds the global Redis Streams messaging configuration.
	RabbitMQ RabbitMQ `validate:"omitempty"` // RabbitMQ holds the global RabbitMQ messaging configuration.
}

// InlineConfig defines an inline messaging configuration for a specific use case.
//
// Only one engine should be configured per instance.
type InlineConfig struct {
	Redis    Redis    `validate:"required_alone"` // Redis holds the inline Redis Streams configuration.
	RabbitMQ RabbitMQ `validate:"required_alone"` // RabbitMQ holds the inline RabbitMQ configuration.
}

// Picker allows a use case to select a messaging engine by name or provide an inline configuration.
//
// The Engine and Config fields are mutually exclusive.
type Picker struct {
	// Engine indicates which globally defined messaging engine to use.
	// Accepted values: "REDIS", "RABBITMQ".
	Engine string `validate:"omitempty,oneof=REDIS RABBITMQ"`

	// Config provides an inline configuration to use instead of a global engine.
	Config InlineConfig `validate:"required_without=Engine,excluded_with=Engine"`
}
