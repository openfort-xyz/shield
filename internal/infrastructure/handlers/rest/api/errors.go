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

	ErrShareNotFound      = &Error{"Share not found", "SH_NOT_FOUND", http.StatusNotFound}
	ErrShareAlreadyExists = &Error{"Share already exists", "SH_EXISTS", http.StatusConflict}

	ErrUserNotFound                = &Error{"User not found", "US_NOT_FOUND", http.StatusNotFound}
	ErrExternalUserNotFound        = &Error{"External user not found", "US_EXT_NOT_FOUND", http.StatusNotFound}
	ErrExternalUserAlreadyExists   = &Error{"External user already exists", "US_EXT_EXISTS", http.StatusConflict}
	ErrEncryptionPartRequired      = &Error{"The requested share have project entropy and encryption part is required", "EC_MISSING", http.StatusConflict}
	ErrEncryptionNotConfigured     = &Error{"Encryption not configured", "EC_MISSING", http.StatusConflict}
	ErrJWKPemConflict              = &Error{"JWK and PEM cannot be set at the same time", "PV_CFG_INVALID", http.StatusConflict}
	ErrInvalidEncryptionPart       = &Error{"Invalid encryption part", "EC_INVALID", http.StatusBadRequest}
	ErrEncryptionPartAlreadyExists = &Error{"Encryption part already exists", "EC_EXISTS", http.StatusConflict}
	ErrAllowedOriginNotFound       = &Error{"Allowed origin not found", "AO_NOT_FOUND", http.StatusNotFound}

	ErrMissingAPIKey       = &Error{"Missing API key", "A_MISSING", http.StatusUnauthorized}
	ErrMissingAPISecret    = &Error{"Missing API secret", "A_MISSING", http.StatusUnauthorized}
	ErrInvalidAPISecret    = &Error{"Invalid API secret", "A_INVALID", http.StatusUnauthorized}
	ErrMissingToken        = &Error{"Missing token", "A_MISSING", http.StatusUnauthorized}
	ErrInvalidToken        = &Error{"Invalid token", "A_INVALID", http.StatusUnauthorized}
	ErrMissingAuthProvider = &Error{"Missing auth provider", "A_MISSING", http.StatusUnauthorized}
	ErrInvalidAuthProvider = &Error{"Invalid auth provider", "A_INVALID", http.StatusUnauthorized}

	ErrInternal = &Error{"Internal error", "INTERNAL", http.StatusInternalServerError}
)

func ErrBadRequestWithMessage(message string) *Error {
	return &Error{message, "BAD_REQUEST", http.StatusBadRequest}
}
