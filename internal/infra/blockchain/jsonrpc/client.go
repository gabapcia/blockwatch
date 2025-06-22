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

var ErrProviderReturnedError = errors.New("provider error")

type response struct {
	JsonRPC string `json:"jsonrpc"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	Result json.RawMessage `json:"result"`
}

func (r response) Err() error {
	if r.Error == nil {
		return nil
	}

	return fmt.Errorf("%w: [%d] - %s", ErrProviderReturnedError, r.Error.Code, r.Error.Message)
}

type client struct {
	providerEndpoint string
	httpClient       *http.Client
}

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

type config struct {
	timeout      time.Duration
	retryWaitMin time.Duration
	retryWaitMax time.Duration
	retryMax     int
}

type Option func(*config)

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

func WithTimeout(d time.Duration) Option {
	return func(c *config) {
		c.timeout = d
	}
}

func WithRetryWaitMin(d time.Duration) Option {
	return func(c *config) {
		c.retryWaitMin = d
	}
}

func WithRetryWaitMax(d time.Duration) Option {
	return func(c *config) {
		c.retryWaitMax = d
	}
}

func WithRetryMax(n int) Option {
	return func(c *config) {
		c.retryMax = n
	}
}
