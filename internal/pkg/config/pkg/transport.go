package pkg

import "time"

// HttpClient defines configuration parameters for HTTP client behavior, including timeouts and retries.
type HttpClient struct {
	// Timeout sets the maximum duration for an HTTP request.
	// Default: 5s
	Timeout time.Duration `default:"5s"`

	// RetryWaitMin defines the minimum wait time between retry attempts.
	// Default: 1s
	RetryWaitMin time.Duration `default:"1s" split_words:"true"`

	// RetryWaitMax defines the maximum wait time between retry attempts.
	// Default: 5s
	RetryWaitMax time.Duration `default:"5s" split_words:"true"`

	// RetryMax specifies the maximum number of retry attempts.
	// Default: 2
	RetryMax int `default:"2" split_words:"true"`
}

// JsonRPC defines configuration for accessing a JSON-RPC provider over HTTP.
type JsonRPC struct {
	// HttpClient holds the HTTP client settings used for JSON-RPC communication.
	HttpClient `validate:"required"`

	// ProviderEndpoint is the full URL of the JSON-RPC provider (e.g., "https://mainnet.infura.io/v3/...").
	ProviderEndpoint string `validate:"required" split_words:"true"`
}
