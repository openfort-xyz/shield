package projectapp

import (
	"errors"

	domainErrors "go.openfort.xyz/shield/internal/core/domain/errors"
)

var (
	ErrProjectNotFound                  = errors.New("project not found")
	ErrNoProviderSpecified              = errors.New("no provider specified")
	ErrProviderMismatch                 = errors.New("provider mismatch")
	ErrKeyTypeNotSpecified              = errors.New("key type not specified")
	ErrInvalidProviderConfig            = errors.New("invalid provider config")
	ErrUnknownProviderType              = errors.New("unknown provider type")
	ErrProviderAlreadyExists            = errors.New("custom authentication already registered for this project")
	ErrProviderNotFound                 = errors.New("custom authentication not found")
	ErrInvalidEncryptionPart            = errors.New("invalid encryption part")
	ErrInvalidEncryptionSession         = errors.New("invalid encryption session")
	ErrEncryptionPartAlreadyExists      = errors.New("encryption part already exists")
	ErrEncryptionNotConfigured          = errors.New("encryption not configured")
	ErrJWKPemConflict                   = errors.New("jwk and pem cannot be set at the same time")
	ErrInvalidPemCertificate            = errors.New("invalid PEM certificate")
	ErrOTPRequired                      = errors.New("OTP is required for this request")
	ErrOTPRateLimitExceeded             = errors.New("rate limit exceeded")
	ErrOTPFailedToGenerate              = errors.New("failed to generate OTP")
	ErrOTPFailedToMarshal               = errors.New("failed to marshal OTP request")
	ErrOTPExpired                       = errors.New("otp was expired")
	ErrOTPInvalidated                   = errors.New("otp invalidated after max failed attempts")
	ErrOTPInvalid                       = errors.New("received otp is invalid")
	ErrOTPUserInfoMissing               = errors.New("neither email nor phone number was provided")
	ErrOTPRecordNotFound                = errors.New("otp record not found")
	ErrEmailIsInvalid                   = errors.New("email is invalid")
	ErrPhoneNumberIsInvalid             = errors.New("phone number is invalid")
	ErrMissingNotificationService       = errors.New("cannot generate OTP because notification service is absent")
	ErrProjectDoesntHave2FA             = errors.New("project doesn't have 2FA enabled")
	ErrProject2FAAlreadyEnabled         = errors.New("project already has 2FA enabled")
	ErrUserContactInformationMismatch   = errors.New("user contact information mismatch")
	ErrNoUserContactInformationProvided = errors.New("no user contact information provided")
	ErrInternal                         = errors.New("internal error")
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

	if errors.Is(err, domainErrors.ErrOTPRateLimitExceeded) {
		return ErrOTPRateLimitExceeded
	}

	if errors.Is(err, domainErrors.ErrOTPFailedToGenerate) {
		return ErrInternal
	}

	if errors.Is(err, domainErrors.ErrOTPFailedToMarshal) {
		return ErrInternal
	}

	if errors.Is(err, domainErrors.ErrOTPExpired) {
		return ErrOTPExpired
	}

	if errors.Is(err, domainErrors.ErrOTPInvalidated) {
		return ErrOTPInvalidated
	}

	if errors.Is(err, domainErrors.ErrOTPInvalid) {
		return ErrOTPInvalid
	}

	return ErrInternal
}
