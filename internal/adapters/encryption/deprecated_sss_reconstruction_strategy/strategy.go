package depsssrec

import (
	"go.openfort.xyz/shield/internal/core/domain/errors"
	"go.openfort.xyz/shield/internal/core/ports/strategies"
	"go.openfort.xyz/shield/pkg/cypher"
)

const (
	MaxReties = 5
)

type SSSReconstructionStrategy struct{}

func NewSSSReconstructionStrategy() strategies.ReconstructionStrategy {
	return &SSSReconstructionStrategy{}
}

func (s *SSSReconstructionStrategy) Split(data string) (storedPart string, projectPart string, err error) {
	for i := 0; i < MaxReties; i++ {
		storedPart, projectPart, err = cypher.SplitEncryptionKey(data)
		if err != nil {
			continue
		}

		err = s.validateSplit(data, storedPart, projectPart)
		if err == nil {
			return
		}
	}

	return
}

func (s *SSSReconstructionStrategy) Reconstruct(storedPart string, projectPart string) (string, error) {
	return cypher.ReconstructEncryptionKey(storedPart, projectPart)
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
