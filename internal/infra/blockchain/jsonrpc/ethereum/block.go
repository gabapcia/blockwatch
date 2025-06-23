// Package ethereum implements the watcher.Blockchain interface for Ethereum-compatible nodes.
// It uses a JSON-RPC client to poll for new blocks and stream blockchain events.
package ethereum

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gabapcia/blockwatch/internal/pkg/types"
	"github.com/gabapcia/blockwatch/internal/watcher"
)

const (
	// averageNumberOfTransactionsPerBlock defines the default buffer size for the event channel.
	averageNumberOfTransactionsPerBlock = 200

	// averageBlockTime defines the expected time between blocks in Ethereum.
	averageBlockTime = 12 * time.Second
)

type (
	// TransactionResponse represents a raw transaction object returned by the Ethereum JSON-RPC API.
	TransactionResponse struct {
		Type                 string `json:"type"`
		ChainID              string `json:"chainId"`
		Nonce                string `json:"nonce"`
		Gas                  string `json:"gas"`
		MaxFeePerGas         string `json:"maxFeePerGas"`
		MaxPriorityFeePerGas string `json:"maxPriorityFeePerGas"`
		To                   string `json:"to"`
		Value                string `json:"value"`
		Input                string `json:"input"`
		R                    string `json:"r"`
		S                    string `json:"s"`
		YParity              string `json:"yParity"`
		V                    string `json:"v"`
		Hash                 string `json:"hash"`
		BlockHash            string `json:"blockHash"`
		BlockNumber          string `json:"blockNumber"`
		TransactionIndex     string `json:"transactionIndex"`
		From                 string `json:"from"`
		GasPrice             string `json:"gasPrice"`
		AccessList           []struct {
			Address     string   `json:"address"`
			StorageKeys []string `json:"storageKeys"`
		} `json:"accessList"`
	}

	// WithdrawalResponse represents a withdrawal entry included in a block (e.g., for validators).
	WithdrawalResponse struct {
		Index          string `json:"index"`
		ValidatorIndex string `json:"validatorIndex"`
		Address        string `json:"address"`
		Amount         string `json:"amount"`
	}

	// BlockResponse represents the full structure of a block returned by the Ethereum JSON-RPC API.
	BlockResponse struct {
		Hash                  string                `json:"hash"`
		ParentHash            string                `json:"parentHash"`
		Sha3Uncles            string                `json:"sha3Uncles"`
		Miner                 string                `json:"miner"`
		StateRoot             string                `json:"stateRoot"`
		TransactionsRoot      string                `json:"transactionsRoot"`
		ReceiptsRoot          string                `json:"receiptsRoot"`
		LogsBloom             string                `json:"logsBloom"`
		Difficulty            string                `json:"difficulty"`
		Number                types.Hex             `json:"number"`
		GasLimit              string                `json:"gasLimit"`
		GasUsed               string                `json:"gasUsed"`
		Timestamp             string                `json:"timestamp"`
		ExtraData             string                `json:"extraData"`
		MixHash               string                `json:"mixHash"`
		Nonce                 string                `json:"nonce"`
		BaseFeePerGas         string                `json:"baseFeePerGas"`
		WithdrawalsRoot       string                `json:"withdrawalsRoot"`
		BlobGasUsed           string                `json:"blobGasUsed"`
		ExcessBlobGas         string                `json:"excessBlobGas"`
		ParentBeaconBlockRoot string                `json:"parentBeaconBlockRoot"`
		RequestsHash          string                `json:"requestsHash"`
		Size                  string                `json:"size"`
		Uncles                []string              `json:"uncles"`
		Transactions          []TransactionResponse `json:"transactions"`
		Withdrawals           []WithdrawalResponse  `json:"withdrawals"`
	}
)

// toWatcherTransaction converts a TransactionResponse to a simplified watcher.Transaction.
func (t TransactionResponse) toWatcherTransaction() watcher.Transaction {
	return watcher.Transaction{
		Hash: t.Hash,
		From: t.From,
		To:   t.To,
	}
}

// toWatcherBlock converts a BlockResponse to a simplified watcher.Block.
func (b BlockResponse) toWatcherBlock() watcher.Block {
	transactions := make([]watcher.Transaction, len(b.Transactions))
	for i, t := range b.Transactions {
		transactions[i] = t.toWatcherTransaction()
	}

	return watcher.Block{
		Number:       b.Number,
		Hash:         b.Hash,
		Transactions: transactions,
	}
}

// getLatestBlockNumber fetches the latest block number from the Ethereum node.
func (c *client) getLatestBlockNumber(ctx context.Context) (types.Hex, error) {
	data, err := c.conn.Fetch(ctx, "eth_blockNumber")
	if err != nil {
		return "", err
	}

	var blockNumber types.Hex
	return blockNumber, json.Unmarshal(data, &blockNumber)
}

// getBlockByNumber retrieves a full block by its number.
func (c *client) getBlockByNumber(ctx context.Context, blockNumber types.Hex) (BlockResponse, error) {
	data, err := c.conn.Fetch(ctx, "eth_getBlockByNumber", blockNumber, true)
	if err != nil {
		return BlockResponse{}, err
	}

	var blockResponse BlockResponse
	return blockResponse, json.Unmarshal(data, &blockResponse)
}

// pollNewBlocks fetches all new blocks from fromBlockNumber up to the latest block number,
// and sends them as BlockchainEvent values to the eventsCh channel.
//
// First, it retrieves the latest block number using eth_blockNumber. If this fails,
// it sends a BlockchainEvent containing the error and returns fromBlockNumber unchanged.
//
// If fromBlockNumber is equal to the latest block number, the function returns immediately
// without emitting any events, as there are no new blocks to process.
//
// Otherwise, for each block in the range [fromBlockNumber, latestBlockNumber], the function:
//   - Fetches the block using eth_getBlockByNumber
//   - Converts the block to the simplified watcher.Block format
//   - Emits a BlockchainEvent to the channel, including the block and any fetch error
//
// This function performs a tight loop over the range and does not include delay or throttling.
// It is intended to be invoked periodically (e.g., by a time.Ticker) to poll for new blocks.
//
// Returns the latest block number that was successfully queried, regardless of errors on individual blocks.
func (c *client) pollNewBlocks(ctx context.Context, fromBlockNumber types.Hex, eventsCh chan<- watcher.BlockchainEvent) types.Hex {
	latestBlockNumber, err := c.getLatestBlockNumber(ctx)
	if err != nil {
		eventsCh <- watcher.BlockchainEvent{Error: err}
		return fromBlockNumber
	}

	if fromBlockNumber == latestBlockNumber {
		return latestBlockNumber
	}

	currentBlockNumber := fromBlockNumber
	for currentBlockNumber.Int() <= latestBlockNumber.Int() {
		block, err := c.getBlockByNumber(ctx, currentBlockNumber)

		eventsCh <- watcher.BlockchainEvent{
			NewBlock: block.toWatcherBlock(),
			Error:    err,
		}

		currentBlockNumber = currentBlockNumber.Add(1)
	}

	return latestBlockNumber
}

// Listen implements the watcher.Blockchain interface.
// It starts polling the Ethereum node for new blocks and emits BlockchainEvent values.
// If startFromBlockNumber is empty, it starts from the latest block at the time of invocation.
// The returned channel will be closed when the context is canceled.
func (c *client) Listen(ctx context.Context, startFromBlockNumber types.Hex) (<-chan watcher.BlockchainEvent, error) {
	if startFromBlockNumber == "" {
		latestBlockNumber, err := c.getLatestBlockNumber(ctx)
		if err != nil {
			return nil, err
		}

		startFromBlockNumber = latestBlockNumber
	}

	eventsCh := make(chan watcher.BlockchainEvent, averageNumberOfTransactionsPerBlock)
	go func() {
		defer close(eventsCh)

		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(averageBlockTime):
				startFromBlockNumber = c.pollNewBlocks(ctx, startFromBlockNumber, eventsCh)
			}
		}
	}()

	return eventsCh, nil
}
