package repositories

import (
	"context"

	"go.openfort.xyz/shield/internal/core/domain/usercontact"
)

type UserContactRepository interface {
	Save(ctx context.Context, project *usercontact.UserContact) error
	GetByUserID(ctx context.Context, userID string) (*usercontact.UserContact, error)
}
