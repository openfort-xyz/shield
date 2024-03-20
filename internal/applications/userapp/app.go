package userapp

import (
	"context"
	"go.openfort.xyz/shield/internal/core/domain/provider"
	"go.openfort.xyz/shield/internal/core/ports/services"
	"go.openfort.xyz/shield/internal/infrastructure/providers"
	"go.openfort.xyz/shield/pkg/ofcontext"
	"go.openfort.xyz/shield/pkg/oflog"
	"log/slog"
	"os"
)

type UserApplication struct {
	userSvc         services.UserService
	shareSvc        services.ShareService
	projectSvc      services.ProjectService
	providerSvc     services.ProviderService
	providerManager *providers.ProviderManager
	logger          *slog.Logger
}

func New(userSvc services.UserService, shareSvc services.ShareService, projectSvc services.ProjectService, providerSvc services.ProviderService, providerManager *providers.ProviderManager) *UserApplication {
	return &UserApplication{
		userSvc:         userSvc,
		shareSvc:        shareSvc,
		projectSvc:      projectSvc,
		providerSvc:     providerSvc,
		providerManager: providerManager,
		logger:          slog.New(oflog.NewContextHandler(slog.NewTextHandler(os.Stdout, nil))).WithGroup("user_application"),
	}
}

func (a *UserApplication) RegisterShare(ctx context.Context, share, token string, providerType provider.Type) error {
	a.logger.InfoContext(ctx, "registering share")

	usrID, err := a.identifyUser(ctx, token, providerType)
	if err != nil {
		return err
	}

	err = a.shareSvc.Create(ctx, usrID, share)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to create share", slog.String("error", err.Error()))
		return err
	}

	return nil
}

func (a *UserApplication) GetShare(ctx context.Context, token string, providerType provider.Type) (string, error) {
	a.logger.InfoContext(ctx, "getting share")

	usrID, err := a.identifyUser(ctx, token, providerType)
	shr, err := a.shareSvc.GetByUserID(ctx, usrID)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to get share by user ID", slog.String("error", err.Error()))
		return "", err
	}

	return shr.Data, nil
}

func (a *UserApplication) identifyUser(ctx context.Context, token string, providerType provider.Type) (string, error) {
	a.logger.InfoContext(ctx, "identifying user")

	apiKey := ofcontext.GetAPIKey(ctx)
	proj, err := a.projectSvc.GetByAPIKey(ctx, apiKey)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to get project by API key", slog.String("error", err.Error()))
		return "", err
	}

	prov, err := a.providerManager.GetProvider(ctx, proj.ID, providerType)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to get provider", slog.String("error", err.Error()))
	}

	externalUserID, err := prov.Identify(ctx, token)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to identify user", slog.String("error", err.Error()))
		return "", err
	}

	usr, err := a.userSvc.GetByExternal(ctx, externalUserID, prov.GetProviderID())
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to get user by external", slog.String("error", err.Error()))
		return "", err
	}

	return usr.ID, nil
}
