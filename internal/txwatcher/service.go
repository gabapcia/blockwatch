package txwatcher

import (
	"context"
	"errors"
	"time"

	"github.com/gabapcia/blockwatch/internal/pkg/logger"
)

var ErrServiceAlreadyStarted = errors.New("service already started")

type Service interface {
}

type service struct {
	idempotencyGuard    IdempotencyGuard
	walletStorage       WalletStorage
	transactionNotifier TransactionNotifier
}

var _ Service = (*service)(nil)

func (s *service) NotifyWatchedTransactions(ctx context.Context, block Block) error {
	ok, err := s.idempotencyGuard.ClaimBlockForTxWatch(ctx, block.Network, block.Hash, 5*time.Minute)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	if err := s.notifyWatchedWalletTransactions(ctx, block.Network, block.Transactions); err != nil {
		return err
	}

	if err := s.idempotencyGuard.MarkBlockTxWatchComplete(ctx, block.Network, block.Hash); err != nil {
		logger.Error(ctx, "error marking block tx watch as complete",
			"block.network", block.Network,
			"block.hash", block.Hash,
			"block.height", block.Height,
			"error", err,
		)
	}

	return nil
}

type config struct {
	idempotencyGuard IdempotencyGuard
}

type Option func(*config)

func New(ws WalletStorage, tn TransactionNotifier, opts ...Option) *service {
	cfg := config{
		idempotencyGuard: nopIdempotencyGuard{},
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	return &service{
		idempotencyGuard:    cfg.idempotencyGuard,
		walletStorage:       ws,
		transactionNotifier: tn,
	}
}

func WithIdempotencyGuard(g IdempotencyGuard) Option {
	return func(c *config) {
		c.idempotencyGuard = g
	}
}
