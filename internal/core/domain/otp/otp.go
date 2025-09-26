package otp

// OTP Request represents a pending OTP verification request
type Request struct {
	OTP              string `json:"otp"`
	CreatedAt        int64  `json:"created_at"`
	FailedAttempts   int    `json:"failed_attempts"`
	SkipVerification bool   `json:"skip_verification"`
}
