package watcher

import (
	"cmp"
	"context"

	"github.com/gabapcia/blockwatch/internal/pkg/resilience/retry"
	"github.com/gabapcia/blockwatch/internal/pkg/types"
)

const (
	errorChannelBufferSize      = 5
	recoveryChannelBufferSize   = 5
	processingChannelBufferSize = 10
)

type NetworkBlock struct {
	Network string
	Block
}

type NetworkError struct {
	Network string
	Height  types.Hex
	Err     error
}

type Service interface {
	Start(ctx context.Context) error
}

type service struct {
	networks          map[string]Blockchain
	walletStorage     WalletStorage
	checkpointStorage CheckpointStorage

	retry retry.Retry
}

var _ Service = (*service)(nil)

func (s *service) startErrorHandler(ctx context.Context, errorCh <-chan NetworkError) error {
	return nil
}

func (s *service) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var (
		recoveryCh   chan NetworkError
		errorCh      = make(chan NetworkError, errorChannelBufferSize)
		processingCh = make(chan NetworkBlock, processingChannelBufferSize)
	)
	defer close(errorCh)
	defer close(processingCh)

	if err := s.startErrorHandler(ctx, errorCh); err != nil {
		return err
	}

	if s.retry != nil {
		recoveryCh = make(chan NetworkError, recoveryChannelBufferSize)
		defer close(recoveryCh)

		s.startRetryFailedBlockFetches(ctx, recoveryCh, processingCh, errorCh)
	}

	errorSubmissionCh := cmp.Or(recoveryCh, errorCh)
	if err := s.startSubscriptions(ctx, processingCh, errorSubmissionCh); err != nil {
		return err
	}

	return s.process(ctx, processingCh)
}

func NewService(walletStorage WalletStorage, checkpointStorage CheckpointStorage, networks map[string]Blockchain, retry retry.Retry) *service {
	return &service{
		retry:             retry,
		networks:          networks,
		walletStorage:     walletStorage,
		checkpointStorage: checkpointStorage,
	}
}
