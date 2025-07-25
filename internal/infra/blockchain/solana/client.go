// Package solana provides an implementation of the chainstream.Blockchain interface
// that interacts with the Solana blockchain through a JSON-RPC client.
//
// This layer is responsible for abstracting the transport mechanism and exposing
// Solana-specific data as normalized domain-level blocks and transactions.
package solana

import (
	"github.com/gabapcia/blockwatch/internal/chainstream"
	"github.com/gabapcia/blockwatch/internal/pkg/transport/jsonrpc"
)

// client implements the chainstream.Blockchain interface using Solana's JSON-RPC protocol.
//
// It wraps a low-level JSON-RPC client and delegates requests to the appropriate RPC methods.
// This component is designed to be infrastructure-specific and hidden behind the domain interface.
type client struct {
	conn jsonrpc.Client // JSON-RPC connection to the Solana node
}

// Compile-time assertion to ensure client implements the Blockchain interface.
var _ chainstream.Blockchain = (*client)(nil)

// NewClient creates a new Solana blockchain client using the provided JSON-RPC connection.
//
// The resulting client conforms to the chainstream.Blockchain interface and should be used
// within the infrastructure layer to stream or fetch blocks from the Solana chain.
//
// Parameters:
//   - conn: an implementation of jsonrpc.Client to communicate with the Solana node.
//
// Returns:
//   - A pointer to a new Solana client instance.
func NewClient(conn jsonrpc.Client) *client {
	return &client{
		conn: conn,
	}
}
