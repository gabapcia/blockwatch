package jsonrpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestResponse_Err(t *testing.T) {
	t.Run("returns nil when Error field is nil", func(t *testing.T) {
		resp := response{
			JsonRPC: "2.0",
			Error:   nil,
			Result:  nil,
		}

		err := resp.Err()
		assert.NoError(t, err, "Err() should return nil when Error field is nil")
	})

	t.Run("returns formatted error when Error field is present", func(t *testing.T) {
		expectedCode := -32601
		expectedMsg := "method not found"

		resp := response{
			JsonRPC: "2.0",
			Error: &struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			}{
				Code:    expectedCode,
				Message: expectedMsg,
			},
		}

		err := resp.Err()

		assert.Error(t, err, "Err() should return an error when Error field is present")
		assert.ErrorIs(t, err, ErrProviderReturnedError, "Err() should wrap ErrProviderReturnedError")
		assert.Contains(t, err.Error(), fmt.Sprintf("[%d]", expectedCode), "error message should include code")
		assert.Contains(t, err.Error(), expectedMsg, "error message should include message")
	})
}

func TestClient_Fetch(t *testing.T) {
	t.Run("successful response returns raw result", func(t *testing.T) {
		expected := map[string]any{"foo": "bar"}
		// Mock server returning a valid JSON-RPC result
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      "1",
				"result":  expected,
			})
		}))
		defer server.Close()

		c := NewClient(server.Client(), server.URL)
		raw, err := c.Fetch(t.Context(), "testMethod", "param1", 2)
		assert.NoError(t, err, "Fetch should not return error on valid JSON-RPC response")

		var actual map[string]any
		err = json.Unmarshal(raw, &actual)
		assert.NoError(t, err, "Unmarshalling raw result should succeed")
		assert.Equal(t, expected, actual, "Fetched result should match expected value")
	})

	t.Run("json-rpc error response returns error", func(t *testing.T) {
		// Mock server returning a JSON-RPC error
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"error": map[string]any{
					"code":    123,
					"message": "failure",
				},
				"id": "1",
			})
		}))
		defer server.Close()

		c := NewClient(server.Client(), server.URL)
		raw, err := c.Fetch(t.Context(), "method")
		assert.Error(t, err, "Fetch should return error for JSON-RPC error response")
		assert.True(t, errors.Is(err, ErrProviderReturnedError), "error should wrap ErrProviderReturnedError")
		assert.Contains(t, err.Error(), "[123]", "error message should contain code")
		assert.Contains(t, err.Error(), "failure", "error message should contain message")
		assert.Empty(t, raw, "raw result should be empty when error occurs")
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		// Mock server returning invalid JSON
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("not valid json"))
		}))
		defer server.Close()

		c := NewClient(server.Client(), server.URL)
		raw, err := c.Fetch(t.Context(), "method")
		assert.Error(t, err, "Fetch should return error for invalid JSON")
		assert.Contains(t, err.Error(), "invalid character", "error message should indicate JSON parsing failure")
		assert.Empty(t, raw, "raw result should be empty when parsing fails")
	})

	t.Run("network error returns error", func(t *testing.T) {
		// Close server immediately to simulate network failure
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		url := server.URL
		server.Close()

		c := NewClient(http.DefaultClient, url)
		raw, err := c.Fetch(t.Context(), "method")
		assert.Error(t, err, "Fetch should return error when network fails")
		assert.Empty(t, raw, "raw result should be empty on network failure")
	})
}

func TestNewClient(t *testing.T) {
	t.Run("returns client with provided HTTP client and endpoint", func(t *testing.T) {
		// given
		customHTTPClient := &http.Client{Timeout: 123 * time.Millisecond}
		endpoint := "http://localhost:8545"

		// when
		c := NewClient(customHTTPClient, endpoint)

		// then
		assert.NotNil(t, c, "NewClient should not return nil")
		assert.Equal(t, customHTTPClient, c.httpClient, "httpClient should be set to the provided value")
		assert.Equal(t, endpoint, c.providerEndpoint, "providerEndpoint should be set to the provided value")
	})
}
