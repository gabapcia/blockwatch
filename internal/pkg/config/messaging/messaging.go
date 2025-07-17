package messaging

const (
	EngineRedis    = "REDIS"
	EngineRabbitMQ = "RABBITMQ"
)

type Engines struct {
	Redis    Redis    `validate:"omitempty"`
	RabbitMQ RabbitMQ `validate:"omitempty"`
}

type InlineConfig struct {
	Redis    Redis    `validate:"required_alone"`
	RabbitMQ RabbitMQ `validate:"required_alone"`
}

type Picker struct {
	Engine string       `validate:"omitempty,oneof=REDIS RABBITMQ"`
	Config InlineConfig `validate:"required_without=Engine,excluded_with=Engine"`
}
