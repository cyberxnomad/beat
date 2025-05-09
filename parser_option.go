package beat

import "time"

type parserOption func(*Parser)

// WithLayout allows to specify custom layout.
func WithLayout(layout []LayoutField) parserOption {
	return func(p *Parser) {
		p.layout = layout
	}
}

// WithDefaultLocation allows to specify default location if no location specified in expression.
func WithDefaultLocation(location *time.Location) parserOption {
	return func(p *Parser) {
		p.defaultLoction = location
	}
}
