package userrepo

import (
	"go.openfort.xyz/shield/internal/core/ports/repositories"
)

type options struct {
	query map[string]interface{}
}

func (r *repository) WithUserID(userID string) repositories.Option {
	return func(opts repositories.Options) {
		opts.(*options).query["user_id"] = userID
	}
}

func (r *repository) WithExternalUserID(externalUserID string) repositories.Option {
	return func(opts repositories.Options) {
		opts.(*options).query["external_user_id"] = externalUserID
	}
}

func (r *repository) WithProviderID(providerID string) repositories.Option {
	return func(opts repositories.Options) {
		opts.(*options).query["provider_id"] = providerID
	}
}
