package jsonrpc

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	t.Run("successful response with result", func(t *testing.T) {
		expected := map[string]any{"hello": "world"}
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"result":  expected,
				"id":      "1",
			})
		}))
		defer mockServer.Close()

		c := NewClient(mockServer.URL)

		result, err := c.Fetch(t.Context(), "dummy_method")
		assert.NoError(t, err)

		var actual map[string]any
		err = json.Unmarshal(result, &actual)
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("response with JSON-RPC error", func(t *testing.T) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"error": map[string]any{
					"code":    -32601,
					"message": "method not found",
				},
				"id": "1",
			})
		}))
		defer mockServer.Close()

		c := NewClient(mockServer.URL)

		result, err := c.Fetch(t.Context(), "nonexistent_method")
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "method not found")
	})

	t.Run("malformed JSON response", func(t *testing.T) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("this is not json"))
		}))
		defer mockServer.Close()

		c := NewClient(mockServer.URL)

		result, err := c.Fetch(t.Context(), "bad_json")
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid character")
	})

	t.Run("network error when server is down", func(t *testing.T) {
		mockServer := httptest.NewServer(nil)
		mockServer.Close() // Immediately close

		c := NewClient(mockServer.URL,
			WithTimeout(1*time.Second),
			WithRetryMax(0),
		)

		result, err := c.Fetch(t.Context(), "network_failure")
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestNewClient(t *testing.T) {
	t.Run("uses default configuration when no options are provided", func(t *testing.T) {
		client := NewClient("http://localhost:8080")

		assert.Equal(t, "http://localhost:8080", client.providerEndpoint)
		assert.NotNil(t, client.httpClient)

		assert.Equal(t, 5*time.Second, client.httpClient.HTTPClient.Timeout, "default timeout should be 5s")
		assert.Equal(t, 1*time.Second, client.httpClient.RetryWaitMin, "default retryWaitMin should be 1s")
		assert.Equal(t, 5*time.Second, client.httpClient.RetryWaitMax, "default retryWaitMax should be 5s")
		assert.Equal(t, 2, client.httpClient.RetryMax, "default retryMax should be 2")
	})

	t.Run("applies all custom options correctly", func(t *testing.T) {
		timeout := 9 * time.Second
		retryWaitMin := 111 * time.Millisecond
		retryWaitMax := 3 * time.Second
		retryMaxAttempts := 7

		c := NewClient(
			"http://localhost:8080",
			WithTimeout(timeout),
			WithRetryWaitMin(retryWaitMin),
			WithRetryWaitMax(retryWaitMax),
			WithRetryMax(retryMaxAttempts),
		)

		assert.Equal(t, "http://localhost:8080", c.providerEndpoint)
		assert.NotNil(t, c.httpClient)

		assert.Equal(t, timeout, c.httpClient.HTTPClient.Timeout, "custom timeout should be applied")
		assert.Equal(t, retryWaitMin, c.httpClient.RetryWaitMin, "custom retryWaitMin should be applied")
		assert.Equal(t, retryWaitMax, c.httpClient.RetryWaitMax, "custom retryWaitMax should be applied")
		assert.Equal(t, retryMaxAttempts, c.httpClient.RetryMax, "custom retryMax should be applied")
	})
}

func TestWithTimeout(t *testing.T) {
	cfg := &config{}
	timeout := 10 * time.Second

	opt := WithTimeout(timeout)
	require.NotNil(t, opt)

	opt(cfg)
	assert.Equal(t, timeout, cfg.timeout, "timeout should be set correctly")
}

func TestWithRetryWaitMin(t *testing.T) {
	cfg := &config{}
	min := 500 * time.Millisecond

	opt := WithRetryWaitMin(min)
	require.NotNil(t, opt)

	opt(cfg)
	assert.Equal(t, min, cfg.retryWaitMin, "retryWaitMin should be set correctly")
}

func TestWithRetryWaitMax(t *testing.T) {
	cfg := &config{}
	max := 8 * time.Second

	opt := WithRetryWaitMax(max)
	require.NotNil(t, opt)

	opt(cfg)
	assert.Equal(t, max, cfg.retryWaitMax, "retryWaitMax should be set correctly")
}

func TestWithRetryMax(t *testing.T) {
	cfg := &config{}
	retries := 5

	opt := WithRetryMax(retries)
	require.NotNil(t, opt)

	opt(cfg)
	assert.Equal(t, retries, cfg.retryMax, "retryMax should be set correctly")
}
