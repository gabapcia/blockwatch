package chainwatch

import (
	"cmp"
	"context"
	"errors"
	"sync"

	"github.com/gabapcia/blockwatch/internal/pkg/logger"
	"github.com/gabapcia/blockwatch/internal/pkg/resilience/retry"
)

var ErrServiceAlreadyStarted = errors.New("service already started")

const (
	errorChannelBufferSize      = 5
	recoveryChannelBufferSize   = 5
	processingChannelBufferSize = 10
)

type Service interface {
	Start(ctx context.Context) (<-chan ObservedBlock, error)
	Close(ctx context.Context)
}

type closeFunc func()
type onDispatchFailureFunc func(ctx context.Context, dispatchFailure BlockDispatchFailure)

type service struct {
	m         sync.Mutex
	started   bool
	closeFunc closeFunc

	networks          map[string]Blockchain
	checkpointStorage CheckpointStorage

	retry             retry.Retry
	onDispatchFailure onDispatchFailureFunc
}

var _ Service = (*service)(nil)

func (s *service) Start(ctx context.Context) (<-chan ObservedBlock, error) {
	s.m.Lock()
	defer s.m.Unlock()

	if s.started {
		return nil, ErrServiceAlreadyStarted
	}

	ctx, cancel := context.WithCancel(ctx)

	var (
		recoveryCh   chan BlockDispatchFailure
		errorCh      = make(chan BlockDispatchFailure, errorChannelBufferSize)
		processingCh = make(chan ObservedBlock, processingChannelBufferSize)
	)

	s.closeFunc = func() {
		cancel()
		close(processingCh)
		if recoveryCh != nil {
			close(recoveryCh)
		}
		close(errorCh)
	}

	s.startHandleDispatchFailures(ctx, errorCh)

	if s.retry != nil {
		recoveryCh = make(chan BlockDispatchFailure, recoveryChannelBufferSize)
		s.startRetryFailedBlockFetches(ctx, recoveryCh, processingCh, errorCh)
	}

	errorSubmissionCh := cmp.Or(recoveryCh, errorCh)
	if err := s.launchAllNetworkSubscriptions(ctx, processingCh, errorSubmissionCh); err != nil {
		s.closeFunc()
		return nil, err
	}

	s.started = true
	return processingCh, nil
}

func (s *service) Close(ctx context.Context) {
	s.m.Lock()
	defer s.m.Unlock()

	s.closeFunc()
	s.started = false
}

type config struct {
	retry             retry.Retry
	checkpointStorage CheckpointStorage
	onDispatchFailure onDispatchFailureFunc
}

type Option func(*config)

func New(networks map[string]Blockchain, opts ...Option) *service {
	cfg := config{
		retry:             nil,
		checkpointStorage: nopCheckpoint{},
		onDispatchFailure: defaultOnDispatchFailureFunc,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	return &service{
		networks:          networks,
		checkpointStorage: cfg.checkpointStorage,
		retry:             cfg.retry,
	}
}

func defaultOnDispatchFailureFunc(ctx context.Context, dispatchFailure BlockDispatchFailure) {
	logger.Error(ctx, "block dispatch failure",
		"block.network", dispatchFailure.Network,
		"block.height", dispatchFailure.Height,
		"block.errors", dispatchFailure.Errors,
	)
}

func WithOnDispatchFailureFunc(f onDispatchFailureFunc) Option {
	return func(c *config) {
		c.onDispatchFailure = f
	}
}

func WithRetry(r retry.Retry) Option {
	return func(c *config) {
		c.retry = r
	}
}

func WithCheckpointStorage(cs CheckpointStorage) Option {
	return func(c *config) {
		c.checkpointStorage = cs
	}
}
