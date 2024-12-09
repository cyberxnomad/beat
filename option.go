package cron

import (
	"time"
)

type Option func(*Cron)

func WithParser(p ScheduleParser) Option {
	return func(c *Cron) {
		c.parser = p
	}
}

func WithRecover(enable bool) Option {
	return func(c *Cron) {
		c.withRecover = enable
	}
}

func WithLocation(location *time.Location) Option {
	return func(c *Cron) {
		c.location = location
	}
}

func WithLogger(log Logger) Option {
	return func(c *Cron) {
		c.log = log
	}
}
