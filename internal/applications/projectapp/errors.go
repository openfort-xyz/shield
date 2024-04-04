package projectapp

import (
	"errors"

	"go.openfort.xyz/shield/internal/core/domain"
)

var (
	ErrProjectNotFound       = errors.New("project not found")
	ErrNoProviderSpecified   = errors.New("no provider specified")
	ErrProviderMismatch      = errors.New("provider mismatch")
	ErrInvalidProviderConfig = errors.New("invalid provider config")
	ErrUnknownProviderType   = errors.New("unknown provider type")
	ErrProviderAlreadyExists = errors.New("custom authentication already registered for this project")
	ErrProviderNotFound      = errors.New("custom authentication not found")
	ErrInvalidEncryptionPart = errors.New("invalid encryption part")
	ErrInternal              = errors.New("internal error")
)

func fromDomainError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, domain.ErrProjectNotFound) {
		return ErrProjectNotFound
	}

	if errors.Is(err, domain.ErrInvalidProviderConfig) {
		return ErrInvalidProviderConfig
	}

	if errors.Is(err, domain.ErrUnknownProviderType) {
		return ErrUnknownProviderType
	}

	if errors.Is(err, domain.ErrProviderAlreadyExists) {
		return ErrProviderAlreadyExists
	}

	if errors.Is(err, domain.ErrProviderNotFound) {
		return ErrProviderNotFound
	}

	return ErrInternal
}
