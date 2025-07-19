package messaging

// RedisConnection defines the configuration required to connect to a Redis server
// for use with Redis Streams as a messaging backend.
type RedisConnection struct {
	// Address is the Redis server address in the format "host:port".
	Address string `env:"ADDRESS" validate:"required"`

	// Username is the optional username used for ACL-based authentication.
	Username string `env:"USERNAME"`

	// Password is the password or token used for authentication.
	Password string `env:"PASSWORD"`

	// DB is the Redis logical database index to use (default is 0).
	DB int `env:"DB"`
}

// RedisPublisher defines the configuration for publishing messages to a Redis Stream.
type RedisPublisher struct {
	// Stream is the name of the Redis Stream to which messages will be published.
	Stream string `env:"STREAM" validate:"required"`
}
