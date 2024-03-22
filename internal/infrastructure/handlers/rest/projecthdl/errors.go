package projecthdl

import (
	"errors"

	"go.openfort.xyz/shield/internal/applications/projectapp"
	"go.openfort.xyz/shield/internal/infrastructure/handlers/rest/api"
)

func fromApplicationError(err error) *api.Error {
	switch {
	case errors.Is(err, projectapp.ErrProjectNotFound):
		return api.ErrProjectNotFound
	case errors.Is(err, projectapp.ErrNoProviderSpecified):
		return api.ErrMissingProvider
	case errors.Is(err, projectapp.ErrProviderMismatch):
		return api.ErrInvalidProviderConfig
	case errors.Is(err, projectapp.ErrInvalidProviderConfig):
		return api.ErrInvalidProviderConfig
	case errors.Is(err, projectapp.ErrUnknownProviderType):
		return api.ErrUnknownProviderType
	case errors.Is(err, projectapp.ErrProviderAlreadyExists):
		return api.ErrProviderAlreadyExists
	case errors.Is(err, projectapp.ErrProviderNotFound):
		return api.ErrProviderNotFound
	default:
		return api.ErrInternal
	}
}