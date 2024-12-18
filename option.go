package cron

import (
	"context"
	"time"
)

type Option func(*Cron)

// WithParser allows to specify custom parser.
func WithParser(p ScheduleParser) Option {
	return func(c *Cron) {
		c.parser = p
	}
}

// WithRecover allows to enable panic recovery in job.
func WithRecover(enable bool) Option {
	return func(c *Cron) {
		c.withRecover = enable
	}
}

// WithLocation allows to specify custom location.
func WithLocation(location *time.Location) Option {
	return func(c *Cron) {
		c.location = location
	}
}

// WithLogger allows to specify custom logger.
func WithLogger(log Logger) Option {
	return func(c *Cron) {
		c.log = log
	}
}

// WithContext allows to specify custom context.
func WithContext(ctx context.Context) Option {
	return func(c *Cron) {
		c.ctx = ctx
	}
}
