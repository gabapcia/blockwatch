package storage

// PostgreSQL defines the configuration required to connect to a PostgreSQL database.
type PostgreSQL struct {
	// DSN is the PostgreSQL connection string in either URL-style or libpq-style format.
	//
	// Examples:
	//   - URL-style:   "postgres://user:pass@host:port/dbname"
	//   - libpq-style: "host=localhost port=5432 user=... password=... dbname=..."
	DSN string `validate:"required"`
}
