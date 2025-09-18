package otp

// OTPRequest represents a pending OTP verification request
type OTPRequest struct {
	OTP            string `json:"otp"`
	CreatedAt      int64  `json:"created_at"`
	FailedAttempts int    `json:"failed_attempts"`
}
