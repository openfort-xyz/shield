package userapp

import (
	"context"
	"go.openfort.xyz/shield/internal/core/ports/services"
	"go.openfort.xyz/shield/internal/infrastructure/providersmgr"
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
	providerManager *providersmgr.Manager
	logger          *slog.Logger
}

func New(userSvc services.UserService, shareSvc services.ShareService, projectSvc services.ProjectService, providerSvc services.ProviderService, providerManager *providersmgr.Manager) *UserApplication {
	return &UserApplication{
		userSvc:         userSvc,
		shareSvc:        shareSvc,
		projectSvc:      projectSvc,
		providerSvc:     providerSvc,
		providerManager: providerManager,
		logger:          slog.New(oflog.NewContextHandler(slog.NewTextHandler(os.Stdout, nil))).WithGroup("user_application"),
	}
}

func (a *UserApplication) RegisterShare(ctx context.Context, share string) error {
	a.logger.InfoContext(ctx, "registering share")

	usrID := ofcontext.GetUserID(ctx)
	err := a.shareSvc.Create(ctx, usrID, share)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to create share", slog.String("error", err.Error()))
		return err
	}

	return nil
}

func (a *UserApplication) GetShare(ctx context.Context) (string, error) {
	a.logger.InfoContext(ctx, "getting share")

	usrID := ofcontext.GetUserID(ctx)
	shr, err := a.shareSvc.GetByUserID(ctx, usrID)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to get share by user ID", slog.String("error", err.Error()))
		return "", err
	}

	return shr.Data, nil
}
