// Package ethereum implements the chainwatch.Blockchain interface for Ethereum-compatible nodes.
// It uses a JSON-RPC client to poll for new blocks and stream blockchain events.
package ethereum

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gabapcia/blockwatch/internal/chainwatch"
	"github.com/gabapcia/blockwatch/internal/pkg/types"
)

const (
	// eventsChannelBufferSize is the buffer size for the BlockchainEvent channel.
	eventsChannelBufferSize = 200

	// averageBlockTime is the expected time between Ethereum blocks.
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

// toWatcherTransaction converts a TransactionResponse into a chainwatch.Transaction.
func (t TransactionResponse) toWatcherTransaction() chainwatch.Transaction {
	return chainwatch.Transaction{
		Hash: t.Hash,
		From: t.From,
		To:   t.To,
	}
}

// toWatcherBlock converts a BlockResponse into a chainwatch.Block.
func (b BlockResponse) toWatcherBlock() chainwatch.Block {
	out := make([]chainwatch.Transaction, len(b.Transactions))
	for i, tx := range b.Transactions {
		out[i] = tx.toWatcherTransaction()
	}

	return chainwatch.Block{
		Height:       b.Number,
		Hash:         b.Hash,
		Transactions: out,
	}
}

// getLatestBlockNumber fetches the latest block height from the node using eth_blockNumber.
func (c *client) getLatestBlockNumber(ctx context.Context) (types.Hex, error) {
	data, err := c.conn.Fetch(ctx, "eth_blockNumber")
	if err != nil {
		return "", err
	}
	var height types.Hex
	return height, json.Unmarshal(data, &height)
}

// getBlockByNumber retrieves a BlockResponse for the given height using eth_getBlockByNumber.
func (c *client) getBlockByNumber(ctx context.Context, height types.Hex) (BlockResponse, error) {
	data, err := c.conn.Fetch(ctx, "eth_getBlockByNumber", height, true)
	if err != nil {
		return BlockResponse{}, err
	}
	var resp BlockResponse
	return resp, json.Unmarshal(data, &resp)
}

// pollNewBlocks retrieves and emits all new blocks from the starting height up to the current latest height.
//
// The function proceeds as follows:
//  1. Fetches the current latest height via getLatestBlockNumber.
//     - On error, it emits a single BlockchainEvent with:
//     • Height: fromHeight
//     • Err: the retrieval error
//     and returns fromHeight unchanged, allowing the caller to retry later.
//  2. If fromHeight is greater than or equal to the latest height, there are
//     no new blocks to process. It returns fromHeight immediately, emitting no events.
//  3. Otherwise, for each height h in the inclusive range [fromHeight, latestHeight]:
//     a. Calls getBlockByNumber(ctx, h).
//     b. Converts the raw BlockResponse to chainwatch.Block via toWatcherBlock.
//     c. Emits a BlockchainEvent on eventsCh containing:
//     • Height: h
//     • Block: the converted block (zero-value if an error occurred)
//     • Err: any fetch or unmarshal error for that block.
//  4. After emitting events for all heights up to latestHeight, it returns
//     latestHeight + 1, indicating the next height to begin from on the
//     following polling iteration.
//
// This function itself does not perform any waiting or backoff; it is intended
// to be invoked periodically (for example, by a time.Ticker in Subscribe),
// which handles scheduling and context cancellation.
func (c *client) pollNewBlocks(ctx context.Context, fromHeight types.Hex, eventsCh chan<- chainwatch.BlockchainEvent) types.Hex {
	latestHeight, err := c.getLatestBlockNumber(ctx)
	if err != nil {
		eventsCh <- chainwatch.BlockchainEvent{Height: fromHeight, Err: err}
		return fromHeight
	}

	if fromHeight >= latestHeight {
		return fromHeight
	}

	for h := fromHeight; h.Int() <= latestHeight.Int(); h = h.Add(1) {
		blockResp, err := c.getBlockByNumber(ctx, h)
		eventsCh <- chainwatch.BlockchainEvent{
			Height: h,
			Block:  blockResp.toWatcherBlock(),
			Err:    err,
		}
	}

	return latestHeight.Add(1)
}

// FetchBlockByHeight retrieves a single block at the specified height.
// It returns the converted chainwatch.Block or an error if fetching fails.
func (c *client) FetchBlockByHeight(ctx context.Context, height types.Hex) (chainwatch.Block, error) {
	resp, err := c.getBlockByNumber(ctx, height)
	if err != nil {
		return chainwatch.Block{}, err
	}

	return resp.toWatcherBlock(), nil
}

// Subscribe begins streaming new blocks starting at fromHeight (inclusive).
// If fromHeight is empty, it first fetches the latest height and starts from there.
// It returns a receive-only channel of BlockchainEvent; the channel is closed when ctx is canceled.
func (c *client) Subscribe(ctx context.Context, fromHeight types.Hex) (<-chan chainwatch.BlockchainEvent, error) {
	if fromHeight.IsEmpty() {
		h, err := c.getLatestBlockNumber(ctx)
		if err != nil {
			return nil, err
		}

		fromHeight = h
	}

	eventsCh := make(chan chainwatch.BlockchainEvent, eventsChannelBufferSize)
	go func() {
		defer close(eventsCh)

		for {
			fromHeight = c.pollNewBlocks(ctx, fromHeight, eventsCh)

			select {
			case <-ctx.Done():
				return
			// delay before next poll to match average block time
			case <-time.After(averageBlockTime):
			}
		}
	}()

	return eventsCh, nil
}
