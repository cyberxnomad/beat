package cron

import "time"

type ParserOption func(*Parser)

func WithLayout(layout []LayoutField) ParserOption {
	return func(p *Parser) {
		p.layout = layout
	}
}

func WithDefaultLocation(location *time.Location) ParserOption {
	return func(p *Parser) {
		p.defaultLoction = location
	}
}
