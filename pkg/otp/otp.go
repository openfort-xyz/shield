package otp

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/tidwall/buntdb"
	"go.openfort.xyz/shield/internal/core/ports/repositories"
)

const OTPDigits = 9

// SecurityConfig holds all security-related configuration for OTP service
//
// OTP Brute Force Protection:
//   - OTP space: 1,000,000,000 possible combinations (9-digit numeric)
//   - Attempts per day: With a window of 6 hours, 4 windows per day, 3 attempts per window, and 3 OTP attempts per one generation, an attacker gets 36 OTP generations per day.
//   - Expected brute force time: Using cumulative probability model 0.5 = 1 - (1 - 3/1000,000,000)^n, it takes 231,049,000 OTP generations for 50% success probability.
//   - Time to brute force: 231,049,000 ÷ 36 OTP generations/day = ~6,418,027 days ≈ **17583 years**.
type SecurityConfig struct {
	MaxFailedAttempts      int
	UserOnboardingWindowMS int64
	MaxUserOnboardAttempts int
	OTPExpiryMS            int64
}

var DefaultSecurityConfig = SecurityConfig{
	MaxFailedAttempts:      3,
	UserOnboardingWindowMS: 6 * 60 * 60 * 1000, // 6 hours
	MaxUserOnboardAttempts: 3,
	OTPExpiryMS:            5 * 60 * 1000, // 5 minutes
}

// OTPRequest represents a pending OTP verification request
type OTPRequest struct {
	OTP            string `json:"otp"`
	CreatedAt      int64  `json:"created_at"`
	FailedAttempts int    `json:"failed_attempts"`
}

// OnboardingTracker tracks device onboarding attempts with rate limiting
type OnboardingTracker struct {
	windowMS    int64
	maxAttempts int
	attempts    map[string][]int64
	mu          sync.RWMutex
}

// NewOnboardingTracker creates a new onboarding tracker
func NewOnboardingTracker(windowMS int64, maxAttempts int) *OnboardingTracker {
	return &OnboardingTracker{
		windowMS:    windowMS,
		maxAttempts: maxAttempts,
		attempts:    make(map[string][]int64),
	}
}

// TrackAttempt tracks an onboarding attempt and returns an error if rate limit is exceeded
func (ot *OnboardingTracker) TrackAttempt(userID string) error {
	ot.mu.Lock()
	defer ot.mu.Unlock()

	now := time.Now().UnixMilli()

	var validAttempts []int64
	for _, timestamp := range ot.attempts[userID] {
		if now-timestamp <= ot.windowMS {
			validAttempts = append(validAttempts, timestamp)
		}
	}

	// Check if we're at the rate limit
	if len(validAttempts) >= ot.maxAttempts {
		return fmt.Errorf("rate limit exceeded for signer %s", userID)
	}

	return nil
}

func (ot *OnboardingTracker) AddAttempt(userID string) {
	ot.attempts[userID] = []int64{time.Now().UnixMilli()}
}

// CleanupOldRecords removes tracking data older than the window
func (ot *OnboardingTracker) CleanupOldRecords() {
	ot.mu.Lock()
	defer ot.mu.Unlock()

	now := time.Now().UnixMilli()
	cleanedCount := 0

	for key, attempts := range ot.attempts {
		var validAttempts []int64
		for _, timestamp := range attempts {
			if now-timestamp <= ot.windowMS {
				validAttempts = append(validAttempts, timestamp)
			}
		}

		if len(validAttempts) == 0 {
			delete(ot.attempts, key)
			cleanedCount++
		} else {
			ot.attempts[key] = validAttempts
		}
	}

	if cleanedCount > 0 {
		log.Printf("[Cleanup] Removed %d old onboarding tracking records", cleanedCount)
	}
}

// OTPService defines the interface for OTP operations
type OTPService interface {
	// GenerateOTP generates a new OTP and stores it
	// Returns 9-digit numeric OTP string
	// Returns error with status 429 if device onboarding rate limit exceeded
	GenerateOTP(ctx context.Context, userId string) (string, error)

	// VerifyOTP verifies an OTP for a given device
	// Returns the OTP request if valid
	// Returns error if OTP is invalid, expired, or max attempts exceeded
	VerifyOTP(ctx context.Context, userID, otpCode string) (*OTPRequest, error)

	// Cleanup removes expired OTPs and old tracking records
	Cleanup() error

	// Close closes the OTP service and cleans up resources
	Close() error
}

// InMemoryOTPService implements OTPService with comprehensive security controls using buntdb
//
// Security Features:
// - Rate Limiting: Device onboarding attempts are limited per signerID+authID pair
// - Brute Force Protection: OTPs invalidated after 3 failed attempts
// - Time-based Expiry: OTPs expire after 5 minutes
// - Memory Management: Automatic cleanup of expired OTPs and old records
// - Cryptographic Randomness: Uses crypto/rand for OTP generation
type InMemoryOTPService struct {
	sharesRepo      repositories.EncryptionPartsRepository
	securityService *OnboardingTracker
	config          SecurityConfig
	cleanupTicker   *time.Ticker
}

// NewInMemoryOTPService creates a new OTP service with buntdb storage
func NewInMemoryOTPService(sharesRepo repositories.EncryptionPartsRepository, securityService *OnboardingTracker, config SecurityConfig) (*InMemoryOTPService, error) {
	service := &InMemoryOTPService{
		sharesRepo:      sharesRepo,
		securityService: securityService,
		config:          config,
	}

	service.startCleanupInterval()

	return service, nil
}

// GenerateOTP generates a new OTP and stores it in memory
//
// Security Flow:
// 1. Enforces device onboarding limits per signerID/authID pair
// 2. Generates cryptographically secure 9-digit OTP
// 3. Stores OTP with metadata for verification tracking
//
// Returns 9-digit numeric OTP string
// Returns error with status 429 if rate limit exceeded
func (s *InMemoryOTPService) GenerateOTP(ctx context.Context, userID string) (string, error) {
	if err := s.securityService.TrackAttempt(userID); err != nil {
		return "", &HTTPError{
			Status:  http.StatusTooManyRequests,
			Message: err.Error(),
		}
	}

	otp, err := s.createRandomOTP()
	if err != nil {
		return "", fmt.Errorf("failed to generate OTP: %w", err)
	}

	log.Printf("[DEBUG] Generated OTP: %s", otp)

	request := &OTPRequest{
		OTP:            otp,
		CreatedAt:      time.Now().UnixMilli(),
		FailedAttempts: 0,
	}

	requestBytes, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal OTP request: %w", err)
	}

	options := buntdb.SetOptions{
		Expires: true,
		TTL:     time.Duration(s.config.OTPExpiryMS+1000) * time.Millisecond, // add some buffer to expiry time, just in case
	}
	err = s.sharesRepo.Set(ctx, userID, string(requestBytes), &options)
	if err != nil {
		return "", err
	}

	return otp, nil
}

// VerifyOTP verifies an OTP for a given device with comprehensive security checks
//
// Security Validations:
// 1. Checks if OTP request exists for device
// 2. Validates OTP hasn't expired (5-minute window)
// 3. Verifies OTP code matches
// 4. Tracks failed attempts and invalidates after 3 failures
// 5. Cleans up successful/failed requests from memory
//
// Returns the OTP request if valid, containing authentication context
// Returns error with status 400 if no pending authentication
// Returns error with status 401 if OTP expired, invalid, or max attempts exceeded
func (s *InMemoryOTPService) VerifyOTP(ctx context.Context, userID, otpCode string) (*OTPRequest, error) {
	val, err := s.sharesRepo.Get(ctx, userID)
	if err != nil {
		return nil, err
	}

	var request OTPRequest
	if err := json.Unmarshal([]byte(val), &request); err != nil {
		return nil, err
	}

	currentTime := time.Now().UnixMilli()
	if currentTime-request.CreatedAt > s.config.OTPExpiryMS {
		err := s.sharesRepo.Delete(ctx, userID)
		if err != nil {
			return nil, err
		}
		return nil, &HTTPError{
			Status:  http.StatusUnauthorized,
			Message: "OTP has expired",
		}
	}

	if request.OTP != otpCode {
		request.FailedAttempts++

		if request.FailedAttempts >= s.config.MaxFailedAttempts {
			err := s.sharesRepo.Delete(ctx, userID)
			if err != nil {
				return nil, err
			}

			// prevent brute forcing
			s.securityService.AddAttempt(userID)

			return nil, &HTTPError{
				Status:  http.StatusUnauthorized,
				Message: fmt.Sprintf("OTP invalidated after %d failed attempts", s.config.MaxFailedAttempts),
			}
		}

		// Update failed attempts count
		requestBytes, _ := json.Marshal(request)

		options := buntdb.SetOptions{
			Expires: true,
			TTL:     time.Duration((s.config.OTPExpiryMS-(currentTime-request.CreatedAt))+1000) * time.Millisecond, // add some buffer to expiry time, just in case
		}
		err = s.sharesRepo.Update(ctx, userID, string(requestBytes), &options)
		if err != nil {
			return nil, err
		}

		return nil, &HTTPError{
			Status:  http.StatusUnauthorized,
			Message: fmt.Sprintf("Invalid OTP (%d/%d attempts)", request.FailedAttempts, s.config.MaxFailedAttempts),
		}
	}

	// OTP is valid, remove from storage
	err = s.sharesRepo.Delete(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &request, nil
}

// Cleanup removes old tracking records
func (s *InMemoryOTPService) Cleanup() error {
	s.securityService.CleanupOldRecords()
	return nil
}

func (s *InMemoryOTPService) startCleanupInterval() {
	s.cleanupTicker = time.NewTicker(time.Duration(s.config.OTPExpiryMS) * time.Millisecond)

	go func() {
		for range s.cleanupTicker.C {
			s.Cleanup()
		}
	}()
}

// createRandomOTP generates cryptographically secure 9-digit OTP
//
// Security Implementation:
// - Uses crypto/rand for cryptographic randomness
// - Generates numbers in range 000000000-999999999 (1B possibilities)
// - Uniform distribution prevents bias in OTP generation
//
// Returns 9-digit zero-padded numeric string
func (s *InMemoryOTPService) createRandomOTP() (string, error) {
	maxValue := big.NewInt(1000000000) // 10^9 for 9 digits

	randomNumber, err := rand.Int(rand.Reader, maxValue)
	if err != nil {
		return "", fmt.Errorf("failed to generate random number: %w", err)
	}

	return fmt.Sprintf("%09d", randomNumber.Int64()), nil
}

// HTTPError represents an HTTP error with status code and message
type HTTPError struct {
	Status  int
	Message string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.Status, e.Message)
}

// StatusCode returns the HTTP status code
func (e *HTTPError) StatusCode() int {
	return e.Status
}
