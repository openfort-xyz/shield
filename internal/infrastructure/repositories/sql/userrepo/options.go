package userrepo

import (
	"go.openfort.xyz/shield/internal/core/domain/provider"
	"go.openfort.xyz/shield/internal/core/ports/repositories"
)

type options struct {
	query map[string]interface{}
}

func (r *repository) WithProviderType(providerType provider.Type) repositories.Option {
	return func(opts repositories.Options) {
		opts.(*options).query["type"] = r.parser.mapProviderTypeToDatabase[providerType]
	}
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

func (r *repository) WithProjectID(projectID string) repositories.Option {
	return func(opts repositories.Options) {
		opts.(*options).query["project_id"] = projectID
	}
}
