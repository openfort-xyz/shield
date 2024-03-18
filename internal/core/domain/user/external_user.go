package user

import "go.openfort.xyz/shield/internal/core/domain/provider"

type ExternalUser struct {
	ID             string
	UserID         string
	ExternalUserID string
	Type           provider.Type
	ProjectID      string
}
