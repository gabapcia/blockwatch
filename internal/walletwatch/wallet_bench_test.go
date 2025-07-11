package walletwatch

import (
	"context"
	"fmt"
	"testing"

	"github.com/gabapcia/blockwatch/internal/pkg/types"
)

// mockWalletStorage is a simple mock implementation for benchmarking
type mockWalletStorage struct {
	watchedWallets map[string][]string // network -> watched addresses
}

func newMockWalletStorage() *mockWalletStorage {
	return &mockWalletStorage{
		watchedWallets: make(map[string][]string),
	}
}

func (m *mockWalletStorage) addWatchedWallets(network string, addresses []string) {
	m.watchedWallets[network] = append(m.watchedWallets[network], addresses...)
}

func (m *mockWalletStorage) FilterWatchedWallets(ctx context.Context, network string, addresses []string) ([]string, error) {
	watched := m.watchedWallets[network]
	watchedSet := types.NewSet(watched...)

	var result []string
	for _, addr := range addresses {
		if _, exists := watchedSet[addr]; exists {
			result = append(result, addr)
		}
	}
	return result, nil
}

// generateTransactions creates test transactions for benchmarking
func generateTransactions(numTxs int, numUniqueAddresses int) []Transaction {
	txs := make([]Transaction, numTxs)

	// Generate a pool of addresses to use
	addresses := make([]string, numUniqueAddresses)
	for i := 0; i < numUniqueAddresses; i++ {
		addresses[i] = fmt.Sprintf("0x%040d", i)
	}

	// Create transactions using addresses from the pool
	for i := 0; i < numTxs; i++ {
		fromIdx := i % numUniqueAddresses
		toIdx := (i + 1) % numUniqueAddresses

		txs[i] = Transaction{
			Hash: fmt.Sprintf("0x%064d", i),
			From: addresses[fromIdx],
			To:   addresses[toIdx],
		}
	}

	return txs
}

// setupBenchmarkService creates a service with mock dependencies for benchmarking
func setupBenchmarkService(watchedAddressesCount int, totalAddressesCount int) (*service, []Transaction) {
	storage := newMockWalletStorage()

	// Add some watched addresses (subset of total addresses)
	watchedAddresses := make([]string, watchedAddressesCount)
	for i := 0; i < watchedAddressesCount; i++ {
		watchedAddresses[i] = fmt.Sprintf("0x%040d", i)
	}
	storage.addWatchedWallets("ethereum", watchedAddresses)

	// Create service
	svc := &service{
		walletStorage: storage,
	}

	// Generate transactions
	txs := generateTransactions(1000, totalAddressesCount)

	return svc, txs
}

// Benchmark getTransactionsByWallet with small dataset
func BenchmarkGetTransactionsByWallet_Small(b *testing.B) {
	svc, txs := setupBenchmarkService(10, 50) // 10 watched out of 50 total addresses
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := svc.getTransactionsByWallet(ctx, "ethereum", txs)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark getTransactionsByWalletV2 with small dataset
func BenchmarkGetTransactionsByWalletV2_Small(b *testing.B) {
	svc, txs := setupBenchmarkService(10, 50) // 10 watched out of 50 total addresses
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := svc.getTransactionsByWalletV2(ctx, "ethereum", txs)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark getTransactionsByWallet with medium dataset
func BenchmarkGetTransactionsByWallet_Medium(b *testing.B) {
	svc, txs := setupBenchmarkService(100, 500) // 100 watched out of 500 total addresses
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := svc.getTransactionsByWallet(ctx, "ethereum", txs)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark getTransactionsByWalletV2 with medium dataset
func BenchmarkGetTransactionsByWalletV2_Medium(b *testing.B) {
	svc, txs := setupBenchmarkService(100, 500) // 100 watched out of 500 total addresses
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := svc.getTransactionsByWalletV2(ctx, "ethereum", txs)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark getTransactionsByWallet with large dataset
func BenchmarkGetTransactionsByWallet_Large(b *testing.B) {
	svc, txs := setupBenchmarkService(1000, 5000) // 1000 watched out of 5000 total addresses
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := svc.getTransactionsByWallet(ctx, "ethereum", txs)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark getTransactionsByWalletV2 with large dataset
func BenchmarkGetTransactionsByWalletV2_Large(b *testing.B) {
	svc, txs := setupBenchmarkService(1000, 5000) // 1000 watched out of 5000 total addresses
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := svc.getTransactionsByWalletV2(ctx, "ethereum", txs)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark with varying transaction counts
func BenchmarkGetTransactionsByWallet_VaryingTxCount(b *testing.B) {
	txCounts := []int{100, 500, 1000, 5000, 10000}

	for _, txCount := range txCounts {
		b.Run(fmt.Sprintf("TxCount_%d", txCount), func(b *testing.B) {
			storage := newMockWalletStorage()
			watchedAddresses := make([]string, 100)
			for i := 0; i < 100; i++ {
				watchedAddresses[i] = fmt.Sprintf("0x%040d", i)
			}
			storage.addWatchedWallets("ethereum", watchedAddresses)

			svc := &service{walletStorage: storage}
			txs := generateTransactions(txCount, 500)
			ctx := context.Background()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := svc.getTransactionsByWallet(ctx, "ethereum", txs)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkGetTransactionsByWalletV2_VaryingTxCount(b *testing.B) {
	txCounts := []int{100, 500, 1000, 5000, 10000}

	for _, txCount := range txCounts {
		b.Run(fmt.Sprintf("TxCount_%d", txCount), func(b *testing.B) {
			storage := newMockWalletStorage()
			watchedAddresses := make([]string, 100)
			for i := 0; i < 100; i++ {
				watchedAddresses[i] = fmt.Sprintf("0x%040d", i)
			}
			storage.addWatchedWallets("ethereum", watchedAddresses)

			svc := &service{walletStorage: storage}
			txs := generateTransactions(txCount, 500)
			ctx := context.Background()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := svc.getTransactionsByWalletV2(ctx, "ethereum", txs)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// Benchmark with varying watched wallet ratios
func BenchmarkGetTransactionsByWallet_VaryingWatchedRatio(b *testing.B) {
	watchedRatios := []float64{0.01, 0.05, 0.1, 0.2, 0.5} // 1%, 5%, 10%, 20%, 50%
	totalAddresses := 1000

	for _, ratio := range watchedRatios {
		watchedCount := int(float64(totalAddresses) * ratio)
		b.Run(fmt.Sprintf("WatchedRatio_%.0f%%", ratio*100), func(b *testing.B) {
			storage := newMockWalletStorage()
			watchedAddresses := make([]string, watchedCount)
			for i := 0; i < watchedCount; i++ {
				watchedAddresses[i] = fmt.Sprintf("0x%040d", i)
			}
			storage.addWatchedWallets("ethereum", watchedAddresses)

			svc := &service{walletStorage: storage}
			txs := generateTransactions(2000, totalAddresses)
			ctx := context.Background()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := svc.getTransactionsByWallet(ctx, "ethereum", txs)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkGetTransactionsByWalletV2_VaryingWatchedRatio(b *testing.B) {
	watchedRatios := []float64{0.01, 0.05, 0.1, 0.2, 0.5} // 1%, 5%, 10%, 20%, 50%
	totalAddresses := 1000

	for _, ratio := range watchedRatios {
		watchedCount := int(float64(totalAddresses) * ratio)
		b.Run(fmt.Sprintf("WatchedRatio_%.0f%%", ratio*100), func(b *testing.B) {
			storage := newMockWalletStorage()
			watchedAddresses := make([]string, watchedCount)
			for i := 0; i < watchedCount; i++ {
				watchedAddresses[i] = fmt.Sprintf("0x%040d", i)
			}
			storage.addWatchedWallets("ethereum", watchedAddresses)

			svc := &service{walletStorage: storage}
			txs := generateTransactions(2000, totalAddresses)
			ctx := context.Background()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := svc.getTransactionsByWalletV2(ctx, "ethereum", txs)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// Memory allocation benchmarks
func BenchmarkGetTransactionsByWallet_MemAlloc(b *testing.B) {
	svc, txs := setupBenchmarkService(100, 500)
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := svc.getTransactionsByWallet(ctx, "ethereum", txs)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetTransactionsByWalletV2_MemAlloc(b *testing.B) {
	svc, txs := setupBenchmarkService(100, 500)
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := svc.getTransactionsByWalletV2(ctx, "ethereum", txs)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Solana-scale benchmarks (25k transactions, up to 50k unique addresses)
func BenchmarkGetTransactionsByWallet_SolanaScale_Small(b *testing.B) {
	// 1% watched wallets (500 out of 50k addresses), 25k transactions
	storage := newMockWalletStorage()
	watchedAddresses := make([]string, 500)
	for i := 0; i < 500; i++ {
		watchedAddresses[i] = fmt.Sprintf("0x%040d", i)
	}
	storage.addWatchedWallets("solana", watchedAddresses)

	svc := &service{walletStorage: storage}
	txs := generateTransactions(25000, 50000)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := svc.getTransactionsByWallet(ctx, "solana", txs)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetTransactionsByWalletV2_SolanaScale_Small(b *testing.B) {
	// 1% watched wallets (500 out of 50k addresses), 25k transactions
	storage := newMockWalletStorage()
	watchedAddresses := make([]string, 500)
	for i := 0; i < 500; i++ {
		watchedAddresses[i] = fmt.Sprintf("0x%040d", i)
	}
	storage.addWatchedWallets("solana", watchedAddresses)

	svc := &service{walletStorage: storage}
	txs := generateTransactions(25000, 50000)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := svc.getTransactionsByWalletV2(ctx, "solana", txs)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetTransactionsByWallet_SolanaScale_Medium(b *testing.B) {
	// 5% watched wallets (2500 out of 50k addresses), 25k transactions
	storage := newMockWalletStorage()
	watchedAddresses := make([]string, 2500)
	for i := 0; i < 2500; i++ {
		watchedAddresses[i] = fmt.Sprintf("0x%040d", i)
	}
	storage.addWatchedWallets("solana", watchedAddresses)

	svc := &service{walletStorage: storage}
	txs := generateTransactions(25000, 50000)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := svc.getTransactionsByWallet(ctx, "solana", txs)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetTransactionsByWalletV2_SolanaScale_Medium(b *testing.B) {
	// 5% watched wallets (2500 out of 50k addresses), 25k transactions
	storage := newMockWalletStorage()
	watchedAddresses := make([]string, 2500)
	for i := 0; i < 2500; i++ {
		watchedAddresses[i] = fmt.Sprintf("0x%040d", i)
	}
	storage.addWatchedWallets("solana", watchedAddresses)

	svc := &service{walletStorage: storage}
	txs := generateTransactions(25000, 50000)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := svc.getTransactionsByWalletV2(ctx, "solana", txs)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetTransactionsByWallet_SolanaScale_Large(b *testing.B) {
	// 10% watched wallets (5000 out of 50k addresses), 25k transactions
	storage := newMockWalletStorage()
	watchedAddresses := make([]string, 5000)
	for i := 0; i < 5000; i++ {
		watchedAddresses[i] = fmt.Sprintf("0x%040d", i)
	}
	storage.addWatchedWallets("solana", watchedAddresses)

	svc := &service{walletStorage: storage}
	txs := generateTransactions(25000, 50000)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := svc.getTransactionsByWallet(ctx, "solana", txs)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetTransactionsByWalletV2_SolanaScale_Large(b *testing.B) {
	// 10% watched wallets (5000 out of 50k addresses), 25k transactions
	storage := newMockWalletStorage()
	watchedAddresses := make([]string, 5000)
	for i := 0; i < 5000; i++ {
		watchedAddresses[i] = fmt.Sprintf("0x%040d", i)
	}
	storage.addWatchedWallets("solana", watchedAddresses)

	svc := &service{walletStorage: storage}
	txs := generateTransactions(25000, 50000)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := svc.getTransactionsByWalletV2(ctx, "solana", txs)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Extreme scale benchmarks (100k addresses)
func BenchmarkGetTransactionsByWallet_ExtremeScale(b *testing.B) {
	// 1% watched wallets (1000 out of 100k addresses), 25k transactions
	storage := newMockWalletStorage()
	watchedAddresses := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		watchedAddresses[i] = fmt.Sprintf("0x%040d", i)
	}
	storage.addWatchedWallets("solana", watchedAddresses)

	svc := &service{walletStorage: storage}
	txs := generateTransactions(25000, 100000)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := svc.getTransactionsByWallet(ctx, "solana", txs)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetTransactionsByWalletV2_ExtremeScale(b *testing.B) {
	// 1% watched wallets (1000 out of 100k addresses), 25k transactions
	storage := newMockWalletStorage()
	watchedAddresses := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		watchedAddresses[i] = fmt.Sprintf("0x%040d", i)
	}
	storage.addWatchedWallets("solana", watchedAddresses)

	svc := &service{walletStorage: storage}
	txs := generateTransactions(25000, 100000)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := svc.getTransactionsByWalletV2(ctx, "solana", txs)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Memory allocation benchmarks for Solana scale
func BenchmarkGetTransactionsByWallet_SolanaScale_MemAlloc(b *testing.B) {
	storage := newMockWalletStorage()
	watchedAddresses := make([]string, 2500)
	for i := 0; i < 2500; i++ {
		watchedAddresses[i] = fmt.Sprintf("0x%040d", i)
	}
	storage.addWatchedWallets("solana", watchedAddresses)

	svc := &service{walletStorage: storage}
	txs := generateTransactions(25000, 50000)
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := svc.getTransactionsByWallet(ctx, "solana", txs)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetTransactionsByWalletV2_SolanaScale_MemAlloc(b *testing.B) {
	storage := newMockWalletStorage()
	watchedAddresses := make([]string, 2500)
	for i := 0; i < 2500; i++ {
		watchedAddresses[i] = fmt.Sprintf("0x%040d", i)
	}
	storage.addWatchedWallets("solana", watchedAddresses)

	svc := &service{walletStorage: storage}
	txs := generateTransactions(25000, 50000)
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := svc.getTransactionsByWalletV2(ctx, "solana", txs)
		if err != nil {
			b.Fatal(err)
		}
	}
}
