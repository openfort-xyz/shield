package sharehdl

import "go.openfort.xyz/shield/internal/core/domain/share"

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
	shr := &share.Share{
		Secret: s.Secret,
		EncryptionParameters: &share.EncryptionParameters{
			Entropy: p.mapEntropyDomain[s.Entropy],
		},
	}

	if s.Salt != "" {
		shr.EncryptionParameters.Salt = s.Salt
	}
	if s.Iterations != 0 {
		shr.EncryptionParameters.Iterations = s.Iterations
	}
	if s.Length != 0 {
		shr.EncryptionParameters.Length = s.Length
	}
	if s.Digest != "" {
		shr.EncryptionParameters.Digest = s.Digest
	}

	return shr
}

func (p *parser) fromDomain(s *share.Share) *Share {
	shr := &Share{
		Secret:  s.Secret,
		Entropy: p.mapDomainEntropy[s.EncryptionParameters.Entropy],
	}

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

	return shr
}
