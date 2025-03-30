package ethereum

import "github.com/gabapcia/blockwatch"

// transactionResponse represents the JSON structure returned from an Ethereum node
// for a transaction. It captures basic details such as the transaction hash,
// sender, receiver, and value.
type transactionResponse struct {
	// Hash is the unique identifier of the transaction.
	Hash string `json:"hash"`
	// From is the address that initiated the transaction.
	From string `json:"from"`
	// To is the address that received the transaction.
	To string `json:"to"`
	// Value represents the amount transferred in the transaction.
	Value string `json:"value"`
}

// toDomain converts a transactionResponse to a blockwatch.Transaction domain model.
// The blockNumber parameter is used to set the block number associated with the transaction.
func (t transactionResponse) toDomain(blockNumber int64) blockwatch.Transaction {
	return blockwatch.Transaction{
		BlockNumber: blockNumber,
		Hash:        t.Hash,
		From:        t.From,
		To:          t.To,
		Value:       t.Value,
	}
}
