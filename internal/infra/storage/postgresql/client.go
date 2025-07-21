package postgresql

import (
	"context"

	"github.com/gabapcia/blockwatch/internal/infra/storage/postgresql/querier"

	"github.com/jackc/pgx/v5/pgxpool"
)

type client struct {
	pool    *pgxpool.Pool
	queries *querier.Queries
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
		queries: querier.New(pool),
	}, nil
}
