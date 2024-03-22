package providersvc

import (
	"context"
	"errors"
	"log/slog"
	"os"

	"go.openfort.xyz/shield/internal/core/domain"
	"go.openfort.xyz/shield/internal/core/domain/provider"
	"go.openfort.xyz/shield/internal/core/ports/repositories"
	"go.openfort.xyz/shield/internal/core/ports/services"
	"go.openfort.xyz/shield/pkg/oflog"
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
		return nil, domain.ErrNoProviderConfig
	}

	switch config.GetType() {
	case provider.TypeCustom:
		customConfig, ok := config.GetConfig().(*services.CustomProviderConfig)
		if !ok {
			return nil, domain.ErrInvalidProviderConfig
		}

		return s.configureCustomProvider(ctx, projectID, customConfig.JWKUrl)
	case provider.TypeOpenfort:
		openfortConfig, ok := config.GetConfig().(*services.OpenfortProviderConfig)
		if !ok {
			return nil, domain.ErrInvalidProviderConfig
		}

		return s.configureOpenfortProvider(ctx, projectID, openfortConfig.OpenfortProject)
	default:
		return nil, domain.ErrUnknownProviderType
	}
}

func (s *service) Get(ctx context.Context, providerID string) (*provider.Provider, error) {
	s.logger.InfoContext(ctx, "getting provider", slog.String("provider_id", providerID))

	prov, err := s.repo.Get(ctx, providerID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get provider", slog.String("error", err.Error()))
		return nil, err
	}

	return prov, nil
}

func (s *service) List(ctx context.Context, projectID string) ([]*provider.Provider, error) {
	s.logger.InfoContext(ctx, "listing providers", slog.String("project_id", projectID))

	provs, err := s.repo.List(ctx, projectID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to list providers", slog.String("error", err.Error()))
		return nil, err
	}

	return provs, nil
}

func (s *service) UpdateConfig(ctx context.Context, config interface{}) error {
	s.logger.InfoContext(ctx, "updating provider config")

	if cfg, ok := config.(*provider.CustomConfig); ok {
		s.logger.InfoContext(ctx, "updating custom provider config", slog.String("provider_id", cfg.ProviderID))
		err := s.repo.UpdateCustom(ctx, cfg)
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to update provider config", slog.String("error", err.Error()))
			return err
		}

		return nil
	}

	if cfg, ok := config.(*provider.OpenfortConfig); ok {
		s.logger.InfoContext(ctx, "updating openfort provider config", slog.String("provider_id", cfg.ProviderID))
		err := s.repo.UpdateOpenfort(ctx, cfg)
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to update provider config", slog.String("error", err.Error()))
			return err
		}

		return nil
	}

	return domain.ErrInvalidProviderConfig
}

func (s *service) Remove(ctx context.Context, projectID string, providerID string) error {
	s.logger.InfoContext(ctx, "removing provider", slog.String("project_id", projectID), slog.String("provider_id", providerID))

	err := s.repo.Delete(ctx, providerID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to delete provider", slog.String("error", err.Error()))
		return err
	}

	return nil
}

func (s *service) configureCustomProvider(ctx context.Context, projectID, jwkURL string) (*provider.Provider, error) {
	s.logger.InfoContext(ctx, "configuring custom provider", slog.String("project_id", projectID))

	prov, err := s.repo.GetByProjectAndType(ctx, projectID, provider.TypeCustom)
	if err != nil && !errors.Is(err, domain.ErrProviderNotFound) {
		s.logger.ErrorContext(ctx, "failed to get provider", slog.String("error", err.Error()))
		return nil, err
	}

	if prov != nil {
		s.logger.ErrorContext(ctx, "provider already exists")
		return nil, domain.ErrProviderAlreadyExists
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

	customAuth := &provider.CustomConfig{
		ProviderID: prov.ID,
		JWK:        jwkURL,
	}
	err = s.repo.CreateCustom(ctx, customAuth)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create custom provider", slog.String("error", err.Error()))
		errD := s.repo.Delete(ctx, prov.ID)
		if errD != nil {
			s.logger.ErrorContext(ctx, "failed to delete provider", slog.String("provider", prov.ID), slog.String("error", errD.Error()))
			err = errors.Join(err, errD)
		}
		return nil, err
	}

	prov.Config = customAuth
	return prov, nil
}

func (s *service) configureOpenfortProvider(ctx context.Context, projectID, openfortProject string) (*provider.Provider, error) {
	s.logger.InfoContext(ctx, "configuring openfort provider", slog.String("project_id", projectID))

	prov, err := s.repo.GetByProjectAndType(ctx, projectID, provider.TypeOpenfort)
	if err != nil && !errors.Is(err, domain.ErrProviderNotFound) {
		s.logger.ErrorContext(ctx, "failed to get provider", slog.String("error", err.Error()))
		return nil, err
	}

	if prov != nil {
		s.logger.ErrorContext(ctx, "provider already exists")
		return nil, domain.ErrProviderAlreadyExists
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

	openfortAuth := &provider.OpenfortConfig{
		ProviderID:     prov.ID,
		PublishableKey: openfortProject,
	}
	err = s.repo.CreateOpenfort(ctx, openfortAuth)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create openfort provider", slog.String("error", err.Error()))
		errD := s.repo.Delete(ctx, prov.ID)
		if errD != nil {
			s.logger.ErrorContext(ctx, "failed to delete provider", slog.String("provider", prov.ID), slog.String("error", errD.Error()))
			err = errors.Join(err, errD)
		}
		return nil, err
	}

	prov.Config = openfortAuth
	return prov, nil
}
