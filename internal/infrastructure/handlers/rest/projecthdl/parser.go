package projecthdl

import (
	"go.openfort.xyz/shield/internal/applications/projectapp"
	"go.openfort.xyz/shield/internal/core/domain/project"
	"go.openfort.xyz/shield/internal/core/domain/provider"
)

type parser struct{}

func newParser() *parser {
	return &parser{}
}

func (p *parser) toCreateProjectResponse(proj *project.Project) *CreateProjectResponse {
	return &CreateProjectResponse{
		ID:        proj.ID,
		Name:      proj.Name,
		APIKey:    proj.APIKey,
		APISecret: proj.APISecret,
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
		opts = append(opts, projectapp.WithCustom(req.Providers.Custom.JWK))
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
	case provider.TypeUnknown:
	}

	return resp
}
