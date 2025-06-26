package watcher

import (
	"context"
	"errors"

	"github.com/gabapcia/blockwatch/internal/pkg/resilience/retry"
	"github.com/gabapcia/blockwatch/internal/pkg/types"
)

const (
	errorChannelBufferSize     = 5
	toProcessChannelBufferSize = 10
)

type NetworkBlock struct {
	Network string
	Block
}

type NetworkError struct {
	Network string
	Height  types.Hex
	err     error
}

type Service interface {
	Start(ctx context.Context) error
}

type service struct {
	retry             retry.Retry
	networks          map[string]Blockchain
	walletStorage     WalletStorage
	checkpointStorage CheckpointStorage
}

var _ Service = (*service)(nil)

func (s *service) startNetworksSubscriptions(ctx context.Context, processingCh chan<- NetworkBlock, recoveryCh chan<- NetworkError) error {
	for network, module := range s.networks {
		fromHeight, err := s.checkpointStorage.LoadLatestCheckpoint(ctx, network)
		if err != nil && !errors.Is(err, ErrNoCheckpointFound) {
			return err
		}

		if !fromHeight.IsEmpty() {
			fromHeight = fromHeight.Add(1)
		}

		eventsCh, err := module.Subscribe(ctx, fromHeight)
		if err != nil {
			return err
		}

		go func() {
			defer close(processingCh)
			defer close(recoveryCh)

			for event := range eventsCh {
				if event.Err != nil {
					networkErr := NetworkError{
						Network: network,
						Height:  event.Height,
						err:     err,
					}

					recoveryCh <- networkErr
					continue
				}

				networkBlock := NetworkBlock{
					Network: network,
					Block:   event.Block,
				}

				select {
				case <-ctx.Done():
					return
				case processingCh <- networkBlock:
				}
			}
		}()
	}

	return nil
}

func (s *service) startRetryFailed(ctx context.Context, processingCh chan<- NetworkBlock, recoveryCh <-chan NetworkError, errorCh chan<- NetworkError) error {
	go func() {
		defer close(errorCh)

		for networkErr := range recoveryCh {
			err := s.retry.Execute(ctx, func() error {
				module := s.networks[networkErr.Network]

				block, err := module.FetchBlockByHeight(ctx, networkErr.Height)
				if err != nil {
					return err
				}

				processingCh <- NetworkBlock{
					Network: networkErr.Network,
					Block:   block,
				}
				return nil
			})

			networkErr.err = errors.Join(networkErr.err, err)

			select {
			case <-ctx.Done():
				return
			case errorCh <- networkErr:
			}
		}
	}()

	return nil
}

func (s *service) startErrorHandler(ctx context.Context, errorCh <-chan NetworkError) error {
	return nil
}

func (s *service) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var (
		errorCh      = make(chan NetworkError)
		recoveryCh   = make(chan NetworkError)
		processingCh = make(chan NetworkBlock)
	)

	if err := s.startErrorHandler(ctx, errorCh); err != nil {
		return err
	}

	if err := s.startRetryFailed(ctx, processingCh, recoveryCh, errorCh); err != nil {
		return err
	}

	if err := s.startNetworksSubscriptions(ctx, processingCh, recoveryCh); err != nil {
		return err
	}

	return s.process(ctx, processingCh)
}

func NewService(
	retry retry.Retry,
	walletStorage WalletStorage,
	checkpointStorage CheckpointStorage,
	networks map[string]Blockchain,
) *service {
	return &service{
		retry:             retry,
		networks:          networks,
		walletStorage:     walletStorage,
		checkpointStorage: checkpointStorage,
	}
}
