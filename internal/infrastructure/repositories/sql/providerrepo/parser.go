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
		},
		mapProviderTypeToDomain: map[Type]provider.Type{
			TypeCustom:   provider.TypeCustom,
			TypeOpenfort: provider.TypeOpenfort,
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

func (p *parser) toDomainProvider(prov Provider) *provider.Provider {
	domainProv := &provider.Provider{
		ID:        prov.ID,
		ProjectID: prov.ProjectID,
		Type:      p.mapProviderTypeToDomain[prov.Type],
	}

	if prov.Custom != nil {
		domainProv.Config = p.toDomainCustomProvider(prov.Custom)
	}

	if prov.Openfort != nil {
		domainProv.Config = p.toDomainOpenfortProvider(prov.Openfort)
	}

	return domainProv
}

func (p *parser) toDatabaseOpenfortProvider(prov *provider.OpenfortConfig) *ProviderOpenfort {
	return &ProviderOpenfort{
		ProviderID:     prov.ProviderID,
		PublishableKey: prov.PublishableKey,
	}
}

func (p *parser) toDomainOpenfortProvider(prov *ProviderOpenfort) *provider.OpenfortConfig {
	return &provider.OpenfortConfig{
		ProviderID:     prov.ProviderID,
		PublishableKey: prov.PublishableKey,
	}
}

func (p *parser) toDatabaseCustomProvider(prov *provider.CustomConfig) *ProviderCustom {
	return &ProviderCustom{
		ProviderID: prov.ProviderID,
		JWKUrl:     prov.JWK,
	}
}

func (p *parser) toDomainCustomProvider(prov *ProviderCustom) *provider.CustomConfig {
	return &provider.CustomConfig{
		ProviderID: prov.ProviderID,
		JWK:        prov.JWKUrl,
	}
}
