package rabbitmq

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gabapcia/blockwatch/internal/walletwatch"

	"github.com/google/uuid"
	"github.com/rabbitmq/amqp091-go"
)

type walletwatchTransactionNotifier struct {
	channel    *amqp091.Channel
	exchange   string
	routingKey string
}

func (c *client) AsWalletwatchTransactionNotifier(exchange, routingKey string) *walletwatchTransactionNotifier {
	return &walletwatchTransactionNotifier{
		channel:    c.channel,
		exchange:   exchange,
		routingKey: routingKey,
	}
}

type (
	walletwatchTransactionMessage struct {
		Hash string `json:"hash"`
		From string `json:"from"`
		To   string `json:"to"`
	}

	walletwatchNotifyTransactionsMessage struct {
		Network       string                          `json:"network"`
		WalletAddress string                          `json:"wallet_address"`
		Transactions  []walletwatchTransactionMessage `json:"transactions"`
	}
)

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

var _ walletwatch.TransactionNotifier = new(walletwatchTransactionNotifier)
