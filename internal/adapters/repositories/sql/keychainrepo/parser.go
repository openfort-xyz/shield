package keychainrepo

import (
	"go.openfort.xyz/shield/internal/core/domain/keychain"
)

type parser struct {
}

func newParser() *parser {
	return &parser{}
}

func (p *parser) toDomain(k *Keychain) *keychain.Keychain {
	return &keychain.Keychain{
		ID:     k.ID,
		UserID: k.UserID,
	}
}

func (p *parser) toDatabase(k *keychain.Keychain) *Keychain {
	return &Keychain{
		ID:     k.ID,
		UserID: k.UserID,
	}
}
