package txwatcher

import (
	"context"
	"errors"
	"time"

	"github.com/gabapcia/blockwatch/internal/pkg/logger"
)

const defaultMaxProcessingTime = 5 * time.Minute

var ErrServiceAlreadyStarted = errors.New("service already started")

type Service interface {
}

type service struct {
	maxProcessingTime time.Duration
	idempotencyGuard  IdempotencyGuard

	walletStorage       WalletStorage
	transactionNotifier TransactionNotifier
}

var _ Service = (*service)(nil)

func (s *service) NotifyWatchedTransactions(ctx context.Context, block Block) error {
	ok, err := s.idempotencyGuard.ClaimBlockForTxWatch(ctx, block.Network, block.Hash, s.maxProcessingTime)
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
	maxProcessingTime time.Duration
	idempotencyGuard  IdempotencyGuard
}

type Option func(*config)

func New(ws WalletStorage, tn TransactionNotifier, opts ...Option) *service {
	cfg := config{
		maxProcessingTime: defaultMaxProcessingTime,
		idempotencyGuard:  nopIdempotencyGuard{},
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	return &service{
		maxProcessingTime:   cfg.maxProcessingTime,
		idempotencyGuard:    cfg.idempotencyGuard,
		walletStorage:       ws,
		transactionNotifier: tn,
	}
}

func WithMaxProcessingTime(d time.Duration) Option {
	return func(c *config) {
		c.maxProcessingTime = d
	}
}

func WithIdempotencyGuard(g IdempotencyGuard) Option {
	return func(c *config) {
		c.idempotencyGuard = g
	}
}
