package redis

import (
	"context"

	redis "github.com/redis/go-redis/v9"
)

type client struct {
	conn *redis.Client
}

func (c *client) Close() error {
	return c.conn.Close()
}

func NewClient(ctx context.Context, addr, username, password string, db int) (*client, error) {
	conn := redis.NewClient(&redis.Options{
		Addr:     addr,
		Username: username,
		Password: password,
		DB:       db,
	})

	if err := conn.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &client{
		conn: conn,
	}, nil
}
