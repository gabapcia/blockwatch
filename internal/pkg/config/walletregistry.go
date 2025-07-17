package config

import "github.com/gabapcia/blockwatch/internal/pkg/config/storage"

type WalletRegistry struct {
	WalletStorage storage.Picker `validate:"required" split_words:"true"`
}
