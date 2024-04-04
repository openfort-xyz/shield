package sharehdl

import (
	"errors"

	"go.openfort.xyz/shield/internal/applications/shareapp"
	"go.openfort.xyz/shield/internal/infrastructure/handlers/rest/api"
)

func fromApplicationError(err error) *api.Error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, shareapp.ErrShareNotFound):
		return api.ErrShareNotFound
	case errors.Is(err, shareapp.ErrShareAlreadyExists):
		return api.ErrShareAlreadyExists
	case errors.Is(err, shareapp.ErrUserNotFound):
		return api.ErrUserNotFound
	case errors.Is(err, shareapp.ErrExternalUserNotFound):
		return api.ErrExternalUserNotFound
	case errors.Is(err, shareapp.ErrExternalUserAlreadyExists):
		return api.ErrExternalUserAlreadyExists
	case errors.Is(err, shareapp.ErrEncryptionPartRequired):
		return api.ErrEncryptionPartRequired
	case errors.Is(err, shareapp.ErrEncryptionNotConfigured):
		return api.ErrEncryptionNotConfigured
	case errors.Is(err, shareapp.ErrInvalidEncryptionPart):
		return api.ErrInvalidEncryptionPart
	default:
		return api.ErrInternal
	}
}
