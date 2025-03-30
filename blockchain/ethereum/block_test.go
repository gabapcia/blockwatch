package ethereum

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gabapcia/blockwatch"
)

func TestBlockResponse_toDomain(t *testing.T) {
	t.Run("Valid Block", func(t *testing.T) {
		blockResp := blockResponse{
			Number: "0x4b7",
			Transactions: []transactionResponse{
				{
					Hash:  "0xabc123",
					From:  "0xfromAddress",
					To:    "0xtoAddress",
					Value: "1000000000000000000",
				},
				{
					Hash:  "0xdef456",
					From:  "0xfromAddress2",
					To:    "0xtoAddress2",
					Value: "2000000000000000000",
				},
			},
		}

		domainBlock, err := blockResp.toDomain()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		expectedBlockNumber := int64(1207)
		if domainBlock.Number != expectedBlockNumber {
			t.Errorf("expected block number %d, got %d", expectedBlockNumber, domainBlock.Number)
		}

		expectedTransactions := []blockwatch.Transaction{
			{
				BlockNumber: expectedBlockNumber,
				Hash:        "0xabc123",
				From:        "0xfromAddress",
				To:          "0xtoAddress",
				Value:       "1000000000000000000",
			},
			{
				BlockNumber: expectedBlockNumber,
				Hash:        "0xdef456",
				From:        "0xfromAddress2",
				To:          "0xtoAddress2",
				Value:       "2000000000000000000",
			},
		}
		if !reflect.DeepEqual(domainBlock.Transactions, expectedTransactions) {
			t.Errorf("expected transactions %+v, got %+v", expectedTransactions, domainBlock.Transactions)
		}
	})

	t.Run("Invalid Hex", func(t *testing.T) {
		blockResp := blockResponse{
			Number: "not-a-hex",
			Transactions: []transactionResponse{
				{
					Hash:  "0xabc123",
					From:  "0xfromAddress",
					To:    "0xtoAddress",
					Value: "1000000000000000000",
				},
			},
		}

		_, err := blockResp.toDomain()
		if err == nil {
			t.Fatal("expected error when converting invalid hex, got nil")
		}
	})

	t.Run("Empty Transactions", func(t *testing.T) {
		blockResp := blockResponse{
			Number:       "0x1a",
			Transactions: []transactionResponse{},
		}

		domainBlock, err := blockResp.toDomain()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		expectedBlockNumber := int64(26)
		if domainBlock.Number != expectedBlockNumber {
			t.Errorf("expected block number %d, got %d", expectedBlockNumber, domainBlock.Number)
		}

		if len(domainBlock.Transactions) != 0 {
			t.Errorf("expected 0 transactions, got %d", len(domainBlock.Transactions))
		}
	})
}

func TestFetchBlockByNumber(t *testing.T) {
	t.Run("SuccessfulResponse", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"result": {"number": "0x10", "transactions": []}}`))
		}))
		defer ts.Close()

		s := &service{
			rpcUrl:     ts.URL,
			httpClient: http.DefaultClient,
		}

		blockNumber := int64(16)
		blockResp, err := s.fetchBlockByNumber(t.Context(), blockNumber)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if blockResp.Number != "0x10" {
			t.Errorf("expected block number %q, got %q", "0x10", blockResp.Number)
		}

		if len(blockResp.Transactions) != 0 {
			t.Errorf("expected 0 transactions, got %d", len(blockResp.Transactions))
		}
	})

	t.Run("Malformed JSON", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`invalid json`))
		}))
		defer ts.Close()

		s := &service{
			rpcUrl:     ts.URL,
			httpClient: http.DefaultClient,
		}

		blockNumber := int64(16)
		if _, err := s.fetchBlockByNumber(t.Context(), blockNumber); err == nil {
			t.Fatal("expected error due to malformed JSON, got nil")
		}
	})
}

func TestGetLatestBlockNumber(t *testing.T) {
	t.Run("Valid Latest Block", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"result": "0x10", "error": null}`))
		}))
		defer ts.Close()

		s := &service{
			rpcUrl:     ts.URL,
			httpClient: http.DefaultClient,
		}

		blockNumber, err := s.GetLatestBlockNumber(t.Context())
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if blockNumber != 16 {
			t.Errorf("expected block number 16, got %d", blockNumber)
		}
	})

	t.Run("MalformedJSON", func(t *testing.T) {
		// Create a test server that returns invalid JSON.
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`invalid json`))
		}))
		defer ts.Close()

		s := &service{
			rpcUrl:     ts.URL,
			httpClient: http.DefaultClient,
		}

		if _, err := s.GetLatestBlockNumber(t.Context()); err == nil {
			t.Fatal("expected error due to malformed JSON, got nil")
		}
	})
}

func TestListBlocksSince(t *testing.T) {
	t.Run("Valid List", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req struct {
				JsonRPC string `json:"jsonrpc"`
				ID      int    `json:"id"`
				Method  string `json:"method"`
				Params  []any  `json:"params"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			switch req.Method {
			case methodFetchLatestBlock:
				w.Write([]byte(`{"result": "0xc", "error": null}`))
			case methodFetchBlockByNumber:
				blockHex, ok := req.Params[0].(string)
				if !ok {
					http.Error(w, "invalid params", http.StatusBadRequest)
					return
				}
				resp := fmt.Sprintf(`{"result": {"number": "%s", "transactions": []}}`, blockHex)
				w.Write([]byte(resp))
			default:
				http.Error(w, "unknown method", http.StatusBadRequest)
			}
		}))
		defer ts.Close()

		s := &service{
			rpcUrl:     ts.URL,
			httpClient: http.DefaultClient,
		}

		blocks, err := s.ListBlocksSince(t.Context(), 10)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(blocks) != 2 {
			t.Fatalf("expected 2 blocks, got %d", len(blocks))
		}

		if blocks[0].Number != 11 {
			t.Errorf("expected first block number 11, got %d", blocks[0].Number)
		}

		if blocks[1].Number != 12 {
			t.Errorf("expected second block number 12, got %d", blocks[1].Number)
		}
	})

	t.Run("Error On Get Latest", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req struct {
				JsonRPC string `json:"jsonrpc"`
				ID      int    `json:"id"`
				Method  string `json:"method"`
				Params  []any  `json:"params"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			if req.Method == methodFetchLatestBlock {
				w.Write([]byte(`invalid json`))
				return
			}

			if req.Method == methodFetchBlockByNumber {
				blockHex, ok := req.Params[0].(string)
				if !ok {
					http.Error(w, "invalid params", http.StatusBadRequest)
					return
				}
				resp := fmt.Sprintf(`{"number": "%s", "transactions": []}`, blockHex)
				w.Write([]byte(resp))
			}
		}))
		defer ts.Close()

		s := &service{
			rpcUrl:     ts.URL,
			httpClient: http.DefaultClient,
		}

		_, err := s.ListBlocksSince(t.Context(), 10)
		if err == nil {
			t.Fatal("expected error due to malformed JSON in GetLatestBlockNumber, got nil")
		}
	})

	t.Run("Error On Fetch Block", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req struct {
				JsonRPC string `json:"jsonrpc"`
				ID      int    `json:"id"`
				Method  string `json:"method"`
				Params  []any  `json:"params"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			switch req.Method {
			case methodFetchLatestBlock:
				w.Write([]byte(`{"result": "0xc", "error": null}`))
			case methodFetchBlockByNumber:
				w.Write([]byte(`invalid json`))
			default:
				http.Error(w, "unknown method", http.StatusBadRequest)
			}
		}))
		defer ts.Close()

		s := &service{
			rpcUrl:     ts.URL,
			httpClient: http.DefaultClient,
		}

		_, err := s.ListBlocksSince(t.Context(), 10)
		if err == nil {
			t.Fatal("expected error due to error in fetchBlockByNumber, got nil")
		}
	})
}
