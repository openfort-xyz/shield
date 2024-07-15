package sssrec

import (
	"encoding/base64"

	sss "go.openfort.xyz/shamir-secret-sharing-go"
	"go.openfort.xyz/shield/internal/core/domain/errors"
	"go.openfort.xyz/shield/internal/core/ports/strategies"
)

const (
	MaxReties = 5
)

type SSSReconstructionStrategy struct{}

func NewSSSReconstructionStrategy() strategies.ReconstructionStrategy {
	return &SSSReconstructionStrategy{}
}

func (s *SSSReconstructionStrategy) Split(data string) (string, string, error) {
	for i := 0; i < MaxReties; i++ {
		rawKey, err := base64.StdEncoding.DecodeString(data)
		if err != nil {
			return "", "", err
		}

		parts, err := sss.Split(2, 2, rawKey)
		if err != nil {
			return "", "", err
		}

		if len(parts) != 2 {
			return "", "", errors.ErrFailedToSplitKey
		}

		storedPart := base64.StdEncoding.EncodeToString(parts[0])
		projectPart := base64.StdEncoding.EncodeToString(parts[1])

		err = s.validateSplit(data, storedPart, projectPart)
		if err == nil {
			return storedPart, projectPart, nil
		}
	}

	return "", "", errors.ErrFailedToSplitKey
}

func (s *SSSReconstructionStrategy) Reconstruct(storedPart string, projectPart string) (string, error) {
	rawStoredPart, err := base64.StdEncoding.DecodeString(storedPart)
	if err != nil {
		return "", err
	}

	// Backward compatibility with old keys
	if len(rawStoredPart) == 32 {
		rawStoredPart = append([]byte{1}, rawStoredPart...)
	}

	rawProjectPart, err := base64.StdEncoding.DecodeString(projectPart)
	if err != nil {
		return "", err
	}
	// Backward compatibility with old keys
	if len(rawProjectPart) == 32 {
		rawProjectPart = append([]byte{2}, rawProjectPart...)
	}

	combined, err := sss.Combine([][]byte{rawStoredPart, rawProjectPart})
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(combined), nil
}

func (s *SSSReconstructionStrategy) validateSplit(data string, storedPart string, projectPart string) error {
	reconstructed, err := s.Reconstruct(storedPart, projectPart)
	if err != nil {
		return err
	}

	if data != reconstructed {
		return errors.ErrReconstructedKeyMismatch
	}

	return nil
}
