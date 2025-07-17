package config

import (
	"time"

	"github.com/gabapcia/blockwatch/internal/pkg/config/messaging"
	"github.com/gabapcia/blockwatch/internal/pkg/config/storage"
)

type WalletWatch struct {
	MaxProcessingTime   time.Duration    `default:"5m" split_words:"true"`
	IdempotencyGuard    *storage.Picker  `validate:"omitempty" split_words:"true"`
	WalletStorage       storage.Picker   `validate:"required" split_words:"true"`
	TransactionNotifier messaging.Picker `validate:"required" split_words:"true"`
}
