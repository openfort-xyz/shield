package errors

import "errors"

var (
	ErrProjectNotFound                 = errors.New("project not found")
	ErrEncryptionPartNotFound          = errors.New("encryption part not found")
	ErrKeyInDBAlreadyExists            = errors.New("key in DB already exist")
	ErrDataInDBNotFound                = errors.New("data in DB not found")
	ErrEncryptionPartAlreadyExists     = errors.New("encryption part already exists")
	ErrEncryptionPartRequired          = errors.New("encryption part is required")
	ErrInvalidEncryptionPart           = errors.New("invalid encryption part")
	ErrInvalidEncryptionSession        = errors.New("invalid encryption session")
	ErrInvalidEncryptionKeyBuilderType = errors.New("invalid encryption key builder type")
	ErrReconstructedKeyMismatch        = errors.New("reconstructed key mismatch")
	ErrProjectPartRequired             = errors.New("project part is required")
	ErrDatabasePartRequired            = errors.New("database part is required")
	ErrFailedToSplitKey                = errors.New("failed to split key")
	ErrOTPVerificationRequired         = errors.New("otp verification required")
	ErrEncryptionPartChanged           = errors.New("encryption part changed during request")
	ErrPendingAPIKeyNotProven          = errors.New("commit must be authenticated with the pending API key")
	ErrAPIKeyRotationExpired           = errors.New("pending API key rotation expired")
	ErrNoPendingAPIKeyRotation         = errors.New("no pending API key rotation")
	ErrAPIKeyRevoked                   = errors.New("API key was rotated")
	ErrPendingAPISecretNotProven       = errors.New("commit must be authenticated with the pending API secret")
	ErrAPISecretRotationExpired        = errors.New("pending API secret rotation expired")
	ErrNoPendingAPISecretRotation      = errors.New("no pending API secret rotation")
)
