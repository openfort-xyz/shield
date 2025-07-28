package projectapp

import (
	"errors"

	domainErrors "go.openfort.xyz/shield/internal/core/domain/errors"
)

var (
	ErrProjectNotFound             = errors.New("project not found")
	ErrNoProviderSpecified         = errors.New("no provider specified")
	ErrProviderMismatch            = errors.New("provider mismatch")
	ErrKeyTypeNotSpecified         = errors.New("key type not specified")
	ErrInvalidProviderConfig       = errors.New("invalid provider config")
	ErrUnknownProviderType         = errors.New("unknown provider type")
	ErrProviderAlreadyExists       = errors.New("custom authentication already registered for this project")
	ErrProviderNotFound            = errors.New("custom authentication not found")
	ErrInvalidEncryptionPart       = errors.New("invalid encryption part")
	ErrInvalidEncryptionSession    = errors.New("invalid encryption session")
	ErrEncryptionPartAlreadyExists = errors.New("encryption part already exists")
	ErrEncryptionNotConfigured     = errors.New("encryption not configured")
	ErrJWKPemConflict              = errors.New("jwk and pem cannot be set at the same time")
	ErrInvalidPemCertificate       = errors.New("invalid PEM certificate")
	ErrInternal                    = errors.New("internal error")
)

func fromDomainError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, domainErrors.ErrProjectNotFound) {
		return ErrProjectNotFound
	}

	if errors.Is(err, domainErrors.ErrInvalidProviderConfig) {
		return ErrInvalidProviderConfig
	}

	if errors.Is(err, domainErrors.ErrUnknownProviderType) {
		return ErrUnknownProviderType
	}

	if errors.Is(err, domainErrors.ErrProviderAlreadyExists) {
		return ErrProviderAlreadyExists
	}

	if errors.Is(err, domainErrors.ErrProviderNotFound) {
		return ErrProviderNotFound
	}

	if errors.Is(err, domainErrors.ErrEncryptionPartNotFound) {
		return ErrEncryptionNotConfigured
	}

	if errors.Is(err, domainErrors.ErrInvalidEncryptionSession) {
		return ErrInvalidEncryptionSession
	}

	if errors.Is(err, domainErrors.ErrInvalidEncryptionPart) {
		return ErrInvalidEncryptionPart
	}
	return ErrInternal
}
