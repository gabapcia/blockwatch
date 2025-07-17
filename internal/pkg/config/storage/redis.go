package storage

// Redis defines the configuration required to connect to a Redis instance
// for use as a storage backend.
type Redis struct {
	Address  string `validate:"required"` // Redis server address "host:port" (required)
	Username string // Optional username for ACL-based Redis instances
	Password string // Password or authentication token
	DB       int    // Redis database index (default 0)
}
