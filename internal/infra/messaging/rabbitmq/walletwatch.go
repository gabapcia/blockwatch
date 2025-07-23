package rabbitmq

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gabapcia/blockwatch/internal/walletwatch"

	"github.com/google/uuid"
	"github.com/rabbitmq/amqp091-go"
)

// walletwatchTransactionNotifier implements the TransactionNotifier interface
// and publishes wallet transactions to a RabbitMQ exchange.
type walletwatchTransactionNotifier struct {
	channel    *amqp091.Channel // channel is the AMQP channel used for publishing messages.
	exchange   string           // exchange is the name of the RabbitMQ exchange where messages will be published.
	routingKey string           // routingKey is the routing key used when publishing to the exchange.
}

// AsWalletwatchTransactionNotifier returns a new instance of walletwatchTransactionNotifier,
// which is responsible for publishing walletwatch transactions using the given exchange and routing key.
func (c *client) AsWalletwatchTransactionNotifier(exchange, routingKey string) walletwatch.TransactionNotifier {
	return &walletwatchTransactionNotifier{
		channel:    c.channel,
		exchange:   exchange,
		routingKey: routingKey,
	}
}

type (
	// walletwatchTransactionMessage represents a simplified transaction structure for publishing.
	walletwatchTransactionMessage struct {
		Hash string `json:"hash"`
		From string `json:"from"`
		To   string `json:"to"`
	}

	// walletwatchNotifyTransactionsMessage defines the message schema sent to RabbitMQ,
	// including the network, wallet address, and a list of transactions.
	walletwatchNotifyTransactionsMessage struct {
		Network       string                          `json:"network"`
		WalletAddress string                          `json:"wallet_address"`
		Transactions  []walletwatchTransactionMessage `json:"transactions"`
	}
)

// makeNotifyTransactionsMessage converts a list of walletwatch.Transaction into the
// message structure expected by the notifier.
func makeNotifyTransactionsMessage(network, wallet string, txs []walletwatch.Transaction) walletwatchNotifyTransactionsMessage {
	transactions := make([]walletwatchTransactionMessage, len(txs))
	for i, tx := range txs {
		transactions[i] = walletwatchTransactionMessage(tx)
	}

	return walletwatchNotifyTransactionsMessage{
		Network:       network,
		WalletAddress: wallet,
		Transactions:  transactions,
	}
}

// NotifyTransactions publishes a wallet transaction event to RabbitMQ.
//
// It serializes the transaction list into JSON and publishes it using the configured
// exchange and routing key. Each message includes metadata such as timestamp and message ID.
//
// Parameters:
//   - ctx: context for timeout and cancellation.
//   - network: the blockchain network name.
//   - wallet: the wallet address involved.
//   - txs: list of transactions associated with the wallet.
//
// Returns:
//   - An error if message publishing or serialization fails.
func (c *walletwatchTransactionNotifier) NotifyTransactions(ctx context.Context, network, wallet string, txs []walletwatch.Transaction) error {
	msg, err := json.Marshal(makeNotifyTransactionsMessage(network, wallet, txs))
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

// Compile-time assertion to ensure walletwatchTransactionNotifier implements TransactionNotifier.
var _ walletwatch.TransactionNotifier = new(walletwatchTransactionNotifier)
