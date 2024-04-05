package api

import (
	"encoding/json"
	"net/http"
)

type Error struct {
	Message string `json:"message"`
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
	ErrProjectNotFound = &Error{"Project not found", http.StatusNotFound}

	ErrUnknownProviderType   = &Error{"Unknown provider type", http.StatusBadRequest}
	ErrMissingProvider       = &Error{"Missing provider", http.StatusBadRequest}
	ErrProviderNotFound      = &Error{"Provider not found", http.StatusNotFound}
	ErrInvalidProviderConfig = &Error{"Invalid provider config", http.StatusBadRequest}
	ErrProviderAlreadyExists = &Error{"Custom authentication already registered for this project", http.StatusConflict}

	ErrShareNotFound      = &Error{"Share not found", http.StatusNotFound}
	ErrShareAlreadyExists = &Error{"Share already exists", http.StatusConflict}

	ErrUserNotFound                = &Error{"User not found", http.StatusNotFound}
	ErrExternalUserNotFound        = &Error{"External user not found", http.StatusNotFound}
	ErrExternalUserAlreadyExists   = &Error{"External user already exists", http.StatusConflict}
	ErrEncryptionPartRequired      = &Error{"The requested share have project entropy and encryption part is required", http.StatusConflict}
	ErrEncryptionNotConfigured     = &Error{"Encryption not configured", http.StatusConflict}
	ErrInvalidEncryptionPart       = &Error{"Invalid encryption part", http.StatusBadRequest}
	ErrEncryptionPartAlreadyExists = &Error{"Encryption part already exists", http.StatusConflict}
	ErrAllowedOriginNotFound       = &Error{"Allowed origin not found", http.StatusNotFound}

	ErrMissingAPIKey       = &Error{"Missing API key", http.StatusUnauthorized}
	ErrMissingAPISecret    = &Error{"Missing API secret", http.StatusUnauthorized}
	ErrInvalidAPISecret    = &Error{"Invalid API secret", http.StatusUnauthorized}
	ErrMissingToken        = &Error{"Missing token", http.StatusUnauthorized}
	ErrInvalidToken        = &Error{"Invalid token", http.StatusUnauthorized}
	ErrMissingAuthProvider = &Error{"Missing auth provider", http.StatusUnauthorized}
	ErrInvalidAuthProvider = &Error{"Invalid auth provider", http.StatusUnauthorized}

	ErrInternal = &Error{"Internal error", http.StatusInternalServerError}
)

func ErrBadRequestWithMessage(message string) *Error {
	return &Error{message, http.StatusBadRequest}
}
