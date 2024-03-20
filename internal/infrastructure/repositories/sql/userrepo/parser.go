package userrepo

import (
	"go.openfort.xyz/shield/internal/core/domain/user"
)

type parser struct {
}

func newParser() *parser {
	return &parser{}
}

func (p *parser) toDomain(u *User) *user.User {
	return &user.User{
		ID:        u.ID,
		ProjectID: u.ProjectID,
	}
}

func (p *parser) toDatabase(u *user.User) *User {
	return &User{
		ID:        u.ID,
		ProjectID: u.ProjectID,
	}
}

func (p *parser) toDomainExternalUser(u *ExternalUser) *user.ExternalUser {
	return &user.ExternalUser{
		ID:             u.ID,
		UserID:         u.UserID,
		ExternalUserID: u.ExternalUserID,
		ProviderID:     u.ProviderID,
	}
}

func (p *parser) toDatabaseExternalUser(u *user.ExternalUser) *ExternalUser {
	return &ExternalUser{
		ID:             u.ID,
		UserID:         u.UserID,
		ExternalUserID: u.ExternalUserID,
		ProviderID:     u.ProviderID,
	}
}
