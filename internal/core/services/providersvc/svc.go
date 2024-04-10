package providersvc

import (
	"context"
	"errors"
	"log/slog"

	"go.openfort.xyz/shield/internal/core/domain"
	"go.openfort.xyz/shield/internal/core/domain/provider"
	"go.openfort.xyz/shield/internal/core/ports/repositories"
	"go.openfort.xyz/shield/internal/core/ports/services"
	"go.openfort.xyz/shield/pkg/logger"
)

type service struct {
	repo   repositories.ProviderRepository
	logger *slog.Logger
}

var _ services.ProviderService = (*service)(nil)

func New(repo repositories.ProviderRepository) services.ProviderService {
	return &service{
		repo:   repo,
		logger: logger.New("provider_service"),
	}
}

func (s *service) Configure(ctx context.Context, prov *provider.Provider) error {
	switch prov.Type {
	case provider.TypeCustom:
		return s.configureCustomProvider(ctx, prov)
	case provider.TypeOpenfort:
		return s.configureOpenfortProvider(ctx, prov)
	default:
		return domain.ErrUnknownProviderType
	}
}

func (s *service) configureCustomProvider(ctx context.Context, prov *provider.Provider) error {
	s.logger.InfoContext(ctx, "configuring custom provider", slog.String("project_id", prov.ProjectID))

	err := s.repo.Create(ctx, prov)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create provider", logger.Error(err))
		return err
	}

	customAuth, ok := prov.Config.(*provider.CustomConfig)
	if !ok {
		s.logger.ErrorContext(ctx, "invalid custom provider config")
		return domain.ErrInvalidProviderConfig
	}

	customAuth.ProviderID = prov.ID

	err = s.repo.CreateCustom(ctx, customAuth)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create custom provider", logger.Error(err))
		errD := s.repo.Delete(ctx, prov.ID)
		if errD != nil {
			s.logger.ErrorContext(ctx, "failed to delete provider", slog.String("provider", prov.ID), logger.Error(errD))
			err = errors.Join(err, errD)
		}
		return err
	}

	return nil
}

func (s *service) configureOpenfortProvider(ctx context.Context, prov *provider.Provider) error {
	s.logger.InfoContext(ctx, "configuring openfort provider", slog.String("project_id", prov.ProjectID))

	err := s.repo.Create(ctx, prov)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create provider", logger.Error(err))
		return err
	}

	openfortAuth, ok := prov.Config.(*provider.OpenfortConfig)
	if !ok {
		s.logger.ErrorContext(ctx, "invalid openfort provider config")
		return domain.ErrInvalidProviderConfig
	}

	openfortAuth.ProviderID = prov.ID

	err = s.repo.CreateOpenfort(ctx, openfortAuth)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create openfort provider", logger.Error(err))
		errD := s.repo.Delete(ctx, prov.ID)
		if errD != nil {
			s.logger.ErrorContext(ctx, "failed to delete provider", slog.String("provider", prov.ID), logger.Error(errD))
			err = errors.Join(err, errD)
		}
		return err
	}

	return nil
}
