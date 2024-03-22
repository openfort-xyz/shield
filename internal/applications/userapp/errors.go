package userapp

import (
	"errors"

	"go.openfort.xyz/shield/internal/core/domain"
)

var (
	ErrShareNotFound             = errors.New("share not found")
	ErrShareAlreadyExists        = errors.New("share already exists")
	ErrUserNotFound              = errors.New("user not found")
	ErrExternalUserNotFound      = errors.New("external user not found")
	ErrExternalUserAlreadyExists = errors.New("external user already exists")
	ErrInternal                  = errors.New("internal error")
)

func fromDomainError(err error) error {
	if errors.Is(err, domain.ErrShareNotFound) {
		return ErrShareNotFound
	}

	if errors.Is(err, domain.ErrShareAlreadyExists) {
		return ErrShareAlreadyExists
	}

	if errors.Is(err, domain.ErrUserNotFound) {
		return ErrUserNotFound
	}

	if errors.Is(err, domain.ErrExternalUserNotFound) {
		return ErrExternalUserNotFound
	}

	if errors.Is(err, domain.ErrExternalUserAlreadyExists) {
		return ErrExternalUserAlreadyExists
	}

	return ErrInternal
}
