package ethereum

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/gabapcia/blockwatch/internal/chainstream"
	jsonrpctest "github.com/gabapcia/blockwatch/internal/pkg/transport/jsonrpc/mocks"
	"github.com/gabapcia/blockwatch/internal/pkg/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTransactionResponse_toWatcherTransaction(t *testing.T) {
	t.Run("converts TransactionResponse to chainstream.Transaction", func(t *testing.T) {
		tr := TransactionResponse{
			Hash: "0xabc123",
			From: "0xfrom",
			To:   "0xto",
		}

		expected := chainstream.Transaction{
			Hash: "0xabc123",
			From: "0xfrom",
			To:   "0xto",
		}

		result := tr.toWatcherTransaction()
		assert.Equal(t, expected, result, "converted transaction should match expected chainstream.Transaction")
	})
}

func TestBlockResponse_toWatcherBlock(t *testing.T) {
	t.Run("converts BlockResponse to chainstream.Block", func(t *testing.T) {
		blockResp := BlockResponse{
			Hash:   "0xblockhash",
			Number: types.Hex("0x10"),
			Transactions: []TransactionResponse{
				{
					Hash: "0x1", From: "0xA", To: "0xB",
				},
				{
					Hash: "0x2", From: "0xC", To: "0xD",
				},
			},
		}

		expected := chainstream.Block{
			Hash:   "0xblockhash",
			Height: types.Hex("0x10"),
			Transactions: []chainstream.Transaction{
				{Hash: "0x1", From: "0xA", To: "0xB"},
				{Hash: "0x2", From: "0xC", To: "0xD"},
			},
		}

		result := blockResp.toWatcherBlock()
		assert.Equal(t, expected, result, "converted block should match expected chainstream.Block")
	})
}

func TestClient_getLatestBlockNumber(t *testing.T) {
	t.Run("returns latest block number successfully", func(t *testing.T) {
		mockClient := new(jsonrpctest.Client)
		raw := json.RawMessage(`"0x10"`)

		mockClient.
			On("Fetch", mock.Anything, "eth_blockNumber").
			Return(raw, nil)

		c := NewClient(mockClient)
		result, err := c.getLatestBlockNumber(t.Context())

		assert.NoError(t, err)
		assert.Equal(t, types.Hex("0x10"), result)

		mockClient.AssertExpectations(t)
	})

	t.Run("returns error when fetch fails", func(t *testing.T) {
		mockClient := new(jsonrpctest.Client)

		mockClient.
			On("Fetch", mock.Anything, "eth_blockNumber").
			Return(nil, errors.New("fetch error"))

		c := NewClient(mockClient)
		result, err := c.getLatestBlockNumber(t.Context())

		assert.Error(t, err)
		assert.Empty(t, result)

		mockClient.AssertExpectations(t)
	})

	t.Run("returns error on invalid response", func(t *testing.T) {
		mockClient := new(jsonrpctest.Client)
		invalidJSON := json.RawMessage(`not-a-hex-string`)

		mockClient.
			On("Fetch", mock.Anything, "eth_blockNumber").
			Return(invalidJSON, nil)

		c := NewClient(mockClient)
		result, err := c.getLatestBlockNumber(t.Context())

		assert.Error(t, err)
		assert.Empty(t, result)

		mockClient.AssertExpectations(t)
	})
}

func TestClient_getBlockByNumber(t *testing.T) {
	t.Run("returns block successfully", func(t *testing.T) {
		mockClient := new(jsonrpctest.Client)

		mockJSON := json.RawMessage(`{
			"hash": "0xabc",
			"number": "0x10",
			"transactions": [
				{"hash": "0x1", "from": "0xA", "to": "0xB"}
			],
			"withdrawals": []
		}`)

		mockClient.
			On("Fetch", mock.Anything, "eth_getBlockByNumber", []any{types.Hex("0x10"), true}).
			Return(mockJSON, nil)

		c := NewClient(mockClient)
		block, err := c.getBlockByNumber(t.Context(), types.Hex("0x10"))

		assert.NoError(t, err)
		assert.Equal(t, types.Hex("0x10"), block.Number)
		assert.Equal(t, "0xabc", block.Hash)
		assert.Len(t, block.Transactions, 1)
		assert.Equal(t, "0x1", block.Transactions[0].Hash)

		mockClient.AssertExpectations(t)
	})

	t.Run("returns error when fetch fails", func(t *testing.T) {
		mockClient := new(jsonrpctest.Client)

		mockClient.
			On("Fetch", mock.Anything, "eth_getBlockByNumber", []any{types.Hex("0x10"), true}).
			Return(nil, errors.New("connection error"))

		c := NewClient(mockClient)
		block, err := c.getBlockByNumber(t.Context(), types.Hex("0x10"))

		assert.Error(t, err)
		assert.Empty(t, block)

		mockClient.AssertExpectations(t)
	})

	t.Run("returns error on invalid json", func(t *testing.T) {
		mockClient := new(jsonrpctest.Client)

		mockJSON := json.RawMessage(`{ invalid-json`)

		mockClient.
			On("Fetch", mock.Anything, "eth_getBlockByNumber", []any{types.Hex("0x10"), true}).
			Return(mockJSON, nil)

		c := NewClient(mockClient)
		block, err := c.getBlockByNumber(t.Context(), types.Hex("0x10"))

		assert.Error(t, err)
		assert.Empty(t, block)

		mockClient.AssertExpectations(t)
	})
}

func TestClient_pollNewBlocks(t *testing.T) {
	t.Run("emits all blocks from fromBlockNumber up to latest, returns latest+1", func(t *testing.T) {
		mockClient := new(jsonrpctest.Client)

		// latest block: 0x13
		mockClient.
			On("Fetch", mock.Anything, "eth_blockNumber").
			Return(json.RawMessage(`"0x13"`), nil)

		for _, hex := range []string{"0x10", "0x11", "0x12", "0x13"} {
			mockClient.
				On("Fetch", mock.Anything, "eth_getBlockByNumber", []any{types.Hex(hex), true}).
				Return(json.RawMessage(`{
					"hash": "0xabc`+hex+`",
					"number": "`+hex+`",
					"transactions": [],
					"withdrawals": []
				}`), nil)
		}

		c := NewClient(mockClient)
		events := make(chan chainstream.BlockchainEvent, 10)

		next := c.pollNewBlocks(t.Context(), types.Hex("0x10"), events)
		assert.Equal(t, types.Hex("0x14"), next, "next block number should be latest + 1 (0x14)")

		close(events)
		var count int
		for ev := range events {
			assert.NoError(t, ev.Err, "event error should be nil")
			expected := types.Hex("0x10").Add(int64(count))
			assert.Equal(t, expected, ev.Height, "block height mismatch at index %d", count)
			assert.Equal(t, expected, ev.Block.Height, "block number mismatch at index %d", count)
			count++
		}
		assert.Equal(t, 4, count, "number of emitted blocks should be 4")
		mockClient.AssertExpectations(t)
	})

	t.Run("returns immediately when fromBlockNumber >= latestBlockNumber", func(t *testing.T) {
		mockClient := new(jsonrpctest.Client)

		mockClient.
			On("Fetch", mock.Anything, "eth_blockNumber").
			Return(json.RawMessage(`"0x20"`), nil)

		c := NewClient(mockClient)
		events := make(chan chainstream.BlockchainEvent, 1)

		next := c.pollNewBlocks(t.Context(), types.Hex("0x20"), events)
		assert.Equal(t, types.Hex("0x20"), next, "next block number should be unchanged when no new blocks")

		close(events)
		assert.Empty(t, events, "no events should be emitted when from >= latest")
		mockClient.AssertExpectations(t)
	})

	t.Run("returns same block number and emits error when latest block fetch fails", func(t *testing.T) {
		mockClient := new(jsonrpctest.Client)

		expectedErr := errors.New("rpc error")
		mockClient.
			On("Fetch", mock.Anything, "eth_blockNumber").
			Return(nil, expectedErr)

		c := NewClient(mockClient)
		events := make(chan chainstream.BlockchainEvent, 1)

		next := c.pollNewBlocks(t.Context(), types.Hex("0x5"), events)
		assert.Equal(t, types.Hex("0x5"), next, "should return fromBlockNumber unchanged on failure")

		close(events)
		ev := <-events
		assert.Empty(t, ev.Block.Hash, "no block should be present when latest fetch fails")
		assert.Equal(t, types.Hex("0x5"), ev.Height, "event should contain the block height")
		assert.ErrorIs(t, ev.Err, expectedErr, "event should contain the fetch error")
		mockClient.AssertExpectations(t)
	})

	t.Run("emits partial success and error for failing block", func(t *testing.T) {
		mockClient := new(jsonrpctest.Client)

		mockClient.
			On("Fetch", mock.Anything, "eth_blockNumber").
			Return(json.RawMessage(`"0x11"`), nil)

		mockClient.
			On("Fetch", mock.Anything, "eth_getBlockByNumber", []any{types.Hex("0x10"), true}).
			Return(json.RawMessage(`{
				"hash": "0xabc10",
				"number": "0x10",
				"transactions": [],
				"withdrawals": []
			}`), nil)

		mockedErr := errors.New("block not found")
		mockClient.
			On("Fetch", mock.Anything, "eth_getBlockByNumber", []any{types.Hex("0x11"), true}).
			Return(nil, mockedErr)

		c := NewClient(mockClient)
		events := make(chan chainstream.BlockchainEvent, 2)

		next := c.pollNewBlocks(t.Context(), types.Hex("0x10"), events)
		assert.Equal(t, types.Hex("0x12"), next, "next block number should be latest + 1")

		close(events)

		ev1 := <-events
		assert.Equal(t, types.Hex("0x10"), ev1.Block.Height, "first event should be for block 0x10")
		assert.Equal(t, types.Hex("0x10"), ev1.Height, "first event should be for block 0x10")
		assert.NoError(t, ev1.Err, "first event should not have error")

		ev2 := <-events
		assert.Empty(t, ev2.Block.Height, "second event should have empty block due to error")
		assert.Equal(t, types.Hex("0x11"), ev2.Height, "second event should be for block 0x11")
		assert.ErrorIs(t, ev2.Err, mockedErr, "second event should contain fetch error")
		mockClient.AssertExpectations(t)
	})
}

func TestClient_FetchBlockByHeight(t *testing.T) {
	t.Run("returns converted block on success", func(t *testing.T) {
		// Arrange: set up mock to return a valid BlockResponse JSON
		mockClient := new(jsonrpctest.Client)
		rawJSON := json.RawMessage(`{
			"hash": "0xabc",
			"number": "0x10",
			"transactions": [{"hash": "0x1", "from": "0xa", "to": "0xb"}],
			"withdrawals": []
		}`)
		// Expect Fetch with correct parameters
		mockClient.On("Fetch", mock.Anything, "eth_getBlockByNumber", []any{types.Hex("0x10"), true}).
			Return(rawJSON, nil)

		client := NewClient(mockClient)

		// Act
		block, err := client.FetchBlockByHeight(t.Context(), types.Hex("0x10"))

		// Assert
		assert.NoError(t, err, "FetchBlockByHeight should not return an error on valid input")
		assert.Equal(t, types.Hex("0x10"), block.Height, "block height should match the requested height")
		assert.Equal(t, "0xabc", block.Hash, "block hash should match the JSON response")
		assert.Len(t, block.Transactions, 1, "there should be exactly one transaction")
		// Validate transaction conversion
		expectedTx := chainstream.Transaction{Hash: "0x1", From: "0xa", To: "0xb"}
		assert.Equal(t, expectedTx, block.Transactions[0], "transaction should be converted correctly")

		mockClient.AssertExpectations(t)
	})

	t.Run("returns error when underlying fetch fails", func(t *testing.T) {
		// Arrange: mock error
		mockClient := new(jsonrpctest.Client)
		expectedErr := errors.New("rpc failure")
		mockClient.On("Fetch", mock.Anything, "eth_getBlockByNumber", []any{types.Hex("0x20"), true}).
			Return(nil, expectedErr)

		client := NewClient(mockClient)

		// Act
		block, err := client.FetchBlockByHeight(t.Context(), types.Hex("0x20"))

		// Assert
		assert.ErrorIs(t, err, expectedErr, "FetchBlockByHeight should return the underlying error")
		assert.Empty(t, block, "block should be empty when an error occurs")

		mockClient.AssertExpectations(t)
	})
}

func TestClient_Subscribe(t *testing.T) {
	t.Run("emits events when start is less than latest", func(t *testing.T) {
		mockClient := new(jsonrpctest.Client)

		// Simulate latest block number = 0x12
		mockClient.
			On("Fetch", mock.Anything, "eth_blockNumber").
			Return(json.RawMessage(`"0x12"`), nil)

		// Simulate fetching blocks 0x10, 0x11, 0x12
		for _, hex := range []string{"0x10", "0x11", "0x12"} {
			mockClient.
				On("Fetch", mock.Anything, "eth_getBlockByNumber", []any{types.Hex(hex), true}).
				Return(json.RawMessage(fmt.Sprintf(`{
					"hash": "0xabc%s",
					"number": "%s",
					"transactions": [],
					"withdrawals": []
				}`, hex, hex)), nil)
		}

		c := NewClient(mockClient)
		ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
		defer cancel()

		eventsCh, err := c.Subscribe(ctx, types.Hex("0x10"))
		assert.NoError(t, err)

		var events []chainstream.BlockchainEvent
		for ev := range eventsCh {
			events = append(events, ev)
		}

		assert.Len(t, events, 3)
		for i, ev := range events {
			expectedNumber := types.Hex("0x10").Add(int64(i))
			assert.Equal(t, expectedNumber, ev.Height)
			assert.Equal(t, expectedNumber, ev.Block.Height)
			assert.NoError(t, ev.Err)
		}

		mockClient.AssertExpectations(t)
	})

	t.Run("no events when start equals latest", func(t *testing.T) {
		mockClient := new(jsonrpctest.Client)

		// Latest block number = 0x20
		mockClient.
			On("Fetch", mock.Anything, "eth_blockNumber").
			Return(json.RawMessage(`"0x20"`), nil)

		c := NewClient(mockClient)
		ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
		defer cancel()

		eventsCh, err := c.Subscribe(ctx, types.Hex("0x20"))
		assert.NoError(t, err)

		count := 0
		for range eventsCh {
			count++
		}
		assert.Zero(t, count, "no events should be emitted when start == latest")

		mockClient.AssertExpectations(t)
	})

	t.Run("emits events when start is empty (uses latest as start)", func(t *testing.T) {
		mockClient := new(jsonrpctest.Client)

		// First fetch: determine latest = 0x15
		mockClient.
			On("Fetch", mock.Anything, "eth_blockNumber").
			Return(json.RawMessage(`"0x15"`), nil).
			Once()

		// Next poll: latest = 0x16, then fetch block 0x15
		mockClient.
			On("Fetch", mock.Anything, "eth_blockNumber").
			Return(json.RawMessage(`"0x16"`), nil).
			Once()

		mockClient.
			On("Fetch", mock.Anything, "eth_getBlockByNumber", []any{types.Hex("0x15"), true}).
			Return(json.RawMessage(`{
				"hash": "0xabc15",
				"number": "0x15",
				"transactions": [],
				"withdrawals": []
			}`), nil).
			Once()

		mockClient.
			On("Fetch", mock.Anything, "eth_getBlockByNumber", []any{types.Hex("0x16"), true}).
			Return(json.RawMessage(`{
				"hash": "0xabc16",
				"number": "0x16",
				"transactions": [],
				"withdrawals": []
			}`), nil).
			Once()

		c := NewClient(mockClient)
		ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
		defer cancel()

		eventsCh, err := c.Subscribe(ctx, "")
		assert.NoError(t, err)

		var events []chainstream.BlockchainEvent
		for ev := range eventsCh {
			events = append(events, ev)
		}

		assert.Len(t, events, 2)

		assert.Equal(t, types.Hex("0x15"), events[0].Block.Height)
		assert.Equal(t, types.Hex("0x15"), events[0].Height)
		assert.NoError(t, events[0].Err)

		assert.Equal(t, types.Hex("0x16"), events[1].Block.Height)
		assert.Equal(t, types.Hex("0x16"), events[1].Height)
		assert.NoError(t, events[1].Err)

		mockClient.AssertExpectations(t)
	})

	t.Run("propagates block fetch errors to channel", func(t *testing.T) {
		var (
			mockClient  = new(jsonrpctest.Client)
			sentinelErr = errors.New("block not found")
		)

		// latest block = 0x11
		mockClient.
			On("Fetch", mock.Anything, "eth_blockNumber").
			Return(json.RawMessage(`"0x11"`), nil)

		// block 0x10 succeeds
		mockClient.
			On("Fetch", mock.Anything, "eth_getBlockByNumber", []any{types.Hex("0x10"), true}).
			Return(json.RawMessage(`{
				"hash": "0xabc10",
				"number": "0x10",
				"transactions": [],
				"withdrawals": []
			}`), nil)

		// block 0x11 fails
		mockClient.
			On("Fetch", mock.Anything, "eth_getBlockByNumber", []any{types.Hex("0x11"), true}).
			Return(nil, sentinelErr)

		c := NewClient(mockClient)
		ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
		defer cancel()

		eventsCh, err := c.Subscribe(ctx, types.Hex("0x10"))
		assert.NoError(t, err)

		var events []chainstream.BlockchainEvent
		for ev := range eventsCh {
			events = append(events, ev)
		}
		assert.Len(t, events, 2)

		// First event ok
		assert.NoError(t, events[0].Err)
		assert.Equal(t, types.Hex("0x10"), events[0].Block.Height)
		assert.Equal(t, types.Hex("0x10"), events[0].Height)

		// Second event should carry sentinelErr
		assert.ErrorIs(t, events[1].Err, sentinelErr)
		assert.Empty(t, events[1].Block.Height)
		assert.Equal(t, types.Hex("0x11"), events[1].Height)

		mockClient.AssertExpectations(t)
	})

	t.Run("returns error if initial getLatestBlockNumber fails", func(t *testing.T) {
		var (
			mockClient  = new(jsonrpctest.Client)
			sentinelErr = errors.New("rpc down")
		)

		// initial fetch fails
		mockClient.
			On("Fetch", mock.Anything, "eth_blockNumber").
			Return(nil, sentinelErr).
			Once()

		c := NewClient(mockClient)
		eventsCh, err := c.Subscribe(t.Context(), "")
		assert.Nil(t, eventsCh)
		assert.ErrorIs(t, err, sentinelErr)

		mockClient.AssertExpectations(t)
	})
}
