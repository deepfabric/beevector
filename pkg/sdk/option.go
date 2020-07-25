package sdk

import (
	"time"
)

// Option sdk option
type Option func(*options)

type options struct {
	batch   int64
	timeout time.Duration
}

func (opts *options) adjust() {
	if opts.batch == 0 {
		opts.batch = 128
	}

	if opts.timeout == 0 {
		opts.timeout = time.Minute
	}
}

// WithBatch with batch
func WithBatch(value int64) Option {
	return func(opts *options) {
		opts.batch = value
	}
}

// WithTimeout with timeout
func WithTimeout(value time.Duration) Option {
	return func(opts *options) {
		opts.timeout = value
	}
}
