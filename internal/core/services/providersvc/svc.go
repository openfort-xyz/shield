package providersvc

import (
	"context"
	"log/slog"

	domainErrors "github.com/openfort-xyz/shield/internal/core/domain/errors"

	"github.com/openfort-xyz/shield/internal/core/domain/project"
	"github.com/openfort-xyz/shield/internal/core/domain/provider"
	"github.com/openfort-xyz/shield/internal/core/ports/repositories"
	"github.com/openfort-xyz/shield/internal/core/ports/services"
	"github.com/openfort-xyz/shield/pkg/logger"
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

func (s *service) Configure(ctx context.Context, projectID string, provs []*provider.Provider, check project.CheckAPISecretFunc) error {
	s.logger.InfoContext(ctx, "configuring providers", slog.String("project_id", projectID))

	for _, prov := range provs {
		var ok bool
		switch prov.Type {
		case provider.TypeCustom:
			_, ok = prov.Config.(*provider.CustomConfig)
		case provider.TypeOpenfort:
			_, ok = prov.Config.(*provider.OpenfortConfig)
		default:
			return domainErrors.ErrUnknownProviderType
		}
		if !ok {
			s.logger.ErrorContext(ctx, "invalid provider config")
			return domainErrors.ErrInvalidProviderConfig
		}
	}

	if err := s.repo.Create(ctx, projectID, provs, check); err != nil {
		s.logger.ErrorContext(ctx, "failed to create providers", logger.Error(err))
		return err
	}
	return nil
}
