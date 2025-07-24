package redis

import (
	"context"
	"encoding/json"

	"github.com/gabapcia/blockwatch/internal/chainstream"

	"github.com/redis/go-redis/v9"
)

type ChainstreamDispatchFailureNotifier = chainstream.DispatchFailureNotifier

// chainstreamDispatchFailureNotifier implements chainstream.DispatchFailureNotifier
// by sending dispatch failure information to a Redis Stream.
type chainstreamDispatchFailureNotifier struct {
	conn   *redis.Client // Redis client connection
	stream string        // Redis Stream name to which failures will be published
}

// AsChainstreamDispatchFailureNotifier returns a chainstreamDispatchFailureNotifier
// that writes block dispatch failures to the specified Redis Stream.
//
// Parameters:
//   - stream: the name of the Redis Stream where the failure messages will be published.
func (c *client) AsChainstreamDispatchFailureNotifier(stream string) chainstream.DispatchFailureNotifier {
	return &chainstreamDispatchFailureNotifier{
		conn:   c.conn,
		stream: stream,
	}
}

// makeBlockDispatchFailureMessage converts a BlockDispatchFailure into a flat map[string]any
// that can be sent as fields in a Redis Stream entry.
func makeBlockDispatchFailureMessage(dispatchFailure chainstream.BlockDispatchFailure) (map[string]any, error) {
	errorList := make([]string, len(dispatchFailure.Errors))
	for i, err := range dispatchFailure.Errors {
		errorList[i] = err.Error()
	}

	errorsData, err := json.Marshal(errorList)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"network": dispatchFailure.Network,
		"height":  dispatchFailure.Height.String(),
		"errors":  string(errorsData),
	}, nil
}

// NotifyDispatchFailure publishes a block dispatch failure event to the configured Redis Stream.
//
// This method implements the chainstream.DispatchFailureNotifier interface.
func (c *chainstreamDispatchFailureNotifier) NotifyDispatchFailure(ctx context.Context, failure chainstream.BlockDispatchFailure) error {
	values, err := makeBlockDispatchFailureMessage(failure)
	if err != nil {
		return err
	}

	return c.conn.XAdd(ctx, &redis.XAddArgs{
		Stream: c.stream,
		ID:     "*",
		Values: values,
	}).Err()
}

// Compile-time check to ensure chainstreamDispatchFailureNotifier implements DispatchFailureNotifier.
var _ chainstream.DispatchFailureNotifier = (*chainstreamDispatchFailureNotifier)(nil)
