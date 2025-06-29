package chainwatch

import (
	"context"
	"errors"
	"sync"

	"github.com/gabapcia/blockwatch/internal/pkg/logger"
	"github.com/gabapcia/blockwatch/internal/pkg/resilience/retry"
	"github.com/gabapcia/blockwatch/internal/pkg/x/chflow"
)

var ErrServiceAlreadyStarted = errors.New("service already started")

const (
	dispatchFailureChannelBufferSize = 5
	retryFailureChannelBufferSize    = 5
	observedBlockChannelBufferSize   = 10
)

type Service interface {
	Start(ctx context.Context) (<-chan ObservedBlock, error)
	Close()
}

type closeFunc func()
type dispatchFailureHandler func(ctx context.Context, dispatchFailure BlockDispatchFailure)

type service struct {
	mu        sync.Mutex
	isStarted bool
	closeFunc closeFunc

	networks          map[string]Blockchain
	checkpointStorage CheckpointStorage

	retry                  retry.Retry
	dispatchFailureHandler dispatchFailureHandler
}

var _ Service = (*service)(nil)

func (s *service) Start(ctx context.Context) (<-chan ObservedBlock, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isStarted {
		return nil, ErrServiceAlreadyStarted
	}

	ctx, cancel := context.WithCancel(ctx)

	var (
		retryFailureCh    chan BlockDispatchFailure
		dispatchFailureCh = make(chan BlockDispatchFailure, dispatchFailureChannelBufferSize)
		observedBlockCh   = make(chan ObservedBlock, observedBlockChannelBufferSize)
	)

	s.closeFunc = func() {
		cancel()
		close(observedBlockCh)
		if retryFailureCh != nil {
			close(retryFailureCh)
		}
		close(dispatchFailureCh)
	}

	s.startHandleDispatchFailures(ctx, dispatchFailureCh)

	if s.retry != nil {
		retryFailureCh = make(chan BlockDispatchFailure, retryFailureChannelBufferSize)
		s.startRetryFailedBlockFetches(ctx, retryFailureCh, observedBlockCh, dispatchFailureCh)
	}

	errorSubmissionCh := chflow.FirstNonNil(retryFailureCh, dispatchFailureCh)
	if err := s.launchAllNetworkSubscriptions(ctx, observedBlockCh, errorSubmissionCh); err != nil {
		s.closeFunc()
		return nil, err
	}

	s.isStarted = true
	return observedBlockCh, nil
}

func (s *service) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closeFunc != nil {
		s.closeFunc()
	}
	s.isStarted = false
	s.closeFunc = nil
}

type config struct {
	retry                  retry.Retry
	checkpointStorage      CheckpointStorage
	dispatchFailureHandler dispatchFailureHandler
}

type Option func(*config)

func New(networks map[string]Blockchain, opts ...Option) *service {
	cfg := config{
		retry:                  nil,
		checkpointStorage:      nopCheckpoint{},
		dispatchFailureHandler: defaultOnDispatchFailure,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	return &service{
		networks:               networks,
		checkpointStorage:      cfg.checkpointStorage,
		retry:                  cfg.retry,
		dispatchFailureHandler: cfg.dispatchFailureHandler,
	}
}

func defaultOnDispatchFailure(ctx context.Context, dispatchFailure BlockDispatchFailure) {
	logger.Error(ctx, "block dispatch failure",
		"block.network", dispatchFailure.Network,
		"block.height", dispatchFailure.Height,
		"block.errors", dispatchFailure.Errors,
	)
}

func WithDispatchFailureHandler(f dispatchFailureHandler) Option {
	return func(c *config) {
		c.dispatchFailureHandler = f
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
