package ethereum

import "github.com/gabapcia/blockwatch"

type transactionResponse struct {
	Hash  string `json:"hash"`
	From  string `json:"from"`
	To    string `json:"to"`
	Value string `json:"value"`
}

func (t transactionResponse) toDomain(blockNumber int64) blockwatch.Transaction {
	return blockwatch.Transaction{
		BlockNumber: blockNumber,
		Hash:        t.Hash,
		From:        t.From,
		To:          t.To,
		Value:       t.Value,
	}
}
