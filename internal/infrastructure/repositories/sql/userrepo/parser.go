package userrepo

import (
	"go.openfort.xyz/shield/internal/core/domain/provider"
	"go.openfort.xyz/shield/internal/core/domain/user"
)

type parser struct {
	mapProviderTypeToDatabase map[provider.Type]Type
	mapProviderTypeToDomain   map[Type]provider.Type
}

func newParser() *parser {
	return &parser{
		mapProviderTypeToDatabase: map[provider.Type]Type{
			provider.TypeCustom:   TypeCustom,
			provider.TypeOpenfort: TypeOpenfort,
			provider.TypeSupabase: TypeSupabase,
		},
		mapProviderTypeToDomain: map[Type]provider.Type{
			TypeCustom:   provider.TypeCustom,
			TypeOpenfort: provider.TypeOpenfort,
			TypeSupabase: provider.TypeSupabase,
		},
	}
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
		Type:           p.mapProviderTypeToDomain[u.Type],
		ProjectID:      u.ProjectID,
	}
}

func (p *parser) toDatabaseExternalUser(u *user.ExternalUser) *ExternalUser {
	return &ExternalUser{
		ID:             u.ID,
		UserID:         u.UserID,
		ExternalUserID: u.ExternalUserID,
		Type:           p.mapProviderTypeToDatabase[u.Type],
		ProjectID:      u.ProjectID,
	}
}
