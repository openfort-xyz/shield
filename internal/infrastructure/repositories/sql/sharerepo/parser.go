package sharerepo

import (
	"go.openfort.xyz/shield/internal/core/domain/share"
)

type parser struct {
}

func newParser() *parser {
	return &parser{}
}

func (p *parser) toDomain(s *Share) *share.Share {
	return &share.Share{
		ID:          s.ID,
		Data:        s.Data,
		UserID:      s.UserID,
		UserEntropy: s.UserEntropy,
		Salt:        s.Salt,
		Iterations:  s.Iterations,
		Length:      s.Length,
		Digest:      s.Digest,
	}
}

func (p *parser) toDatabase(s *share.Share) *Share {
	return &Share{
		ID:          s.ID,
		Data:        s.Data,
		UserID:      s.UserID,
		UserEntropy: s.UserEntropy,
		Salt:        s.Salt,
		Iterations:  s.Iterations,
		Length:      s.Length,
		Digest:      s.Digest,
	}
}
