package solana

import (
	"encoding/json"
	"errors"
	"testing"

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
