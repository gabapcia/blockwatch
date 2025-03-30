package ethereum

import (
	"reflect"
	"testing"

	"github.com/gabapcia/blockwatch"
)

func TestTransactionResponse_toDomain(t *testing.T) {
	var (
		blockNum int64 = 12345
		txResp         = transactionResponse{
			Hash:  "0xabc123",
			From:  "0xfromAddress",
			To:    "0xtoAddress",
			Value: "1000000000000000000",
		}

		domainTx = txResp.toDomain(blockNum)
		expected = blockwatch.Transaction{
			BlockNumber: blockNum,
			Hash:        "0xabc123",
			From:        "0xfromAddress",
			To:          "0xtoAddress",
			Value:       "1000000000000000000",
		}
	)

	if !reflect.DeepEqual(domainTx, expected) {
		t.Errorf("Expected %+v, got %+v", expected, domainTx)
	}
}
