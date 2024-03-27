package userhdl

import (
	"errors"

	"go.openfort.xyz/shield/internal/applications/userapp"
	"go.openfort.xyz/shield/internal/infrastructure/handlers/rest/api"
)

func fromApplicationError(err error) *api.Error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, userapp.ErrShareNotFound):
		return api.ErrShareNotFound
	case errors.Is(err, userapp.ErrShareAlreadyExists):
		return api.ErrShareAlreadyExists
	case errors.Is(err, userapp.ErrUserNotFound):
		return api.ErrUserNotFound
	case errors.Is(err, userapp.ErrExternalUserNotFound):
		return api.ErrExternalUserNotFound
	case errors.Is(err, userapp.ErrExternalUserAlreadyExists):
		return api.ErrExternalUserAlreadyExists
	default:
		return api.ErrInternal
	}
}
