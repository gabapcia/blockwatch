package watcher

import (
	"cmp"
	"context"

	"github.com/gabapcia/blockwatch/internal/pkg/resilience/retry"
)

const (
	errorChannelBufferSize      = 5
	recoveryChannelBufferSize   = 5
	processingChannelBufferSize = 10
)

type Service interface {
	Start(ctx context.Context) error
}

type service struct {
	networks          map[string]Blockchain
	checkpointStorage CheckpointStorage

	retry retry.Retry
}

var _ Service = (*service)(nil)

func (s *service) startErrorHandler(ctx context.Context, errorsCh <-chan BlockDispatchFailure) error {
	return nil
}

func (s *service) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var (
		recoveryCh   chan BlockDispatchFailure
		errorCh      = make(chan BlockDispatchFailure, errorChannelBufferSize)
		processingCh = make(chan blockProcessingState, processingChannelBufferSize)
	)
	defer close(errorCh)
	defer close(processingCh)

	if err := s.startErrorHandler(ctx, errorCh); err != nil {
		return err
	}

	if s.retry != nil {
		recoveryCh = make(chan BlockDispatchFailure, recoveryChannelBufferSize)
		defer close(recoveryCh)

		s.startRetryFailedBlockFetches(ctx, recoveryCh, processingCh, errorCh)
	}

	errorSubmissionCh := cmp.Or(recoveryCh, errorCh)
	if err := s.launchAllNetworkSubscriptions(ctx, processingCh, errorSubmissionCh); err != nil {
		return err
	}

	return s.process(ctx, processingCh)
}

func NewService(checkpointStorage CheckpointStorage, networks map[string]Blockchain, retry retry.Retry) *service {
	return &service{
		retry:             retry,
		networks:          networks,
		checkpointStorage: checkpointStorage,
	}
}
