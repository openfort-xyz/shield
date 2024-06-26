package contexter

type ContextKey string

const (
	ContextKeyRequestID ContextKey = "request-id"
	ContextKeyProjectID ContextKey = "project-id"
	ContextKeyAPIKey    ContextKey = "api-key"
	ContextKeyAPISecret ContextKey = "api-secret"
	ContextKeyUserID    ContextKey = "user-id"
)
