package storage

// Redis defines the configuration required to connect to a Redis instance
// for use as a storage backend.
type Redis struct {
	Address  string `env:"ADDRESS" validate:"required"` // Redis server address "host:port" (required)
	Username string `env:"USERNAME"`                    // Optional username for ACL-based Redis instances
	Password string `env:"PASSWORD"`                    // Password or authentication token
	DB       int    `env:"DB"`                          // Redis database index (default 0)
}
