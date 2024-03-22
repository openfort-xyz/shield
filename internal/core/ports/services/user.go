package services

import (
	"context"

	"go.openfort.xyz/shield/internal/core/domain/user"
)

type UserService interface {
	Create(ctx context.Context, projectID string) (*user.User, error)
	Get(ctx context.Context, userID string) (*user.User, error)
	GetByExternal(ctx context.Context, externalUserID, providerID string) (*user.User, error)
	CreateExternal(ctx context.Context, projectID, userID, externalUserID, providerID string) (*user.ExternalUser, error)
}
