package providerrepo

import "go.openfort.xyz/shield/internal/core/domain/provider"

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

func (p *parser) toDatabaseProvider(prov *provider.Provider) *Provider {
	return &Provider{
		ID:        prov.ID,
		ProjectID: prov.ProjectID,
		Type:      p.mapProviderTypeToDatabase[prov.Type],
	}
}

func (p *parser) toDomainProvider(prov *Provider) *provider.Provider {
	return &provider.Provider{
		ID:        prov.ID,
		ProjectID: prov.ProjectID,
		Type:      p.mapProviderTypeToDomain[prov.Type],
	}
}

func (p *parser) toDatabaseOpenfortProvider(prov *provider.Openfort) *ProviderOpenfort {
	return &ProviderOpenfort{
		ProviderID:      prov.ProviderID,
		OpenfortProject: prov.OpenfortProjectID,
	}
}

func (p *parser) toDomainOpenfortProvider(prov *ProviderOpenfort) *provider.Openfort {
	return &provider.Openfort{
		ProviderID:        prov.ProviderID,
		OpenfortProjectID: prov.OpenfortProject,
	}
}

func (p *parser) toDatabaseSupabaseProvider(prov *provider.Supabase) *ProviderSupabase {
	return &ProviderSupabase{
		ProviderID:      prov.ProviderID,
		SupabaseProject: prov.SupabaseProjectReference,
	}
}

func (p *parser) toDomainSupabaseProvider(prov *ProviderSupabase) *provider.Supabase {
	return &provider.Supabase{
		ProviderID:               prov.ProviderID,
		SupabaseProjectReference: prov.SupabaseProject,
	}
}

func (p *parser) toDatabaseCustomProvider(prov *provider.Custom) *ProviderCustom {
	return &ProviderCustom{
		ProviderID: prov.ProviderID,
		JWKUrl:     prov.JWK,
	}
}

func (p *parser) toDomainCustomProvider(prov *ProviderCustom) *provider.Custom {
	return &provider.Custom{
		ProviderID: prov.ProviderID,
		JWK:        prov.JWKUrl,
	}
}
