package share

// EncryptionPart is an encryption session record: the project's half of the
// key, held for one use in the project it was minted for. UserID is the
// external user ID it was minted for.
type EncryptionPart struct {
	EncPart     string `json:"encryption_part"`
	ProjectID   string `json:"project_id"`
	UserID      string `json:"user_id"`
	OTPVerified bool   `json:"otp_verified"`
}

// EncryptionSessionKey is the store key of a session. Session IDs reach the
// store from callers, and OTP records share it, so sessions get their own
// prefix.
func EncryptionSessionKey(sessionID string) string {
	return "encryption_session:" + sessionID
}
