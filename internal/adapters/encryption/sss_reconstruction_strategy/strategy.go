package sss_reconstruction_strategy

import (
	"go.openfort.xyz/shield/internal/core/ports/strategies"
	"go.openfort.xyz/shield/pkg/cypher"
)

type SSSReconstructionStrategy struct{}

func NewSSSReconstructionStrategy() strategies.ReconstructionStrategy {
	return &SSSReconstructionStrategy{}
}

func (s *SSSReconstructionStrategy) Split(data string) (storedPart string, projectPart string, err error) {
	return cypher.SplitEncryptionKey(data)
}

func (s *SSSReconstructionStrategy) Reconstruct(storedPart string, projectPart string) (string, error) {
	return cypher.ReconstructEncryptionKey(storedPart, projectPart)
}
