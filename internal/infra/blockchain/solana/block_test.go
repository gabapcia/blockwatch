package solana

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/gabapcia/blockwatch/internal/chainstream"
	jsonrpctest "github.com/gabapcia/blockwatch/internal/pkg/transport/jsonrpc/mocks"
	"github.com/gabapcia/blockwatch/internal/pkg/x/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTransaction_toStreamTransactions(t *testing.T) {
	t.Run("should convert a transaction with one transfer instruction", func(t *testing.T) {
		tx := transaction{
			Transaction: innerTransaction{
				Signatures: []string{"sig1"},
				Message: message{
					Instructions: []instruction{
						{
							Parsed: &parsedInstruction{
								Type: "transfer",
								Info: parsedInfo{
									Source:      "from1",
									Destination: "to1",
									Lamports:    100,
								},
							},
						},
					},
				},
			},
		}

		expected := []chainstream.Transaction{
			{Hash: "sig1", From: "from1", To: "to1"},
		}

		assert.Equal(t, expected, tx.toStreamTransactions())
	})

	t.Run("should convert a transaction with multiple transfer instructions", func(t *testing.T) {
		tx := transaction{
			Transaction: innerTransaction{
				Signatures: []string{"sig2"},
				Message: message{
					Instructions: []instruction{
						{
							Parsed: &parsedInstruction{
								Type: "transfer",
								Info: parsedInfo{
									Source:      "from2",
									Destination: "to2",
									Lamports:    200,
								},
							},
						},
						{
							Parsed: &parsedInstruction{
								Type: "transfer",
								Info: parsedInfo{
									Source:      "from3",
									Destination: "to3",
									Lamports:    300,
								},
							},
						},
					},
				},
			},
		}

		expected := []chainstream.Transaction{
			{Hash: "sig2", From: "from2", To: "to2"},
			{Hash: "sig2", From: "from3", To: "to3"},
		}

		assert.Equal(t, expected, tx.toStreamTransactions())
	})

	t.Run("should return empty slice if transaction has no signatures", func(t *testing.T) {
		tx := transaction{
			Transaction: innerTransaction{
				Signatures: []string{},
			},
		}

		assert.Empty(t, tx.toStreamTransactions())
	})

	t.Run("should ignore non-transfer instructions", func(t *testing.T) {
		tx := transaction{
			Transaction: innerTransaction{
				Signatures: []string{"sig3"},
				Message: message{
					Instructions: []instruction{
						{
							Parsed: &parsedInstruction{
								Type: "createAccount",
							},
						},
					},
				},
			},
		}

		assert.Empty(t, tx.toStreamTransactions())
	})

	t.Run("should handle mixed transfer and non-transfer instructions", func(t *testing.T) {
		tx := transaction{
			Transaction: innerTransaction{
				Signatures: []string{"sig4"},
				Message: message{
					Instructions: []instruction{
						{
							Parsed: &parsedInstruction{
								Type: "createAccount",
							},
						},
						{
							Parsed: &parsedInstruction{
								Type: "transfer",
								Info: parsedInfo{
									Source:      "from4",
									Destination: "to4",
									Lamports:    400,
								},
							},
						},
					},
				},
			},
		}

		expected := []chainstream.Transaction{
			{Hash: "sig4", From: "from4", To: "to4"},
		}

		assert.Equal(t, expected, tx.toStreamTransactions())
	})

	t.Run("should handle instructions with nil Parsed field", func(t *testing.T) {
		tx := transaction{
			Transaction: innerTransaction{
				Signatures: []string{"sig5"},
				Message: message{
					Instructions: []instruction{
						{
							Parsed: nil,
						},
					},
				},
			},
		}

		assert.Empty(t, tx.toStreamTransactions())
	})
}

func TestBlockResult_toStreamBlock(t *testing.T) {
	t.Run("should convert block with multiple transactions to stream block", func(t *testing.T) {
		br := blockResult{
			BlockHeight: 123,
			Blockhash:   "hash123",
			Transactions: []transaction{
				{
					Transaction: innerTransaction{
						Signatures: []string{"sig1"},
						Message: message{
							Instructions: []instruction{
								{
									Parsed: &parsedInstruction{
										Type: "transfer",
										Info: parsedInfo{Source: "from1", Destination: "to1"},
									},
								},
							},
						},
					},
				},
				{
					Transaction: innerTransaction{
						Signatures: []string{"sig2"},
						Message: message{
							Instructions: []instruction{
								{
									Parsed: &parsedInstruction{
										Type: "transfer",
										Info: parsedInfo{Source: "from2", Destination: "to2"},
									},
								},
								{
									Parsed: &parsedInstruction{
										Type: "createAccount",
									},
								},
							},
						},
					},
				},
			},
		}

		expected := chainstream.Block{
			Height: "0x7b",
			Hash:   "hash123",
			Transactions: []chainstream.Transaction{
				{Hash: "sig1", From: "from1", To: "to1"},
				{Hash: "sig2", From: "from2", To: "to2"},
			},
		}

		assert.Equal(t, expected, br.toStreamBlock())
	})

	t.Run("should handle block with no transactions", func(t *testing.T) {
		br := blockResult{
			BlockHeight:  456,
			Blockhash:    "hash456",
			Transactions: []transaction{},
		}

		expected := chainstream.Block{
			Height:       "0x1c8",
			Hash:         "hash456",
			Transactions: []chainstream.Transaction{},
		}

		assert.Equal(t, expected, br.toStreamBlock())
	})

	t.Run("should handle block with transactions but no transfer instructions", func(t *testing.T) {
		br := blockResult{
			BlockHeight: 789,
			Blockhash:   "hash789",
			Transactions: []transaction{
				{
					Transaction: innerTransaction{
						Signatures: []string{"sig3"},
						Message: message{
							Instructions: []instruction{
								{
									Parsed: &parsedInstruction{
										Type: "createAccount",
									},
								},
							},
						},
					},
				},
			},
		}

		expected := chainstream.Block{
			Height:       "0x315",
			Hash:         "hash789",
			Transactions: []chainstream.Transaction{},
		}

		assert.Equal(t, expected, br.toStreamBlock())
	})
}

func TestClient_getLatestBlockHeight(t *testing.T) {
	t.Run("returns latest block height successfully", func(t *testing.T) {
		mockClient := jsonrpctest.NewClient(t)
		raw := json.RawMessage(`123`)

		mockClient.
			EXPECT().
			Fetch(mock.Anything, "getBlockHeight", mock.Anything).
			Return(raw, nil)

		c := NewClient(mockClient)
		result, err := c.getLatestBlockHeight(t.Context())

		assert.NoError(t, err)
		assert.Equal(t, types.Hex("0x7b"), result)
	})

	t.Run("returns error when fetch fails", func(t *testing.T) {
		mockClient := jsonrpctest.NewClient(t)

		mockClient.
			EXPECT().
			Fetch(mock.Anything, "getBlockHeight", mock.Anything).
			Return(nil, errors.New("fetch error"))

		c := NewClient(mockClient)
		result, err := c.getLatestBlockHeight(t.Context())

		assert.Error(t, err)
		assert.Empty(t, result)
	})

	t.Run("returns error on invalid response", func(t *testing.T) {
		mockClient := jsonrpctest.NewClient(t)
		invalidJSON := json.RawMessage(`"invalid"`)

		mockClient.
			EXPECT().
			Fetch(mock.Anything, "getBlockHeight", mock.Anything).
			Return(invalidJSON, nil)

		c := NewClient(mockClient)
		result, err := c.getLatestBlockHeight(t.Context())

		assert.Error(t, err)
		assert.Empty(t, result)
	})
}

func TestClient_getBlockByHeight(t *testing.T) {
	t.Run("returns block successfully", func(t *testing.T) {
		mockClient := jsonrpctest.NewClient(t)

		mockJSON := json.RawMessage(`{
			"blockHeight": 123,
			"blockhash": "hash123",
			"transactions": [
				{
					"transaction": {
						"signatures": ["sig1"],
						"message": {
							"instructions": [
								{
									"parsed": {
										"type": "transfer",
										"info": { "source": "from1", "destination": "to1" }
									}
								}
							]
						}
					}
				}
			]
		}`)

		mockClient.
			EXPECT().
			Fetch(mock.Anything, "getBlock", mock.Anything).
			Return(mockJSON, nil).
			Once()

		c := NewClient(mockClient)
		block, err := c.getBlockByHeight(t.Context(), types.Hex("0x7b"))

		assert.NoError(t, err)
		assert.Equal(t, uint64(123), block.BlockHeight)
		assert.Equal(t, "hash123", block.Blockhash)
		assert.Len(t, block.Transactions, 1)
		assert.Equal(t, "sig1", block.Transactions[0].Transaction.Signatures[0])
	})

	t.Run("returns error when fetch fails", func(t *testing.T) {
		mockClient := jsonrpctest.NewClient(t)

		mockClient.
			EXPECT().
			Fetch(mock.Anything, "getBlock", mock.Anything).
			Return(nil, errors.New("connection error")).
			Once()

		c := NewClient(mockClient)
		block, err := c.getBlockByHeight(t.Context(), types.Hex("0x7b"))

		assert.Error(t, err)
		assert.Empty(t, block)
	})

	t.Run("returns error on invalid json", func(t *testing.T) {
		mockClient := jsonrpctest.NewClient(t)

		mockJSON := json.RawMessage(`{ invalid-json`)

		mockClient.
			EXPECT().
			Fetch(mock.Anything, "getBlock", mock.Anything).
			Return(mockJSON, nil).
			Once()

		c := NewClient(mockClient)
		block, err := c.getBlockByHeight(t.Context(), types.Hex("0x7b"))

		assert.Error(t, err)
		assert.Empty(t, block)
	})
}

func TestClient_pollNewBlocks(t *testing.T) {
	t.Run("should poll and return new blocks successfully", func(t *testing.T) {
		mockClient := jsonrpctest.NewClient(t)
		c := NewClient(mockClient)

		fromHeight := types.Hex("0x63")
		latestHeight := int64(100)

		mockClient.
			EXPECT().
			Fetch(mock.Anything, "getBlockHeight", mock.Anything).
			Return(json.RawMessage(`100`), nil).
			Once()

		for i := fromHeight.Int(); i <= latestHeight; i++ {
			mockBlock := blockResult{BlockHeight: uint64(i), Blockhash: "some hash"}
			mockBlockJSON, _ := json.Marshal(mockBlock)
			mockClient.
				EXPECT().
				Fetch(mock.Anything, "getBlock", mock.Anything).
				Return(json.RawMessage(mockBlockJSON), nil).
				Once()
		}

		eventsCh := make(chan chainstream.BlockchainEvent, 5)
		newHeight := c.pollNewBlocks(t.Context(), fromHeight, eventsCh)
		close(eventsCh)

		assert.Equal(t, types.Hex("0x65"), newHeight)
		assert.Len(t, eventsCh, 2)

		event1 := <-eventsCh
		assert.Equal(t, types.Hex("0x63"), event1.Height)
		assert.NoError(t, event1.Err)

		event2 := <-eventsCh
		assert.Equal(t, types.Hex("0x64"), event2.Height)
		assert.NoError(t, event2.Err)
	})

	t.Run("should return fromHeight when getLatestBlockHeight fails", func(t *testing.T) {
		mockClient := jsonrpctest.NewClient(t)
		c := NewClient(mockClient)

		fromHeight := types.Hex("0x64")
		expectedErr := errors.New("rpc error")

		mockClient.
			EXPECT().
			Fetch(mock.Anything, "getBlockHeight", mock.Anything).
			Return(nil, expectedErr).
			Once()

		eventsCh := make(chan chainstream.BlockchainEvent, 1)
		newHeight := c.pollNewBlocks(t.Context(), fromHeight, eventsCh)
		close(eventsCh)

		assert.Equal(t, fromHeight, newHeight)
		assert.Len(t, eventsCh, 1)

		event := <-eventsCh
		assert.Equal(t, fromHeight, event.Height)
		assert.Equal(t, expectedErr, event.Err)
	})

	t.Run("should continue polling when getBlockByHeight fails for one block", func(t *testing.T) {
		mockClient := jsonrpctest.NewClient(t)
		c := NewClient(mockClient)

		fromHeight := types.Hex("0x63")
		expectedErr := errors.New("block not found")

		mockClient.
			EXPECT().
			Fetch(mock.Anything, "getBlockHeight", mock.Anything).
			Return(json.RawMessage(`100`), nil).
			Once()

		mockBlock := blockResult{BlockHeight: 99, Blockhash: "some hash"}
		mockBlockJSON, _ := json.Marshal(mockBlock)
		mockClient.
			EXPECT().
			Fetch(mock.Anything, "getBlock", mock.Anything).
			Return(json.RawMessage(mockBlockJSON), nil).
			Once()

		mockClient.
			EXPECT().
			Fetch(mock.Anything, "getBlock", mock.Anything).
			Return(nil, expectedErr).
			Once()

		eventsCh := make(chan chainstream.BlockchainEvent, 2)
		newHeight := c.pollNewBlocks(t.Context(), fromHeight, eventsCh)
		close(eventsCh)

		assert.Equal(t, types.Hex("0x65"), newHeight)
		assert.Len(t, eventsCh, 2)

		event1 := <-eventsCh
		assert.Equal(t, fromHeight, event1.Height)
		assert.NoError(t, event1.Err)

		event2 := <-eventsCh
		assert.Equal(t, fromHeight.Add(1), event2.Height)
		assert.Equal(t, expectedErr, event2.Err)
	})

	t.Run("should not poll when fromHeight is greater than latestHeight", func(t *testing.T) {
		mockClient := jsonrpctest.NewClient(t)
		c := NewClient(mockClient)

		fromHeight := types.Hex("0x65")

		mockClient.
			EXPECT().
			Fetch(mock.Anything, "getBlockHeight", mock.Anything).
			Return(json.RawMessage(`100`), nil).
			Once()

		eventsCh := make(chan chainstream.BlockchainEvent)
		newHeight := c.pollNewBlocks(t.Context(), fromHeight, eventsCh)

		assert.Equal(t, fromHeight, newHeight)
	})
}

func TestClient_FetchBlockByHeight(t *testing.T) {
	t.Run("should fetch block by height successfully", func(t *testing.T) {
		mockClient := jsonrpctest.NewClient(t)
		c := NewClient(mockClient)

		height := types.Hex("0x64")
		mockBlock := blockResult{
			BlockHeight: 100,
			Blockhash:   "some hash",
			Transactions: []transaction{
				{
					Transaction: innerTransaction{
						Signatures: []string{"sig1"},
						Message: message{
							Instructions: []instruction{
								{
									Parsed: &parsedInstruction{
										Type: "transfer",
										Info: parsedInfo{Source: "from1", Destination: "to1"},
									},
								},
							},
						},
					},
				},
			},
		}
		mockBlockJSON, _ := json.Marshal(mockBlock)
		mockClient.
			EXPECT().
			Fetch(mock.Anything, "getBlock", mock.Anything).
			Return(json.RawMessage(mockBlockJSON), nil).
			Once()

		block, err := c.FetchBlockByHeight(t.Context(), height)

		assert.NoError(t, err)
		assert.Equal(t, height, block.Height)
		assert.Equal(t, "some hash", block.Hash)
		assert.Len(t, block.Transactions, 1)
		assert.Equal(t, "sig1", block.Transactions[0].Hash)
	})

	t.Run("should return error when getBlockByHeight fails", func(t *testing.T) {
		mockClient := jsonrpctest.NewClient(t)
		c := NewClient(mockClient)

		height := types.Hex("0x64")
		expectedErr := errors.New("block not found")

		mockClient.
			EXPECT().
			Fetch(mock.Anything, "getBlock", mock.Anything).
			Return(nil, expectedErr).
			Once()

		block, err := c.FetchBlockByHeight(t.Context(), height)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Empty(t, block)
	})
}

func TestClient_Subscribe(t *testing.T) {
	t.Run("should subscribe and stream blocks successfully", func(t *testing.T) {
		mockClient := jsonrpctest.NewClient(t)
		c := NewClient(mockClient)

		fromHeight := types.Hex("0x63")

		// Mock getLatestBlockHeight calls
		mockClient.
			EXPECT().
			Fetch(mock.Anything, "getBlockHeight", mock.Anything).
			Return(json.RawMessage(`100`), nil).
			Once()

		// Mock getBlockByHeight calls for blocks 99 and 100
		for i := int64(99); i <= 100; i++ {
			mockBlock := blockResult{
				BlockHeight: uint64(i),
				Blockhash:   "hash" + types.HexFromInt(i).String(),
				Transactions: []transaction{
					{
						Transaction: innerTransaction{
							Signatures: []string{"sig" + types.HexFromInt(i).String()},
							Message: message{
								Instructions: []instruction{
									{
										Parsed: &parsedInstruction{
											Type: "transfer",
											Info: parsedInfo{
												Source:      "from" + types.HexFromInt(i).String(),
												Destination: "to" + types.HexFromInt(i).String(),
												Lamports:    uint64(i * 100),
											},
										},
									},
								},
							},
						},
					},
				},
			}
			mockBlockJSON, _ := json.Marshal(mockBlock)
			mockClient.
				EXPECT().
				Fetch(mock.Anything, "getBlock", mock.Anything).
				Return(json.RawMessage(mockBlockJSON), nil).
				Once()
		}

		ctx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
		defer cancel()

		eventsCh, err := c.Subscribe(ctx, fromHeight)
		assert.NoError(t, err)
		assert.NotNil(t, eventsCh)

		// Collect events from the first polling cycle
		var events []chainstream.BlockchainEvent
		timeout := time.After(150 * time.Millisecond)

	collectLoop:
		for {
			select {
			case event, ok := <-eventsCh:
				if !ok {
					break collectLoop
				}
				events = append(events, event)
				if len(events) >= 2 {
					break collectLoop
				}
			case <-timeout:
				break collectLoop
			}
		}

		// Cancel context to stop subscription
		cancel()

		// Wait for channel to close
		for range eventsCh {
			// Drain remaining events
		}

		// Verify we got the expected events
		assert.GreaterOrEqual(t, len(events), 2)

		// Check first event
		assert.Equal(t, types.Hex("0x63"), events[0].Height)
		assert.NoError(t, events[0].Err)
		assert.Equal(t, "hash0x63", events[0].Block.Hash)
		assert.Len(t, events[0].Block.Transactions, 1)
		assert.Equal(t, "sig0x63", events[0].Block.Transactions[0].Hash)

		// Check second event
		assert.Equal(t, types.Hex("0x64"), events[1].Height)
		assert.NoError(t, events[1].Err)
		assert.Equal(t, "hash0x64", events[1].Block.Hash)
		assert.Len(t, events[1].Block.Transactions, 1)
		assert.Equal(t, "sig0x64", events[1].Block.Transactions[0].Hash)
	})

	t.Run("should return error when initial getLatestBlockHeight fails", func(t *testing.T) {
		mockClient := jsonrpctest.NewClient(t)
		c := NewClient(mockClient)

		expectedErr := errors.New("rpc connection failed")
		mockClient.
			EXPECT().
			Fetch(mock.Anything, "getBlockHeight", mock.Anything).
			Return(nil, expectedErr).
			Once()

		ctx := t.Context()
		eventsCh, err := c.Subscribe(ctx, types.Hex(""))

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, eventsCh)
	})

	t.Run("should use latest block height when fromHeight is empty", func(t *testing.T) {
		mockClient := jsonrpctest.NewClient(t)
		c := NewClient(mockClient)

		// Mock getLatestBlockHeight to return 150
		mockClient.
			EXPECT().
			Fetch(mock.Anything, "getBlockHeight", mock.Anything).
			Return(json.RawMessage(`150`), nil).
			Maybe() // Allow multiple calls since Subscribe polls continuously

		ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
		defer cancel()

		eventsCh, err := c.Subscribe(ctx, types.Hex(""))
		assert.NoError(t, err)
		assert.NotNil(t, eventsCh)

		// Wait for context to timeout and channel to close
		for range eventsCh {
			// Drain any events
		}
	})

	t.Run("should stop subscription when context is cancelled", func(t *testing.T) {
		mockClient := jsonrpctest.NewClient(t)
		c := NewClient(mockClient)

		fromHeight := types.Hex("0x64")

		// Mock getLatestBlockHeight - allow multiple calls since Subscribe polls continuously
		mockClient.
			EXPECT().
			Fetch(mock.Anything, "getBlockHeight", mock.Anything).
			Return(json.RawMessage(`100`), nil).
			Maybe()

		// Mock getBlockByHeight - allow multiple calls
		mockBlock := blockResult{
			BlockHeight:  100,
			Blockhash:    "hash100",
			Transactions: []transaction{},
		}
		mockBlockJSON, _ := json.Marshal(mockBlock)
		mockClient.
			EXPECT().
			Fetch(mock.Anything, "getBlock", mock.Anything).
			Return(json.RawMessage(mockBlockJSON), nil).
			Maybe()

		ctx, cancel := context.WithCancel(t.Context())

		eventsCh, err := c.Subscribe(ctx, fromHeight)
		assert.NoError(t, err)
		assert.NotNil(t, eventsCh)

		// Give some time for subscription to start
		time.Sleep(50 * time.Millisecond)

		// Cancel context
		cancel()

		// Channel should close within reasonable time
		select {
		case _, ok := <-eventsCh:
			if ok {
				// Drain any remaining events
				for range eventsCh {
				}
			}
		case <-time.After(500 * time.Millisecond):
			t.Fatal("Channel should have been closed")
		}
	})

	t.Run("should handle block fetch errors during polling", func(t *testing.T) {
		mockClient := jsonrpctest.NewClient(t)
		c := NewClient(mockClient)

		fromHeight := types.Hex("0x63")

		// Mock getLatestBlockHeight
		mockClient.
			EXPECT().
			Fetch(mock.Anything, "getBlockHeight", mock.Anything).
			Return(json.RawMessage(`100`), nil).
			Once()

		// First block succeeds
		mockBlock1 := blockResult{
			BlockHeight:  99,
			Blockhash:    "hash99",
			Transactions: []transaction{},
		}
		mockBlockJSON1, _ := json.Marshal(mockBlock1)
		mockClient.
			EXPECT().
			Fetch(mock.Anything, "getBlock", mock.Anything).
			Return(json.RawMessage(mockBlockJSON1), nil).
			Once()

		// Second block fails
		blockErr := errors.New("block not found")
		mockClient.
			EXPECT().
			Fetch(mock.Anything, "getBlock", mock.Anything).
			Return(nil, blockErr).
			Once()

		ctx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
		defer cancel()

		eventsCh, err := c.Subscribe(ctx, fromHeight)
		assert.NoError(t, err)
		assert.NotNil(t, eventsCh)

		// Collect events
		var events []chainstream.BlockchainEvent
		timeout := time.After(150 * time.Millisecond)

	blockErrorLoop:
		for {
			select {
			case event, ok := <-eventsCh:
				if !ok {
					break blockErrorLoop
				}
				events = append(events, event)
				if len(events) >= 2 {
					break blockErrorLoop
				}
			case <-timeout:
				break blockErrorLoop
			}
		}

		cancel()
		// Drain channel
		for range eventsCh {
		}

		// Should have 2 events
		assert.Len(t, events, 2)

		// First event should be successful
		assert.Equal(t, types.Hex("0x63"), events[0].Height)
		assert.NoError(t, events[0].Err)
		assert.Equal(t, "hash99", events[0].Block.Hash)

		// Second event should have error
		assert.Equal(t, types.Hex("0x64"), events[1].Height)
		assert.Equal(t, blockErr, events[1].Err)
		// When there's an error, the block will have zero values
		assert.Equal(t, "", events[1].Block.Hash)
		assert.Equal(t, types.Hex("0x0"), events[1].Block.Height)
	})

	t.Run("should not poll when already at latest height", func(t *testing.T) {
		mockClient := jsonrpctest.NewClient(t)
		c := NewClient(mockClient)

		fromHeight := types.Hex("0x64") // 100 in decimal

		// Mock getLatestBlockHeight to return same height
		mockClient.
			EXPECT().
			Fetch(mock.Anything, "getBlockHeight", mock.Anything).
			Return(json.RawMessage(`100`), nil).
			Maybe() // Allow multiple calls since Subscribe polls continuously

		ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
		defer cancel()

		eventsCh, err := c.Subscribe(ctx, fromHeight)
		assert.NoError(t, err)
		assert.NotNil(t, eventsCh)

		// Should not receive any events since we're already at latest height
		// Just wait for context timeout and channel close
		for range eventsCh {
			// Drain any unexpected events
		}
	})
}
