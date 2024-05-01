package logger

import (
	"context"
	"log/slog"
	"os"

	"go.openfort.xyz/shield/pkg/contexter"
)

// New creates a new standard logger with a context handler.
func New(name string) *slog.Logger {
	return slog.New(NewContextHandler(name, slog.NewTextHandler(os.Stdout, nil)))
}

// Error returns an attribute for an error string value.
func Error(err error) slog.Attr {
	return slog.String("error", err.Error())
}

// ContextHandler is a logger handler that adds context attributes to log records.
type ContextHandler struct {
	name        string
	baseHandler slog.Handler
}

// NewContextHandler creates a new context handler.
func NewContextHandler(name string, baseHandler slog.Handler) *ContextHandler {
	return &ContextHandler{
		name:        name,
		baseHandler: baseHandler,
	}
}

var _ slog.Handler = (*ContextHandler)(nil)

// Enabled wraps the base handler's Enabled method.
func (c *ContextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return c.baseHandler.Enabled(ctx, level)
}

// Handle warps the base handler's Handle method and adds context attributes to the log record.
func (c *ContextHandler) Handle(ctx context.Context, record slog.Record) error {
	record.Add(slog.String("logger", c.name))
	if projID := contexter.GetProjectID(ctx); projID != "" {
		record.Add(slog.String(ProjectID, projID))
	}

	if reqID := contexter.GetRequestID(ctx); reqID != "" {
		record.Add(slog.String(RequestID, reqID))
	}

	return c.baseHandler.Handle(ctx, record)
}

// WithAttrs wraps the base handler's WithAttrs method.
func (c *ContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return c.baseHandler.WithAttrs(attrs)
}

// WithGroup wraps the base handler's WithGroup method.
func (c *ContextHandler) WithGroup(name string) slog.Handler {
	return c.baseHandler.WithGroup(name)
}
