package beat

import (
	"context"
	"time"
)

type Option func(*Beat)

// WithParser allows to specify custom parser.
func WithParser(p ScheduleParser) Option {
	return func(b *Beat) {
		b.parser = p
	}
}

// WithRecover allows to enable panic recovery in job.
func WithRecover(enable bool) Option {
	return func(b *Beat) {
		b.withRecover = enable
	}
}

// WithLocation allows to specify custom location.
func WithLocation(location *time.Location) Option {
	return func(b *Beat) {
		b.location = location
	}
}

// WithLogger allows to specify custom logger.
func WithLogger(log Logger) Option {
	return func(b *Beat) {
		b.log = log
	}
}

// WithContext allows to specify custom context.
func WithContext(ctx context.Context) Option {
	return func(b *Beat) {
		b.ctx = ctx
	}
}
