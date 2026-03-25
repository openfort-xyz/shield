package services

import (
	"context"

	"github.com/openfort-xyz/shield/internal/core/domain/user"
)

type UserService interface {
	GetOrCreate(ctx context.Context, projectID, externalUserID, providerID string) (*user.User, error)
}
