package cron

import "time"

type ParserOption func(*Parser)

// WithLayout allows to specify custom layout.
func WithLayout(layout []LayoutField) ParserOption {
	return func(p *Parser) {
		p.layout = layout
	}
}

// WithDefaultLocation allows to specify default location if no location specified in expression.
func WithDefaultLocation(location *time.Location) ParserOption {
	return func(p *Parser) {
		p.defaultLoction = location
	}
}
