package ethereum

import (
	"net/http"
	"time"

	"github.com/gabapcia/blockwatch"
)

const DefaultURL = "https://ethereum-rpc.publicnode.com"

type service struct {
	rpcUrl     string
	httpClient *http.Client
}

var _ blockwatch.Blockchain = (*service)(nil)

type (
	config struct {
		url     string
		timeout time.Duration
	}

	Option func(*config)
)

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

func WithURL(url string) Option {
	return func(c *config) {
		c.url = url
	}
}

func WithTimeout(d time.Duration) Option {
	return func(c *config) {
		c.timeout = d
	}
}
