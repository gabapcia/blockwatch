package solana_test

import (
	"testing"

	"github.com/gabapcia/blockwatch/internal/infra/blockchain/solana"
	jsonrpctest "github.com/gabapcia/blockwatch/internal/pkg/transport/jsonrpc/mocks"

	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	var (
		conn   = jsonrpctest.NewClient(t)
		client = solana.NewClient(conn)
	)

	assert.NotNil(t, client)
}
