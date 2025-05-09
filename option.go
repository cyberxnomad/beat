package beat

import (
	"context"
	"time"
)

type option func(*Beat)

// WithParser allows to specify custom parser.
func WithParser(p ScheduleParser) option {
	return func(b *Beat) {
		b.parser = p
	}
}

// WithRecover allows to enable panic recovery in job.
func WithRecover(enable bool) option {
	return func(b *Beat) {
		b.withRecover = enable
	}
}

// WithLocation allows to specify custom location.
func WithLocation(location *time.Location) option {
	return func(b *Beat) {
		b.location = location
	}
}

// WithLogger allows to specify custom logger.
func WithLogger(log Logger) option {
	return func(b *Beat) {
		b.log = log
	}
}

// WithContext allows to specify custom context.
func WithContext(ctx context.Context) option {
	return func(b *Beat) {
		b.ctx = ctx
	}
}
