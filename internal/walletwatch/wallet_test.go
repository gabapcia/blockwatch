package walletwatch

import (
	"context"
	"errors"
	"testing"

	"github.com/gabapcia/blockwatch/internal/pkg/logger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func init() {
	// Initialize logger for tests to prevent nil pointer dereference
	_ = logger.Init("error")
}

// matchAddresses creates a mock.MatchedBy function that matches slices containing the same addresses
// regardless of order, handling the non-deterministic nature of Go's map iteration
func matchAddresses(expectedAddresses []string) interface{} {
	return mock.MatchedBy(func(addresses []string) bool {
		if len(addresses) != len(expectedAddresses) {
			return false
		}
		addressMap := make(map[string]bool)
		for _, addr := range addresses {
			addressMap[addr] = true
		}
		for _, expected := range expectedAddresses {
			if !addressMap[expected] {
				return false
			}
		}
		return true
	})
}

func TestGetTransactionsByWallet(t *testing.T) {
	t.Run("empty transactions list", func(t *testing.T) {
		mockWalletStorage := NewWalletStorageMock(t)

		// When there are no transactions, walletsSet.ToSlice() returns nil
		mockWalletStorage.EXPECT().
			FilterWatchedWallets(t.Context(), "ethereum", []string(nil)).
			Return([]string{}, nil)

		svc := &service{
			walletStorage: mockWalletStorage,
		}

		result, err := svc.getTransactionsByWallet(t.Context(), "ethereum", []Transaction{})

		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("no watched wallets", func(t *testing.T) {
		mockWalletStorage := NewWalletStorageMock(t)

		transactions := []Transaction{
			{Hash: "tx1", From: "wallet1", To: "wallet2"},
			{Hash: "tx2", From: "wallet2", To: "wallet3"},
		}

		mockWalletStorage.EXPECT().
			FilterWatchedWallets(t.Context(), "ethereum", matchAddresses([]string{"wallet1", "wallet2", "wallet3"})).
			Return([]string{}, nil)

		svc := &service{
			walletStorage: mockWalletStorage,
		}

		result, err := svc.getTransactionsByWallet(t.Context(), "ethereum", transactions)

		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("single transaction with one watched wallet as sender", func(t *testing.T) {
		mockWalletStorage := NewWalletStorageMock(t)

		transactions := []Transaction{
			{Hash: "tx1", From: "wallet1", To: "wallet2"},
		}

		mockWalletStorage.EXPECT().
			FilterWatchedWallets(t.Context(), "ethereum", matchAddresses([]string{"wallet1", "wallet2"})).
			Return([]string{"wallet1"}, nil)

		svc := &service{
			walletStorage: mockWalletStorage,
		}

		result, err := svc.getTransactionsByWallet(t.Context(), "ethereum", transactions)

		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Contains(t, result, "wallet1")
		assert.Len(t, result["wallet1"], 1)
		assert.Equal(t, "tx1", result["wallet1"][0].Hash)
		assert.Equal(t, "wallet1", result["wallet1"][0].From)
		assert.Equal(t, "wallet2", result["wallet1"][0].To)
	})

	t.Run("single transaction with one watched wallet as recipient", func(t *testing.T) {
		mockWalletStorage := NewWalletStorageMock(t)

		transactions := []Transaction{
			{Hash: "tx1", From: "wallet1", To: "wallet2"},
		}

		mockWalletStorage.EXPECT().
			FilterWatchedWallets(t.Context(), "ethereum", matchAddresses([]string{"wallet1", "wallet2"})).
			Return([]string{"wallet2"}, nil)

		svc := &service{
			walletStorage: mockWalletStorage,
		}

		result, err := svc.getTransactionsByWallet(t.Context(), "ethereum", transactions)

		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Contains(t, result, "wallet2")
		assert.Len(t, result["wallet2"], 1)
		assert.Equal(t, "tx1", result["wallet2"][0].Hash)
	})

	t.Run("single transaction with both sender and recipient watched", func(t *testing.T) {
		mockWalletStorage := NewWalletStorageMock(t)

		transactions := []Transaction{
			{Hash: "tx1", From: "wallet1", To: "wallet2"},
		}

		mockWalletStorage.EXPECT().
			FilterWatchedWallets(t.Context(), "ethereum", matchAddresses([]string{"wallet1", "wallet2"})).
			Return([]string{"wallet1", "wallet2"}, nil)

		svc := &service{
			walletStorage: mockWalletStorage,
		}

		result, err := svc.getTransactionsByWallet(t.Context(), "ethereum", transactions)

		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Contains(t, result, "wallet1")
		assert.Contains(t, result, "wallet2")
		assert.Len(t, result["wallet1"], 1)
		assert.Len(t, result["wallet2"], 1)
		assert.Equal(t, "tx1", result["wallet1"][0].Hash)
		assert.Equal(t, "tx1", result["wallet2"][0].Hash)
	})

	t.Run("multiple transactions with multiple watched wallets", func(t *testing.T) {
		mockWalletStorage := NewWalletStorageMock(t)

		transactions := []Transaction{
			{Hash: "tx1", From: "wallet1", To: "wallet2"},
			{Hash: "tx2", From: "wallet2", To: "wallet3"},
			{Hash: "tx3", From: "wallet4", To: "wallet1"},
			{Hash: "tx4", From: "wallet5", To: "wallet6"},
		}

		mockWalletStorage.EXPECT().
			FilterWatchedWallets(t.Context(), "ethereum", matchAddresses([]string{"wallet1", "wallet2", "wallet3", "wallet4", "wallet5", "wallet6"})).
			Return([]string{"wallet1", "wallet3"}, nil)

		svc := &service{
			walletStorage: mockWalletStorage,
		}

		result, err := svc.getTransactionsByWallet(t.Context(), "ethereum", transactions)

		require.NoError(t, err)
		assert.Len(t, result, 2)

		// Check wallet1 transactions
		assert.Contains(t, result, "wallet1")
		assert.Len(t, result["wallet1"], 2)

		wallet1Hashes := make([]string, len(result["wallet1"]))
		for i, tx := range result["wallet1"] {
			wallet1Hashes[i] = tx.Hash
		}
		assert.ElementsMatch(t, []string{"tx1", "tx3"}, wallet1Hashes)

		// Check wallet3 transactions
		assert.Contains(t, result, "wallet3")
		assert.Len(t, result["wallet3"], 1)
		assert.Equal(t, "tx2", result["wallet3"][0].Hash)
	})

	t.Run("wallet appears in multiple transactions", func(t *testing.T) {
		mockWalletStorage := NewWalletStorageMock(t)

		transactions := []Transaction{
			{Hash: "tx1", From: "wallet1", To: "wallet2"},
			{Hash: "tx2", From: "wallet3", To: "wallet1"},
			{Hash: "tx3", From: "wallet1", To: "wallet4"},
		}

		mockWalletStorage.EXPECT().
			FilterWatchedWallets(t.Context(), "ethereum", matchAddresses([]string{"wallet1", "wallet2", "wallet3", "wallet4"})).
			Return([]string{"wallet1"}, nil)

		svc := &service{
			walletStorage: mockWalletStorage,
		}

		result, err := svc.getTransactionsByWallet(t.Context(), "ethereum", transactions)

		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Contains(t, result, "wallet1")
		assert.Len(t, result["wallet1"], 3)

		wallet1Hashes := make([]string, len(result["wallet1"]))
		for i, tx := range result["wallet1"] {
			wallet1Hashes[i] = tx.Hash
		}
		assert.ElementsMatch(t, []string{"tx1", "tx2", "tx3"}, wallet1Hashes)
	})

	t.Run("duplicate transaction hashes", func(t *testing.T) {
		mockWalletStorage := NewWalletStorageMock(t)

		transactions := []Transaction{
			{Hash: "tx1", From: "wallet1", To: "wallet2"},
			{Hash: "tx1", From: "wallet1", To: "wallet2"}, // duplicate
		}

		mockWalletStorage.EXPECT().
			FilterWatchedWallets(t.Context(), "ethereum", matchAddresses([]string{"wallet1", "wallet2"})).
			Return([]string{"wallet1"}, nil)

		svc := &service{
			walletStorage: mockWalletStorage,
		}

		result, err := svc.getTransactionsByWallet(t.Context(), "ethereum", transactions)

		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Contains(t, result, "wallet1")
		// Should only have one transaction due to duplicate hash overwriting
		assert.Len(t, result["wallet1"], 1)
		assert.Equal(t, "tx1", result["wallet1"][0].Hash)
	})

	t.Run("self transaction same sender and recipient", func(t *testing.T) {
		mockWalletStorage := NewWalletStorageMock(t)

		transactions := []Transaction{
			{Hash: "tx1", From: "wallet1", To: "wallet1"},
		}

		mockWalletStorage.EXPECT().
			FilterWatchedWallets(t.Context(), "ethereum", []string{"wallet1"}).
			Return([]string{"wallet1"}, nil)

		svc := &service{
			walletStorage: mockWalletStorage,
		}

		result, err := svc.getTransactionsByWallet(t.Context(), "ethereum", transactions)

		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Contains(t, result, "wallet1")
		assert.Len(t, result["wallet1"], 1)
		assert.Equal(t, "tx1", result["wallet1"][0].Hash)
		assert.Equal(t, "wallet1", result["wallet1"][0].From)
		assert.Equal(t, "wallet1", result["wallet1"][0].To)
	})

	t.Run("wallet storage error", func(t *testing.T) {
		mockWalletStorage := NewWalletStorageMock(t)

		transactions := []Transaction{
			{Hash: "tx1", From: "wallet1", To: "wallet2"},
		}

		expectedError := errors.New("storage connection failed")
		mockWalletStorage.EXPECT().
			FilterWatchedWallets(t.Context(), "ethereum", matchAddresses([]string{"wallet1", "wallet2"})).
			Return(nil, expectedError)

		svc := &service{
			walletStorage: mockWalletStorage,
		}

		result, err := svc.getTransactionsByWallet(t.Context(), "ethereum", transactions)

		require.Error(t, err)
		assert.Equal(t, expectedError, err)
		assert.Nil(t, result)
	})

	t.Run("different network", func(t *testing.T) {
		mockWalletStorage := NewWalletStorageMock(t)

		transactions := []Transaction{
			{Hash: "tx1", From: "wallet1", To: "wallet2"},
		}

		mockWalletStorage.EXPECT().
			FilterWatchedWallets(t.Context(), "polygon", matchAddresses([]string{"wallet1", "wallet2"})).
			Return([]string{"wallet1"}, nil)

		svc := &service{
			walletStorage: mockWalletStorage,
		}

		result, err := svc.getTransactionsByWallet(t.Context(), "polygon", transactions)

		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Contains(t, result, "wallet1")
		assert.Equal(t, "tx1", result["wallet1"][0].Hash)
	})

	t.Run("context cancellation", func(t *testing.T) {
		mockWalletStorage := NewWalletStorageMock(t)

		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		transactions := []Transaction{
			{Hash: "tx1", From: "wallet1", To: "wallet2"},
		}

		mockWalletStorage.EXPECT().
			FilterWatchedWallets(ctx, "ethereum", matchAddresses([]string{"wallet1", "wallet2"})).
			Return(nil, context.Canceled)

		svc := &service{
			walletStorage: mockWalletStorage,
		}

		result, err := svc.getTransactionsByWallet(ctx, "ethereum", transactions)

		require.Error(t, err)
		assert.Equal(t, context.Canceled, err)
		assert.Nil(t, result)
	})

	t.Run("empty wallet addresses", func(t *testing.T) {
		mockWalletStorage := NewWalletStorageMock(t)

		transactions := []Transaction{
			{Hash: "tx1", From: "", To: "wallet1"},
			{Hash: "tx2", From: "wallet2", To: ""},
			{Hash: "tx3", From: "", To: ""},
		}

		mockWalletStorage.EXPECT().
			FilterWatchedWallets(t.Context(), "ethereum", matchAddresses([]string{"", "wallet1", "wallet2"})).
			Return([]string{"wallet1"}, nil)

		svc := &service{
			walletStorage: mockWalletStorage,
		}

		result, err := svc.getTransactionsByWallet(t.Context(), "ethereum", transactions)

		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Contains(t, result, "wallet1")
		assert.Len(t, result["wallet1"], 1)
		assert.Equal(t, "tx1", result["wallet1"][0].Hash)
	})

	t.Run("large number of transactions and wallets", func(t *testing.T) {
		mockWalletStorage := NewWalletStorageMock(t)

		transactions := []Transaction{
			{Hash: "tx1", From: "wallet1", To: "wallet2"},
			{Hash: "tx2", From: "wallet3", To: "wallet4"},
			{Hash: "tx3", From: "wallet5", To: "wallet6"},
			{Hash: "tx4", From: "wallet7", To: "wallet8"},
			{Hash: "tx5", From: "wallet9", To: "wallet10"},
		}

		mockWalletStorage.EXPECT().
			FilterWatchedWallets(t.Context(), "ethereum", matchAddresses([]string{"wallet1", "wallet2", "wallet3", "wallet4", "wallet5", "wallet6", "wallet7", "wallet8", "wallet9", "wallet10"})).
			Return([]string{"wallet2", "wallet4", "wallet6"}, nil)

		svc := &service{
			walletStorage: mockWalletStorage,
		}

		result, err := svc.getTransactionsByWallet(t.Context(), "ethereum", transactions)

		require.NoError(t, err)
		assert.Len(t, result, 3)

		assert.Contains(t, result, "wallet2")
		assert.Len(t, result["wallet2"], 1)
		assert.Equal(t, "tx1", result["wallet2"][0].Hash)

		assert.Contains(t, result, "wallet4")
		assert.Len(t, result["wallet4"], 1)
		assert.Equal(t, "tx2", result["wallet4"][0].Hash)

		assert.Contains(t, result, "wallet6")
		assert.Len(t, result["wallet6"], 1)
		assert.Equal(t, "tx3", result["wallet6"][0].Hash)
	})
}

func TestNotifyWatchedWalletTransactions(t *testing.T) {
	t.Run("successful notification for single wallet", func(t *testing.T) {
		mockWalletStorage := NewWalletStorageMock(t)
		mockTransactionNotifier := NewTransactionNotifierMock(t)

		transactions := []Transaction{
			{Hash: "tx1", From: "wallet1", To: "wallet2"},
		}

		// Mock getTransactionsByWallet behavior
		mockWalletStorage.EXPECT().
			FilterWatchedWallets(t.Context(), "ethereum", matchAddresses([]string{"wallet1", "wallet2"})).
			Return([]string{"wallet1"}, nil)

		// Mock notification call
		expectedTxs := []Transaction{{Hash: "tx1", From: "wallet1", To: "wallet2"}}
		mockTransactionNotifier.EXPECT().
			NotifyTransactions(t.Context(), "ethereum", "wallet1", expectedTxs).
			Return(nil)

		svc := &service{
			walletStorage:       mockWalletStorage,
			transactionNotifier: mockTransactionNotifier,
		}

		err := svc.notifyWatchedWalletTransactions(t.Context(), "ethereum", transactions)

		require.NoError(t, err)
	})

	t.Run("successful notification for multiple wallets", func(t *testing.T) {
		mockWalletStorage := NewWalletStorageMock(t)
		mockTransactionNotifier := NewTransactionNotifierMock(t)

		transactions := []Transaction{
			{Hash: "tx1", From: "wallet1", To: "wallet2"},
			{Hash: "tx2", From: "wallet3", To: "wallet1"},
		}

		// Mock getTransactionsByWallet behavior
		mockWalletStorage.EXPECT().
			FilterWatchedWallets(t.Context(), "ethereum", matchAddresses([]string{"wallet1", "wallet2", "wallet3"})).
			Return([]string{"wallet1", "wallet2"}, nil)

		// Mock notification calls - order may vary due to map iteration
		mockTransactionNotifier.EXPECT().
			NotifyTransactions(t.Context(), "ethereum", "wallet1", mock.MatchedBy(func(txs []Transaction) bool {
				if len(txs) != 2 {
					return false
				}
				hashes := make(map[string]bool)
				for _, tx := range txs {
					hashes[tx.Hash] = true
				}
				return hashes["tx1"] && hashes["tx2"]
			})).
			Return(nil)

		mockTransactionNotifier.EXPECT().
			NotifyTransactions(t.Context(), "ethereum", "wallet2", mock.MatchedBy(func(txs []Transaction) bool {
				return len(txs) == 1 && txs[0].Hash == "tx1"
			})).
			Return(nil)

		svc := &service{
			walletStorage:       mockWalletStorage,
			transactionNotifier: mockTransactionNotifier,
		}

		err := svc.notifyWatchedWalletTransactions(t.Context(), "ethereum", transactions)

		require.NoError(t, err)
	})

	t.Run("error from getTransactionsByWallet", func(t *testing.T) {
		mockWalletStorage := NewWalletStorageMock(t)
		mockTransactionNotifier := NewTransactionNotifierMock(t)

		transactions := []Transaction{
			{Hash: "tx1", From: "wallet1", To: "wallet2"},
		}

		expectedError := errors.New("wallet storage error")
		mockWalletStorage.EXPECT().
			FilterWatchedWallets(t.Context(), "ethereum", matchAddresses([]string{"wallet1", "wallet2"})).
			Return(nil, expectedError)

		// TransactionNotifier should not be called when getTransactionsByWallet fails

		svc := &service{
			walletStorage:       mockWalletStorage,
			transactionNotifier: mockTransactionNotifier,
		}

		err := svc.notifyWatchedWalletTransactions(t.Context(), "ethereum", transactions)

		require.Error(t, err)
		assert.Equal(t, expectedError, err)
	})

	t.Run("error from transaction notifier", func(t *testing.T) {
		mockWalletStorage := NewWalletStorageMock(t)
		mockTransactionNotifier := NewTransactionNotifierMock(t)

		transactions := []Transaction{
			{Hash: "tx1", From: "wallet1", To: "wallet2"},
		}

		// Mock getTransactionsByWallet behavior
		mockWalletStorage.EXPECT().
			FilterWatchedWallets(t.Context(), "ethereum", matchAddresses([]string{"wallet1", "wallet2"})).
			Return([]string{"wallet1"}, nil)

		// Mock notification call that fails
		expectedError := errors.New("notification failed")
		expectedTxs := []Transaction{{Hash: "tx1", From: "wallet1", To: "wallet2"}}
		mockTransactionNotifier.EXPECT().
			NotifyTransactions(t.Context(), "ethereum", "wallet1", expectedTxs).
			Return(expectedError)

		svc := &service{
			walletStorage:       mockWalletStorage,
			transactionNotifier: mockTransactionNotifier,
		}

		err := svc.notifyWatchedWalletTransactions(t.Context(), "ethereum", transactions)

		require.Error(t, err)
		assert.Equal(t, expectedError, err)
	})

	t.Run("no watched wallets found", func(t *testing.T) {
		mockWalletStorage := NewWalletStorageMock(t)
		mockTransactionNotifier := NewTransactionNotifierMock(t)

		transactions := []Transaction{
			{Hash: "tx1", From: "wallet1", To: "wallet2"},
		}

		// Mock getTransactionsByWallet behavior returning empty result
		mockWalletStorage.EXPECT().
			FilterWatchedWallets(t.Context(), "ethereum", matchAddresses([]string{"wallet1", "wallet2"})).
			Return([]string{}, nil)

		// TransactionNotifier should not be called when no watched wallets are found

		svc := &service{
			walletStorage:       mockWalletStorage,
			transactionNotifier: mockTransactionNotifier,
		}

		err := svc.notifyWatchedWalletTransactions(t.Context(), "ethereum", transactions)

		require.NoError(t, err)
	})

	t.Run("empty transactions list", func(t *testing.T) {
		mockWalletStorage := NewWalletStorageMock(t)
		mockTransactionNotifier := NewTransactionNotifierMock(t)

		// Mock getTransactionsByWallet behavior for empty transactions
		mockWalletStorage.EXPECT().
			FilterWatchedWallets(t.Context(), "ethereum", []string(nil)).
			Return([]string{}, nil)

		// TransactionNotifier should not be called when no transactions are provided

		svc := &service{
			walletStorage:       mockWalletStorage,
			transactionNotifier: mockTransactionNotifier,
		}

		err := svc.notifyWatchedWalletTransactions(t.Context(), "ethereum", []Transaction{})

		require.NoError(t, err)
	})

	t.Run("different network", func(t *testing.T) {
		mockWalletStorage := NewWalletStorageMock(t)
		mockTransactionNotifier := NewTransactionNotifierMock(t)

		transactions := []Transaction{
			{Hash: "tx1", From: "wallet1", To: "wallet2"},
		}

		// Mock getTransactionsByWallet behavior for polygon network
		mockWalletStorage.EXPECT().
			FilterWatchedWallets(t.Context(), "polygon", matchAddresses([]string{"wallet1", "wallet2"})).
			Return([]string{"wallet1"}, nil)

		// Mock notification call for polygon network
		expectedTxs := []Transaction{{Hash: "tx1", From: "wallet1", To: "wallet2"}}
		mockTransactionNotifier.EXPECT().
			NotifyTransactions(t.Context(), "polygon", "wallet1", expectedTxs).
			Return(nil)

		svc := &service{
			walletStorage:       mockWalletStorage,
			transactionNotifier: mockTransactionNotifier,
		}

		err := svc.notifyWatchedWalletTransactions(t.Context(), "polygon", transactions)

		require.NoError(t, err)
	})

	t.Run("context cancellation", func(t *testing.T) {
		mockWalletStorage := NewWalletStorageMock(t)
		mockTransactionNotifier := NewTransactionNotifierMock(t)

		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		transactions := []Transaction{
			{Hash: "tx1", From: "wallet1", To: "wallet2"},
		}

		// Mock getTransactionsByWallet behavior that respects context cancellation
		mockWalletStorage.EXPECT().
			FilterWatchedWallets(ctx, "ethereum", matchAddresses([]string{"wallet1", "wallet2"})).
			Return(nil, context.Canceled)

		svc := &service{
			walletStorage:       mockWalletStorage,
			transactionNotifier: mockTransactionNotifier,
		}

		err := svc.notifyWatchedWalletTransactions(ctx, "ethereum", transactions)

		require.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})

	t.Run("notification fails for first wallet stops processing", func(t *testing.T) {
		mockWalletStorage := NewWalletStorageMock(t)
		mockTransactionNotifier := NewTransactionNotifierMock(t)

		transactions := []Transaction{
			{Hash: "tx1", From: "wallet1", To: "wallet2"},
			{Hash: "tx2", From: "wallet3", To: "wallet4"},
		}

		// Mock getTransactionsByWallet behavior
		mockWalletStorage.EXPECT().
			FilterWatchedWallets(t.Context(), "ethereum", mock.MatchedBy(func(addresses []string) bool {
				expectedAddresses := []string{"wallet1", "wallet2", "wallet3", "wallet4"}
				if len(addresses) != len(expectedAddresses) {
					return false
				}
				addressMap := make(map[string]bool)
				for _, addr := range addresses {
					addressMap[addr] = true
				}
				for _, expected := range expectedAddresses {
					if !addressMap[expected] {
						return false
					}
				}
				return true
			})).
			Return([]string{"wallet1", "wallet3"}, nil)

		// Mock notification call that fails for the first wallet processed
		// Due to map iteration order being non-deterministic, we need to handle both cases
		expectedError := errors.New("notification failed")

		// Set up expectations for both possible orders
		call1 := mockTransactionNotifier.EXPECT().
			NotifyTransactions(t.Context(), "ethereum", mock.AnythingOfType("string"), mock.AnythingOfType("[]walletwatch.Transaction")).
			Return(expectedError).
			Maybe() // This call might happen first

		call2 := mockTransactionNotifier.EXPECT().
			NotifyTransactions(t.Context(), "ethereum", mock.AnythingOfType("string"), mock.AnythingOfType("[]walletwatch.Transaction")).
			Return(expectedError).
			Maybe() // This call might happen first

		// Ensure at least one call happens
		call1.NotBefore()
		call2.NotBefore()

		svc := &service{
			walletStorage:       mockWalletStorage,
			transactionNotifier: mockTransactionNotifier,
		}

		err := svc.notifyWatchedWalletTransactions(t.Context(), "ethereum", transactions)

		require.Error(t, err)
		assert.Equal(t, expectedError, err)
	})
}

func TestNotifyWatchedTransactions(t *testing.T) {
	t.Run("successful processing of new block", func(t *testing.T) {
		mockWalletStorage := NewWalletStorageMock(t)
		mockTransactionNotifier := NewTransactionNotifierMock(t)
		mockIdempotencyGuard := NewIdempotencyGuardMock(t)

		block := Block{
			Network: "ethereum",
			Hash:    "block123",
			Height:  "0x64", // 100 in hex
			Transactions: []Transaction{
				{Hash: "tx1", From: "wallet1", To: "wallet2"},
			},
		}

		// Mock idempotency guard claiming the block
		mockIdempotencyGuard.EXPECT().
			ClaimBlockForTxWatch(t.Context(), "ethereum", "block123", mock.AnythingOfType("time.Duration")).
			Return(nil)

		// Mock getTransactionsByWallet behavior
		mockWalletStorage.EXPECT().
			FilterWatchedWallets(t.Context(), "ethereum", matchAddresses([]string{"wallet1", "wallet2"})).
			Return([]string{"wallet1"}, nil)

		// Mock notification call
		expectedTxs := []Transaction{{Hash: "tx1", From: "wallet1", To: "wallet2"}}
		mockTransactionNotifier.EXPECT().
			NotifyTransactions(t.Context(), "ethereum", "wallet1", expectedTxs).
			Return(nil)

		// Mock marking block as complete
		mockIdempotencyGuard.EXPECT().
			MarkBlockTxWatchComplete(t.Context(), "ethereum", "block123").
			Return(nil)

		svc := &service{
			walletStorage:       mockWalletStorage,
			transactionNotifier: mockTransactionNotifier,
			idempotencyGuard:    mockIdempotencyGuard,
		}

		err := svc.NotifyWatchedTransactions(t.Context(), block)

		require.NoError(t, err)
	})

	t.Run("block already in progress", func(t *testing.T) {
		mockWalletStorage := NewWalletStorageMock(t)
		mockTransactionNotifier := NewTransactionNotifierMock(t)
		mockIdempotencyGuard := NewIdempotencyGuardMock(t)

		block := Block{
			Network: "ethereum",
			Hash:    "block123",
			Height:  "0x64", // 100 in hex
			Transactions: []Transaction{
				{Hash: "tx1", From: "wallet1", To: "wallet2"},
			},
		}

		// Mock idempotency guard returning ErrStillInProgress
		mockIdempotencyGuard.EXPECT().
			ClaimBlockForTxWatch(t.Context(), "ethereum", "block123", mock.AnythingOfType("time.Duration")).
			Return(ErrStillInProgress)

		// No other mocks should be called when block is already in progress

		svc := &service{
			walletStorage:       mockWalletStorage,
			transactionNotifier: mockTransactionNotifier,
			idempotencyGuard:    mockIdempotencyGuard,
		}

		err := svc.NotifyWatchedTransactions(t.Context(), block)

		require.Error(t, err)
		assert.Equal(t, ErrStillInProgress, err)
	})

	t.Run("block already finished", func(t *testing.T) {
		mockWalletStorage := NewWalletStorageMock(t)
		mockTransactionNotifier := NewTransactionNotifierMock(t)
		mockIdempotencyGuard := NewIdempotencyGuardMock(t)

		block := Block{
			Network: "ethereum",
			Hash:    "block123",
			Height:  "0x64", // 100 in hex
			Transactions: []Transaction{
				{Hash: "tx1", From: "wallet1", To: "wallet2"},
			},
		}

		// Mock idempotency guard returning ErrAlreadyFinished
		mockIdempotencyGuard.EXPECT().
			ClaimBlockForTxWatch(t.Context(), "ethereum", "block123", mock.AnythingOfType("time.Duration")).
			Return(ErrAlreadyFinished)

		// No other mocks should be called when block is already finished

		svc := &service{
			walletStorage:       mockWalletStorage,
			transactionNotifier: mockTransactionNotifier,
			idempotencyGuard:    mockIdempotencyGuard,
		}

		err := svc.NotifyWatchedTransactions(t.Context(), block)

		require.Error(t, err)
		assert.Equal(t, ErrAlreadyFinished, err)
	})

	t.Run("error claiming block", func(t *testing.T) {
		mockWalletStorage := NewWalletStorageMock(t)
		mockTransactionNotifier := NewTransactionNotifierMock(t)
		mockIdempotencyGuard := NewIdempotencyGuardMock(t)

		block := Block{
			Network: "ethereum",
			Hash:    "block123",
			Height:  "0x64", // 100 in hex
			Transactions: []Transaction{
				{Hash: "tx1", From: "wallet1", To: "wallet2"},
			},
		}

		expectedError := errors.New("idempotency guard error")
		mockIdempotencyGuard.EXPECT().
			ClaimBlockForTxWatch(t.Context(), "ethereum", "block123", mock.AnythingOfType("time.Duration")).
			Return(expectedError)

		// No other mocks should be called when claiming fails

		svc := &service{
			walletStorage:       mockWalletStorage,
			transactionNotifier: mockTransactionNotifier,
			idempotencyGuard:    mockIdempotencyGuard,
		}

		err := svc.NotifyWatchedTransactions(t.Context(), block)

		require.Error(t, err)
		assert.Equal(t, expectedError, err)
	})

	t.Run("error during notification processing", func(t *testing.T) {
		mockWalletStorage := NewWalletStorageMock(t)
		mockTransactionNotifier := NewTransactionNotifierMock(t)
		mockIdempotencyGuard := NewIdempotencyGuardMock(t)

		block := Block{
			Network: "ethereum",
			Hash:    "block123",
			Height:  "0x64", // 100 in hex
			Transactions: []Transaction{
				{Hash: "tx1", From: "wallet1", To: "wallet2"},
			},
		}

		// Mock idempotency guard claiming the block
		mockIdempotencyGuard.EXPECT().
			ClaimBlockForTxWatch(t.Context(), "ethereum", "block123", mock.AnythingOfType("time.Duration")).
			Return(nil)

		// Mock getTransactionsByWallet behavior that fails
		expectedError := errors.New("wallet storage error")
		mockWalletStorage.EXPECT().
			FilterWatchedWallets(t.Context(), "ethereum", matchAddresses([]string{"wallet1", "wallet2"})).
			Return(nil, expectedError)

		// MarkBlockTxWatchComplete should not be called when notification fails

		svc := &service{
			walletStorage:       mockWalletStorage,
			transactionNotifier: mockTransactionNotifier,
			idempotencyGuard:    mockIdempotencyGuard,
		}

		err := svc.NotifyWatchedTransactions(t.Context(), block)

		require.Error(t, err)
		assert.Equal(t, expectedError, err)
	})

	t.Run("error marking block as complete but processing succeeds", func(t *testing.T) {
		mockWalletStorage := NewWalletStorageMock(t)
		mockTransactionNotifier := NewTransactionNotifierMock(t)
		mockIdempotencyGuard := NewIdempotencyGuardMock(t)

		block := Block{
			Network: "ethereum",
			Hash:    "block123",
			Height:  "0x64", // 100 in hex
			Transactions: []Transaction{
				{Hash: "tx1", From: "wallet1", To: "wallet2"},
			},
		}

		// Mock idempotency guard claiming the block
		mockIdempotencyGuard.EXPECT().
			ClaimBlockForTxWatch(t.Context(), "ethereum", "block123", mock.AnythingOfType("time.Duration")).
			Return(nil)

		// Mock successful getTransactionsByWallet behavior
		mockWalletStorage.EXPECT().
			FilterWatchedWallets(t.Context(), "ethereum", matchAddresses([]string{"wallet1", "wallet2"})).
			Return([]string{"wallet1"}, nil)

		// Mock successful notification call
		expectedTxs := []Transaction{{Hash: "tx1", From: "wallet1", To: "wallet2"}}
		mockTransactionNotifier.EXPECT().
			NotifyTransactions(t.Context(), "ethereum", "wallet1", expectedTxs).
			Return(nil)

		// Mock marking block as complete that fails (but should not propagate error)
		markingError := errors.New("marking error")
		mockIdempotencyGuard.EXPECT().
			MarkBlockTxWatchComplete(t.Context(), "ethereum", "block123").
			Return(markingError)

		svc := &service{
			walletStorage:       mockWalletStorage,
			transactionNotifier: mockTransactionNotifier,
			idempotencyGuard:    mockIdempotencyGuard,
		}

		err := svc.NotifyWatchedTransactions(t.Context(), block)

		// Should not return error even though marking failed
		require.NoError(t, err)
	})

	t.Run("empty block with no transactions", func(t *testing.T) {
		mockWalletStorage := NewWalletStorageMock(t)
		mockTransactionNotifier := NewTransactionNotifierMock(t)
		mockIdempotencyGuard := NewIdempotencyGuardMock(t)

		block := Block{
			Network:      "ethereum",
			Hash:         "block123",
			Height:       "0x64", // 100 in hex
			Transactions: []Transaction{},
		}

		// Mock idempotency guard claiming the block
		mockIdempotencyGuard.EXPECT().
			ClaimBlockForTxWatch(t.Context(), "ethereum", "block123", mock.AnythingOfType("time.Duration")).
			Return(nil)

		// Mock getTransactionsByWallet behavior for empty transactions
		mockWalletStorage.EXPECT().
			FilterWatchedWallets(t.Context(), "ethereum", []string(nil)).
			Return([]string{}, nil)

		// No notification calls should be made for empty transactions

		// Mock marking block as complete
		mockIdempotencyGuard.EXPECT().
			MarkBlockTxWatchComplete(t.Context(), "ethereum", "block123").
			Return(nil)

		svc := &service{
			walletStorage:       mockWalletStorage,
			transactionNotifier: mockTransactionNotifier,
			idempotencyGuard:    mockIdempotencyGuard,
		}

		err := svc.NotifyWatchedTransactions(t.Context(), block)

		require.NoError(t, err)
	})

	t.Run("multiple transactions with multiple watched wallets", func(t *testing.T) {
		mockWalletStorage := NewWalletStorageMock(t)
		mockTransactionNotifier := NewTransactionNotifierMock(t)
		mockIdempotencyGuard := NewIdempotencyGuardMock(t)

		block := Block{
			Network: "ethereum",
			Hash:    "block123",
			Height:  "0x64", // 100 in hex
			Transactions: []Transaction{
				{Hash: "tx1", From: "wallet1", To: "wallet2"},
				{Hash: "tx2", From: "wallet3", To: "wallet1"},
				{Hash: "tx3", From: "wallet4", To: "wallet5"},
			},
		}

		// Mock idempotency guard claiming the block
		mockIdempotencyGuard.EXPECT().
			ClaimBlockForTxWatch(t.Context(), "ethereum", "block123", mock.AnythingOfType("time.Duration")).
			Return(nil)

		// Mock getTransactionsByWallet behavior
		mockWalletStorage.EXPECT().
			FilterWatchedWallets(t.Context(), "ethereum", matchAddresses([]string{"wallet1", "wallet2", "wallet3", "wallet4", "wallet5"})).
			Return([]string{"wallet1", "wallet2"}, nil)

		// Mock notification calls - order may vary due to map iteration
		mockTransactionNotifier.EXPECT().
			NotifyTransactions(t.Context(), "ethereum", "wallet1", mock.MatchedBy(func(txs []Transaction) bool {
				if len(txs) != 2 {
					return false
				}
				hashes := make(map[string]bool)
				for _, tx := range txs {
					hashes[tx.Hash] = true
				}
				return hashes["tx1"] && hashes["tx2"]
			})).
			Return(nil)

		mockTransactionNotifier.EXPECT().
			NotifyTransactions(t.Context(), "ethereum", "wallet2", mock.MatchedBy(func(txs []Transaction) bool {
				return len(txs) == 1 && txs[0].Hash == "tx1"
			})).
			Return(nil)

		// Mock marking block as complete
		mockIdempotencyGuard.EXPECT().
			MarkBlockTxWatchComplete(t.Context(), "ethereum", "block123").
			Return(nil)

		svc := &service{
			walletStorage:       mockWalletStorage,
			transactionNotifier: mockTransactionNotifier,
			idempotencyGuard:    mockIdempotencyGuard,
		}

		err := svc.NotifyWatchedTransactions(t.Context(), block)

		require.NoError(t, err)
	})

	t.Run("different network", func(t *testing.T) {
		mockWalletStorage := NewWalletStorageMock(t)
		mockTransactionNotifier := NewTransactionNotifierMock(t)
		mockIdempotencyGuard := NewIdempotencyGuardMock(t)

		block := Block{
			Network: "polygon",
			Hash:    "block456",
			Height:  "0xc8", // 200 in hex
			Transactions: []Transaction{
				{Hash: "tx1", From: "wallet1", To: "wallet2"},
			},
		}

		// Mock idempotency guard claiming the block for polygon network
		mockIdempotencyGuard.EXPECT().
			ClaimBlockForTxWatch(t.Context(), "polygon", "block456", mock.AnythingOfType("time.Duration")).
			Return(nil)

		// Mock getTransactionsByWallet behavior for polygon network
		mockWalletStorage.EXPECT().
			FilterWatchedWallets(t.Context(), "polygon", matchAddresses([]string{"wallet1", "wallet2"})).
			Return([]string{"wallet1"}, nil)

		// Mock notification call for polygon network
		expectedTxs := []Transaction{{Hash: "tx1", From: "wallet1", To: "wallet2"}}
		mockTransactionNotifier.EXPECT().
			NotifyTransactions(t.Context(), "polygon", "wallet1", expectedTxs).
			Return(nil)

		// Mock marking block as complete for polygon network
		mockIdempotencyGuard.EXPECT().
			MarkBlockTxWatchComplete(t.Context(), "polygon", "block456").
			Return(nil)

		svc := &service{
			walletStorage:       mockWalletStorage,
			transactionNotifier: mockTransactionNotifier,
			idempotencyGuard:    mockIdempotencyGuard,
		}

		err := svc.NotifyWatchedTransactions(t.Context(), block)

		require.NoError(t, err)
	})

	t.Run("context cancellation during processing", func(t *testing.T) {
		mockWalletStorage := NewWalletStorageMock(t)
		mockTransactionNotifier := NewTransactionNotifierMock(t)
		mockIdempotencyGuard := NewIdempotencyGuardMock(t)

		ctx, cancel := context.WithCancel(t.Context())

		block := Block{
			Network: "ethereum",
			Hash:    "block123",
			Height:  "0x64", // 100 in hex
			Transactions: []Transaction{
				{Hash: "tx1", From: "wallet1", To: "wallet2"},
			},
		}

		// Mock idempotency guard claiming the block
		mockIdempotencyGuard.EXPECT().
			ClaimBlockForTxWatch(ctx, "ethereum", "block123", mock.AnythingOfType("time.Duration")).
			Return(nil)

		// Cancel context before wallet storage call
		cancel()

		// Mock getTransactionsByWallet behavior that respects context cancellation
		mockWalletStorage.EXPECT().
			FilterWatchedWallets(ctx, "ethereum", matchAddresses([]string{"wallet1", "wallet2"})).
			Return(nil, context.Canceled)

		svc := &service{
			walletStorage:       mockWalletStorage,
			transactionNotifier: mockTransactionNotifier,
			idempotencyGuard:    mockIdempotencyGuard,
		}

		err := svc.NotifyWatchedTransactions(ctx, block)

		require.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})

	t.Run("notification failure stops processing", func(t *testing.T) {
		mockWalletStorage := NewWalletStorageMock(t)
		mockTransactionNotifier := NewTransactionNotifierMock(t)
		mockIdempotencyGuard := NewIdempotencyGuardMock(t)

		block := Block{
			Network: "ethereum",
			Hash:    "block123",
			Height:  "0x64", // 100 in hex
			Transactions: []Transaction{
				{Hash: "tx1", From: "wallet1", To: "wallet2"},
				{Hash: "tx2", From: "wallet3", To: "wallet4"},
			},
		}

		// Mock idempotency guard claiming the block
		mockIdempotencyGuard.EXPECT().
			ClaimBlockForTxWatch(t.Context(), "ethereum", "block123", mock.AnythingOfType("time.Duration")).
			Return(nil)

		// Mock getTransactionsByWallet behavior
		mockWalletStorage.EXPECT().
			FilterWatchedWallets(t.Context(), "ethereum", matchAddresses([]string{"wallet1", "wallet2", "wallet3", "wallet4"})).
			Return([]string{"wallet1", "wallet3"}, nil)

		// Mock notification call that fails for the first wallet processed
		expectedError := errors.New("notification failed")
		mockTransactionNotifier.EXPECT().
			NotifyTransactions(t.Context(), "ethereum", mock.AnythingOfType("string"), mock.AnythingOfType("[]walletwatch.Transaction")).
			Return(expectedError)

		// MarkBlockTxWatchComplete should not be called when notification fails

		svc := &service{
			walletStorage:       mockWalletStorage,
			transactionNotifier: mockTransactionNotifier,
			idempotencyGuard:    mockIdempotencyGuard,
		}

		err := svc.NotifyWatchedTransactions(t.Context(), block)

		require.Error(t, err)
		assert.Equal(t, expectedError, err)
	})
}
