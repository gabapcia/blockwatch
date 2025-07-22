package postgresql

import (
	"context"

	"github.com/gabapcia/blockwatch/internal/infra/storage/postgresql/querier/monitoredwallets"
	"github.com/gabapcia/blockwatch/internal/infra/storage/postgresql/querier/walletwatchidempotency"

	"github.com/jackc/pgx/v5/pgxpool"
)

type client struct {
	pool *pgxpool.Pool

	monitoredWallets       *monitoredwallets.Queries
	walletwatchIdempotency *walletwatchidempotency.Queries
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
		pool:                   pool,
		monitoredWallets:       monitoredwallets.New(pool),
		walletwatchIdempotency: walletwatchidempotency.New(pool),
	}, nil
}
