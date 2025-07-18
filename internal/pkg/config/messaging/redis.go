package messaging

// Redis defines the configuration required to use Redis Streams as a messaging backend.
type Redis struct {
	Address  string `env:"ADDRESS" validate:"required"` // Address is the Redis server address in the format "host:port".
	Username string `env:"USERNAME"`                    // Username is the optional username for ACL-based authentication.
	Password string `env:"PASSWORD"`                    // Password is the password or token for authentication.
	DB       int    `env:"DB"`                          // DB is the Redis database index (default is 0).
}
