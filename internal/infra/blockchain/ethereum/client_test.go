package ethereum

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/gabapcia/blockwatch/internal/pkg/transport/jsonrpc/jsonrpctest"
	"github.com/gabapcia/blockwatch/internal/watcher"
)

func TestNewClient(t *testing.T) {
	t.Run("returns valid ethereum client with jsonrpc mock", func(t *testing.T) {
		mockConn := new(jsonrpctest.Client)
		c := NewClient(mockConn)

		assert.NotNil(t, c, "NewClient should not return nil")
		assert.Equal(t, mockConn, c.conn, "conn field should be assigned correctly")

		// Compile-time interface check
		var _ watcher.Blockchain = c
	})
}
