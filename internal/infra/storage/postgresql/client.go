package postgresql

import (
	"context"

	"github.com/gabapcia/blockwatch/internal/infra/storage/postgresql/internal/sqlc"

	"github.com/jackc/pgx/v5/pgxpool"
)

type client struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

func (c client) Close() {
	c.pool.Close()
}

func New(ctx context.Context, dsn string) (*client, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}

	return &client{
		pool:    pool,
		queries: sqlc.New(pool),
	}, nil
}
