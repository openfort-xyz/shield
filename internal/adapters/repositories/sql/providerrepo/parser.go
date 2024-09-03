package providerrepo

import "go.openfort.xyz/shield/internal/core/domain/provider"

type parser struct {
	mapProviderTypeToDatabase map[provider.Type]Type
	mapProviderTypeToDomain   map[Type]provider.Type
	mapKeyTypeToDomain        map[KeyType]provider.KeyType
	mapKeyTypeToDatabase      map[provider.KeyType]KeyType
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
		mapKeyTypeToDomain: map[KeyType]provider.KeyType{
			KeyTypeRSA: provider.KeyTypeRSA,
			KeyTypeEC:  provider.KeyTypeECDSA,
			KeyTypeEd:  provider.KeyTypeEd25519,
		},
		mapKeyTypeToDatabase: map[provider.KeyType]KeyType{
			provider.KeyTypeRSA:     KeyTypeRSA,
			provider.KeyTypeECDSA:   KeyTypeEC,
			provider.KeyTypeEd25519: KeyTypeEd,
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
	var jwkUrl *string
	if prov.JWK != "" {
		jwkUrl = &prov.JWK
	}

	var pem *string
	if prov.PEM != "" {
		pem = &prov.PEM
	}

	var keyType *KeyType
	if keyTypeStr := p.mapKeyTypeToDatabase[prov.KeyType]; keyTypeStr != "" {
		keyType = &keyTypeStr
	}
	return &ProviderCustom{
		ProviderID: prov.ProviderID,
		JWKUrl:     jwkUrl,
		PEM:        pem,
		KeyType:    keyType,
	}
}

func (p *parser) toUpdateCustomProviderMap(prov *provider.CustomConfig) map[string]interface{} {
	updates := make(map[string]interface{})
	if prov.JWK != "" {
		updates["jwk_url"] = prov.JWK
		updates["pem_cert"] = nil
		updates["key_type"] = nil
	} else if prov.PEM != "" {
		updates["pem_cert"] = prov.PEM
		updates["jwk_url"] = nil
		if keyType := p.mapKeyTypeToDatabase[prov.KeyType]; keyType != "" {
			updates["key_type"] = keyType
		}
	}
	return updates
}

func (p *parser) toDomainCustomProvider(prov *ProviderCustom) *provider.CustomConfig {
	jwk := ""
	if prov.JWKUrl != nil {
		jwk = *prov.JWKUrl
	}

	pem := ""
	if prov.PEM != nil {
		pem = *prov.PEM
	}

	keyType := provider.KeyTypeUnknown
	if prov.KeyType != nil {
		keyType = p.mapKeyTypeToDomain[*prov.KeyType]
	}
	return &provider.CustomConfig{
		ProviderID: prov.ProviderID,
		JWK:        jwk,
		PEM:        pem,
		KeyType:    keyType,
	}
}
