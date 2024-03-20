package ofcontext

import "context"

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

func WithAPIKey(ctx context.Context, apiKey string) context.Context {
	return context.WithValue(ctx, ContextKeyAPIKey, apiKey)
}

func GetAPIKey(ctx context.Context) string {
	apiKey, ok := ctx.Value(ContextKeyAPIKey).(string)
	if !ok {
		return ""
	}

	return apiKey
}

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

func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, ContextKeyUserID, userID)
}

func GetUserID(ctx context.Context) string {
	userID, ok := ctx.Value(ContextKeyUserID).(string)
	if !ok {
		return ""
	}

	return userID
}
