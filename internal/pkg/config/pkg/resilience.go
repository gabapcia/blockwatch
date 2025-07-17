package pkg

import "time"

type Retry struct {
	Attempts uint          `default:"3"`
	Delay    time.Duration `default:"1s"`
	MaxDelay time.Duration `default:"5s" split_words:"true"`
}
