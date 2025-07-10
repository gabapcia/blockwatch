package blockproc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/gabapcia/blockwatch/internal/chainstream"
	"github.com/gabapcia/blockwatch/internal/pkg/logger"
	"github.com/gabapcia/blockwatch/internal/pkg/types"
	"github.com/gabapcia/blockwatch/internal/walletwatch"
	walletwatchtest "github.com/gabapcia/blockwatch/internal/walletwatch/mocks"
)

func TestMapObservedToWalletBlock(t *testing.T) {
	t.Run("maps observed block with single transaction", func(t *testing.T) {
		// Custom types for test
		observedBlock := chainstream.ObservedBlock{
			Network: "ethereum",
			Block: chainstream.Block{
				Height: types.Hex("0x1a"),
				Hash:   "0xabcdef123456789",
				Transactions: []chainstream.Transaction{
					{
						Hash: "0x111",
						From: "0xfrom1",
						To:   "0xto1",
					},
				},
			},
		}

		expected := walletwatch.Block{
			Network: "ethereum",
			Height:  types.Hex("0x1a"),
			Hash:    "0xabcdef123456789",
			Transactions: []walletwatch.Transaction{
				{
					Hash: "0x111",
					From: "0xfrom1",
					To:   "0xto1",
				},
			},
		}

		result := mapObservedToWalletBlock(observedBlock)

		assert.Equal(t, expected.Network, result.Network)
		assert.Equal(t, expected.Height, result.Height)
		assert.Equal(t, expected.Hash, result.Hash)
		require.Len(t, result.Transactions, len(expected.Transactions))
		assert.Equal(t, expected.Transactions[0].Hash, result.Transactions[0].Hash)
		assert.Equal(t, expected.Transactions[0].From, result.Transactions[0].From)
		assert.Equal(t, expected.Transactions[0].To, result.Transactions[0].To)
	})

	t.Run("maps observed block with multiple transactions", func(t *testing.T) {
		// Custom types for test
		observedBlock := chainstream.ObservedBlock{
			Network: "polygon",
			Block: chainstream.Block{
				Height: types.Hex("0xff"),
				Hash:   "0x987654321abcdef",
				Transactions: []chainstream.Transaction{
					{
						Hash: "0x111",
						From: "0xfrom1",
						To:   "0xto1",
					},
					{
						Hash: "0x222",
						From: "0xfrom2",
						To:   "0xto2",
					},
					{
						Hash: "0x333",
						From: "0xfrom3",
						To:   "0xto3",
					},
				},
			},
		}

		result := mapObservedToWalletBlock(observedBlock)

		assert.Equal(t, "polygon", result.Network)
		assert.Equal(t, types.Hex("0xff"), result.Height)
		assert.Equal(t, "0x987654321abcdef", result.Hash)
		require.Len(t, result.Transactions, 3)

		// Verify all transactions are mapped correctly
		expectedTxs := []walletwatch.Transaction{
			{Hash: "0x111", From: "0xfrom1", To: "0xto1"},
			{Hash: "0x222", From: "0xfrom2", To: "0xto2"},
			{Hash: "0x333", From: "0xfrom3", To: "0xto3"},
		}

		for i, expectedTx := range expectedTxs {
			assert.Equal(t, expectedTx.Hash, result.Transactions[i].Hash)
			assert.Equal(t, expectedTx.From, result.Transactions[i].From)
			assert.Equal(t, expectedTx.To, result.Transactions[i].To)
		}
	})

	t.Run("maps observed block with no transactions", func(t *testing.T) {
		// Custom types for test
		observedBlock := chainstream.ObservedBlock{
			Network: "bitcoin",
			Block: chainstream.Block{
				Height:       types.Hex("0x0"),
				Hash:         "0xemptyblock",
				Transactions: []chainstream.Transaction{},
			},
		}

		result := mapObservedToWalletBlock(observedBlock)

		assert.Equal(t, "bitcoin", result.Network)
		assert.Equal(t, types.Hex("0x0"), result.Height)
		assert.Equal(t, "0xemptyblock", result.Hash)
		assert.Len(t, result.Transactions, 0)
		assert.NotNil(t, result.Transactions)
	})

	t.Run("maps observed block with nil transactions", func(t *testing.T) {
		// Custom types for test
		observedBlock := chainstream.ObservedBlock{
			Network: "solana",
			Block: chainstream.Block{
				Height:       types.Hex("0x1"),
				Hash:         "0xniltxblock",
				Transactions: nil,
			},
		}

		result := mapObservedToWalletBlock(observedBlock)

		assert.Equal(t, "solana", result.Network)
		assert.Equal(t, types.Hex("0x1"), result.Height)
		assert.Equal(t, "0xniltxblock", result.Hash)
		assert.Len(t, result.Transactions, 0)
		assert.NotNil(t, result.Transactions)
	})

	t.Run("maps observed block with empty string values", func(t *testing.T) {
		// Custom types for test
		observedBlock := chainstream.ObservedBlock{
			Network: "",
			Block: chainstream.Block{
				Height: types.Hex(""),
				Hash:   "",
				Transactions: []chainstream.Transaction{
					{
						Hash: "",
						From: "",
						To:   "",
					},
				},
			},
		}

		result := mapObservedToWalletBlock(observedBlock)

		assert.Equal(t, "", result.Network)
		assert.Equal(t, types.Hex(""), result.Height)
		assert.Equal(t, "", result.Hash)
		require.Len(t, result.Transactions, 1)
		assert.Equal(t, "", result.Transactions[0].Hash)
		assert.Equal(t, "", result.Transactions[0].From)
		assert.Equal(t, "", result.Transactions[0].To)
	})

	t.Run("preserves transaction order", func(t *testing.T) {
		// Custom types for test
		transactions := []chainstream.Transaction{
			{Hash: "0x001", From: "0xa", To: "0xb"},
			{Hash: "0x002", From: "0xc", To: "0xd"},
			{Hash: "0x003", From: "0xe", To: "0xf"},
			{Hash: "0x004", From: "0xg", To: "0xh"},
			{Hash: "0x005", From: "0xi", To: "0xj"},
		}

		observedBlock := chainstream.ObservedBlock{
			Network: "ethereum",
			Block: chainstream.Block{
				Height:       types.Hex("0x100"),
				Hash:         "0xordertest",
				Transactions: transactions,
			},
		}

		result := mapObservedToWalletBlock(observedBlock)

		require.Len(t, result.Transactions, len(transactions))

		for i, originalTx := range transactions {
			assert.Equal(t, originalTx.Hash, result.Transactions[i].Hash)
			assert.Equal(t, originalTx.From, result.Transactions[i].From)
			assert.Equal(t, originalTx.To, result.Transactions[i].To)
		}
	})

	t.Run("creates independent transaction slice", func(t *testing.T) {
		// Custom types for test
		originalTxs := []chainstream.Transaction{
			{Hash: "0x111", From: "0xfrom1", To: "0xto1"},
		}

		observedBlock := chainstream.ObservedBlock{
			Network: "ethereum",
			Block: chainstream.Block{
				Height:       types.Hex("0x1"),
				Hash:         "0xtest",
				Transactions: originalTxs,
			},
		}

		result := mapObservedToWalletBlock(observedBlock)

		// Verify that modifying the original slice doesn't affect the result
		originalTxs[0].Hash = "0xmodified"

		assert.Equal(t, "0x111", result.Transactions[0].Hash)
		assert.NotEqual(t, originalTxs[0].Hash, result.Transactions[0].Hash)
	})
}

func TestHandleWalletActivity(t *testing.T) {
	// Initialize logger for tests
	err := logger.Init("error")
	require.NoError(t, err)

	t.Run("processes single block successfully", func(t *testing.T) {
		// Custom types and structs for test
		mockWalletwatch := walletwatchtest.NewService(t)
		svc := &service{
			walletwatch: mockWalletwatch,
		}

		observedBlock := chainstream.ObservedBlock{
			Network: "ethereum",
			Block: chainstream.Block{
				Height: types.Hex("0x1a"),
				Hash:   "0xabcdef123456789",
				Transactions: []chainstream.Transaction{
					{
						Hash: "0x111",
						From: "0xfrom1",
						To:   "0xto1",
					},
				},
			},
		}

		expectedWalletBlock := walletwatch.Block{
			Network: "ethereum",
			Height:  types.Hex("0x1a"),
			Hash:    "0xabcdef123456789",
			Transactions: []walletwatch.Transaction{
				{
					Hash: "0x111",
					From: "0xfrom1",
					To:   "0xto1",
				},
			},
		}

		blocksCh := make(chan chainstream.ObservedBlock, 1)
		blocksCh <- observedBlock
		close(blocksCh)

		mockWalletwatch.EXPECT().NotifyWatchedTransactions(mock.MatchedBy(func(ctx context.Context) bool {
			return ctx != nil
		}), expectedWalletBlock).Return(nil)

		ctx := t.Context()
		svc.handleWalletActivity(ctx, blocksCh)

		mockWalletwatch.AssertExpectations(t)
	})

	t.Run("processes multiple blocks successfully", func(t *testing.T) {
		// Custom types and structs for test
		mockWalletwatch := walletwatchtest.NewService(t)
		svc := &service{
			walletwatch: mockWalletwatch,
		}

		blocks := []chainstream.ObservedBlock{
			{
				Network: "ethereum",
				Block: chainstream.Block{
					Height: types.Hex("0x1"),
					Hash:   "0xblock1",
					Transactions: []chainstream.Transaction{
						{Hash: "0x111", From: "0xa", To: "0xb"},
					},
				},
			},
			{
				Network: "polygon",
				Block: chainstream.Block{
					Height: types.Hex("0x2"),
					Hash:   "0xblock2",
					Transactions: []chainstream.Transaction{
						{Hash: "0x222", From: "0xc", To: "0xd"},
						{Hash: "0x333", From: "0xe", To: "0xf"},
					},
				},
			},
		}

		blocksCh := make(chan chainstream.ObservedBlock, len(blocks))
		for _, block := range blocks {
			blocksCh <- block
		}
		close(blocksCh)

		// Set up expectations for each block
		for _, block := range blocks {
			expectedWalletBlock := mapObservedToWalletBlock(block)
			mockWalletwatch.EXPECT().NotifyWatchedTransactions(mock.MatchedBy(func(ctx context.Context) bool {
				return ctx != nil
			}), expectedWalletBlock).Return(nil)
		}

		ctx := t.Context()
		svc.handleWalletActivity(ctx, blocksCh)

		mockWalletwatch.AssertExpectations(t)
	})

	t.Run("handles walletwatch error gracefully", func(t *testing.T) {
		// Custom types and structs for test
		mockWalletwatch := walletwatchtest.NewService(t)
		svc := &service{
			walletwatch: mockWalletwatch,
		}

		observedBlock := chainstream.ObservedBlock{
			Network: "ethereum",
			Block: chainstream.Block{
				Height: types.Hex("0x1a"),
				Hash:   "0xabcdef123456789",
				Transactions: []chainstream.Transaction{
					{
						Hash: "0x111",
						From: "0xfrom1",
						To:   "0xto1",
					},
				},
			},
		}

		expectedWalletBlock := mapObservedToWalletBlock(observedBlock)
		walletwatchError := errors.New("walletwatch service error")

		blocksCh := make(chan chainstream.ObservedBlock, 1)
		blocksCh <- observedBlock
		close(blocksCh)

		mockWalletwatch.EXPECT().NotifyWatchedTransactions(mock.MatchedBy(func(ctx context.Context) bool {
			return ctx != nil
		}), expectedWalletBlock).Return(walletwatchError)

		ctx := t.Context()
		// Function should not panic or return error, just log the error
		svc.handleWalletActivity(ctx, blocksCh)

		mockWalletwatch.AssertExpectations(t)
	})

	t.Run("exits when context is cancelled", func(t *testing.T) {
		// Custom types and structs for test
		mockWalletwatch := walletwatchtest.NewService(t)
		svc := &service{
			walletwatch: mockWalletwatch,
		}

		blocksCh := make(chan chainstream.ObservedBlock)

		ctx, cancel := context.WithCancel(t.Context())
		cancel() // Cancel immediately

		// Function should exit immediately due to cancelled context
		svc.handleWalletActivity(ctx, blocksCh)

		// No expectations should be called since context was cancelled
		mockWalletwatch.AssertExpectations(t)
	})

	t.Run("exits when channel is closed", func(t *testing.T) {
		// Custom types and structs for test
		mockWalletwatch := walletwatchtest.NewService(t)
		svc := &service{
			walletwatch: mockWalletwatch,
		}

		blocksCh := make(chan chainstream.ObservedBlock)
		close(blocksCh) // Close channel immediately

		ctx := t.Context()
		// Function should exit immediately due to closed channel
		svc.handleWalletActivity(ctx, blocksCh)

		// No expectations should be called since channel was closed
		mockWalletwatch.AssertExpectations(t)
	})

	t.Run("handles context timeout gracefully", func(t *testing.T) {
		// Custom types and structs for test
		mockWalletwatch := walletwatchtest.NewService(t)
		svc := &service{
			walletwatch: mockWalletwatch,
		}

		blocksCh := make(chan chainstream.ObservedBlock)

		ctx, cancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
		defer cancel()

		// Function should exit when context times out
		svc.handleWalletActivity(ctx, blocksCh)

		// No expectations should be called since context timed out
		mockWalletwatch.AssertExpectations(t)
	})

	t.Run("processes block with empty transactions", func(t *testing.T) {
		// Custom types and structs for test
		mockWalletwatch := walletwatchtest.NewService(t)
		svc := &service{
			walletwatch: mockWalletwatch,
		}

		observedBlock := chainstream.ObservedBlock{
			Network: "bitcoin",
			Block: chainstream.Block{
				Height:       types.Hex("0x0"),
				Hash:         "0xemptyblock",
				Transactions: []chainstream.Transaction{},
			},
		}

		expectedWalletBlock := walletwatch.Block{
			Network:      "bitcoin",
			Height:       types.Hex("0x0"),
			Hash:         "0xemptyblock",
			Transactions: []walletwatch.Transaction{},
		}

		blocksCh := make(chan chainstream.ObservedBlock, 1)
		blocksCh <- observedBlock
		close(blocksCh)

		mockWalletwatch.EXPECT().NotifyWatchedTransactions(mock.MatchedBy(func(ctx context.Context) bool {
			return ctx != nil
		}), expectedWalletBlock).Return(nil)

		ctx := t.Context()
		svc.handleWalletActivity(ctx, blocksCh)

		mockWalletwatch.AssertExpectations(t)
	})

	t.Run("continues processing after walletwatch error", func(t *testing.T) {
		// Custom types and structs for test
		mockWalletwatch := walletwatchtest.NewService(t)
		svc := &service{
			walletwatch: mockWalletwatch,
		}

		blocks := []chainstream.ObservedBlock{
			{
				Network: "ethereum",
				Block: chainstream.Block{
					Height: types.Hex("0x1"),
					Hash:   "0xblock1",
					Transactions: []chainstream.Transaction{
						{Hash: "0x111", From: "0xa", To: "0xb"},
					},
				},
			},
			{
				Network: "ethereum",
				Block: chainstream.Block{
					Height: types.Hex("0x2"),
					Hash:   "0xblock2",
					Transactions: []chainstream.Transaction{
						{Hash: "0x222", From: "0xc", To: "0xd"},
					},
				},
			},
		}

		blocksCh := make(chan chainstream.ObservedBlock, len(blocks))
		for _, block := range blocks {
			blocksCh <- block
		}
		close(blocksCh)

		// First block fails, second block succeeds
		expectedWalletBlock1 := mapObservedToWalletBlock(blocks[0])
		expectedWalletBlock2 := mapObservedToWalletBlock(blocks[1])

		mockWalletwatch.EXPECT().NotifyWatchedTransactions(mock.MatchedBy(func(ctx context.Context) bool {
			return ctx != nil
		}), expectedWalletBlock1).Return(errors.New("first block error"))

		mockWalletwatch.EXPECT().NotifyWatchedTransactions(mock.MatchedBy(func(ctx context.Context) bool {
			return ctx != nil
		}), expectedWalletBlock2).Return(nil)

		ctx := t.Context()
		svc.handleWalletActivity(ctx, blocksCh)

		mockWalletwatch.AssertExpectations(t)
	})
}

func TestStartHandleWalletActivity(t *testing.T) {
	// Initialize logger for tests
	err := logger.Init("error")
	require.NoError(t, err)

	t.Run("starts goroutine that processes blocks", func(t *testing.T) {
		// Custom types and structs for test
		mockWalletwatch := walletwatchtest.NewService(t)
		svc := &service{
			walletwatch: mockWalletwatch,
		}

		observedBlock := chainstream.ObservedBlock{
			Network: "ethereum",
			Block: chainstream.Block{
				Height: types.Hex("0x1a"),
				Hash:   "0xabcdef123456789",
				Transactions: []chainstream.Transaction{
					{
						Hash: "0x111",
						From: "0xfrom1",
						To:   "0xto1",
					},
				},
			},
		}

		expectedWalletBlock := walletwatch.Block{
			Network: "ethereum",
			Height:  types.Hex("0x1a"),
			Hash:    "0xabcdef123456789",
			Transactions: []walletwatch.Transaction{
				{
					Hash: "0x111",
					From: "0xfrom1",
					To:   "0xto1",
				},
			},
		}

		blocksCh := make(chan chainstream.ObservedBlock, 1)

		mockWalletwatch.EXPECT().NotifyWatchedTransactions(mock.MatchedBy(func(ctx context.Context) bool {
			return ctx != nil
		}), expectedWalletBlock).Return(nil)

		ctx := t.Context()

		// Start the goroutine
		svc.startHandleWalletActivity(ctx, blocksCh)

		// Send a block to verify the goroutine is processing
		blocksCh <- observedBlock
		close(blocksCh)

		// Give the goroutine time to process
		time.Sleep(50 * time.Millisecond)

		mockWalletwatch.AssertExpectations(t)
	})

	t.Run("starts goroutine that handles context cancellation", func(t *testing.T) {
		// Custom types and structs for test
		mockWalletwatch := walletwatchtest.NewService(t)
		svc := &service{
			walletwatch: mockWalletwatch,
		}

		blocksCh := make(chan chainstream.ObservedBlock)

		ctx, cancel := context.WithCancel(t.Context())

		// Start the goroutine
		svc.startHandleWalletActivity(ctx, blocksCh)

		// Cancel the context immediately
		cancel()

		// Give the goroutine time to exit
		time.Sleep(50 * time.Millisecond)

		// No expectations should be called since context was cancelled
		mockWalletwatch.AssertExpectations(t)
	})

	t.Run("starts goroutine that handles closed channel", func(t *testing.T) {
		// Custom types and structs for test
		mockWalletwatch := walletwatchtest.NewService(t)
		svc := &service{
			walletwatch: mockWalletwatch,
		}

		blocksCh := make(chan chainstream.ObservedBlock)

		ctx := t.Context()

		// Start the goroutine
		svc.startHandleWalletActivity(ctx, blocksCh)

		// Close the channel immediately
		close(blocksCh)

		// Give the goroutine time to exit
		time.Sleep(50 * time.Millisecond)

		// No expectations should be called since channel was closed
		mockWalletwatch.AssertExpectations(t)
	})

	t.Run("starts goroutine that processes multiple blocks", func(t *testing.T) {
		// Custom types and structs for test
		mockWalletwatch := walletwatchtest.NewService(t)
		svc := &service{
			walletwatch: mockWalletwatch,
		}

		blocks := []chainstream.ObservedBlock{
			{
				Network: "ethereum",
				Block: chainstream.Block{
					Height: types.Hex("0x1"),
					Hash:   "0xblock1",
					Transactions: []chainstream.Transaction{
						{Hash: "0x111", From: "0xa", To: "0xb"},
					},
				},
			},
			{
				Network: "polygon",
				Block: chainstream.Block{
					Height: types.Hex("0x2"),
					Hash:   "0xblock2",
					Transactions: []chainstream.Transaction{
						{Hash: "0x222", From: "0xc", To: "0xd"},
					},
				},
			},
		}

		blocksCh := make(chan chainstream.ObservedBlock, len(blocks))

		// Set up expectations for each block
		for _, block := range blocks {
			expectedWalletBlock := mapObservedToWalletBlock(block)
			mockWalletwatch.EXPECT().NotifyWatchedTransactions(mock.MatchedBy(func(ctx context.Context) bool {
				return ctx != nil
			}), expectedWalletBlock).Return(nil)
		}

		ctx := t.Context()

		// Start the goroutine
		svc.startHandleWalletActivity(ctx, blocksCh)

		// Send blocks to verify the goroutine is processing
		for _, block := range blocks {
			blocksCh <- block
		}
		close(blocksCh)

		// Give the goroutine time to process all blocks
		time.Sleep(100 * time.Millisecond)

		mockWalletwatch.AssertExpectations(t)
	})

	t.Run("starts goroutine that handles walletwatch errors", func(t *testing.T) {
		// Custom types and structs for test
		mockWalletwatch := walletwatchtest.NewService(t)
		svc := &service{
			walletwatch: mockWalletwatch,
		}

		observedBlock := chainstream.ObservedBlock{
			Network: "ethereum",
			Block: chainstream.Block{
				Height: types.Hex("0x1a"),
				Hash:   "0xabcdef123456789",
				Transactions: []chainstream.Transaction{
					{
						Hash: "0x111",
						From: "0xfrom1",
						To:   "0xto1",
					},
				},
			},
		}

		expectedWalletBlock := mapObservedToWalletBlock(observedBlock)
		walletwatchError := errors.New("walletwatch service error")

		blocksCh := make(chan chainstream.ObservedBlock, 1)

		mockWalletwatch.EXPECT().NotifyWatchedTransactions(mock.MatchedBy(func(ctx context.Context) bool {
			return ctx != nil
		}), expectedWalletBlock).Return(walletwatchError)

		ctx := t.Context()

		// Start the goroutine
		svc.startHandleWalletActivity(ctx, blocksCh)

		// Send a block that will cause an error
		blocksCh <- observedBlock
		close(blocksCh)

		// Give the goroutine time to process and handle the error
		time.Sleep(50 * time.Millisecond)

		mockWalletwatch.AssertExpectations(t)
	})
}
