package rabbitmq

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gabapcia/blockwatch/internal/chainstream"
	"github.com/gabapcia/blockwatch/internal/pkg/x/types"

	"github.com/google/uuid"
	"github.com/rabbitmq/amqp091-go"
)

// chainstreamDispatchFailureNotifier implements chainstream.DispatchFailureNotifier
// and publishes unrecoverable block dispatch failures to a RabbitMQ exchange.
type chainstreamDispatchFailureNotifier struct {
	channel    *amqp091.Channel // AMQP channel used to publish messages.
	exchange   string           // Exchange to which failure messages will be published.
	routingKey string           // Routing key used for publishing failure messages.
}

// AsChainstreamDispatchFailureNotifier returns an implementation of
// chainstream.DispatchFailureNotifier that emits messages to RabbitMQ.
//
// This allows a chainstream service to route unrecoverable dispatch failures
// to external consumers (e.g., monitoring systems) via RabbitMQ.
func (c *Client) AsChainstreamDispatchFailureNotifier(exchange, routingKey string) *chainstreamDispatchFailureNotifier {
	return &chainstreamDispatchFailureNotifier{
		channel:    c.channel,
		exchange:   exchange,
		routingKey: routingKey,
	}
}

// chainstreamBlockDispatchFailureMessage represents the message structure
// used to encode block dispatch failures in JSON format.
//
// Errors are encoded as a list of error messages (string) for compatibility
// with external systems and easier JSON serialization.
type chainstreamBlockDispatchFailureMessage struct {
	Network string    `json:"network"` // Blockchain network where the failure occurred.
	Height  types.Hex `json:"height"`  // Block height associated with the failure.
	Errors  []string  `json:"errors"`  // List of stringified error messages.
}

// makeBlockDispatchFailureMessage converts a chainstream.BlockDispatchFailure
// into a serializable message payload with stringified errors.
func makeBlockDispatchFailureMessage(failure chainstream.BlockDispatchFailure) chainstreamBlockDispatchFailureMessage {
	errorList := make([]string, len(failure.Errors))
	for i, err := range failure.Errors {
		errorList[i] = err.Error()
	}

	return chainstreamBlockDispatchFailureMessage{
		Network: failure.Network,
		Height:  failure.Height,
		Errors:  errorList,
	}
}

// NotifyDispatchFailure publishes a JSON-formatted failure message to RabbitMQ.
//
// It marshals the failure into a message, sets appropriate metadata (e.g., timestamp,
// message ID), and uses the configured exchange and routing key to dispatch it.
//
// If serialization or publishing fails, an error is returned.
func (c *chainstreamDispatchFailureNotifier) NotifyDispatchFailure(ctx context.Context, failure chainstream.BlockDispatchFailure) error {
	msg, err := json.Marshal(makeBlockDispatchFailureMessage(failure))
	if err != nil {
		return err
	}

	return c.channel.PublishWithContext(ctx,
		c.exchange,
		c.routingKey,
		false, false,
		amqp091.Publishing{
			Timestamp:   time.Now(),
			MessageId:   uuid.Must(uuid.NewV7()).String(),
			ContentType: "application/json",
			Body:        msg,
		},
	)
}

// Compile-time assertion to ensure the notifier implements the interface.
var _ chainstream.DispatchFailureNotifier = new(chainstreamDispatchFailureNotifier)
