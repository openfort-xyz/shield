package project

import (
	"encoding/hex"
	"time"
)

type Project struct {
	ID             string
	Name           string
	APIKey         string
	APISecret      string
	EncryptionPart string
	Enable2FA      bool
	SMSRateLimit   int64
	EmailRateLimit int64

	// PendingAPISecret is the bcrypt hash of a prepared, not yet committed,
	// replacement API secret; it only authenticates the secret commit route.
	PendingAPISecret          *string
	PendingAPISecretExpiresAt *time.Time
}

type WithRateLimit struct {
	ID             string
	Name           string
	APIKey         string
	APISecret      string
	EncryptionPart string
	Enable2FA      bool
	SMSRateLimit   int64
	EmailRateLimit int64
}

// APISecretBytes returns the bytes an API secret is bcrypt-hashed as: new
// secrets are hex-encoded 32 bytes, legacy secrets are UUIDs used as is.
func APISecretBytes(apiSecret string) []byte {
	if decoded, err := hex.DecodeString(apiSecret); err == nil {
		return decoded
	}
	return []byte(apiSecret)
}

type RateLimit struct {
	ProjectID            string
	SMSRequestsPerHour   int64
	EmailRequestsPerHour int64
}
