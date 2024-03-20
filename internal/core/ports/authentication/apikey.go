package authentication

import "context"

type APIKeyAuthenticator interface {
	Authenticate(ctx context.Context, apiKey string) (projectID string, err error)
}
