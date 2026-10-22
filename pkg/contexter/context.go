package contexter

import (
	"context"

	"github.com/openfort-xyz/shield/internal/core/domain/project"
)

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, ContextKeyRequestID, requestID)
}

func GetRequestID(ctx context.Context) string {
	reqID, ok := ctx.Value(ContextKeyRequestID).(string)
	if !ok {
		return ""
	}

	return reqID
}

func WithProjectID(ctx context.Context, projectID string) context.Context {
	return context.WithValue(ctx, ContextKeyProjectID, projectID)
}

func GetProjectID(ctx context.Context) string {
	projectID, ok := ctx.Value(ContextKeyProjectID).(string)
	if !ok {
		return ""
	}

	return projectID
}

func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, ContextKeyUserID, userID)
}

func WithExternalUserID(ctx context.Context, externalUserID string) context.Context {
	return context.WithValue(ctx, ContextExternalUserID, externalUserID)
}

func GetUserID(ctx context.Context) string {
	userID, ok := ctx.Value(ContextKeyUserID).(string)
	if !ok {
		return ""
	}

	return userID
}

func GetExternalUserID(ctx context.Context) string {
	userID, ok := ctx.Value(ContextExternalUserID).(string)
	if !ok {
		return ""
	}

	return userID
}

func WithProject(ctx context.Context, proj *project.Project) context.Context {
	return context.WithValue(ctx, ContextKeyProject, proj)
}

func GetProject(ctx context.Context) *project.Project {
	proj, ok := ctx.Value(ContextKeyProject).(*project.Project)
	if !ok {
		return nil
	}

	return proj
}

// WithAPIKey records the API key the request authenticated with.
func WithAPIKey(ctx context.Context, apiKey string) context.Context {
	return context.WithValue(ctx, ContextKeyAPIKey, apiKey)
}

// WithAPISecret records the API secret the request authenticated with. Only
// the secret rotation reads it; never log it.
func WithAPISecret(ctx context.Context, apiSecret string) context.Context {
	return context.WithValue(ctx, ContextKeyAPISecret, apiSecret)
}

func GetAPISecret(ctx context.Context) string {
	apiSecret, ok := ctx.Value(ContextKeyAPISecret).(string)
	if !ok {
		return ""
	}

	return apiSecret
}

func GetAPIKey(ctx context.Context) string {
	apiKey, ok := ctx.Value(ContextKeyAPIKey).(string)
	if !ok {
		return ""
	}

	return apiKey
}
