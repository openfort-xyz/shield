package projectsvc

import "errors"

var (
	ErrProjectNotFound = errors.New("project not found")
	ErrProjectExists   = errors.New("project exists")
)
