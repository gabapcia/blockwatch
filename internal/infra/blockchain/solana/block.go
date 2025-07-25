// Package solana implements a low-level data source that interacts with the Solana blockchain.
//
// It provides block and transaction retrieval using Solana's JSON-RPC interface,
// and translates blockchain data into domain-specific types for the `chainstream` layer.
//
// Only finalized blocks are processed, and transactions of type `system.transfer`
// are extracted and normalized into internal transfer events.
package solana

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gabapcia/blockwatch/internal/chainstream"
	"github.com/gabapcia/blockwatch/internal/pkg/x/types"
)

const (
	// eventsChannelBufferSize defines the buffer size for the event channel
	// used when streaming new blocks to subscribers.
	eventsChannelBufferSize = 200

	// averageBlockTime approximates Solana’s block production interval.
	averageBlockTime = 400 * time.Millisecond

	// waitTimeBetweenRequests controls the polling frequency for new blocks.
	waitTimeBetweenRequests = 10 * averageBlockTime
)

// Internal response models used for decoding Solana RPC JSON structures.
// See: https://solana.com/docs/rpc/http/getblock and https://solana.com/docs/rpc/json-structures#transactions
type (
	// blockResult represents the full response from the `getBlock` endpoint,
	// including block metadata and the list of transactions it contains.
	blockResult struct {
		BlockHeight       uint64        `json:"blockHeight"`       // Height of the block in the blockchain
		BlockTime         *int64        `json:"blockTime"`         // Optional UNIX timestamp of block (nullable)
		Blockhash         string        `json:"blockhash"`         // Unique hash that identifies this block
		ParentSlot        uint64        `json:"parentSlot"`        // Slot number of the parent block
		PreviousBlockhash string        `json:"previousBlockhash"` // Hash of the immediately preceding block
		Transactions      []transaction `json:"transactions"`      // List of transactions included in this block
	}

	// transaction wraps the actual transaction content along with metadata.
	transaction struct {
		Meta        map[string]any   `json:"meta"`        // Transaction metadata (status, logs, fee info, etc.)
		Transaction innerTransaction `json:"transaction"` // Core transaction data including instructions and signatures
	}

	// innerTransaction represents the inner structure of a transaction payload.
	innerTransaction struct {
		Signatures []string `json:"signatures"` // List of transaction signatures (first is treated as transaction ID)
		Message    message  `json:"message"`    // The actual instructions and involved accounts
	}

	// message defines the execution plan of a transaction, including the accounts
	// it touches and the ordered set of instructions to be run.
	message struct {
		AccountKeys     []accountKey  `json:"accountKeys"`     // All accounts referenced during execution
		Instructions    []instruction `json:"instructions"`    // List of ordered instructions within the transaction
		RecentBlockhash string        `json:"recentBlockhash"` // Reference to a recent block to prevent replay
	}

	// accountKey holds metadata about a public key used during the transaction.
	accountKey struct {
		Pubkey   string `json:"pubkey"`   // Base58 public key
		Signer   bool   `json:"signer"`   // Indicates if this key signed the transaction
		Source   string `json:"source"`   // Origin of the key (e.g., "transaction")
		Writable bool   `json:"writable"` // Indicates if the account can be modified
	}

	// instruction represents a single instruction invoked by the transaction.
	instruction struct {
		Parsed      *parsedInstruction `json:"parsed"`      // Optional decoded structure (only available for supported programs)
		Program     string             `json:"program"`     // Human-readable name of the program (e.g., "system")
		ProgramID   string             `json:"programId"`   // Unique identifier of the program on-chain
		StackHeight *int               `json:"stackHeight"` // Optional call depth information (nullable)
	}

	// parsedInstruction contains the decoded, high-level interpretation of a Solana instruction.
	parsedInstruction struct {
		Info parsedInfo `json:"info"` // Instruction arguments (typically source, destination, lamports)
		Type string     `json:"type"` // Type of operation (e.g., "transfer")
	}

	// parsedInfo holds decoded values for instructions like `system.transfer`.
	parsedInfo struct {
		Source      string `json:"source"`      // Public key of the sender
		Destination string `json:"destination"` // Public key of the recipient
		Lamports    uint64 `json:"lamports"`    // Amount of lamports to be transferred
	}
)

// toStreamTransactions extracts all valid transfer instructions from a given transaction,
// returning them as normalized chainstream.Transaction entries.
//
// Non-transfer instructions and transactions without signatures are ignored.
func (t transaction) toStreamTransactions() []chainstream.Transaction {
	txs := make([]chainstream.Transaction, 0)

	if len(t.Transaction.Signatures) == 0 {
		return txs
	}

	transactionHash := t.Transaction.Signatures[0]

	for _, instruction := range t.Transaction.Message.Instructions {
		if instruction.Parsed == nil || instruction.Parsed.Type != "transfer" {
			continue
		}

		txs = append(txs, chainstream.Transaction{
			Hash: transactionHash,
			From: instruction.Parsed.Info.Source,
			To:   instruction.Parsed.Info.Destination,
		})
	}

	return txs
}

// toStreamBlock transforms a full blockResult into a domain-level chainstream.Block,
// flattening all valid transfer instructions from its constituent transactions.
func (b blockResult) toStreamBlock() chainstream.Block {
	txs := make([]chainstream.Transaction, 0)
	for _, tx := range b.Transactions {
		txs = append(txs, tx.toStreamTransactions()...)
	}

	return chainstream.Block{
		Height:       types.HexFromInt(int64(b.BlockHeight)),
		Hash:         b.Blockhash,
		Transactions: txs,
	}
}

// getLatestBlockHeight fetches the latest finalized block height from the Solana node.
//
// Returns the height wrapped in a types.Hex representation.
func (c *client) getLatestBlockHeight(ctx context.Context) (types.Hex, error) {
	data, err := c.conn.Fetch(ctx, "getBlockHeight", map[string]any{
		"commitment": "finalized",
	})
	if err != nil {
		return "", err
	}

	var height int64
	if err := json.Unmarshal(data, &height); err != nil {
		return "", err
	}

	return types.HexFromInt(height), nil
}

// getBlockByHeight retrieves the full data for a specific block height
// using "getBlock" with parsed instruction support.
//
// It returns the decoded blockResult struct.
func (c *client) getBlockByHeight(ctx context.Context, height types.Hex) (blockResult, error) {
	data, err := c.conn.Fetch(ctx, "getBlock", height.Int(), map[string]any{
		"commitment":         "finalized",
		"encoding":           "jsonParsed",
		"transactionDetails": "full",
	})
	if err != nil {
		return blockResult{}, err
	}

	var resp blockResult
	return resp, json.Unmarshal(data, &resp)
}

// pollNewBlocks scans from the given height up to the latest finalized block,
// emitting normalized events into the provided channel.
//
// Any block fetch error is captured per event without halting the process.
func (c *client) pollNewBlocks(ctx context.Context, fromHeight types.Hex, eventsCh chan<- chainstream.BlockchainEvent) types.Hex {
	latestHeight, err := c.getLatestBlockHeight(ctx)
	if err != nil {
		eventsCh <- chainstream.BlockchainEvent{Height: fromHeight, Err: err}
		return fromHeight
	}

	if fromHeight >= latestHeight {
		return fromHeight
	}

	for h := fromHeight; h.Int() <= latestHeight.Int(); h = h.Add(1) {
		blockResp, err := c.getBlockByHeight(ctx, h)
		eventsCh <- chainstream.BlockchainEvent{
			Height: h,
			Block:  blockResp.toStreamBlock(),
			Err:    err,
		}
	}

	return latestHeight.Add(1)
}

// FetchBlockByHeight retrieves and transforms a specific block into domain format.
func (c *client) FetchBlockByHeight(ctx context.Context, height types.Hex) (chainstream.Block, error) {
	resp, err := c.getBlockByHeight(ctx, height)
	if err != nil {
		return chainstream.Block{}, err
	}

	return resp.toStreamBlock(), nil
}

// Subscribe continuously polls for new finalized blocks starting from the given height,
// streaming them into a buffered channel as chainstream.BlockchainEvent values.
//
// It respects cancellation via context and uses fixed delays between polling rounds.
func (c *client) Subscribe(ctx context.Context, fromHeight types.Hex) (<-chan chainstream.BlockchainEvent, error) {
	if fromHeight.IsEmpty() {
		h, err := c.getLatestBlockHeight(ctx)
		if err != nil {
			return nil, err
		}

		fromHeight = h
	}

	eventsCh := make(chan chainstream.BlockchainEvent, eventsChannelBufferSize)
	go func() {
		defer close(eventsCh)

		for {
			fromHeight = c.pollNewBlocks(ctx, fromHeight, eventsCh)

			select {
			case <-ctx.Done():
				return
			case <-time.After(waitTimeBetweenRequests):
			}
		}
	}()

	return eventsCh, nil
}
