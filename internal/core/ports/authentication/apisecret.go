package authentication

import "context"

type APISecretAuthenticator interface {
	Authenticate(ctx context.Context, apiKey, apiSecret string) (projectID string, err error)
}
