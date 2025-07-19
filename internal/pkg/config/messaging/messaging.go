package messaging

import (
	"reflect"

	"github.com/gabapcia/blockwatch/internal/pkg/validator"
)

func init() {
	// Register the custom struct-level validator for the Picker struct.
	// This ensures validation logic beyond simple field tags.
	validator.RegisterStructValidation(validatePickerStruct, Picker{}, &Picker{})
}

// validatePickerStruct performs cross-field validation for the Picker struct.
// It ensures that the MessagePublisher configuration matches the selected engine,
// whether defined globally (via Engine) or inline (via InlineConfig).
func validatePickerStruct(sl validator.StructLevel) {
	var picker Picker

	// Normalize the struct: support both Picker and *Picker types.
	switch v := sl.Current().Interface().(type) {
	case Picker:
		picker = v
	case *Picker:
		if v == nil {
			return // nil pointer, nothing to validate
		}
		picker = *v
	default:
		return // unsupported type
	}

	// Validate according to selected or inline engine
	switch {
	// Case: Redis engine selected via name or inline config
	case picker.Engine == EngineRedis || picker.InlineConfig.Redis != nil:
		if picker.MessagePublisher.Redis == nil {
			// Redis is selected but no Redis publisher is configured
			sl.ReportError(reflect.ValueOf(picker.MessagePublisher), "MessagePublisher.Redis", "Redis", "required", "")
		}

	// Case: RabbitMQ engine selected via name or inline config
	case picker.Engine == EngineRabbitMQ || picker.InlineConfig.RabbitMQ != nil:
		if picker.MessagePublisher.RabbitMQ == nil {
			// RabbitMQ is selected but no RabbitMQ publisher is configured
			sl.ReportError(reflect.ValueOf(picker.MessagePublisher), "MessagePublisher.RabbitMQ", "RabbitMQ", "required", "")
		}

	// Case: No engine provided in any form
	default:
		sl.ReportError(reflect.ValueOf(picker.InlineConfig), "InlineConfig", "InlineConfig", "required_engine", "")
		return
	}

	// ✅ How to add support for a new messaging engine in the future:
	//
	// 1. Add a constant for the engine name (e.g., EngineKafka = "KAFKA").
	// 2. Extend Engines, InlineConfig, and MessagePublisher structs with Kafka fields.
	// 3. Add a new validation case here:
	//    case picker.Engine == EngineKafka || picker.InlineConfig.Kafka != nil:
	//        if picker.MessagePublisher.Kafka == nil {
	//            sl.ReportError(..., "MessagePublisher.Kafka", "Kafka", "required", "")
	//        }
}

// Supported messaging engine identifiers used for selection and validation.
const (
	// EngineRedis represents Redis Streams as a messaging backend.
	EngineRedis = "REDIS"

	// EngineRabbitMQ represents RabbitMQ as a messaging backend.
	EngineRabbitMQ = "RABBITMQ"
)

// Engines contains global/shared messaging engine configurations.
type Engines struct {
	// Redis holds the global Redis Streams messaging configuration.
	Redis *RedisConnection `env:",prefix=REDIS_" validate:"omitempty"`

	// RabbitMQ holds the global RabbitMQ messaging configuration.
	RabbitMQ *RabbitMQConnection `env:",prefix=RABBITMQ_" validate:"omitempty"`
}

// MessagePublisher defines the message publication configuration for a specific use case.
//
// Only one backend (Redis or RabbitMQ) must be configured per instance.
type MessagePublisher struct {
	// Redis holds the Redis Streams publication settings.
	Redis *RedisPublisher `env:",prefix=REDIS_" validate:"omitempty,required_alone"`

	// RabbitMQ holds the RabbitMQ publication settings.
	RabbitMQ *RabbitMQPublisher `env:",prefix=RABBITMQ_" validate:"omitempty,required_alone"`
}

// InlineConfig defines an inline messaging configuration for a specific use case.
//
// Only one engine should be configured per instance.
type InlineConfig struct {
	// Redis holds the inline Redis Streams configuration.
	Redis *RedisConnection `env:",prefix=REDIS_" validate:"omitempty,required_alone"`

	// RabbitMQ holds the inline RabbitMQ configuration.
	RabbitMQ *RabbitMQConnection `env:",prefix=RABBITMQ_" validate:"omitempty,required_alone"`
}

// Picker allows a use case to select a messaging engine by name or provide an inline configuration.
//
// The Engine and InlineConfig fields are mutually exclusive.
// MessagePublisher must be set and correspond to the selected backend.
type Picker struct {
	// Engine indicates which globally defined messaging engine to use.
	// Accepted values: "REDIS", "RABBITMQ".
	Engine string `env:"ENGINE" validate:"required_without=InlineConfig,omitempty,oneof=REDIS RABBITMQ"`

	// InlineConfig provides an inline configuration to use instead of a global engine.
	InlineConfig `validate:"required_without=Engine"`

	// MessagePublisher holds the message publication configuration for the selected engine.
	MessagePublisher `validate:"required"`
}
