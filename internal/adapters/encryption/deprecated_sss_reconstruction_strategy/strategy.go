package depsssrec

import (
	"github.com/openfort-xyz/shield/internal/core/domain/errors"
	"github.com/openfort-xyz/shield/internal/core/domain/project"
	"github.com/openfort-xyz/shield/internal/core/ports/strategies"
	"github.com/openfort-xyz/shield/pkg/cypher"
)

const (
	MaxReties = 5
)

type SSSReconstructionStrategy struct{}

func NewSSSReconstructionStrategy() strategies.ReconstructionStrategy {
	return &SSSReconstructionStrategy{}
}

func (s *SSSReconstructionStrategy) Split(data string) (storedPart project.StoredPart, projectPart project.ProjectPart, err error) {
	for i := 0; i < MaxReties; i++ {
		var stored, proj string
		stored, proj, err = cypher.SplitEncryptionKey(data)
		if err != nil {
			continue
		}
		storedPart, projectPart = project.StoredPart(stored), project.ProjectPart(proj)

		err = s.validateSplit(data, storedPart, projectPart)
		if err == nil {
			return
		}
	}

	return
}

func (s *SSSReconstructionStrategy) Reconstruct(storedPart project.StoredPart, projectPart project.ProjectPart) (string, error) {
	return cypher.ReconstructEncryptionKey(string(storedPart), string(projectPart))
}

func (s *SSSReconstructionStrategy) validateSplit(data string, storedPart project.StoredPart, projectPart project.ProjectPart) error {
	reconstructed, err := s.Reconstruct(storedPart, projectPart)
	if err != nil {
		return err
	}

	if data != reconstructed {
		return errors.ErrReconstructedKeyMismatch
	}

	return nil
}
