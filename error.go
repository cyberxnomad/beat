package beat

import "errors"

var (
	ErrInvalidExp  = errors.New("invalid expression")
	ErrJobExist    = errors.New("job already exists")
	ErrJobNotExist = errors.New("job does not exists")
)
