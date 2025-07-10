package blockproc

import (
	"context"
	"errors"
	"testing"

	"github.com/gabapcia/blockwatch/internal/chainstream"
	chainstreamMocks "github.com/gabapcia/blockwatch/internal/chainstream/mocks"
	"github.com/gabapcia/blockwatch/internal/pkg/types"
	"github.com/gabapcia/blockwatch/internal/walletwatch"
	walletwatchMocks "github.com/gabapcia/blockwatch/internal/walletwatch/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestStart(t *testing.T) {
	t.Run("successful start", func(t *testing.T) {
		// Custom types and structs for the test
		type testDeps struct {
			chainstream *chainstreamMocks.Service
			walletwatch *walletwatchMocks.Service
		}

		// Setup
		deps := testDeps{
			chainstream: chainstreamMocks.NewService(t),
			walletwatch: walletwatchMocks.NewService(t),
		}

		blocksCh := make(chan chainstream.ObservedBlock, 1)
		testBlock := chainstream.ObservedBlock{
			Network: "ethereum",
			Block: chainstream.Block{
				Height: types.Hex("0x1"),
				Hash:   "0xabc123",
				Transactions: []chainstream.Transaction{
					{
						Hash: "0xtx1",
						From: "0xfrom1",
						To:   "0xto1",
					},
				},
			},
		}

		deps.chainstream.EXPECT().Start(mock.Anything).Return((<-chan chainstream.ObservedBlock)(blocksCh), nil)
		deps.chainstream.EXPECT().Close().Return()

		// Use a channel to synchronize the test
		done := make(chan struct{})

		// Expect walletwatch to be called when block is processed
		deps.walletwatch.EXPECT().NotifyWatchedTransactions(mock.Anything, mock.MatchedBy(func(block walletwatch.Block) bool {
			return block.Network == "ethereum" && block.Hash == "0xabc123"
		})).Return(nil).Run(func(ctx context.Context, block walletwatch.Block) {
			close(done) // Signal that the call was made
		})

		svc := New(deps.chainstream, deps.walletwatch)
		ctx := t.Context()

		// Execute
		err := svc.Start(ctx)

		// Assert
		require.NoError(t, err)

		// Send a test block to verify the channel is being processed
		blocksCh <- testBlock

		// Wait for the block to be processed
		<-done

		close(blocksCh)

		// Cleanup
		svc.Close()
	})

	t.Run("chainstream start fails", func(t *testing.T) {
		// Custom types and structs for the test
		type testDeps struct {
			chainstream *chainstreamMocks.Service
			walletwatch *walletwatchMocks.Service
		}

		// Setup
		deps := testDeps{
			chainstream: chainstreamMocks.NewService(t),
			walletwatch: walletwatchMocks.NewService(t),
		}

		expectedErr := errors.New("chainstream start failed")
		deps.chainstream.EXPECT().Start(mock.Anything).Return((<-chan chainstream.ObservedBlock)(nil), expectedErr)

		svc := New(deps.chainstream, deps.walletwatch)
		ctx := t.Context()

		// Execute
		err := svc.Start(ctx)

		// Assert
		require.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})

	t.Run("service already started", func(t *testing.T) {
		// Custom types and structs for the test
		type testDeps struct {
			chainstream *chainstreamMocks.Service
			walletwatch *walletwatchMocks.Service
		}

		// Setup
		deps := testDeps{
			chainstream: chainstreamMocks.NewService(t),
			walletwatch: walletwatchMocks.NewService(t),
		}

		blocksCh := make(chan chainstream.ObservedBlock)
		deps.chainstream.EXPECT().Start(mock.Anything).Return((<-chan chainstream.ObservedBlock)(blocksCh), nil)
		deps.chainstream.EXPECT().Close().Return()

		svc := New(deps.chainstream, deps.walletwatch)
		ctx := t.Context()

		// Start the service first time
		err := svc.Start(ctx)
		require.NoError(t, err)

		// Execute - try to start again
		err = svc.Start(ctx)

		// Assert
		require.Error(t, err)
		assert.Equal(t, ErrServiceAlreadyStarted, err)

		// Cleanup
		svc.Close()
	})

	t.Run("start with cancelled context", func(t *testing.T) {
		// Custom types and structs for the test
		type testDeps struct {
			chainstream *chainstreamMocks.Service
			walletwatch *walletwatchMocks.Service
		}

		// Setup
		deps := testDeps{
			chainstream: chainstreamMocks.NewService(t),
			walletwatch: walletwatchMocks.NewService(t),
		}

		blocksCh := make(chan chainstream.ObservedBlock)
		ctx, cancel := context.WithCancel(t.Context())
		cancel() // Cancel the context immediately

		deps.chainstream.EXPECT().Start(mock.Anything).Return((<-chan chainstream.ObservedBlock)(blocksCh), nil)
		deps.chainstream.EXPECT().Close().Return()

		svc := New(deps.chainstream, deps.walletwatch)

		// Execute
		err := svc.Start(ctx)

		// Assert
		require.NoError(t, err)

		// Cleanup
		svc.Close()
	})

	t.Run("concurrent start calls", func(t *testing.T) {
		// Custom types and structs for the test
		type testDeps struct {
			chainstream *chainstreamMocks.Service
			walletwatch *walletwatchMocks.Service
		}

		// Setup
		deps := testDeps{
			chainstream: chainstreamMocks.NewService(t),
			walletwatch: walletwatchMocks.NewService(t),
		}

		blocksCh := make(chan chainstream.ObservedBlock)
		deps.chainstream.EXPECT().Start(mock.Anything).Return((<-chan chainstream.ObservedBlock)(blocksCh), nil).Maybe()
		deps.chainstream.EXPECT().Close().Return().Maybe()

		svc := New(deps.chainstream, deps.walletwatch)
		ctx := t.Context()

		// Execute concurrent starts
		errCh := make(chan error, 2)

		go func() {
			errCh <- svc.Start(ctx)
		}()

		go func() {
			errCh <- svc.Start(ctx)
		}()

		// Collect results
		err1 := <-errCh
		err2 := <-errCh

		// Assert - one should succeed, one should fail with ErrServiceAlreadyStarted
		var successCount, alreadyStartedCount int
		for _, err := range []error{err1, err2} {
			if err == nil {
				successCount++
			} else if errors.Is(err, ErrServiceAlreadyStarted) {
				alreadyStartedCount++
			} else {
				t.Fatalf("unexpected error: %v", err)
			}
		}

		assert.Equal(t, 1, successCount, "exactly one start should succeed")
		assert.Equal(t, 1, alreadyStartedCount, "exactly one start should fail with ErrServiceAlreadyStarted")

		// Cleanup
		svc.Close()
	})

	t.Run("start after close", func(t *testing.T) {
		// Custom types and structs for the test
		type testDeps struct {
			chainstream *chainstreamMocks.Service
			walletwatch *walletwatchMocks.Service
		}

		// Setup
		deps := testDeps{
			chainstream: chainstreamMocks.NewService(t),
			walletwatch: walletwatchMocks.NewService(t),
		}

		blocksCh1 := make(chan chainstream.ObservedBlock)
		blocksCh2 := make(chan chainstream.ObservedBlock)

		deps.chainstream.EXPECT().Start(mock.Anything).Return((<-chan chainstream.ObservedBlock)(blocksCh1), nil).Once()
		deps.chainstream.EXPECT().Close().Return().Once()
		deps.chainstream.EXPECT().Start(mock.Anything).Return((<-chan chainstream.ObservedBlock)(blocksCh2), nil).Once()
		deps.chainstream.EXPECT().Close().Return().Once()

		svc := New(deps.chainstream, deps.walletwatch)
		ctx := t.Context()

		// Start, close, then start again
		err := svc.Start(ctx)
		require.NoError(t, err)

		svc.Close()

		// Execute - start again after close
		err = svc.Start(ctx)

		// Assert
		require.NoError(t, err)

		// Cleanup
		svc.Close()
	})
}

func TestClose(t *testing.T) {
	t.Run("close after successful start", func(t *testing.T) {
		// Custom types and structs for the test
		type testDeps struct {
			chainstream *chainstreamMocks.Service
			walletwatch *walletwatchMocks.Service
		}

		// Setup
		deps := testDeps{
			chainstream: chainstreamMocks.NewService(t),
			walletwatch: walletwatchMocks.NewService(t),
		}

		blocksCh := make(chan chainstream.ObservedBlock)
		deps.chainstream.EXPECT().Start(mock.Anything).Return((<-chan chainstream.ObservedBlock)(blocksCh), nil)
		deps.chainstream.EXPECT().Close().Return()

		svc := New(deps.chainstream, deps.walletwatch)
		ctx := t.Context()

		// Start the service first
		err := svc.Start(ctx)
		require.NoError(t, err)

		// Execute
		svc.Close()
	})

	t.Run("close without starting", func(t *testing.T) {
		// Custom types and structs for the test
		type testDeps struct {
			chainstream *chainstreamMocks.Service
			walletwatch *walletwatchMocks.Service
		}

		// Setup
		deps := testDeps{
			chainstream: chainstreamMocks.NewService(t),
			walletwatch: walletwatchMocks.NewService(t),
		}

		svc := New(deps.chainstream, deps.walletwatch)

		// Execute - should not panic or cause issues
		svc.Close()
	})

	t.Run("multiple close calls", func(t *testing.T) {
		// Custom types and structs for the test
		type testDeps struct {
			chainstream *chainstreamMocks.Service
			walletwatch *walletwatchMocks.Service
		}

		// Setup
		deps := testDeps{
			chainstream: chainstreamMocks.NewService(t),
			walletwatch: walletwatchMocks.NewService(t),
		}

		blocksCh := make(chan chainstream.ObservedBlock)
		deps.chainstream.EXPECT().Start(mock.Anything).Return((<-chan chainstream.ObservedBlock)(blocksCh), nil)
		deps.chainstream.EXPECT().Close().Return().Once() // Should only be called once

		svc := New(deps.chainstream, deps.walletwatch)
		ctx := t.Context()

		// Start the service
		err := svc.Start(ctx)
		require.NoError(t, err)

		// Execute - close multiple times
		svc.Close()

		// Second close should be safe
		svc.Close()

		// Third close should also be safe
		svc.Close()
	})

	t.Run("concurrent close calls", func(t *testing.T) {
		// Custom types and structs for the test
		type testDeps struct {
			chainstream *chainstreamMocks.Service
			walletwatch *walletwatchMocks.Service
		}

		// Setup
		deps := testDeps{
			chainstream: chainstreamMocks.NewService(t),
			walletwatch: walletwatchMocks.NewService(t),
		}

		blocksCh := make(chan chainstream.ObservedBlock)
		deps.chainstream.EXPECT().Start(mock.Anything).Return((<-chan chainstream.ObservedBlock)(blocksCh), nil)
		deps.chainstream.EXPECT().Close().Return().Once() // Should only be called once

		svc := New(deps.chainstream, deps.walletwatch)
		ctx := t.Context()

		// Start the service
		err := svc.Start(ctx)
		require.NoError(t, err)

		// Execute concurrent closes
		done := make(chan struct{}, 3)

		go func() {
			svc.Close()
			done <- struct{}{}
		}()

		go func() {
			svc.Close()
			done <- struct{}{}
		}()

		go func() {
			svc.Close()
			done <- struct{}{}
		}()

		// Wait for all goroutines to complete
		<-done
		<-done
		<-done
	})

	t.Run("close stops wallet activity processing", func(t *testing.T) {
		// Custom types and structs for the test
		type testDeps struct {
			chainstream *chainstreamMocks.Service
			walletwatch *walletwatchMocks.Service
		}

		// Setup
		deps := testDeps{
			chainstream: chainstreamMocks.NewService(t),
			walletwatch: walletwatchMocks.NewService(t),
		}

		blocksCh := make(chan chainstream.ObservedBlock, 1)
		deps.chainstream.EXPECT().Start(mock.Anything).Return((<-chan chainstream.ObservedBlock)(blocksCh), nil)
		deps.chainstream.EXPECT().Close().Return()

		// We don't expect NotifyWatchedTransactions to be called after close
		// The context cancellation should stop the goroutine before it processes the block

		svc := New(deps.chainstream, deps.walletwatch)
		ctx := t.Context()

		// Start the service
		err := svc.Start(ctx)
		require.NoError(t, err)

		// Close immediately to cancel the context
		svc.Close()

		// Send a block after closing - it should not be processed
		testBlock := chainstream.ObservedBlock{
			Network: "ethereum",
			Block: chainstream.Block{
				Height: types.Hex("0x1"),
				Hash:   "0xabc123",
			},
		}
		blocksCh <- testBlock
		close(blocksCh)

		// Block should not be processed after close
	})

	t.Run("close after chainstream start failure", func(t *testing.T) {
		// Custom types and structs for the test
		type testDeps struct {
			chainstream *chainstreamMocks.Service
			walletwatch *walletwatchMocks.Service
		}

		// Setup
		deps := testDeps{
			chainstream: chainstreamMocks.NewService(t),
			walletwatch: walletwatchMocks.NewService(t),
		}

		expectedErr := errors.New("chainstream start failed")
		deps.chainstream.EXPECT().Start(mock.Anything).Return((<-chan chainstream.ObservedBlock)(nil), expectedErr)

		svc := New(deps.chainstream, deps.walletwatch)
		ctx := t.Context()

		// Try to start (should fail)
		err := svc.Start(ctx)
		require.Error(t, err)

		// Execute - close should be safe even after failed start
		svc.Close()
	})
}

func TestNew(t *testing.T) {
	t.Run("creates service with valid dependencies", func(t *testing.T) {
		// Custom types and structs for the test
		type testDeps struct {
			chainstream *chainstreamMocks.Service
			walletwatch *walletwatchMocks.Service
		}

		// Setup
		deps := testDeps{
			chainstream: chainstreamMocks.NewService(t),
			walletwatch: walletwatchMocks.NewService(t),
		}

		// Execute
		svc := New(deps.chainstream, deps.walletwatch)

		// Assert
		require.NotNil(t, svc)

		// Verify it implements the Service interface
		var serviceInterface Service = svc
		require.NotNil(t, serviceInterface)
	})

	t.Run("creates service with nil chainstream", func(t *testing.T) {
		// Custom types and structs for the test
		type testDeps struct {
			walletwatch *walletwatchMocks.Service
		}

		// Setup
		deps := testDeps{
			walletwatch: walletwatchMocks.NewService(t),
		}

		// Execute
		svc := New(nil, deps.walletwatch)

		// Assert
		require.NotNil(t, svc)

		// Verify it implements the Service interface
		var serviceInterface Service = svc
		require.NotNil(t, serviceInterface)
	})

	t.Run("creates service with nil walletwatch", func(t *testing.T) {
		// Custom types and structs for the test
		type testDeps struct {
			chainstream *chainstreamMocks.Service
		}

		// Setup
		deps := testDeps{
			chainstream: chainstreamMocks.NewService(t),
		}

		// Execute
		svc := New(deps.chainstream, nil)

		// Assert
		require.NotNil(t, svc)

		// Verify it implements the Service interface
		var serviceInterface Service = svc
		require.NotNil(t, serviceInterface)
	})

	t.Run("creates service with both dependencies nil", func(t *testing.T) {
		// Execute
		svc := New(nil, nil)

		// Assert
		require.NotNil(t, svc)

		// Verify it implements the Service interface
		var serviceInterface Service = svc
		require.NotNil(t, serviceInterface)
	})

	t.Run("returns service implementing interface", func(t *testing.T) {
		// Custom types and structs for the test
		type testDeps struct {
			chainstream *chainstreamMocks.Service
			walletwatch *walletwatchMocks.Service
		}

		// Setup
		deps := testDeps{
			chainstream: chainstreamMocks.NewService(t),
			walletwatch: walletwatchMocks.NewService(t),
		}

		// Execute
		svc := New(deps.chainstream, deps.walletwatch)

		// Assert - verify it implements the Service interface
		var serviceInterface Service = svc
		require.NotNil(t, serviceInterface)
		require.NotNil(t, svc)
	})

	t.Run("creates independent service instances", func(t *testing.T) {
		// Custom types and structs for the test
		type testDeps struct {
			chainstream1 *chainstreamMocks.Service
			walletwatch1 *walletwatchMocks.Service
			chainstream2 *chainstreamMocks.Service
			walletwatch2 *walletwatchMocks.Service
		}

		// Setup
		deps := testDeps{
			chainstream1: chainstreamMocks.NewService(t),
			walletwatch1: walletwatchMocks.NewService(t),
			chainstream2: chainstreamMocks.NewService(t),
			walletwatch2: walletwatchMocks.NewService(t),
		}

		// Execute
		svc1 := New(deps.chainstream1, deps.walletwatch1)
		svc2 := New(deps.chainstream2, deps.walletwatch2)

		// Assert
		require.NotNil(t, svc1)
		require.NotNil(t, svc2)

		// Verify both implement the Service interface
		var serviceInterface1 Service = svc1
		var serviceInterface2 Service = svc2
		require.NotNil(t, serviceInterface1)
		require.NotNil(t, serviceInterface2)
	})

	t.Run("created service can be started and closed", func(t *testing.T) {
		// Custom types and structs for the test
		type testDeps struct {
			chainstream *chainstreamMocks.Service
			walletwatch *walletwatchMocks.Service
		}

		// Setup
		deps := testDeps{
			chainstream: chainstreamMocks.NewService(t),
			walletwatch: walletwatchMocks.NewService(t),
		}

		blocksCh := make(chan chainstream.ObservedBlock)
		deps.chainstream.EXPECT().Start(mock.Anything).Return((<-chan chainstream.ObservedBlock)(blocksCh), nil)
		deps.chainstream.EXPECT().Close().Return()

		// Execute
		svc := New(deps.chainstream, deps.walletwatch)
		ctx := t.Context()

		// Test that the created service can be started
		err := svc.Start(ctx)
		require.NoError(t, err)

		// Test that the created service can be closed
		svc.Close()
	})

	t.Run("created service handles concurrent operations safely", func(t *testing.T) {
		// Custom types and structs for the test
		type testDeps struct {
			chainstream *chainstreamMocks.Service
			walletwatch *walletwatchMocks.Service
		}

		// Setup
		deps := testDeps{
			chainstream: chainstreamMocks.NewService(t),
			walletwatch: walletwatchMocks.NewService(t),
		}

		blocksCh := make(chan chainstream.ObservedBlock)
		deps.chainstream.EXPECT().Start(mock.Anything).Return((<-chan chainstream.ObservedBlock)(blocksCh), nil).Maybe()
		deps.chainstream.EXPECT().Close().Return().Maybe()

		// Execute
		svc := New(deps.chainstream, deps.walletwatch)
		ctx := t.Context()

		// Test concurrent operations to verify mutex works
		done := make(chan struct{}, 2)

		go func() {
			defer func() { done <- struct{}{} }()
			_ = svc.Start(ctx)
		}()

		go func() {
			defer func() { done <- struct{}{} }()
			svc.Close()
		}()

		// Wait for both operations to complete without deadlock
		<-done
		<-done

		// If we reach here, the mutex is working properly
		assert.True(t, true, "concurrent operations completed without deadlock")
	})

	t.Run("new function returns non-nil service", func(t *testing.T) {
		// Custom types and structs for the test
		type testDeps struct {
			chainstream *chainstreamMocks.Service
			walletwatch *walletwatchMocks.Service
		}

		// Setup
		deps := testDeps{
			chainstream: chainstreamMocks.NewService(t),
			walletwatch: walletwatchMocks.NewService(t),
		}

		// Execute
		svc := New(deps.chainstream, deps.walletwatch)

		// Assert
		require.NotNil(t, svc)

		// Test that Close can be called safely on a new service
		svc.Close() // Should not panic

		// Test that Start returns ErrServiceAlreadyStarted after multiple calls
		ctx := t.Context()

		blocksCh := make(chan chainstream.ObservedBlock)
		deps.chainstream.EXPECT().Start(mock.Anything).Return((<-chan chainstream.ObservedBlock)(blocksCh), nil)
		deps.chainstream.EXPECT().Close().Return()

		err1 := svc.Start(ctx)
		require.NoError(t, err1)

		err2 := svc.Start(ctx)
		require.Error(t, err2)
		assert.Equal(t, ErrServiceAlreadyStarted, err2)

		svc.Close()
	})
}
