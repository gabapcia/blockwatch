package messaging

// RabbitMQConnection defines the connection parameters required to establish
// a connection with a RabbitMQ broker.
type RabbitMQConnection struct {
	// URI is the AMQP connection string (e.g., "amqp://user:pass@host:5672/vhost").
	URI string `env:"URI" validate:"required"`
}

// RabbitMQPublisher defines the configuration required to publish messages
// to a RabbitMQ exchange using a specific routing key.
type RabbitMQPublisher struct {
	// Exchange is the name of the exchange to which messages will be published.
	// If empty, the default exchange is used.
	Exchange string `env:"EXCHANGE"`

	// RoutingKey defines the routing key to use when publishing messages.
	RoutingKey string `env:"ROUTING_KEY" validate:"required"`
}
