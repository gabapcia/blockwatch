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
	methodFetchLatestBlock   = "eth_blockNumber"
	methodFetchBlockByNumber = "eth_getBlockByNumber"
)

type blockResponse struct {
	Number       string                `json:"number"`
	Transactions []transactionResponse `json:"transactions"`
}

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
