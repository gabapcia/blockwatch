package pkg

import "time"

type HttpClient struct {
	Timeout      time.Duration `default:"5s"`
	RetryWaitMin time.Duration `default:"1s" split_words:"true"`
	RetryWaitMax time.Duration `default:"5s" split_words:"true"`
	RetryMax     int           `default:"2" split_words:"true"`
}

type JsonRPC struct {
	HttpClient       `validate:"required"`
	ProviderEndpoint string `validate:"required" split_words:"true"`
}
