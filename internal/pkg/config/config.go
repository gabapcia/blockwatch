package config

import (
	"context"

	"github.com/gabapcia/blockwatch/internal/pkg/config/messaging"
	"github.com/gabapcia/blockwatch/internal/pkg/config/pkg"
	"github.com/gabapcia/blockwatch/internal/pkg/config/storage"

	"github.com/kelseyhightower/envconfig"
)

type Engines struct {
	Storage   storage.Engines
	Messaging messaging.Engines
}

type Config struct {
	Log     pkg.Logger
	Engines Engines

	Walletregistry WalletRegistry
	Walletwatch    WalletWatch
	Chainstream    ChainStream
}

func Load(ctx context.Context) (Config, error) {
	var config Config
	if err := envconfig.Process("", &config); err != nil {
		return Config{}, err
	}

	return config, nil
}
