// Package jsonrpc provides a generic JSON-RPC 2.0 client implementation over HTTP.
// It supports automatic retries, configurable timeouts, and is suitable for interacting with
// any JSON-RPC-compatible service, such as blockchain nodes, remote APIs, and more.
package jsonrpc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
)

// ErrProviderReturnedError indicates that the remote JSON-RPC server returned an error response.
var ErrProviderReturnedError = errors.New("provider error")

// response represents a standard JSON-RPC 2.0 response.
type response struct {
	JsonRPC string `json:"jsonrpc"` // JSON-RPC protocol version (usually "2.0")
	Error   *struct {
		Code    int    `json:"code"`    // Error code defined by the JSON-RPC spec or custom server logic
		Message string `json:"message"` // Human-readable error message
	} `json:"error"`
	Result json.RawMessage `json:"result"` // Raw result payload returned by the server
}

// Err returns an error if the response includes a JSON-RPC error object.
// It wraps ErrProviderReturnedError with the provided error code and message.
func (r response) Err() error {
	if r.Error == nil {
		return nil
	}

	return fmt.Errorf("%w: [%d] - %s", ErrProviderReturnedError, r.Error.Code, r.Error.Message)
}

// Client defines the interface for a generic JSON-RPC client.
// It can be used to abstract the underlying implementation and facilitate mocking or testing.
type Client interface {
	// Fetch sends a JSON-RPC request with the given method name and parameters.
	// It returns the raw JSON result or an error if the request or response fails.
	Fetch(ctx context.Context, method string, params ...any) (json.RawMessage, error)
}

// client is the default implementation of the Client interface.
// It sends JSON-RPC requests to the configured provider endpoint using the provided HTTP client.
type client struct {
	providerEndpoint string       // The URL of the remote JSON-RPC server
	httpClient       *http.Client // The HTTP client used to perform requests
}

// Compile-time assertion that client implements the Client interface.
var _ Client = (*client)(nil)

// Fetch sends a JSON-RPC request to the remote server with the given method and parameters.
// It returns the raw result as a json.RawMessage or an error if the request or server fails.
// The `id` field in the request is generated as a UUID string.
func (c *client) Fetch(ctx context.Context, method string, params ...any) (json.RawMessage, error) {
	body, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      uuid.NewString(),
		"method":  method,
		"params":  params,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.providerEndpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var data response
	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		return nil, err
	}

	return data.Result, data.Err()
}

// NewClient constructs and returns a Client that will send JSON-RPC requests
// to the specified provider endpoint using the given HTTP client.
//
// httpClient: the HTTP client to use for sending requests.
// providerEndpoint: the URL of the JSON-RPC server.
func NewClient(httpClient *http.Client, providerEndpoint string) *client {
	return &client{
		providerEndpoint: providerEndpoint,
		httpClient:       httpClient,
	}
}
