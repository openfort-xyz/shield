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
