package api

import (
	"encoding/json"
	"net/http"
)

type Error struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
	Status  int    `json:"-"`
}

func (e *Error) Error() string {
	return e.Message
}

func RespondWithError(w http.ResponseWriter, err *Error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.Status)
	_ = json.NewEncoder(w).Encode(err)
}

var (
	ErrProjectNotFound = &Error{"Project not found", "PJ_NOT_FOUND", http.StatusNotFound}

	ErrUnknownProviderType   = &Error{"Unknown provider type", "PV_UNKNOWN", http.StatusBadRequest}
	ErrMissingProvider       = &Error{"Missing provider", "PV_MISSING", http.StatusBadRequest}
	ErrProviderNotFound      = &Error{"Provider not found", "PV_NOT_FOUND", http.StatusNotFound}
	ErrInvalidProviderConfig = &Error{"Invalid provider config", "PV_CFG_INVALID", http.StatusBadRequest}
	ErrMissingKeyType        = &Error{"Missing key type", "PV_CFG_INVALID", http.StatusBadRequest}
	ErrProviderAlreadyExists = &Error{"Custom authentication already registered for this project", "PV_EXISTS", http.StatusConflict}
	ErrMissingUserID         = &Error{"Missing user ID", "US_ID_MISSING", http.StatusBadRequest}

	ErrShareNotFound      = &Error{"Share not found", "SH_NOT_FOUND", http.StatusNotFound}
	ErrShareAlreadyExists = &Error{"Share already exists", "SH_EXISTS", http.StatusConflict}

	ErrPreRegisterUser = &Error{"Failed to pre-register user", "US_PREREG_FAILED", http.StatusInternalServerError}

	ErrUserNotFound                = &Error{"User not found", "US_NOT_FOUND", http.StatusNotFound}
	ErrExternalUserNotFound        = &Error{"External user not found", "US_EXT_NOT_FOUND", http.StatusNotFound}
	ErrExternalUserAlreadyExists   = &Error{"External user already exists", "US_EXT_EXISTS", http.StatusConflict}
	ErrEncryptionPartRequired      = &Error{"The requested share have project entropy and encryption part is required", "EC_MISSING", http.StatusConflict}
	ErrEncryptionNotConfigured     = &Error{"Encryption not configured", "EC_MISSING", http.StatusConflict}
	ErrJWKPemConflict              = &Error{"JWK and PEM cannot be set at the same time", "PV_CFG_INVALID", http.StatusConflict}
	ErrInvalidPemCertificate       = &Error{"Invalid PEM certificate", "PV_CFG_INVALID", http.StatusBadRequest}
	ErrInvalidEncryptionPart       = &Error{"Invalid encryption part", "EC_INVALID", http.StatusBadRequest}
	ErrInvalidEncryptionSession    = &Error{"Invalid encryption session", "EC_INVALID", http.StatusBadRequest}
	ErrEncryptionPartAlreadyExists = &Error{"Encryption part already exists", "EC_EXISTS", http.StatusConflict}

	ErrMissingAPIKey         = &Error{"Missing API key", "A_MISSING", http.StatusUnauthorized}
	ErrMissingAPISecret      = &Error{"Missing API secret", "A_MISSING", http.StatusUnauthorized}
	ErrInvalidAPICredentials = &Error{"Invalid API key or API secret", "A_INVALID", http.StatusUnauthorized}
	ErrMissingToken          = &Error{"Missing token", "A_MISSING", http.StatusUnauthorized}
	ErrInvalidToken          = &Error{"Invalid token", "A_INVALID", http.StatusUnauthorized}
	ErrMissingAuthProvider   = &Error{"Missing auth provider", "A_MISSING", http.StatusUnauthorized}
	ErrInvalidAuthProvider   = &Error{"Invalid auth provider", "A_INVALID", http.StatusUnauthorized}

	ErrOTPRequired          = &Error{"OTP is required for this request", "OTP_MISSING", http.StatusPreconditionRequired}
	ErrOTPRateLimitExceeded = &Error{"Rate limit exceeded for user to generate OTP", "OTP_RATE_LIMIT", http.StatusTooManyRequests}
	ErrOTPExpired           = &Error{"OTP is expired", "OTP_EXPIRED", http.StatusUnprocessableEntity}
	ErrOTPInvalidated       = &Error{"OTP invalidated after max failed attempts", "OTP_INVALIDATED", http.StatusBadRequest}
	ErrOTPInvalid           = &Error{"Received otp is invalid", "OTP_INVALID", http.StatusBadRequest}
	ErrOTPUserInfoMissing   = &Error{"Missing user information like email or phone number", "OTP_USER_INFO_MISSING", http.StatusBadRequest}

	ErrEmailIsInvalid       = &Error{"Provided Email is invalid", "EMAIL_INVALID", http.StatusBadRequest}
	ErrPhoneNumberIsInvalid = &Error{"Provided phone number is invalid", "PHONE_INVALID", http.StatusBadRequest}

	ErrMissingNotificationService = &Error{"Missing notification service", "MISSING_NOTIFICATION_SERV", http.StatusInternalServerError}

	ErrInternal = &Error{"Internal error", "INTERNAL", http.StatusInternalServerError}
)

func ErrBadRequestWithMessage(message string) *Error {
	return &Error{message, "BAD_REQUEST", http.StatusBadRequest}
}
