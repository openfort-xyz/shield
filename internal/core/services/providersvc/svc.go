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

func (s *service) Configure(ctx context.Context, projectID string, config services.ProviderConfig) error {
	if config == nil {
		return ErrNoProviderConfig
	}

	switch config.GetType() {
	case provider.TypeCustom:
		customConfig, ok := config.GetConfig().(*services.CustomProviderConfig)
		if !ok {
			return ErrInvalidProviderConfig
		}

		return s.configureCustomAuthentication(ctx, projectID, customConfig.JWKUrl)
	case provider.TypeOpenfort:
		openfortConfig, ok := config.GetConfig().(*services.OpenfortProviderConfig)
		if !ok {
			return ErrInvalidProviderConfig
		}

		return s.configureOpenfortAuthentication(ctx, projectID, openfortConfig.OpenfortProject)
	case provider.TypeSupabase:
		supabaseConfig, ok := config.GetConfig().(*services.SupabaseProviderConfig)
		if !ok {
			return ErrInvalidProviderConfig
		}

		return s.configureSupabaseAuthentication(ctx, projectID, supabaseConfig.SupabaseProject)
	default:
		return ErrUnknownProviderType
	}
}

func (s *service) configureCustomAuthentication(ctx context.Context, projectID, jwkUrl string) error {
	s.logger.InfoContext(ctx, "configuring custom authentication", slog.String("project_id", projectID))

	customAuth, err := s.repo.GetCustom(ctx, projectID)
	if err != nil && !errors.Is(err, repositories.ErrCustomProviderNotFound) {
		s.logger.ErrorContext(ctx, "failed to get custom authentication", slog.String("error", err.Error()))
		return err
	}

	if customAuth != nil {
		s.logger.ErrorContext(ctx, "custom authentication already exists")
		return ErrCustomProviderAlreadyExists
	}

	customAuth = &provider.Custom{
		ProjectID: projectID,
		JWK:       jwkUrl,
	}
	err = s.repo.CreateCustom(ctx, customAuth)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create custom authentication", slog.String("error", err.Error()))
		return err
	}

	return nil
}

func (s *service) configureOpenfortAuthentication(ctx context.Context, projectID, openfortProject string) error {
	s.logger.InfoContext(ctx, "configuring openfort authentication", slog.String("project_id", projectID))

	openfortAuth, err := s.repo.GetOpenfort(ctx, projectID)
	if err != nil && !errors.Is(err, repositories.ErrOpenfortProviderNotFound) {
		s.logger.ErrorContext(ctx, "failed to get openfort authentication", slog.String("error", err.Error()))
		return err
	}

	if openfortAuth != nil {
		s.logger.ErrorContext(ctx, "openfort authentication already exists")
		return ErrOpenfortProviderAlreadyExists
	}

	openfortAuth = &provider.Openfort{
		ProjectID:         projectID,
		OpenfortProjectID: openfortProject,
	}
	err = s.repo.CreateOpenfort(ctx, openfortAuth)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create openfort authentication", slog.String("error", err.Error()))
		return err
	}

	return nil
}

func (s *service) configureSupabaseAuthentication(ctx context.Context, projectID, supabaseProject string) error {
	s.logger.InfoContext(ctx, "configuring supabase authentication", slog.String("project_id", projectID))

	supabaseAuth, err := s.repo.GetSupabase(ctx, projectID)
	if err != nil && !errors.Is(err, repositories.ErrSupabaseProviderNotFound) {
		s.logger.ErrorContext(ctx, "failed to get supabase authentication", slog.String("error", err.Error()))
		return err
	}

	if supabaseAuth != nil {
		s.logger.ErrorContext(ctx, "supabase authentication already exists")
		return ErrSupabaseProviderAlreadyExists
	}

	supabaseAuth = &provider.Supabase{
		ProjectID:                projectID,
		SupabaseProjectReference: supabaseProject,
	}
	err = s.repo.CreateSupabase(ctx, supabaseAuth)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create supabase authentication", slog.String("error", err.Error()))
		return err
	}

	return nil
}
