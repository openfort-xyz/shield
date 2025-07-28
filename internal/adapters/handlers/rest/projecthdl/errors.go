package projecthdl

import (
	"errors"

	"go.openfort.xyz/shield/internal/adapters/handlers/rest/api"
	"go.openfort.xyz/shield/internal/applications/projectapp"
)

func fromApplicationError(err error) *api.Error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, projectapp.ErrProjectNotFound):
		return api.ErrProjectNotFound
	case errors.Is(err, projectapp.ErrNoProviderSpecified):
		return api.ErrMissingProvider
	case errors.Is(err, projectapp.ErrProviderMismatch):
		return api.ErrInvalidProviderConfig
	case errors.Is(err, projectapp.ErrKeyTypeNotSpecified):
		return api.ErrMissingKeyType
	case errors.Is(err, projectapp.ErrInvalidProviderConfig):
		return api.ErrInvalidProviderConfig
	case errors.Is(err, projectapp.ErrUnknownProviderType):
		return api.ErrUnknownProviderType
	case errors.Is(err, projectapp.ErrProviderAlreadyExists):
		return api.ErrProviderAlreadyExists
	case errors.Is(err, projectapp.ErrProviderNotFound):
		return api.ErrProviderNotFound
	case errors.Is(err, projectapp.ErrInvalidEncryptionPart):
		return api.ErrInvalidEncryptionPart
	case errors.Is(err, projectapp.ErrInvalidEncryptionSession):
		return api.ErrInvalidEncryptionSession
	case errors.Is(err, projectapp.ErrEncryptionPartAlreadyExists):
		return api.ErrEncryptionPartAlreadyExists
	case errors.Is(err, projectapp.ErrEncryptionNotConfigured):
		return api.ErrEncryptionNotConfigured
	case errors.Is(err, projectapp.ErrJWKPemConflict):
		return api.ErrJWKPemConflict
	case errors.Is(err, projectapp.ErrInvalidPemCertificate):
		return api.ErrInvalidPemCertificate
	default:
		return api.ErrInternal
	}
}
