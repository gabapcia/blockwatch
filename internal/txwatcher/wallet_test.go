package txwatcher

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gabapcia/blockwatch/internal/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	// Initialize logger for tests to prevent nil pointer dereference
	_ = logger.Init("error")
}

func TestService_notifyWatchedWalletTransactions(t *testing.T) {
	type contextKey string

	const testContextKey contextKey = "test"

	t.Run("single wallet with multiple transactions", func(t *testing.T) {
		ctx := context.WithValue(t.Context(), testContextKey, "single_wallet")
		walletStorage := NewWalletStorageMock(t)
		transactionNotifier := NewTransactionNotifierMock(t)

		transactions := []Transaction{
			{Hash: "tx1", From: "wallet1", To: "wallet2"},
			{Hash: "tx2", From: "wallet3", To: "wallet1"},
		}

		walletTxs := map[string][]Transaction{
			"wallet1": {
				{Hash: "tx1", From: "wallet1", To: "wallet2"},
				{Hash: "tx2", From: "wallet3", To: "wallet1"},
			},
		}

		walletStorage.EXPECT().
			GetTransactionsByWallet(ctx, "ethereum", transactions).
			Return(walletTxs, nil).
			Once()

		transactionNotifier.EXPECT().
			NotifyTransactions(ctx, "ethereum", "wallet1", walletTxs["wallet1"]).
			Return(nil).
			Once()

		svc := &service{
			walletStorage:       walletStorage,
			transactionNotifier: transactionNotifier,
		}

		err := svc.notifyWatchedWalletTransactions(ctx, "ethereum", transactions)
		require.NoError(t, err)
	})

	t.Run("multiple wallets with transactions", func(t *testing.T) {
		ctx := context.WithValue(t.Context(), testContextKey, "multiple_wallets")
		walletStorage := NewWalletStorageMock(t)
		transactionNotifier := NewTransactionNotifierMock(t)

		transactions := []Transaction{
			{Hash: "tx1", From: "wallet1", To: "wallet2"},
			{Hash: "tx2", From: "wallet2", To: "wallet3"},
			{Hash: "tx3", From: "wallet4", To: "wallet1"},
		}

		walletTxs := map[string][]Transaction{
			"wallet1": {
				{Hash: "tx1", From: "wallet1", To: "wallet2"},
				{Hash: "tx3", From: "wallet4", To: "wallet1"},
			},
			"wallet2": {
				{Hash: "tx1", From: "wallet1", To: "wallet2"},
				{Hash: "tx2", From: "wallet2", To: "wallet3"},
			},
		}

		walletStorage.EXPECT().
			GetTransactionsByWallet(ctx, "polygon", transactions).
			Return(walletTxs, nil).
			Once()

		transactionNotifier.EXPECT().
			NotifyTransactions(ctx, "polygon", "wallet1", walletTxs["wallet1"]).
			Return(nil).
			Once()

		transactionNotifier.EXPECT().
			NotifyTransactions(ctx, "polygon", "wallet2", walletTxs["wallet2"]).
			Return(nil).
			Once()

		svc := &service{
			walletStorage:       walletStorage,
			transactionNotifier: transactionNotifier,
		}

		err := svc.notifyWatchedWalletTransactions(ctx, "polygon", transactions)
		require.NoError(t, err)
	})

	t.Run("no watched wallets found", func(t *testing.T) {
		ctx := context.WithValue(t.Context(), testContextKey, "no_watched_wallets")
		walletStorage := NewWalletStorageMock(t)
		transactionNotifier := NewTransactionNotifierMock(t)

		transactions := []Transaction{
			{Hash: "tx1", From: "wallet1", To: "wallet2"},
			{Hash: "tx2", From: "wallet3", To: "wallet4"},
		}

		walletStorage.EXPECT().
			GetTransactionsByWallet(ctx, "solana", transactions).
			Return(map[string][]Transaction{}, nil).
			Once()

		svc := &service{
			walletStorage:       walletStorage,
			transactionNotifier: transactionNotifier,
		}

		err := svc.notifyWatchedWalletTransactions(ctx, "solana", transactions)
		require.NoError(t, err)
	})

	t.Run("empty transactions list", func(t *testing.T) {
		ctx := context.WithValue(t.Context(), testContextKey, "empty_transactions")
		walletStorage := NewWalletStorageMock(t)
		transactionNotifier := NewTransactionNotifierMock(t)

		transactions := []Transaction{}

		walletStorage.EXPECT().
			GetTransactionsByWallet(ctx, "ethereum", transactions).
			Return(map[string][]Transaction{}, nil).
			Once()

		svc := &service{
			walletStorage:       walletStorage,
			transactionNotifier: transactionNotifier,
		}

		err := svc.notifyWatchedWalletTransactions(ctx, "ethereum", transactions)
		require.NoError(t, err)
	})

	t.Run("wallet storage database connection failed", func(t *testing.T) {
		ctx := context.WithValue(t.Context(), testContextKey, "db_connection_failed")
		walletStorage := NewWalletStorageMock(t)
		transactionNotifier := NewTransactionNotifierMock(t)

		transactions := []Transaction{
			{Hash: "tx1", From: "wallet1", To: "wallet2"},
		}

		expectedErr := errors.New("database connection failed")
		walletStorage.EXPECT().
			GetTransactionsByWallet(ctx, "ethereum", transactions).
			Return(nil, expectedErr).
			Once()

		svc := &service{
			walletStorage:       walletStorage,
			transactionNotifier: transactionNotifier,
		}

		err := svc.notifyWatchedWalletTransactions(ctx, "ethereum", transactions)
		require.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})

	t.Run("wallet storage timeout", func(t *testing.T) {
		ctx := context.WithValue(t.Context(), testContextKey, "wallet_storage_timeout")
		walletStorage := NewWalletStorageMock(t)
		transactionNotifier := NewTransactionNotifierMock(t)

		transactions := []Transaction{
			{Hash: "tx1", From: "wallet1", To: "wallet2"},
		}

		expectedErr := errors.New("operation timed out")
		walletStorage.EXPECT().
			GetTransactionsByWallet(ctx, "ethereum", transactions).
			Return(nil, expectedErr).
			Once()

		svc := &service{
			walletStorage:       walletStorage,
			transactionNotifier: transactionNotifier,
		}

		err := svc.notifyWatchedWalletTransactions(ctx, "ethereum", transactions)
		require.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})

	t.Run("notification service unavailable", func(t *testing.T) {
		ctx := context.WithValue(t.Context(), testContextKey, "notifier_unavailable")
		walletStorage := NewWalletStorageMock(t)
		transactionNotifier := NewTransactionNotifierMock(t)

		transactions := []Transaction{
			{Hash: "tx1", From: "wallet1", To: "wallet2"},
		}

		walletTxs := map[string][]Transaction{
			"wallet1": transactions,
		}

		walletStorage.EXPECT().
			GetTransactionsByWallet(ctx, "ethereum", transactions).
			Return(walletTxs, nil).
			Once()

		expectedErr := errors.New("notification service unavailable")
		transactionNotifier.EXPECT().
			NotifyTransactions(ctx, "ethereum", "wallet1", transactions).
			Return(expectedErr).
			Once()

		svc := &service{
			walletStorage:       walletStorage,
			transactionNotifier: transactionNotifier,
		}

		err := svc.notifyWatchedWalletTransactions(ctx, "ethereum", transactions)
		require.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})

	t.Run("notification fails on one wallet", func(t *testing.T) {
		ctx := context.WithValue(t.Context(), testContextKey, "notification_fails")
		walletStorage := NewWalletStorageMock(t)
		transactionNotifier := NewTransactionNotifierMock(t)

		transactions := []Transaction{
			{Hash: "tx1", From: "wallet1", To: "wallet2"},
		}

		walletTxs := map[string][]Transaction{
			"wallet1": {{Hash: "tx1", From: "wallet1", To: "wallet2"}},
		}

		walletStorage.EXPECT().
			GetTransactionsByWallet(ctx, "ethereum", transactions).
			Return(walletTxs, nil).
			Once()

		expectedErr := errors.New("rate limit exceeded")
		transactionNotifier.EXPECT().
			NotifyTransactions(ctx, "ethereum", "wallet1", walletTxs["wallet1"]).
			Return(expectedErr).
			Once()

		svc := &service{
			walletStorage:       walletStorage,
			transactionNotifier: transactionNotifier,
		}

		err := svc.notifyWatchedWalletTransactions(ctx, "ethereum", transactions)
		require.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})

	t.Run("context cancelled during wallet storage call", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel() // Cancel immediately

		walletStorage := NewWalletStorageMock(t)
		transactionNotifier := NewTransactionNotifierMock(t)

		transactions := []Transaction{
			{Hash: "tx1", From: "wallet1", To: "wallet2"},
		}

		walletStorage.EXPECT().
			GetTransactionsByWallet(ctx, "ethereum", transactions).
			Return(nil, context.Canceled).
			Once()

		svc := &service{
			walletStorage:       walletStorage,
			transactionNotifier: transactionNotifier,
		}

		err := svc.notifyWatchedWalletTransactions(ctx, "ethereum", transactions)
		require.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})

	t.Run("context cancelled during notification call", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())

		walletStorage := NewWalletStorageMock(t)
		transactionNotifier := NewTransactionNotifierMock(t)

		transactions := []Transaction{
			{Hash: "tx1", From: "wallet1", To: "wallet2"},
		}

		walletTxs := map[string][]Transaction{
			"wallet1": transactions,
		}

		walletStorage.EXPECT().
			GetTransactionsByWallet(ctx, "ethereum", transactions).
			Return(walletTxs, nil).
			Once()

		transactionNotifier.EXPECT().
			NotifyTransactions(ctx, "ethereum", "wallet1", transactions).
			Run(func(ctx context.Context, network, wallet string, txs []Transaction) {
				cancel() // Cancel during the notification call
			}).
			Return(context.Canceled).
			Once()

		svc := &service{
			walletStorage:       walletStorage,
			transactionNotifier: transactionNotifier,
		}

		err := svc.notifyWatchedWalletTransactions(ctx, "ethereum", transactions)
		require.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})

	t.Run("nil transactions slice", func(t *testing.T) {
		ctx := context.WithValue(t.Context(), testContextKey, "nil_transactions")
		walletStorage := NewWalletStorageMock(t)
		transactionNotifier := NewTransactionNotifierMock(t)

		walletStorage.EXPECT().
			GetTransactionsByWallet(ctx, "ethereum", []Transaction(nil)).
			Return(map[string][]Transaction{}, nil).
			Once()

		svc := &service{
			walletStorage:       walletStorage,
			transactionNotifier: transactionNotifier,
		}

		err := svc.notifyWatchedWalletTransactions(ctx, "ethereum", nil)
		require.NoError(t, err)
	})

	t.Run("empty network string", func(t *testing.T) {
		ctx := context.WithValue(t.Context(), testContextKey, "empty_network")
		walletStorage := NewWalletStorageMock(t)
		transactionNotifier := NewTransactionNotifierMock(t)

		transactions := []Transaction{
			{Hash: "tx1", From: "wallet1", To: "wallet2"},
		}

		walletStorage.EXPECT().
			GetTransactionsByWallet(ctx, "", transactions).
			Return(map[string][]Transaction{}, nil).
			Once()

		svc := &service{
			walletStorage:       walletStorage,
			transactionNotifier: transactionNotifier,
		}

		err := svc.notifyWatchedWalletTransactions(ctx, "", transactions)
		require.NoError(t, err)
	})

	t.Run("wallet with empty transaction list", func(t *testing.T) {
		ctx := context.WithValue(t.Context(), testContextKey, "empty_wallet_txs")
		walletStorage := NewWalletStorageMock(t)
		transactionNotifier := NewTransactionNotifierMock(t)

		transactions := []Transaction{
			{Hash: "tx1", From: "wallet1", To: "wallet2"},
		}

		// Wallet storage returns a wallet with empty transaction list
		walletTxs := map[string][]Transaction{
			"wallet1": {},
		}

		walletStorage.EXPECT().
			GetTransactionsByWallet(ctx, "ethereum", transactions).
			Return(walletTxs, nil).
			Once()

		transactionNotifier.EXPECT().
			NotifyTransactions(ctx, "ethereum", "wallet1", []Transaction{}).
			Return(nil).
			Once()

		svc := &service{
			walletStorage:       walletStorage,
			transactionNotifier: transactionNotifier,
		}

		err := svc.notifyWatchedWalletTransactions(ctx, "ethereum", transactions)
		require.NoError(t, err)
	})

	t.Run("realistic scenario with multiple networks and wallets", func(t *testing.T) {
		ctx := context.WithValue(t.Context(), testContextKey, "realistic_scenario")
		walletStorage := NewWalletStorageMock(t)
		transactionNotifier := NewTransactionNotifierMock(t)

		// Simulate a block with various transactions
		transactions := []Transaction{
			{Hash: "0x1", From: "0xabc123", To: "0xdef456"},
			{Hash: "0x2", From: "0xdef456", To: "0x789xyz"},
			{Hash: "0x3", From: "0x789xyz", To: "0xabc123"},
			{Hash: "0x4", From: "0xunwatched1", To: "0xunwatched2"},
		}

		// Only some wallets are watched
		watchedWalletTxs := map[string][]Transaction{
			"0xabc123": {
				{Hash: "0x1", From: "0xabc123", To: "0xdef456"},
				{Hash: "0x3", From: "0x789xyz", To: "0xabc123"},
			},
			"0xdef456": {
				{Hash: "0x1", From: "0xabc123", To: "0xdef456"},
				{Hash: "0x2", From: "0xdef456", To: "0x789xyz"},
			},
		}

		walletStorage.EXPECT().
			GetTransactionsByWallet(ctx, "ethereum", transactions).
			Return(watchedWalletTxs, nil).
			Once()

		// Expect notifications for each watched wallet
		transactionNotifier.EXPECT().
			NotifyTransactions(ctx, "ethereum", "0xabc123", watchedWalletTxs["0xabc123"]).
			Return(nil).
			Once()

		transactionNotifier.EXPECT().
			NotifyTransactions(ctx, "ethereum", "0xdef456", watchedWalletTxs["0xdef456"]).
			Return(nil).
			Once()

		svc := &service{
			walletStorage:       walletStorage,
			transactionNotifier: transactionNotifier,
		}

		err := svc.notifyWatchedWalletTransactions(ctx, "ethereum", transactions)
		require.NoError(t, err)
	})
}

func TestService_NotifyWatchedTransactions(t *testing.T) {
	type contextKey string

	const testContextKey contextKey = "test"

	t.Run("successful processing with idempotency guard", func(t *testing.T) {
		ctx := context.WithValue(t.Context(), testContextKey, "successful_processing")
		walletStorage := NewWalletStorageMock(t)
		transactionNotifier := NewTransactionNotifierMock(t)
		idempotencyGuard := NewIdempotencyGuardMock(t)

		block := Block{
			Network: "ethereum",
			Height:  "0x1a",
			Hash:    "0xabc123",
			Transactions: []Transaction{
				{Hash: "tx1", From: "wallet1", To: "wallet2"},
				{Hash: "tx2", From: "wallet3", To: "wallet1"},
			},
		}

		walletTxs := map[string][]Transaction{
			"wallet1": {
				{Hash: "tx1", From: "wallet1", To: "wallet2"},
				{Hash: "tx2", From: "wallet3", To: "wallet1"},
			},
		}

		// Expect idempotency guard to claim the block
		idempotencyGuard.EXPECT().
			ClaimBlockForTxWatch(ctx, "ethereum", "0xabc123", defaultMaxProcessingTime).
			Return(nil).
			Once()

		// Expect wallet storage to be called
		walletStorage.EXPECT().
			GetTransactionsByWallet(ctx, "ethereum", block.Transactions).
			Return(walletTxs, nil).
			Once()

		// Expect notification to be sent
		transactionNotifier.EXPECT().
			NotifyTransactions(ctx, "ethereum", "wallet1", walletTxs["wallet1"]).
			Return(nil).
			Once()

		// Expect idempotency guard to mark as complete
		idempotencyGuard.EXPECT().
			MarkBlockTxWatchComplete(ctx, "ethereum", "0xabc123").
			Return(nil).
			Once()

		svc := &service{
			maxProcessingTime:   defaultMaxProcessingTime,
			idempotencyGuard:    idempotencyGuard,
			walletStorage:       walletStorage,
			transactionNotifier: transactionNotifier,
		}

		err := svc.NotifyWatchedTransactions(ctx, block)
		require.NoError(t, err)
	})

	t.Run("block already in progress", func(t *testing.T) {
		ctx := context.WithValue(t.Context(), testContextKey, "already_in_progress")
		walletStorage := NewWalletStorageMock(t)
		transactionNotifier := NewTransactionNotifierMock(t)
		idempotencyGuard := NewIdempotencyGuardMock(t)

		block := Block{
			Network: "ethereum",
			Height:  "0x1a",
			Hash:    "0xabc123",
			Transactions: []Transaction{
				{Hash: "tx1", From: "wallet1", To: "wallet2"},
			},
		}

		// Expect idempotency guard to return ErrStillInProgress
		idempotencyGuard.EXPECT().
			ClaimBlockForTxWatch(ctx, "ethereum", "0xabc123", defaultMaxProcessingTime).
			Return(ErrStillInProgress).
			Once()

		svc := &service{
			maxProcessingTime:   defaultMaxProcessingTime,
			idempotencyGuard:    idempotencyGuard,
			walletStorage:       walletStorage,
			transactionNotifier: transactionNotifier,
		}

		err := svc.NotifyWatchedTransactions(ctx, block)
		require.Error(t, err)
		assert.Equal(t, ErrStillInProgress, err)
	})

	t.Run("block already finished", func(t *testing.T) {
		ctx := context.WithValue(t.Context(), testContextKey, "already_finished")
		walletStorage := NewWalletStorageMock(t)
		transactionNotifier := NewTransactionNotifierMock(t)
		idempotencyGuard := NewIdempotencyGuardMock(t)

		block := Block{
			Network: "solana",
			Height:  "0x2b",
			Hash:    "0xdef456",
			Transactions: []Transaction{
				{Hash: "tx1", From: "wallet1", To: "wallet2"},
			},
		}

		// Expect idempotency guard to return ErrAlreadyFinished
		idempotencyGuard.EXPECT().
			ClaimBlockForTxWatch(ctx, "solana", "0xdef456", defaultMaxProcessingTime).
			Return(ErrAlreadyFinished).
			Once()

		svc := &service{
			maxProcessingTime:   defaultMaxProcessingTime,
			idempotencyGuard:    idempotencyGuard,
			walletStorage:       walletStorage,
			transactionNotifier: transactionNotifier,
		}

		err := svc.NotifyWatchedTransactions(ctx, block)
		require.Error(t, err)
		assert.Equal(t, ErrAlreadyFinished, err)
	})

	t.Run("idempotency guard claim fails with other error", func(t *testing.T) {
		ctx := context.WithValue(t.Context(), testContextKey, "claim_fails")
		walletStorage := NewWalletStorageMock(t)
		transactionNotifier := NewTransactionNotifierMock(t)
		idempotencyGuard := NewIdempotencyGuardMock(t)

		block := Block{
			Network: "polygon",
			Height:  "0x3c",
			Hash:    "0x789xyz",
			Transactions: []Transaction{
				{Hash: "tx1", From: "wallet1", To: "wallet2"},
			},
		}

		expectedErr := errors.New("redis connection failed")
		idempotencyGuard.EXPECT().
			ClaimBlockForTxWatch(ctx, "polygon", "0x789xyz", defaultMaxProcessingTime).
			Return(expectedErr).
			Once()

		svc := &service{
			maxProcessingTime:   defaultMaxProcessingTime,
			idempotencyGuard:    idempotencyGuard,
			walletStorage:       walletStorage,
			transactionNotifier: transactionNotifier,
		}

		err := svc.NotifyWatchedTransactions(ctx, block)
		require.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})

	t.Run("wallet storage fails after successful claim", func(t *testing.T) {
		ctx := context.WithValue(t.Context(), testContextKey, "wallet_storage_fails")
		walletStorage := NewWalletStorageMock(t)
		transactionNotifier := NewTransactionNotifierMock(t)
		idempotencyGuard := NewIdempotencyGuardMock(t)

		block := Block{
			Network: "ethereum",
			Height:  "0x4d",
			Hash:    "0xabc789",
			Transactions: []Transaction{
				{Hash: "tx1", From: "wallet1", To: "wallet2"},
			},
		}

		// Expect idempotency guard to claim successfully
		idempotencyGuard.EXPECT().
			ClaimBlockForTxWatch(ctx, "ethereum", "0xabc789", defaultMaxProcessingTime).
			Return(nil).
			Once()

		// Expect wallet storage to fail
		expectedErr := errors.New("database connection timeout")
		walletStorage.EXPECT().
			GetTransactionsByWallet(ctx, "ethereum", block.Transactions).
			Return(nil, expectedErr).
			Once()

		svc := &service{
			maxProcessingTime:   defaultMaxProcessingTime,
			idempotencyGuard:    idempotencyGuard,
			walletStorage:       walletStorage,
			transactionNotifier: transactionNotifier,
		}

		err := svc.NotifyWatchedTransactions(ctx, block)
		require.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})

	t.Run("notification fails after successful claim", func(t *testing.T) {
		ctx := context.WithValue(t.Context(), testContextKey, "notification_fails")
		walletStorage := NewWalletStorageMock(t)
		transactionNotifier := NewTransactionNotifierMock(t)
		idempotencyGuard := NewIdempotencyGuardMock(t)

		block := Block{
			Network: "ethereum",
			Height:  "0x5e",
			Hash:    "0xdef789",
			Transactions: []Transaction{
				{Hash: "tx1", From: "wallet1", To: "wallet2"},
			},
		}

		walletTxs := map[string][]Transaction{
			"wallet1": {
				{Hash: "tx1", From: "wallet1", To: "wallet2"},
			},
		}

		// Expect idempotency guard to claim successfully
		idempotencyGuard.EXPECT().
			ClaimBlockForTxWatch(ctx, "ethereum", "0xdef789", defaultMaxProcessingTime).
			Return(nil).
			Once()

		// Expect wallet storage to succeed
		walletStorage.EXPECT().
			GetTransactionsByWallet(ctx, "ethereum", block.Transactions).
			Return(walletTxs, nil).
			Once()

		// Expect notification to fail
		expectedErr := errors.New("notification service unavailable")
		transactionNotifier.EXPECT().
			NotifyTransactions(ctx, "ethereum", "wallet1", walletTxs["wallet1"]).
			Return(expectedErr).
			Once()

		svc := &service{
			maxProcessingTime:   defaultMaxProcessingTime,
			idempotencyGuard:    idempotencyGuard,
			walletStorage:       walletStorage,
			transactionNotifier: transactionNotifier,
		}

		err := svc.NotifyWatchedTransactions(ctx, block)
		require.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})

	t.Run("mark complete fails but processing succeeds", func(t *testing.T) {
		ctx := context.WithValue(t.Context(), testContextKey, "mark_complete_fails")
		walletStorage := NewWalletStorageMock(t)
		transactionNotifier := NewTransactionNotifierMock(t)
		idempotencyGuard := NewIdempotencyGuardMock(t)

		block := Block{
			Network: "ethereum",
			Height:  "0x6f",
			Hash:    "0x123abc",
			Transactions: []Transaction{
				{Hash: "tx1", From: "wallet1", To: "wallet2"},
			},
		}

		walletTxs := map[string][]Transaction{
			"wallet1": {
				{Hash: "tx1", From: "wallet1", To: "wallet2"},
			},
		}

		// Expect idempotency guard to claim successfully
		idempotencyGuard.EXPECT().
			ClaimBlockForTxWatch(ctx, "ethereum", "0x123abc", defaultMaxProcessingTime).
			Return(nil).
			Once()

		// Expect wallet storage to succeed
		walletStorage.EXPECT().
			GetTransactionsByWallet(ctx, "ethereum", block.Transactions).
			Return(walletTxs, nil).
			Once()

		// Expect notification to succeed
		transactionNotifier.EXPECT().
			NotifyTransactions(ctx, "ethereum", "wallet1", walletTxs["wallet1"]).
			Return(nil).
			Once()

		// Expect mark complete to fail (but this should not cause the function to fail)
		markCompleteErr := errors.New("failed to mark complete")
		idempotencyGuard.EXPECT().
			MarkBlockTxWatchComplete(ctx, "ethereum", "0x123abc").
			Return(markCompleteErr).
			Once()

		svc := &service{
			maxProcessingTime:   defaultMaxProcessingTime,
			idempotencyGuard:    idempotencyGuard,
			walletStorage:       walletStorage,
			transactionNotifier: transactionNotifier,
		}

		// Should succeed despite mark complete failing
		err := svc.NotifyWatchedTransactions(ctx, block)
		require.NoError(t, err)
	})

	t.Run("empty block with no transactions", func(t *testing.T) {
		ctx := context.WithValue(t.Context(), testContextKey, "empty_block")
		walletStorage := NewWalletStorageMock(t)
		transactionNotifier := NewTransactionNotifierMock(t)
		idempotencyGuard := NewIdempotencyGuardMock(t)

		block := Block{
			Network:      "ethereum",
			Height:       "0x7a",
			Hash:         "0x456def",
			Transactions: []Transaction{},
		}

		// Expect idempotency guard to claim successfully
		idempotencyGuard.EXPECT().
			ClaimBlockForTxWatch(ctx, "ethereum", "0x456def", defaultMaxProcessingTime).
			Return(nil).
			Once()

		// Expect wallet storage to be called with empty transactions
		walletStorage.EXPECT().
			GetTransactionsByWallet(ctx, "ethereum", []Transaction{}).
			Return(map[string][]Transaction{}, nil).
			Once()

		// Expect mark complete to succeed
		idempotencyGuard.EXPECT().
			MarkBlockTxWatchComplete(ctx, "ethereum", "0x456def").
			Return(nil).
			Once()

		svc := &service{
			maxProcessingTime:   defaultMaxProcessingTime,
			idempotencyGuard:    idempotencyGuard,
			walletStorage:       walletStorage,
			transactionNotifier: transactionNotifier,
		}

		err := svc.NotifyWatchedTransactions(ctx, block)
		require.NoError(t, err)
	})

	t.Run("context cancelled during processing", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())

		walletStorage := NewWalletStorageMock(t)
		transactionNotifier := NewTransactionNotifierMock(t)
		idempotencyGuard := NewIdempotencyGuardMock(t)

		block := Block{
			Network: "ethereum",
			Height:  "0x8b",
			Hash:    "0x789def",
			Transactions: []Transaction{
				{Hash: "tx1", From: "wallet1", To: "wallet2"},
			},
		}

		// Expect idempotency guard to claim successfully
		idempotencyGuard.EXPECT().
			ClaimBlockForTxWatch(ctx, "ethereum", "0x789def", defaultMaxProcessingTime).
			Return(nil).
			Once()

		// Cancel context during wallet storage call
		walletStorage.EXPECT().
			GetTransactionsByWallet(ctx, "ethereum", block.Transactions).
			Run(func(ctx context.Context, network string, txs []Transaction) {
				cancel()
			}).
			Return(nil, context.Canceled).
			Once()

		svc := &service{
			maxProcessingTime:   defaultMaxProcessingTime,
			idempotencyGuard:    idempotencyGuard,
			walletStorage:       walletStorage,
			transactionNotifier: transactionNotifier,
		}

		err := svc.NotifyWatchedTransactions(ctx, block)
		require.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})

	t.Run("multiple wallets with different networks", func(t *testing.T) {
		ctx := context.WithValue(t.Context(), testContextKey, "multiple_wallets_different_networks")
		walletStorage := NewWalletStorageMock(t)
		transactionNotifier := NewTransactionNotifierMock(t)
		idempotencyGuard := NewIdempotencyGuardMock(t)

		block := Block{
			Network: "polygon",
			Height:  "0x9c",
			Hash:    "0xabc456",
			Transactions: []Transaction{
				{Hash: "tx1", From: "0xwallet1", To: "0xwallet2"},
				{Hash: "tx2", From: "0xwallet2", To: "0xwallet3"},
				{Hash: "tx3", From: "0xwallet4", To: "0xwallet1"},
			},
		}

		walletTxs := map[string][]Transaction{
			"0xwallet1": {
				{Hash: "tx1", From: "0xwallet1", To: "0xwallet2"},
				{Hash: "tx3", From: "0xwallet4", To: "0xwallet1"},
			},
			"0xwallet2": {
				{Hash: "tx1", From: "0xwallet1", To: "0xwallet2"},
				{Hash: "tx2", From: "0xwallet2", To: "0xwallet3"},
			},
		}

		// Expect idempotency guard to claim successfully
		idempotencyGuard.EXPECT().
			ClaimBlockForTxWatch(ctx, "polygon", "0xabc456", defaultMaxProcessingTime).
			Return(nil).
			Once()

		// Expect wallet storage to succeed
		walletStorage.EXPECT().
			GetTransactionsByWallet(ctx, "polygon", block.Transactions).
			Return(walletTxs, nil).
			Once()

		// Expect notifications for both wallets
		transactionNotifier.EXPECT().
			NotifyTransactions(ctx, "polygon", "0xwallet1", walletTxs["0xwallet1"]).
			Return(nil).
			Once()

		transactionNotifier.EXPECT().
			NotifyTransactions(ctx, "polygon", "0xwallet2", walletTxs["0xwallet2"]).
			Return(nil).
			Once()

		// Expect mark complete to succeed
		idempotencyGuard.EXPECT().
			MarkBlockTxWatchComplete(ctx, "polygon", "0xabc456").
			Return(nil).
			Once()

		svc := &service{
			maxProcessingTime:   defaultMaxProcessingTime,
			idempotencyGuard:    idempotencyGuard,
			walletStorage:       walletStorage,
			transactionNotifier: transactionNotifier,
		}

		err := svc.NotifyWatchedTransactions(ctx, block)
		require.NoError(t, err)
	})

	t.Run("custom max processing time", func(t *testing.T) {
		ctx := context.WithValue(t.Context(), testContextKey, "custom_max_processing_time")
		walletStorage := NewWalletStorageMock(t)
		transactionNotifier := NewTransactionNotifierMock(t)
		idempotencyGuard := NewIdempotencyGuardMock(t)

		customMaxProcessingTime := 10 * time.Minute

		block := Block{
			Network: "ethereum",
			Height:  "0xad",
			Hash:    "0xdef123",
			Transactions: []Transaction{
				{Hash: "tx1", From: "wallet1", To: "wallet2"},
			},
		}

		walletTxs := map[string][]Transaction{
			"wallet1": {
				{Hash: "tx1", From: "wallet1", To: "wallet2"},
			},
		}

		// Expect idempotency guard to be called with custom max processing time
		idempotencyGuard.EXPECT().
			ClaimBlockForTxWatch(ctx, "ethereum", "0xdef123", customMaxProcessingTime).
			Return(nil).
			Once()

		walletStorage.EXPECT().
			GetTransactionsByWallet(ctx, "ethereum", block.Transactions).
			Return(walletTxs, nil).
			Once()

		transactionNotifier.EXPECT().
			NotifyTransactions(ctx, "ethereum", "wallet1", walletTxs["wallet1"]).
			Return(nil).
			Once()

		idempotencyGuard.EXPECT().
			MarkBlockTxWatchComplete(ctx, "ethereum", "0xdef123").
			Return(nil).
			Once()

		svc := &service{
			maxProcessingTime:   customMaxProcessingTime,
			idempotencyGuard:    idempotencyGuard,
			walletStorage:       walletStorage,
			transactionNotifier: transactionNotifier,
		}

		err := svc.NotifyWatchedTransactions(ctx, block)
		require.NoError(t, err)
	})
}
