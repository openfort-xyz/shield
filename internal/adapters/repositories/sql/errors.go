package sql

import "errors"

var (
	ErrMissingConfig      = errors.New("missing config")
	ErrMissingDriver      = errors.New("missing driver")
	ErrDriverNotSupported = errors.New("driver not supported")
)
