package postgresql

import (
	"errors"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// isUniqueViolation checks whether the given error is a PostgreSQL unique constraint violation.
//
// This is typically used to detect duplicate key errors when inserting or updating rows
// that must be unique (e.g., on a UNIQUE index or constraint).
//
// Parameters:
//   - err: the error to inspect.
//
// Returns:
//   - true if the error is a PostgreSQL unique violation (SQLSTATE 23505), false otherwise.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation
}

// isNotFoundError checks whether the given error indicates that a query returned no rows.
//
// This is typically used to detect the absence of data when performing a SELECT query
// with SQLC using the `:one` tag, which expects exactly one row.
//
// Parameters:
//   - err: the error to inspect.
//
// Returns:
//   - true if the error is pgx.ErrNoRows, false otherwise.
func isNotFoundError(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
