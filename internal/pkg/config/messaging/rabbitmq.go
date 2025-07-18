package messaging

// RabbitMQ defines the configuration required to connect to a RabbitMQ broker.
type RabbitMQ struct {
	URI        string `env:"URI" validate:"required"` // URI is the AMQP connection string (e.g., "amqp://user:pass@host:5672/vhost").
	Exchange   string `env:"EXCHANGE"`
	RoutingKey string `env:"ROUTING_KEY" validate:"required"`
}
