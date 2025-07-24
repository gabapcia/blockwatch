package postgresql

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
)

func TestIsUniqueViolation(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		result := isUniqueViolation(nil)
		assert.False(t, result)
	})

	t.Run("non-postgresql error", func(t *testing.T) {
		err := errors.New("some generic error")
		result := isUniqueViolation(err)
		assert.False(t, result)
	})

	t.Run("postgresql error but not unique violation", func(t *testing.T) {
		err := &pgconn.PgError{
			Code: pgerrcode.NotNullViolation,
		}
		result := isUniqueViolation(err)
		assert.False(t, result)
	})

	t.Run("postgresql unique violation error", func(t *testing.T) {
		err := &pgconn.PgError{
			Code: pgerrcode.UniqueViolation,
		}
		result := isUniqueViolation(err)
		assert.True(t, result)
	})
}

func TestIsNotFoundError(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		result := isNotFoundError(nil)
		assert.False(t, result)
	})

	t.Run("non-pgx error", func(t *testing.T) {
		err := errors.New("some generic error")
		result := isNotFoundError(err)
		assert.False(t, result)
	})

	t.Run("pgx no rows error", func(t *testing.T) {
		result := isNotFoundError(pgx.ErrNoRows)
		assert.True(t, result)
	})

	t.Run("wrapped pgx no rows error", func(t *testing.T) {
		wrappedErr := fmt.Errorf("query failed: %w", pgx.ErrNoRows)
		result := isNotFoundError(wrappedErr)
		assert.True(t, result)
	})
}
