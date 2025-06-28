package ethereum

import (
	"testing"

	"github.com/gabapcia/blockwatch/internal/chainwatch"
	jsonrpctest "github.com/gabapcia/blockwatch/internal/pkg/transport/jsonrpc/mocks"

	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	t.Run("returns valid ethereum client with jsonrpc mock", func(t *testing.T) {
		mockConn := new(jsonrpctest.Client)
		c := NewClient(mockConn)

		assert.NotNil(t, c, "NewClient should not return nil")
		assert.Equal(t, mockConn, c.conn, "conn field should be assigned correctly")

		// Compile-time interface check
		var _ chainwatch.Blockchain = c
	})
}
