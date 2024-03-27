package userapp

import (
	"context"
	"log/slog"
	"os"

	"go.openfort.xyz/shield/internal/core/domain/share"
	"go.openfort.xyz/shield/internal/core/ports/services"
	"go.openfort.xyz/shield/internal/infrastructure/providersmgr"
	"go.openfort.xyz/shield/pkg/ofcontext"
	"go.openfort.xyz/shield/pkg/oflog"
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

func (a *UserApplication) RegisterShare(ctx context.Context, secret string, userEntropy bool, parameters *EncryptionParameters) error {
	a.logger.InfoContext(ctx, "registering share")
	usrID := ofcontext.GetUserID(ctx)

	shre := &share.Share{
		Data:        secret,
		UserID:      usrID,
		UserEntropy: userEntropy,
	}
	if parameters != nil {
		shre.Salt = parameters.Salt
		shre.Iterations = parameters.Iterations
		shre.Length = parameters.Length
		shre.Digest = parameters.Digest
	}
	err := a.shareSvc.Create(ctx, shre)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to create share", slog.String("error", err.Error()))
		return fromDomainError(err)
	}

	return nil
}

func (a *UserApplication) GetShare(ctx context.Context) (*share.Share, error) {
	a.logger.InfoContext(ctx, "getting share")

	usrID := ofcontext.GetUserID(ctx)
	shr, err := a.shareSvc.GetByUserID(ctx, usrID)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to get share by user ID", slog.String("error", err.Error()))
		return nil, fromDomainError(err)
	}

	return shr, nil
}
