package projectapp

import (
	"context"
	"go.openfort.xyz/shield/internal/core/domain"
	"go.openfort.xyz/shield/internal/core/domain/project"
	"go.openfort.xyz/shield/internal/core/domain/provider"
	"go.openfort.xyz/shield/internal/core/ports/services"
	"go.openfort.xyz/shield/pkg/ofcontext"
	"go.openfort.xyz/shield/pkg/oflog"
	"log/slog"
	"os"
)

type ProjectApplication struct {
	projectSvc  services.ProjectService
	providerSvc services.ProviderService
	logger      *slog.Logger
}

func New(projectSvc services.ProjectService, providerSvc services.ProviderService) *ProjectApplication {
	return &ProjectApplication{
		projectSvc:  projectSvc,
		providerSvc: providerSvc,
		logger:      slog.New(oflog.NewContextHandler(slog.NewTextHandler(os.Stdout, nil))).WithGroup("project_application"),
	}
}

func (a *ProjectApplication) CreateProject(ctx context.Context, name string) (*project.Project, error) {
	a.logger.InfoContext(ctx, "creating project")

	proj, err := a.projectSvc.Create(ctx, name)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to create project", slog.String("error", err.Error()))
		return nil, err
	}

	return proj, nil
}

func (a *ProjectApplication) GetProject(ctx context.Context) (*project.Project, error) {
	a.logger.InfoContext(ctx, "getting project")

	projectID := ofcontext.GetProjectID(ctx)

	proj, err := a.projectSvc.Get(ctx, projectID)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to get project", slog.String("error", err.Error()))
		return nil, err
	}

	return proj, nil
}

func (a *ProjectApplication) AddProviders(ctx context.Context, opts ...ProviderOption) ([]*provider.Provider, error) {
	a.logger.InfoContext(ctx, "adding providers")

	projectID := ofcontext.GetProjectID(ctx)

	cfg := &providerConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	var providers []*provider.Provider
	if cfg.jwkUrl != nil {
		prov, err := a.providerSvc.Configure(ctx, projectID, &services.CustomProviderConfig{JWKUrl: *cfg.jwkUrl})
		if err != nil {
			a.logger.ErrorContext(ctx, "failed to configure custom provider", slog.String("error", err.Error()))
			return nil, err
		}

		providers = append(providers, prov)
	}

	if cfg.openfortProject != nil {
		prov, err := a.providerSvc.Configure(ctx, projectID, &services.OpenfortProviderConfig{OpenfortProject: *cfg.openfortProject})
		if err != nil {
			a.logger.ErrorContext(ctx, "failed to configure openfort provider", slog.String("error", err.Error()))
			return nil, err
		}

		providers = append(providers, prov)
	}

	if cfg.supabaseProject != nil {
		prov, err := a.providerSvc.Configure(ctx, projectID, &services.SupabaseProviderConfig{SupabaseProject: *cfg.supabaseProject})
		if err != nil {
			a.logger.ErrorContext(ctx, "failed to configure supabase provider", slog.String("error", err.Error()))
			return nil, err
		}

		providers = append(providers, prov)
	}

	if len(providers) == 0 {
		return nil, ErrNoProviderSpecified
	}

	return providers, nil
}

func (a *ProjectApplication) GetProviders(ctx context.Context) ([]*provider.Provider, error) {
	a.logger.InfoContext(ctx, "listing providers")

	projectID := ofcontext.GetProjectID(ctx)

	providers, err := a.providerSvc.List(ctx, projectID)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to list providers", slog.String("error", err.Error()))
		return nil, err
	}

	return providers, nil
}

func (a *ProjectApplication) GetProviderDetail(ctx context.Context, providerID string) (*provider.Provider, error) { // TODO return provider detail (custom, openfort, supabase)
	a.logger.InfoContext(ctx, "getting provider detail")

	projectID := ofcontext.GetProjectID(ctx)

	prov, err := a.providerSvc.Get(ctx, providerID)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to get provider", slog.String("error", err.Error()))
		return nil, err
	}

	if prov.ProjectID != projectID {
		a.logger.ErrorContext(ctx, "unauthorized access, trying to access provider from different project", slog.String("project_id", projectID), slog.String("provider_project_id", prov.ProjectID))
		return nil, domain.ErrProviderNotFound
	}

	return prov, nil
}

func (a *ProjectApplication) UpdateProvider(ctx context.Context) error {
	// TODO implement
	return nil
}

func (a *ProjectApplication) RemoveProvider(ctx context.Context, providerID string) error { // TODO delete external users
	a.logger.InfoContext(ctx, "removing provider")

	projectID := ofcontext.GetProjectID(ctx)

	err := a.providerSvc.Remove(ctx, projectID, providerID)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to remove provider", slog.String("error", err.Error()))
		return err
	}

	return nil
}
