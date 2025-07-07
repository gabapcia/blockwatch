// Package ethereum provides an implementation of the watcher.Blockchain interface
// for Ethereum-compatible nodes using a JSON-RPC client.
package ethereum

import (
	"github.com/gabapcia/blockwatch/internal/chainstream"
	"github.com/gabapcia/blockwatch/internal/pkg/transport/jsonrpc"
)

// client implements the watcher.Blockchain interface for Ethereum-based networks.
// It communicates with an Ethereum node via a JSON-RPC client.
type client struct {
	conn jsonrpc.Client // Underlying JSON-RPC client used to interact with the Ethereum node
}

// Ensure client implements the watcher.Blockchain interface at compile time.
var _ chainstream.Blockchain = (*client)(nil)

// NewClient creates a new Ethereum blockchain client using the provided JSON-RPC connection.
// The returned client can be used to listen for new blocks and interact with the network.
func NewClient(conn jsonrpc.Client) *client {
	return &client{
		conn: conn,
	}
}
