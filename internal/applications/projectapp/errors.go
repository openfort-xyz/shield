package projectapp

import "errors"

var (
	ErrProjectNotFound     = errors.New("project not found")
	ErrNoProviderSpecified = errors.New("no provider specified")
)

// TODO: parse service errors
