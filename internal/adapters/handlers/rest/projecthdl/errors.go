package projecthdl

import (
	"errors"

	"github.com/openfort-xyz/shield/internal/adapters/handlers/rest/api"
	"github.com/openfort-xyz/shield/internal/applications/projectapp"
)

// applicationErrorMapping maps application-layer errors to their API-layer counterparts.
// The list is iterated in order; the first error that matches via errors.Is wins.
var applicationErrorMapping = []struct {
	app error
	api *api.Error
}{
	{projectapp.ErrProjectNotFound, api.ErrProjectNotFound},
	{projectapp.ErrNoProviderSpecified, api.ErrMissingProvider},
	{projectapp.ErrProviderMismatch, api.ErrInvalidProviderConfig},
	{projectapp.ErrKeyTypeNotSpecified, api.ErrMissingKeyType},
	{projectapp.ErrInvalidProviderConfig, api.ErrInvalidProviderConfig},
	{projectapp.ErrUnknownProviderType, api.ErrUnknownProviderType},
	{projectapp.ErrProviderAlreadyExists, api.ErrProviderAlreadyExists},
	{projectapp.ErrProviderNotFound, api.ErrProviderNotFound},
	{projectapp.ErrInvalidEncryptionPart, api.ErrInvalidEncryptionPart},
	{projectapp.ErrInvalidEncryptionSession, api.ErrInvalidEncryptionSession},
	{projectapp.ErrEncryptionPartAlreadyExists, api.ErrEncryptionPartAlreadyExists},
	{projectapp.ErrEncryptionNotConfigured, api.ErrEncryptionNotConfigured},
	{projectapp.ErrJWKPemConflict, api.ErrJWKPemConflict},
	{projectapp.ErrInvalidPemCertificate, api.ErrInvalidPemCertificate},
	{projectapp.ErrOTPRequired, api.ErrOTPRequired},
	{projectapp.ErrOTPRateLimitExceeded, api.ErrOTPRateLimitExceeded},
	{projectapp.ErrOTPExpired, api.ErrOTPExpired},
	{projectapp.ErrOTPInvalidated, api.ErrOTPInvalidated},
	{projectapp.ErrOTPInvalid, api.ErrOTPInvalid},
	{projectapp.ErrOTPUserInfoMissing, api.ErrOTPUserInfoMissing},
	{projectapp.ErrOTPMissing, api.ErrOTPMissing},
	{projectapp.ErrEmailIsInvalid, api.ErrEmailIsInvalid},
	{projectapp.ErrPhoneNumberIsInvalid, api.ErrPhoneNumberIsInvalid},
	{projectapp.ErrMissingNotificationService, api.ErrMissingNotificationService},
	{projectapp.ErrProjectDoesntHave2FA, api.ErrProjectDoesntHave2FA},
	{projectapp.ErrProject2FAAlreadyEnabled, api.ErrProject2FAAlreadyEnabled},
	{projectapp.ErrOTPRecordNotFound, api.ErrOTPRecordNotFound},
	{projectapp.ErrUserContactInformationMismatch, api.ErrUserContactInformationMismatch},
	{projectapp.ErrNoUserContactInformationProvided, api.ErrOTPUserInfoMissing},
}

func fromApplicationError(err error) *api.Error {
	if err == nil {
		return nil
	}

	for _, m := range applicationErrorMapping {
		if errors.Is(err, m.app) {
			return m.api
		}
	}

	return api.ErrInternal
}
