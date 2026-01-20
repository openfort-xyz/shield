package errors

import "errors"

var (
	ErrOTPRateLimitExceeded = errors.New("rate limit exceeded")
	ErrOTPFailedToGenerate  = errors.New("failed to generate OTP")
	ErrOTPFailedToMarshal   = errors.New("failed to marshal OTP request")
	ErrOTPExpired           = errors.New("otp was expired")
	ErrOTPInvalidated       = errors.New("otp invalidated after max failed attempts")
	ErrOTPInvalid           = errors.New("received otp is invalid")
	ErrOTPMissing           = errors.New("otp is missing")
)
