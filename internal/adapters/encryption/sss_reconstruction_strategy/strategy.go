package sss_reconstruction_strategy

import (
	"errors"
	"go.openfort.xyz/shield/internal/core/ports/strategies"
	"go.openfort.xyz/shield/pkg/cypher"
)

type SSSReconstructionStrategy struct{}

func NewSSSReconstructionStrategy() strategies.ReconstructionStrategy {
	return &SSSReconstructionStrategy{}
}

func (s *SSSReconstructionStrategy) Split(data string) ([]string, error) {
	firstPart, secondPart, err := cypher.SplitEncryptionKey(data)
	if err != nil {
		return nil, err
	}

	return []string{firstPart, secondPart}, nil
}

func (s *SSSReconstructionStrategy) Reconstruct(parts []string) (string, error) {
	if len(parts) != 2 {
		return "", errors.New("invalid number of parts") //TODO extract error
	}

	return cypher.ReconstructEncryptionKey(parts[0], parts[1])
}
