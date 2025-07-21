package redis

import (
	"context"

	"github.com/gabapcia/blockwatch/internal/chainstream"

	"github.com/redis/go-redis/v9"
)

type chainstreamDispatchFailureNotifier struct {
	conn   *redis.Client
	stream string
}

func (c *client) AsChainstreamDispatchFailureNotifier(stream string) *chainstreamDispatchFailureNotifier {
	return &chainstreamDispatchFailureNotifier{
		conn:   c.conn,
		stream: stream,
	}
}

func makeBlockDispatchFailureMessage(dispatchFailure chainstream.BlockDispatchFailure) map[string]any {
	errorList := make([]string, len(dispatchFailure.Errors))
	for i, err := range dispatchFailure.Errors {
		errorList[i] = err.Error()
	}

	return map[string]any{
		"network": dispatchFailure.Network,
		"height":  dispatchFailure.Height,
		"errors":  errorList,
	}
}

func (c *chainstreamDispatchFailureNotifier) NotifyDispatchFailure(ctx context.Context, failure chainstream.BlockDispatchFailure) error {
	return c.conn.XAdd(ctx, &redis.XAddArgs{
		Stream: c.stream,
		ID:     "*",
		Values: makeBlockDispatchFailureMessage(failure),
	}).Err()
}

var _ chainstream.DispatchFailureNotifier = new(chainstreamDispatchFailureNotifier)
