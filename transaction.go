package blockwatch

import "context"

type Transaction struct {
	BlockNumber int64
	Hash        string
	From        string
	To          string
	Value       string
}

type TransactionStorage interface {
	SaveAddressTransactions(ctx context.Context, address string, transactions []Transaction) error
	ListAddressTransactions(ctx context.Context, address string) ([]Transaction, error)
}
