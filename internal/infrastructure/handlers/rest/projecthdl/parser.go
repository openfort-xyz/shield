package projecthdl

import (
	"go.openfort.xyz/shield/internal/applications/projectapp"
	"go.openfort.xyz/shield/internal/core/domain/project"
	"go.openfort.xyz/shield/internal/core/domain/provider"
)

type parser struct {
	mapKeyTypeToDomain   map[KeyType]provider.KeyType
	mapKeyTypeToResponse map[provider.KeyType]KeyType
}

func newParser() *parser {
	return &parser{
		mapKeyTypeToDomain: map[KeyType]provider.KeyType{
			KeyTypeRSA:     provider.KeyTypeRSA,
			KeyTypeECDSA:   provider.KeyTypeECDSA,
			KeyTypeEd25519: provider.KeyTypeEd25519,
		},
		mapKeyTypeToResponse: map[provider.KeyType]KeyType{
			provider.KeyTypeRSA:     KeyTypeRSA,
			provider.KeyTypeECDSA:   KeyTypeECDSA,
			provider.KeyTypeEd25519: KeyTypeEd25519,
		},
	}
}

func (p *parser) toCreateProjectResponse(proj *project.Project) *CreateProjectResponse {
	return &CreateProjectResponse{
		ID:             proj.ID,
		Name:           proj.Name,
		APIKey:         proj.APIKey,
		APISecret:      proj.APISecret,
		EncryptionPart: proj.EncryptionPart,
	}
}

func (p *parser) toGetProjectResponse(proj *project.Project) *GetProjectResponse {
	return &GetProjectResponse{
		ID:   proj.ID,
		Name: proj.Name,
	}
}

func (p *parser) fromAddProvidersRequest(req *AddProvidersRequest) []projectapp.ProviderOption {
	opts := make([]projectapp.ProviderOption, 0)

	if req.Providers.Openfort != nil && req.Providers.Openfort.PublishableKey != "" {
		opts = append(opts, projectapp.WithOpenfort(req.Providers.Openfort.PublishableKey))
	}

	if req.Providers.Custom != nil && req.Providers.Custom.JWK != "" {
		opts = append(opts, projectapp.WithCustomJWK(req.Providers.Custom.JWK))
	}

	if req.Providers.Custom != nil && req.Providers.Custom.PEM != "" {
		opts = append(opts, projectapp.WithCustomPEM(req.Providers.Custom.PEM, p.mapKeyTypeToDomain[req.Providers.Custom.KeyType]))
	}

	return opts
}

func (p *parser) toAddProvidersResponse(providers []*provider.Provider) *AddProvidersResponse {
	resp := &AddProvidersResponse{
		Providers: make([]*ProviderResponse, 0, len(providers)),
	}

	for _, prov := range providers {
		resp.Providers = append(resp.Providers, &ProviderResponse{
			ProviderID: prov.ID,
			Type:       prov.Type.String(),
		})
	}

	return resp
}

func (p *parser) toGetProvidersResponse(providers []*provider.Provider) *GetProvidersResponse {
	resp := &GetProvidersResponse{
		Providers: make([]*ProviderResponse, 0, len(providers)),
	}

	for _, prov := range providers {
		resp.Providers = append(resp.Providers, &ProviderResponse{
			ProviderID: prov.ID,
			Type:       prov.Type.String(),
		})
	}

	return resp
}

func (p *parser) toGetProviderResponse(prov *provider.Provider) *GetProviderResponse {
	resp := &GetProviderResponse{
		ProviderID: prov.ID,
		Type:       prov.Type.String(),
	}

	switch prov.Type {
	case provider.TypeOpenfort:
		resp.PublishableKey = prov.Config.(*provider.OpenfortConfig).PublishableKey
	case provider.TypeCustom:
		resp.JWK = prov.Config.(*provider.CustomConfig).JWK
		resp.PEM = prov.Config.(*provider.CustomConfig).PEM
		resp.KeyType = p.mapKeyTypeToResponse[prov.Config.(*provider.CustomConfig).KeyType]
	case provider.TypeUnknown:
	}

	return resp
}
