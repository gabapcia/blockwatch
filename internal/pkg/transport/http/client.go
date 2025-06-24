// Package http provides a configurable HTTP client with retry logic.
// It wraps the retryablehttp.Client from HashiCorp and exposes functional
// options for customizing timeouts and retry behavior.
package http

import (
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

// config holds internal settings for the HTTP client.
type config struct {
	timeout      time.Duration // maximum duration for a single HTTP request
	retryWaitMin time.Duration // minimum delay between retry attempts
	retryWaitMax time.Duration // maximum delay between retry attempts
	retryMax     int           // maximum number of retry attempts
}

// Option defines a functional option for configuring the HTTP client.
type Option func(*config)

// NewClient creates and returns a retryablehttp.Client configured with
// the provided options. If no options are given, default values are used:
//
//   - timeout:      5 seconds
//   - retryWaitMin: 1 second
//   - retryWaitMax: 5 seconds
//   - retryMax:     2 retries
func NewClient(opts ...Option) *retryablehttp.Client {
	cfg := config{
		timeout:      5 * time.Second,
		retryWaitMin: 1 * time.Second,
		retryWaitMax: 5 * time.Second,
		retryMax:     2,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	client := retryablehttp.NewClient()
	client.Logger = nil
	client.HTTPClient.Timeout = cfg.timeout
	client.RetryWaitMin = cfg.retryWaitMin
	client.RetryWaitMax = cfg.retryWaitMax
	client.RetryMax = cfg.retryMax
	return client
}

// WithTimeout sets the maximum duration allowed for a single HTTP request.
// Default: 5 seconds.
func WithTimeout(d time.Duration) Option {
	return func(c *config) {
		c.timeout = d
	}
}

// WithRetryWaitMin sets the minimum delay between retry attempts.
// Default: 1 second.
func WithRetryWaitMin(d time.Duration) Option {
	return func(c *config) {
		c.retryWaitMin = d
	}
}

// WithRetryWaitMax sets the maximum delay between retry attempts.
// Default: 5 seconds.
func WithRetryWaitMax(d time.Duration) Option {
	return func(c *config) {
		c.retryWaitMax = d
	}
}

// WithRetryMax sets the maximum number of retry attempts for failed requests.
// Default: 2 retries.
func WithRetryMax(n int) Option {
	return func(c *config) {
		c.retryMax = n
	}
}
