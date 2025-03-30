package ethereum

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/gabapcia/blockwatch"
)

const (
	// methodFetchLatestBlock is the JSON-RPC method used to fetch the latest block number.
	methodFetchLatestBlock = "eth_blockNumber"
	// methodFetchBlockByNumber is the JSON-RPC method used to fetch a block by its number.
	methodFetchBlockByNumber = "eth_getBlockByNumber"
)

// blockResponse represents the JSON structure returned by the Ethereum JSON-RPC endpoint
// for a block. It includes the block number in hexadecimal format and a list of transactions.
type blockResponse struct {
	// Number is the block number in hexadecimal string format (e.g., "0x4b7").
	Number string `json:"number"`
	// Transactions is a slice of transactionResponse objects contained within the block.
	Transactions []transactionResponse `json:"transactions"`
}

// toDomain converts a blockResponse to the domain model blockwatch.Block.
// It parses the block number from its hexadecimal representation and converts
// each transaction to a blockwatch.Transaction.
func (b blockResponse) toDomain() (blockwatch.Block, error) {
	blockNumber, err := strconv.ParseInt(strings.TrimPrefix(b.Number, "0x"), 16, 64)
	if err != nil {
		return blockwatch.Block{}, err
	}

	transactions := make([]blockwatch.Transaction, len(b.Transactions))
	for i, tx := range b.Transactions {
		transactions[i] = tx.toDomain(blockNumber)
	}

	return blockwatch.Block{
		Number:       blockNumber,
		Transactions: transactions,
	}, nil
}

// fetchBlockByNumber sends a JSON-RPC request to fetch the block data corresponding to the
// given block number. It converts the block number to its hexadecimal representation,
// calls the fetch method with the appropriate parameters, and unmarshals the JSON response
// into a blockResponse.
func (s *service) fetchBlockByNumber(ctx context.Context, blockNumber int64) (blockResponse, error) {
	var (
		blockHex     = strconv.FormatInt(blockNumber, 16)
		blockHexRepr = fmt.Sprintf("0x%s", blockHex)
	)

	result, err := s.fetch(ctx, methodFetchBlockByNumber, blockHexRepr, true)
	if err != nil {
		return blockResponse{}, err
	}

	var block blockResponse
	if err = json.Unmarshal(result, &block); err != nil {
		return blockResponse{}, err
	}

	return block, nil
}

// GetLatestBlockNumber fetches the latest block number from the Ethereum JSON-RPC endpoint.
// It sends a request using the "eth_blockNumber" method, unmarshals the hexadecimal response,
// and converts it into an int64.
func (s *service) GetLatestBlockNumber(ctx context.Context) (int64, error) {
	result, err := s.fetch(ctx, methodFetchLatestBlock)
	if err != nil {
		return 0, err
	}

	var hexStr string
	if err = json.Unmarshal(result, &hexStr); err != nil {
		return 0, err
	}

	return strconv.ParseInt(strings.TrimPrefix(hexStr, "0x"), 16, 64)
}

// ListBlocksSince retrieves all blocks from one after the specified target block number up to the latest block.
// It first determines the latest block number via GetLatestBlockNumber, then iterates over each block number
// from targetBlockNumber+1 to the latest block number. For each block, it fetches the block data, converts
// it to the domain model, and appends it to a slice of blockwatch.Block.
func (s *service) ListBlocksSince(ctx context.Context, targetBlockNumber int64) ([]blockwatch.Block, error) {
	latestBlockNumber, err := s.GetLatestBlockNumber(ctx)
	if err != nil {
		return nil, err
	}

	blocks := make([]blockwatch.Block, 0)
	for blockNumber := targetBlockNumber + 1; blockNumber <= latestBlockNumber; blockNumber++ {
		blockData, err := s.fetchBlockByNumber(ctx, blockNumber)
		if err != nil {
			return nil, err
		}

		block, err := blockData.toDomain()
		if err != nil {
			return nil, err
		}

		blocks = append(blocks, block)
	}

	return blocks, nil
}
