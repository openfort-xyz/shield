package contexter

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
