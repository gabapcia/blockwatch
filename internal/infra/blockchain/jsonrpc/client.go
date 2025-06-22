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
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-retryablehttp"
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

// client is a reusable JSON-RPC client over HTTP.
// It handles encoding requests, sending them, decoding responses, and retry logic.
type client struct {
	providerEndpoint string       // The URL of the remote JSON-RPC server
	httpClient       *http.Client // The HTTP client used to perform requests
}

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
	defer req.Body.Close()

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

// config holds optional configuration parameters for the JSON-RPC client.
type config struct {
	timeout      time.Duration // Maximum time to wait for a HTTP request
	retryWaitMin time.Duration // Minimum delay between retries
	retryWaitMax time.Duration // Maximum delay between retries
	retryMax     int           // Maximum number of retry attempts
}

// Option defines a functional option type used to customize the client configuration.
type Option func(*config)

// NewClient creates a new JSON-RPC client pointing to the specified server endpoint.
// Optional configuration parameters can be supplied using functional options such as WithTimeout.
// It includes retry support via the retryablehttp package.
func NewClient(providerEndpoint string, opts ...Option) *client {
	cfg := config{
		timeout:      5 * time.Second,
		retryWaitMin: 1 * time.Second,
		retryWaitMax: 5 * time.Second,
		retryMax:     2,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	httpClient := retryablehttp.NewClient()
	httpClient.Logger = nil
	httpClient.HTTPClient.Timeout = cfg.timeout
	httpClient.RetryWaitMin = cfg.retryWaitMin
	httpClient.RetryWaitMax = cfg.retryWaitMax
	httpClient.RetryMax = cfg.retryMax

	return &client{
		providerEndpoint: providerEndpoint,
		httpClient:       httpClient.StandardClient(),
	}
}

// WithTimeout configures the maximum duration for a single HTTP request.
//
// Default: 5 seconds.
func WithTimeout(d time.Duration) Option {
	return func(c *config) {
		c.timeout = d
	}
}

// WithRetryWaitMin configures the minimum wait duration between retry attempts.
//
// Default: 1 second.
func WithRetryWaitMin(d time.Duration) Option {
	return func(c *config) {
		c.retryWaitMin = d
	}
}

// WithRetryWaitMax configures the maximum wait duration between retry attempts.
//
// Default: 5 seconds.
func WithRetryWaitMax(d time.Duration) Option {
	return func(c *config) {
		c.retryWaitMax = d
	}
}

// WithRetryMax configures the maximum number of retry attempts for failed requests.
//
// Default: 2 retries.
func WithRetryMax(n int) Option {
	return func(c *config) {
		c.retryMax = n
	}
}
