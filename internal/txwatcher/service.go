package txwatcher

import (
	"errors"
	"time"
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
