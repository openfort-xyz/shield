package shareapp

import (
	"errors"
	domainErrors "go.openfort.xyz/shield/internal/core/domain/errors"
)

var (
	ErrShareNotFound             = errors.New("share not found")
	ErrShareAlreadyExists        = errors.New("share already exists")
	ErrUserNotFound              = errors.New("user not found")
	ErrExternalUserNotFound      = errors.New("external user not found")
	ErrExternalUserAlreadyExists = errors.New("external user already exists")
	ErrEncryptionPartRequired    = errors.New("encryption part is required")
	ErrEncryptionNotConfigured   = errors.New("encryption not configured")
	ErrInvalidEncryptionPart     = errors.New("invalid encryption part")
	ErrInternal                  = errors.New("internal error")
)

func fromDomainError(err error) error {
	if errors.Is(err, domainErrors.ErrShareNotFound) {
		return ErrShareNotFound
	}

	if errors.Is(err, domainErrors.ErrShareAlreadyExists) {
		return ErrShareAlreadyExists
	}

	if errors.Is(err, domainErrors.ErrEncryptionPartRequired) {
		return ErrEncryptionPartRequired
	}

	if errors.Is(err, domainErrors.ErrEncryptionPartNotFound) {
		return ErrEncryptionNotConfigured
	}

	return ErrInternal
}
