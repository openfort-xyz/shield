package providersvc

import (
	"context"
	"errors"
	"go.openfort.xyz/shield/internal/core/domain/provider"
	"go.openfort.xyz/shield/internal/core/ports/repositories"
	"go.openfort.xyz/shield/internal/core/ports/services"
	"go.openfort.xyz/shield/pkg/oflog"
	"log/slog"
	"os"
)

type service struct {
	repo   repositories.ProviderRepository
	logger *slog.Logger
}

var _ services.ProviderService = (*service)(nil)

func New(repo repositories.ProviderRepository) services.ProviderService {
	return &service{
		repo:   repo,
		logger: slog.New(oflog.NewContextHandler(slog.NewTextHandler(os.Stdout, nil))).WithGroup("provider_service"),
	}
}

func (s *service) Configure(ctx context.Context, projectID string, config services.ProviderConfig) (*provider.Provider, error) {
	if config == nil {
		return nil, ErrNoProviderConfig
	}

	switch config.GetType() {
	case provider.TypeCustom:
		customConfig, ok := config.GetConfig().(*services.CustomProviderConfig)
		if !ok {
			return nil, ErrInvalidProviderConfig
		}

		return s.configureCustomProvider(ctx, projectID, customConfig.JWKUrl)
	case provider.TypeOpenfort:
		openfortConfig, ok := config.GetConfig().(*services.OpenfortProviderConfig)
		if !ok {
			return nil, ErrInvalidProviderConfig
		}

		return s.configureOpenfortProvider(ctx, projectID, openfortConfig.OpenfortProject)
	case provider.TypeSupabase:
		supabaseConfig, ok := config.GetConfig().(*services.SupabaseProviderConfig)
		if !ok {
			return nil, ErrInvalidProviderConfig
		}

		return s.configureSupabaseAuthentication(ctx, projectID, supabaseConfig.SupabaseProject)
	default:
		return nil, ErrUnknownProviderType
	}
}

func (s *service) configureCustomProvider(ctx context.Context, projectID, jwkUrl string) (*provider.Provider, error) {
	s.logger.InfoContext(ctx, "configuring custom provider", slog.String("project_id", projectID))

	prov, err := s.repo.GetByProjectAndType(ctx, projectID, provider.TypeCustom)
	if err != nil && !errors.Is(err, repositories.ErrProviderNotFound) {
		s.logger.ErrorContext(ctx, "failed to get provider", slog.String("error", err.Error()))
		return nil, err
	}

	if prov != nil {
		s.logger.ErrorContext(ctx, "provider already exists", slog.String("error", err.Error()))
		return nil, ErrProviderAlreadyExists
	}

	prov = &provider.Provider{
		ProjectID: projectID,
		Type:      provider.TypeCustom,
	}
	err = s.repo.Create(ctx, prov)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create provider", slog.String("error", err.Error()))
		return nil, err
	}

	customAuth := &provider.Custom{
		ProviderID: prov.ID,
		JWK:        jwkUrl,
	}
	err = s.repo.CreateCustom(ctx, customAuth)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create custom provider", slog.String("error", err.Error()))
		errD := s.repo.Delete(ctx, prov.ID)
		if errD != nil {
			s.logger.ErrorContext(ctx, "failed to delete provider", slog.String("provider", prov.ID), slog.String("error", errD.Error()))
			errors.Join(err, errD)
		}
		return nil, err
	}

	return prov, nil
}

func (s *service) configureOpenfortProvider(ctx context.Context, projectID, openfortProject string) (*provider.Provider, error) {
	s.logger.InfoContext(ctx, "configuring openfort provider", slog.String("project_id", projectID))

	prov, err := s.repo.GetByProjectAndType(ctx, projectID, provider.TypeOpenfort)
	if err != nil && !errors.Is(err, repositories.ErrProviderNotFound) {
		s.logger.ErrorContext(ctx, "failed to get provider", slog.String("error", err.Error()))
		return nil, err
	}

	if prov != nil {
		s.logger.ErrorContext(ctx, "provider already exists", slog.String("error", err.Error()))
		return nil, ErrProviderAlreadyExists
	}

	prov = &provider.Provider{
		ProjectID: projectID,
		Type:      provider.TypeOpenfort,
	}
	err = s.repo.Create(ctx, prov)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create provider", slog.String("error", err.Error()))
		return nil, err
	}

	openfortAuth := &provider.Openfort{
		ProviderID:        prov.ID,
		OpenfortProjectID: openfortProject,
	}
	err = s.repo.CreateOpenfort(ctx, openfortAuth)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create openfort provider", slog.String("error", err.Error()))
		errD := s.repo.Delete(ctx, prov.ID)
		if errD != nil {
			s.logger.ErrorContext(ctx, "failed to delete provider", slog.String("provider", prov.ID), slog.String("error", errD.Error()))
			errors.Join(err, errD)
		}
		return nil, err
	}

	return prov, nil
}

func (s *service) configureSupabaseAuthentication(ctx context.Context, projectID, supabaseProject string) (*provider.Provider, error) {
	s.logger.InfoContext(ctx, "configuring supabase authentication", slog.String("project_id", projectID))

	prov, err := s.repo.GetByProjectAndType(ctx, projectID, provider.TypeCustom)
	if err != nil && !errors.Is(err, repositories.ErrProviderNotFound) {
		s.logger.ErrorContext(ctx, "failed to get provider", slog.String("error", err.Error()))
		return nil, err
	}

	if prov != nil {
		s.logger.ErrorContext(ctx, "provider already exists", slog.String("error", err.Error()))
		return nil, ErrProviderAlreadyExists
	}

	prov = &provider.Provider{
		ProjectID: projectID,
		Type:      provider.TypeSupabase,
	}
	err = s.repo.Create(ctx, prov)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create provider", slog.String("error", err.Error()))
		return nil, err
	}

	supabaseAuth := &provider.Supabase{
		ProviderID:               prov.ID,
		SupabaseProjectReference: supabaseProject,
	}
	err = s.repo.CreateSupabase(ctx, supabaseAuth)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create supabase provider", slog.String("error", err.Error()))
		errD := s.repo.Delete(ctx, prov.ID)
		if errD != nil {
			s.logger.ErrorContext(ctx, "failed to delete provider", slog.String("provider", prov.ID), slog.String("error", errD.Error()))
			errors.Join(err, errD)
		}
		return nil, err
	}

	return prov, nil
}
