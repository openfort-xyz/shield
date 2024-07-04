package errors

import "errors"

var (
	ErrProjectNotFound             = errors.New("project not found")
	ErrEncryptionPartNotFound      = errors.New("encryption part not found")
	ErrEncryptionPartAlreadyExists = errors.New("encryption part already exists")
	ErrEncryptionPartRequired      = errors.New("encryption part is required")
)
