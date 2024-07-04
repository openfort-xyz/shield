package sharerepo

import (
	"go.openfort.xyz/shield/internal/core/domain/share"
)

type parser struct {
	mapEntropyDomain map[Entropy]share.Entropy
	mapDomainEntropy map[share.Entropy]Entropy
}

func newParser() *parser {
	return &parser{
		mapEntropyDomain: map[Entropy]share.Entropy{
			EntropyNone:    share.EntropyNone,
			EntropyUser:    share.EntropyUser,
			EntropyProject: share.EntropyProject,
		},
		mapDomainEntropy: map[share.Entropy]Entropy{
			share.EntropyNone:    EntropyNone,
			share.EntropyUser:    EntropyUser,
			share.EntropyProject: EntropyProject,
		},
	}
}

func (p *parser) toDomain(s *Share) *share.Share {
	encryptionParameters := &share.EncryptionParameters{
		Entropy: p.mapEntropyDomain[s.Entropy],
	}

	if s.Salt != "" {
		encryptionParameters.Salt = s.Salt
	}
	if s.Iterations != 0 {
		encryptionParameters.Iterations = s.Iterations
	}
	if s.Length != 0 {
		encryptionParameters.Length = s.Length
	}
	if s.Digest != "" {
		encryptionParameters.Digest = s.Digest
	}

	return &share.Share{
		ID:                   s.ID,
		Secret:               s.Data,
		UserID:               s.UserID,
		EncryptionParameters: encryptionParameters,
	}
}

func (p *parser) toDatabase(s *share.Share) *Share {
	shr := &Share{
		ID:     s.ID,
		Data:   s.Secret,
		UserID: s.UserID,
	}

	if s.EncryptionParameters != nil {
		shr.Entropy = p.mapDomainEntropy[s.EncryptionParameters.Entropy]
		if s.EncryptionParameters.Salt != "" {
			shr.Salt = s.EncryptionParameters.Salt
		}
		if s.EncryptionParameters.Iterations != 0 {
			shr.Iterations = s.EncryptionParameters.Iterations
		}
		if s.EncryptionParameters.Length != 0 {
			shr.Length = s.EncryptionParameters.Length
		}
		if s.EncryptionParameters.Digest != "" {
			shr.Digest = s.EncryptionParameters.Digest
		}
	} else {
		shr.Entropy = EntropyNone
	}

	return shr
}
