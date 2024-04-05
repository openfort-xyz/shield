package logger

import (
	"context"
	"log/slog"
	"os"

	"go.openfort.xyz/shield/pkg/contexter"
)

func New(name string) *slog.Logger {
	return slog.New(NewContextHandler(slog.NewTextHandler(os.Stdout, nil))).WithGroup(name)
}

func Error(err error) slog.Attr {
	return slog.String("error", err.Error())
}

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
	if projID := contexter.GetProjectID(ctx); projID != "" {
		record.Add(slog.String(ProjectID, projID))
	}

	if reqID := contexter.GetRequestID(ctx); reqID != "" {
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
