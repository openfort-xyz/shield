package projectapp

import "errors"

var (
	ErrProjectNotFound     = errors.New("project not found")
	ErrNoProviderSpecified = errors.New("no provider specified")
	ErrProviderMismatch    = errors.New("provider mismatch")
)

// TODO: parse service errors
