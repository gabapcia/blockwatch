// Package blockproc coordinates the block-level processing pipeline,
// combining multiple downstream workflows (e.g., wallet activity monitoring)
// into a unified orchestration layer.
package blockproc

import (
	"context"
	"errors"
	"sync"

	"github.com/gabapcia/blockwatch/internal/chainstream"
	"github.com/gabapcia/blockwatch/internal/walletwatch"
)

// ErrServiceAlreadyStarted is returned if Start is called more than once.
//
// The service must be started only once per lifecycle.
var ErrServiceAlreadyStarted = errors.New("service already started")

// Service defines the blockproc lifecycle and coordination entrypoint.
//
// It orchestrates downstream components such as chainstream and walletwatch
// to handle observed blocks and route them through appropriate processing flows.
type Service interface {
	// Start begins the block processing pipeline by launching chainstream
	// and wiring all downstream workflows (e.g., wallet activity handling).
	//
	// Returns ErrServiceAlreadyStarted if Start is called more than once.
	// Call Close to shut down all background routines.
	Start(ctx context.Context) error

	// Close shuts down the blockproc service and cancels all active routines.
	// It is safe to call Close even if the service was never started.
	Close()
}

// closeFunc defines a cleanup routine to stop background goroutines and dependencies.
type closeFunc func()

// service is the internal implementation of the blockproc Service interface.
//
// It wires together chainstream (for observing new blocks) and walletwatch
// (for detecting and notifying wallet-related activity).
type service struct {
	mu        sync.Mutex // protects lifecycle state
	isStarted bool       // ensures Start is called only once
	closeFunc closeFunc  // cancels context and cleans up dependencies

	chainstream chainstream.Service // source of blockchain blocks
	walletwatch walletwatch.Service // wallet activity workflow
}

// Compile-time check to ensure *service implements the Service interface.
var _ Service = new(service)

// Start initializes the block processing service.
//
// It starts the chainstream subscription, receives observed blocks,
// and wires them into all wallet-related processing logic.
//
// Returns an error if startup fails or the service was already started.
func (s *service) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isStarted {
		return ErrServiceAlreadyStarted
	}

	ctx, cancel := context.WithCancel(ctx)

	blocksCh, err := s.chainstream.Start(ctx)
	if err != nil {
		cancel()
		return err
	}

	s.startHandleWalletActivity(ctx, blocksCh)

	s.closeFunc = func() {
		cancel()
		s.chainstream.Close()
	}
	s.isStarted = true
	return nil
}

// Close shuts down all processing routines and dependencies.
//
// It cancels the chainstream subscription and stops internal goroutines.
func (s *service) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closeFunc != nil {
		s.closeFunc()
	}

	s.closeFunc = nil
	s.isStarted = false
}

// New creates a new instance of the blockproc service,
// wiring the chainstream source with walletwatch processing.
func New(s chainstream.Service, w walletwatch.Service) *service {
	return &service{
		chainstream: s,
		walletwatch: w,
	}
}
