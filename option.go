package cron

import "github.com/go-kratos/kratos/v2/log"

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

func WithLogger(log *log.Helper) Option {
	return func(c *Cron) {
		c.log = log
	}
}
