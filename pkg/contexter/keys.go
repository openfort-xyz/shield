package contexter

type ContextKey string

const (
	ContextKeyRequestID   ContextKey = "request-id"
	ContextKeyProjectID   ContextKey = "project-id"
	ContextKeyProject     ContextKey = "project"
	ContextKeyAPIKey      ContextKey = "api-key"
	ContextKeyAPISecret   ContextKey = "api-secret"
	ContextKeyUserID      ContextKey = "user-id"
	ContextExternalUserID ContextKey = "external-user-id"
)
