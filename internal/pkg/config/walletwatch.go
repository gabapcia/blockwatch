package config

import (
	"time"

	"github.com/gabapcia/blockwatch/internal/pkg/config/messaging"
	"github.com/gabapcia/blockwatch/internal/pkg/config/storage"
)

// WalletWatch defines configuration options for the wallet transaction watcher use case.
type WalletWatch struct {
	// MaxProcessingTime sets the maximum allowed time for processing a single block.
	// Default: 5m
	MaxProcessingTime time.Duration `default:"5m" split_words:"true"`

	// IdempotencyGuard selects the storage backend used to prevent duplicate block processing.
	IdempotencyGuard *storage.Picker `validate:"omitempty" split_words:"true"`

	// WalletStorage selects the storage backend used to retrieve registered wallets.
	WalletStorage storage.Picker `validate:"required" split_words:"true"`

	// TransactionNotifier selects the messaging backend used to publish detected transactions.
	TransactionNotifier messaging.Picker `validate:"required" split_words:"true"`
}
