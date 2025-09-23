package otp

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.openfort.xyz/shield/internal/adapters/repositories/mocks/encryptionpartsmockrepo"
	"go.openfort.xyz/shield/internal/core/domain/errors"
	"go.openfort.xyz/shield/internal/core/domain/otp"
)

type TestClock struct {
	currentTime time.Time
}

func (tc *TestClock) Now() time.Time {
	return tc.currentTime
}

func (tc *TestClock) SetNewTime(t time.Time) {
	tc.currentTime = t
}

func TestOtp(t *testing.T) {
	t.Run("Positive OTP verification", func(t *testing.T) {
		ctx := context.TODO()
		ass := assert.New(t)

		tClock := TestClock{}
		testStartTime := time.Now()

		tClock.SetNewTime(testStartTime)

		encryptionPartsRepo := new(encryptionpartsmockrepo.MockEncryptionPartsRepository)
		encryptionPartsRepo.ExpectedCalls = nil
		encryptionPartsRepo.On("Set", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		encryptionPartsRepo.On("Delete", mock.Anything, mock.Anything).Return(nil)
		encryptionPartsRepo.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

		config := OnboardingTrackerConfig{
			WindowMS:              DefaultSecurityConfig.UserOnboardingWindowMS,
			OTPGenerationWindowMS: DefaultSecurityConfig.OTPGenerationWindowMS,
			MaxAttempts:           DefaultSecurityConfig.MaxUserOnboardAttempts,
		}
		otpOnbTracker := NewOnboardingTracker(config, &tClock)

		otpService, err := NewInMemoryOTPService(encryptionPartsRepo, otpOnbTracker, DefaultSecurityConfig, &tClock)
		if err != nil {
			panic(err)
		}

		testUserID := "testUserID12345"

		newOtp, err := otpService.GenerateOTP(ctx, testUserID)
		if err != nil {
			panic(err)
		}

		otpReq := otp.Request{
			OTP:            newOtp,
			CreatedAt:      tClock.Now().UnixMilli(),
			FailedAttempts: 0,
		}
		marshaledReq, err := json.Marshal(otpReq)
		if err != nil {
			panic(err)
		}

		encryptionPartsRepo.On("Get", mock.Anything, mock.Anything).Return(string(marshaledReq), nil)

		verifyResp, err := otpService.VerifyOTP(ctx, testUserID, newOtp)
		if err != nil {
			panic(err)
		}

		ass.NotNil(verifyResp)
	})

	t.Run("Negative OTP verification", func(t *testing.T) {
		ctx := context.TODO()
		ass := assert.New(t)

		tClock := TestClock{}
		testStartTime := time.Now()

		tClock.SetNewTime(testStartTime)

		encryptionPartsRepo := new(encryptionpartsmockrepo.MockEncryptionPartsRepository)
		encryptionPartsRepo.ExpectedCalls = nil
		encryptionPartsRepo.On("Set", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		encryptionPartsRepo.On("Delete", mock.Anything, mock.Anything).Return(nil)
		encryptionPartsRepo.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

		config := OnboardingTrackerConfig{
			WindowMS:              DefaultSecurityConfig.UserOnboardingWindowMS,
			OTPGenerationWindowMS: DefaultSecurityConfig.OTPGenerationWindowMS,
			MaxAttempts:           DefaultSecurityConfig.MaxUserOnboardAttempts,
		}
		otpOnbTracker := NewOnboardingTracker(config, &tClock)

		otpService, err := NewInMemoryOTPService(encryptionPartsRepo, otpOnbTracker, DefaultSecurityConfig, &tClock)
		if err != nil {
			panic(err)
		}

		testUserID := "testUserID12345"

		newOtp, err := otpService.GenerateOTP(ctx, testUserID)
		if err != nil {
			panic(err)
		}

		otpReq := otp.Request{
			OTP:            "123",
			CreatedAt:      tClock.Now().UnixMilli(),
			FailedAttempts: 0,
		}
		marshaledReq, err := json.Marshal(otpReq)
		if err != nil {
			panic(err)
		}

		encryptionPartsRepo.On("Get", mock.Anything, mock.Anything).Return(string(marshaledReq), nil)

		_, err = otpService.VerifyOTP(ctx, testUserID, newOtp)

		ass.ErrorIs(err, errors.ErrOTPInvalid)
	})

	t.Run("Expired OTP", func(t *testing.T) {
		ctx := context.TODO()
		ass := assert.New(t)

		tClock := TestClock{}
		testStartTime := time.Now()

		tClock.SetNewTime(testStartTime)

		encryptionPartsRepo := new(encryptionpartsmockrepo.MockEncryptionPartsRepository)
		encryptionPartsRepo.ExpectedCalls = nil
		encryptionPartsRepo.On("Set", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		encryptionPartsRepo.On("Delete", mock.Anything, mock.Anything).Return(nil)
		encryptionPartsRepo.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

		config := OnboardingTrackerConfig{
			WindowMS:              DefaultSecurityConfig.UserOnboardingWindowMS,
			OTPGenerationWindowMS: DefaultSecurityConfig.OTPGenerationWindowMS,
			MaxAttempts:           DefaultSecurityConfig.MaxUserOnboardAttempts,
		}
		otpOnbTracker := NewOnboardingTracker(config, &tClock)

		otpService, err := NewInMemoryOTPService(encryptionPartsRepo, otpOnbTracker, DefaultSecurityConfig, &tClock)
		if err != nil {
			panic(err)
		}

		testUserID := "testUserID12345"

		newOtp, err := otpService.GenerateOTP(ctx, testUserID)
		if err != nil {
			panic(err)
		}

		otpReq := otp.Request{
			OTP:            newOtp,
			CreatedAt:      tClock.Now().Add(-1 * time.Hour).UnixMilli(),
			FailedAttempts: 0,
		}
		marshaledReq, err := json.Marshal(otpReq)
		if err != nil {
			panic(err)
		}

		encryptionPartsRepo.On("Get", mock.Anything, mock.Anything).Return(string(marshaledReq), nil)

		_, err = otpService.VerifyOTP(ctx, testUserID, newOtp)

		ass.ErrorIs(err, errors.ErrOTPExpired)
	})
}

func TestOtpBruteForce(t *testing.T) {
	t.Run("BruteForce OTP", func(t *testing.T) {
		ctx := context.TODO()
		ass := assert.New(t)

		tClock := TestClock{}
		testStartTime := time.Now()

		tClock.SetNewTime(testStartTime)

		encryptionPartsRepo := new(encryptionpartsmockrepo.MockEncryptionPartsRepository)
		encryptionPartsRepo.ExpectedCalls = nil
		encryptionPartsRepo.On("Set", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		encryptionPartsRepo.On("Delete", mock.Anything, mock.Anything).Return(nil)
		encryptionPartsRepo.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

		config := OnboardingTrackerConfig{
			WindowMS:              DefaultSecurityConfig.UserOnboardingWindowMS,
			OTPGenerationWindowMS: DefaultSecurityConfig.OTPGenerationWindowMS,
			MaxAttempts:           DefaultSecurityConfig.MaxUserOnboardAttempts,
		}
		otpOnbTracker := NewOnboardingTracker(config, &tClock)

		otpService, err := NewInMemoryOTPService(encryptionPartsRepo, otpOnbTracker, DefaultSecurityConfig, &tClock)
		if err != nil {
			panic(err)
		}

		testUserID := "testUserID12345"

		newOtp, err := otpService.GenerateOTP(ctx, testUserID)
		if err != nil {
			panic(err)
		}

		for i := 0; i < 3; i++ {
			otpReq := otp.Request{
				OTP:            "123",
				CreatedAt:      tClock.Now().UnixMilli(),
				FailedAttempts: 2,
			}
			marshaledReq, err := json.Marshal(otpReq)
			if err != nil {
				panic(err)
			}

			encryptionPartsRepo.On("Get", mock.Anything, mock.Anything).Return(string(marshaledReq), nil)

			_, err = otpService.VerifyOTP(ctx, testUserID, newOtp)

			ass.ErrorIs(err, errors.ErrOTPInvalidated)
		}

		_, err = otpService.GenerateOTP(ctx, testUserID)

		ass.ErrorIs(err, errors.ErrOTPRateLimitExceeded)
	})

	t.Run("OTP next window", func(t *testing.T) {
		ctx := context.TODO()
		ass := assert.New(t)

		tClock := TestClock{}
		testStartTime := time.Now()

		tClock.SetNewTime(testStartTime)

		encryptionPartsRepo := new(encryptionpartsmockrepo.MockEncryptionPartsRepository)
		encryptionPartsRepo.ExpectedCalls = nil
		encryptionPartsRepo.On("Set", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		encryptionPartsRepo.On("Delete", mock.Anything, mock.Anything).Return(nil)
		encryptionPartsRepo.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

		config := OnboardingTrackerConfig{
			WindowMS:              DefaultSecurityConfig.UserOnboardingWindowMS,
			OTPGenerationWindowMS: DefaultSecurityConfig.OTPGenerationWindowMS,
			MaxAttempts:           DefaultSecurityConfig.MaxUserOnboardAttempts,
		}
		otpOnbTracker := NewOnboardingTracker(config, &tClock)

		otpService, err := NewInMemoryOTPService(encryptionPartsRepo, otpOnbTracker, DefaultSecurityConfig, &tClock)
		if err != nil {
			panic(err)
		}

		testUserID := "testUserID12345"

		newOtp, err := otpService.GenerateOTP(ctx, testUserID)
		if err != nil {
			panic(err)
		}

		for i := 0; i < 3; i++ {
			otpReq := otp.Request{
				OTP:            "123",
				CreatedAt:      tClock.Now().UnixMilli(),
				FailedAttempts: 2,
			}
			marshaledReq, err := json.Marshal(otpReq)
			if err != nil {
				panic(err)
			}

			encryptionPartsRepo.On("Get", mock.Anything, mock.Anything).Return(string(marshaledReq), nil)

			_, err = otpService.VerifyOTP(ctx, testUserID, newOtp)

			ass.ErrorIs(err, errors.ErrOTPInvalidated)
		}

		_, err = otpService.GenerateOTP(ctx, testUserID)

		ass.ErrorIs(err, errors.ErrOTPRateLimitExceeded)

		tClock.SetNewTime(tClock.Now().Add(7 * time.Hour))

		_, err = otpService.GenerateOTP(ctx, testUserID)

		ass.Nil(err)
	})
}
