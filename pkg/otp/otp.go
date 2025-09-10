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

	"go.openfort.xyz/shield/internal/core/ports/repositories"
)

const OTPDigits = 9

// SecurityConfig holds all security-related configuration for OTP service
//
// OTP Brute Force Protection:
//   - OTP space: 1,000,000,000 possible combinations (9-digit numeric)
//   - Attempts per day: With a window of 6 hours, 4 windows per day, 3 onboard attempts per window,
//     and 3 OTP attempts per onboarding, an attacker gets 12 OTP generations per day.
//   - Expected brute force time: Using cumulative probability model 0.5 = 1 - (1 - 3/1,000,000,000)^n,
//     it takes 231,049,000 OTP generations for 50% success probability.
//   - Time to brute force: 231,049,000 � 12 OTP generations/day = ~19,254,000 days H 53,000 years.
type SecurityConfig struct {
	MaxFailedAttempts        int
	DeviceOnboardingWindowMS int64
	MaxDeviceOnboardAttempts int
	OTPExpiryMS              int64
	OTPCleanupGracePeriodMS  int64
}

var DefaultSecurityConfig = SecurityConfig{
	MaxFailedAttempts:        3,
	DeviceOnboardingWindowMS: 6 * 60 * 60 * 1000, // 6 hours
	MaxDeviceOnboardAttempts: 3,
	OTPExpiryMS:              5 * 60 * 1000,  // 5 minutes
	OTPCleanupGracePeriodMS:  60 * 60 * 1000, // 1 hour
}

// TODO: move to types somewhere
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
func (ot *OnboardingTracker) TrackAttempt(signerID string) error {
	ot.mu.Lock()
	defer ot.mu.Unlock()

	now := time.Now().UnixMilli()

	// Clean up old attempts outside the window
	var validAttempts []int64
	for _, timestamp := range ot.attempts[signerID] {
		if now-timestamp <= ot.windowMS {
			validAttempts = append(validAttempts, timestamp)
		}
	}

	// Check if we're at the rate limit
	if len(validAttempts) >= ot.maxAttempts {
		return fmt.Errorf("rate limit exceeded for signer %s", signerID)
	}

	// Add current attempt
	validAttempts = append(validAttempts, now)
	ot.attempts[signerID] = validAttempts

	return nil
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
	GenerateOTP(signerID, authID, deviceID string) (string, error)

	// VerifyOTP verifies an OTP for a given device
	// Returns the OTP request if valid
	// Returns error if OTP is invalid, expired, or max attempts exceeded
	VerifyOTP(deviceID, otpCode string) (*OTPRequest, error)

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
	stopCleanup     chan struct{}
	mu              sync.Mutex
}

var (
	instance *InMemoryOTPService
	once     sync.Once
)

// GetInstance returns singleton instance of InMemoryOTPService with default security configuration
// func GetInstance() (*InMemoryOTPService, error) {
// 	var err error
// 	once.Do(func() {
// 		security := NewOnboardingTracker(
// 			DefaultSecurityConfig.DeviceOnboardingWindowMS,
// 			DefaultSecurityConfig.MaxDeviceOnboardAttempts,
// 		)
// 		instance, err = NewInMemoryOTPService(security, DefaultSecurityConfig)
// 	})
// 	return instance, err
// }

// NewInMemoryOTPService creates a new OTP service with buntdb storage
func NewInMemoryOTPService(sharesRepo repositories.EncryptionPartsRepository, securityService *OnboardingTracker, config SecurityConfig) (*InMemoryOTPService, error) {
	service := &InMemoryOTPService{
		sharesRepo:      sharesRepo,
		securityService: securityService,
		config:          config,
		stopCleanup:     make(chan struct{}),
	}

	// service.startCleanupInterval()
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
func (s *InMemoryOTPService) GenerateOTP(ctx context.Context, signerID string) (string, error) {
	if err := s.securityService.TrackAttempt(signerID); err != nil {
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

	// TODO: add this stuff there
	// &buntdb.SetOptions{
	// 		Expires: true,
	// 		TTL:     time.Duration(s.config.OTPExpiryMS) * time.Millisecond,
	// 	}
	err = s.sharesRepo.Set(ctx, signerID, string(requestBytes))
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
func (s *InMemoryOTPService) VerifyOTP(ctx context.Context, signerID, otpCode string) (*OTPRequest, error) {
	var request *OTPRequest

	val, err := s.sharesRepo.Get(ctx, signerID)
	if err != nil {
		return nil, err
	}

	var req OTPRequest
	if err := json.Unmarshal([]byte(val), &req); err != nil {
		return nil, err
	}

	if err != nil {
		return nil, fmt.Errorf("failed to deserialize OTP request: %w", err)
	}

	currentTime := time.Now().UnixMilli()
	if currentTime-request.CreatedAt > s.config.OTPExpiryMS {
		err := s.sharesRepo.Delete(ctx, signerID)
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
			err := s.sharesRepo.Delete(ctx, signerID)
			if err != nil {
				return nil, err
			}
			return nil, &HTTPError{
				Status:  http.StatusUnauthorized,
				Message: fmt.Sprintf("OTP invalidated after %d failed attempts", s.config.MaxFailedAttempts),
			}
		}

		// Update failed attempts count
		requestBytes, _ := json.Marshal(request)
		// TODO: add TTL
		// &buntdb.SetOptions{
		// 		Expires: true,
		// 		TTL:     time.Duration(s.config.OTPExpiryMS-(currentTime-request.CreatedAt)) * time.Millisecond,
		// 	}
		err = s.sharesRepo.Update(ctx, signerID, string(requestBytes))
		if err != nil {
			return nil, err
		}

		return nil, &HTTPError{
			Status:  http.StatusUnauthorized,
			Message: fmt.Sprintf("Invalid OTP (%d/%d attempts)", request.FailedAttempts, s.config.MaxFailedAttempts),
		}
	}

	// OTP is valid, remove from storage
	err = s.sharesRepo.Delete(ctx, signerID)
	if err != nil {
		return nil, err
	}

	return request, nil
}

// Cleanup removes expired OTPs and old tracking records
// func (s *InMemoryOTPService) Cleanup() error {
// 	s.cleanupExpiredOTPs()
// 	s.securityService.CleanupOldRecords()
// 	return nil
// }

// Close closes the OTP service and cleans up resources
// func (s *InMemoryOTPService) Close() error {
// 	s.mu.Lock()
// 	defer s.mu.Unlock()

// 	if s.cleanupTicker != nil {
// 		s.cleanupTicker.Stop()
// 		close(s.stopCleanup)
// 		s.cleanupTicker = nil
// 	}

// 	return s.db.Close()
// }

// func (s *InMemoryOTPService) startCleanupInterval() {
// 	s.cleanupTicker = time.NewTicker(time.Duration(s.config.OTPExpiryMS) * time.Millisecond)

// 	go func() {
// 		for {
// 			select {
// 			case <-s.cleanupTicker.C:
// 				s.Cleanup()
// 			case <-s.stopCleanup:
// 				return
// 			}
// 		}
// 	}()
// }

// func (s *InMemoryOTPService) cleanupExpiredOTPs() {
// 	currentTime := time.Now().UnixMilli()
// 	expiredCount := 0
// 	extendedExpiryTime := s.config.OTPExpiryMS + s.config.OTPCleanupGracePeriodMS

// 	s.db.Update(func(tx *buntdb.Tx) error {
// 		var toDelete []string

// 		tx.Ascend("", func(key, value string) bool {
// 			var request OTPRequest
// 			if err := json.Unmarshal([]byte(value), &request); err != nil {
// 				// If we can't unmarshal, delete it
// 				toDelete = append(toDelete, key)
// 				expiredCount++
// 				return true
// 			}

// 			if currentTime-request.CreatedAt > extendedExpiryTime {
// 				toDelete = append(toDelete, key)
// 				expiredCount++
// 			}
// 			return true
// 		})

// 		for _, key := range toDelete {
// 			tx.Delete(key)
// 		}

// 		return nil
// 	})

// 	if expiredCount > 0 {
// 		log.Printf("[Cleanup] Removed %d expired OTPs from memory", expiredCount)
// 	}
// }

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
