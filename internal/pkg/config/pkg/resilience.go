package pkg

import "time"

// Retry defines retry behavior settings for operations that can fail and be retried.
type Retry struct {
	// Attempts specifies the maximum number of retry attempts.
	// Default: 3
	Attempts uint `env:"ATTEMPTS, default=3"`

	// Delay is the initial wait duration before the first retry attempt.
	// Default: 1s
	Delay time.Duration `env:"DELAY, default=1s"`

	// MaxDelay defines the maximum delay between retries.
	// Default: 5s
	MaxDelay time.Duration `env:"MAX_DELAY, default=5s"`
}
