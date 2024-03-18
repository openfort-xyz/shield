package oflog

import (
	"context"
	"go.openfort.xyz/shield/pkg/ofcontext"
	"log/slog"
)

type ContextHandler struct {
	baseHandler slog.Handler
}

func NewContextHandler(baseHandler slog.Handler) *ContextHandler {
	return &ContextHandler{
		baseHandler: baseHandler,
	}
}

var _ slog.Handler = (*ContextHandler)(nil)

func (c *ContextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return c.baseHandler.Enabled(ctx, level)
}

func (c *ContextHandler) Handle(ctx context.Context, record slog.Record) error {
	if projID := ofcontext.GetProjectID(ctx); projID != "" {
		record.Add(slog.String(ProjectID, projID))
	}

	if reqID := ofcontext.GetRequestID(ctx); reqID != "" {
		record.Add(slog.String(RequestID, reqID))
	}

	return c.baseHandler.Handle(ctx, record)
}

func (c *ContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return c.baseHandler.WithAttrs(attrs)
}

func (c *ContextHandler) WithGroup(name string) slog.Handler {
	return c.baseHandler.WithGroup(name)
}
