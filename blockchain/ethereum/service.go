package ethereum

import (
	"net/http"
	"time"

	"github.com/gabapcia/blockwatch"
)

// DefaultURL is the default Ethereum JSON-RPC endpoint URL.
const DefaultURL = "https://ethereum-rpc.publicnode.com"

// service implements the blockwatch.Blockchain interface and encapsulates
// the logic for communicating with the Ethereum blockchain.
type service struct {
	rpcUrl     string
	httpClient *http.Client
}

// Ensure that *service satisfies the blockwatch.Blockchain interface.
var _ blockwatch.Blockchain = (*service)(nil)

type (
	// config holds the configuration settings for the Ethereum service.
	config struct {
		// url specifies the Ethereum JSON-RPC endpoint.
		url string
		// timeout defines the HTTP request timeout duration.
		timeout time.Duration
	}

	// Option defines a function type that modifies the configuration for the Ethereum service.
	Option func(*config)
)

// New creates and returns a new instance of the Ethereum service using the provided options.
// If no options are provided, it defaults to using DefaultURL and a 5-second timeout.
func New(opts ...Option) *service {
	cfg := config{
		url:     DefaultURL,
		timeout: 5 * time.Second,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	return &service{
		rpcUrl: cfg.url,
		httpClient: &http.Client{
			Timeout: cfg.timeout,
		},
	}
}

// WithURL returns an Option that sets a custom URL for the Ethereum JSON-RPC endpoint.
func WithURL(url string) Option {
	return func(c *config) {
		c.url = url
	}
}

// WithTimeout returns an Option that sets a custom timeout for the HTTP client.
func WithTimeout(d time.Duration) Option {
	return func(c *config) {
		c.timeout = d
	}
}
