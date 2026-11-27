package shareapp

import "github.com/openfort-xyz/shield/internal/core/ports/factories"

type options struct {
	encryptionPart        *string
	encryptionSession     *string
	sessionPolicy         factories.EncryptionSessionPolicy
	preregistration       bool
	recoveryChangeSession string
}

// partSource names where the caller's half of the encryption key came from,
// so a log line can attribute a key mismatch to the right input.
func (o options) partSource() string {
	switch {
	case o.encryptionPart != nil && *o.encryptionPart != "":
		return "part"
	case o.encryptionSession != nil && *o.encryptionSession != "":
		return "session"
	default:
		return ""
	}
}

type Option func(*options)

func WithEncryptionPart(encryptionPart string) Option {
	return func(o *options) {
		o.encryptionPart = &encryptionPart
	}
}

func WithRecoveryChangeSession(sessionID string) Option {
	return func(o *options) {
		o.recoveryChangeSession = sessionID
	}
}

func WithEncryptionSession(encryptionSession string) Option {
	return func(o *options) {
		o.encryptionSession = &encryptionSession
	}
}

// AsPreregistration marks registration authenticated with the project's API secret.
func AsPreregistration() Option {
	return func(o *options) {
		o.preregistration = true
	}
}
