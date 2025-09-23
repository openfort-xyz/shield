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
)
