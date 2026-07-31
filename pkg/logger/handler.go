package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/openfort-xyz/shield/pkg/contexter"
)

// LevelEnvVar is the environment variable read by ConfigureFromEnv.
const LevelEnvVar = "LOG_LEVEL"

var (
	// levelVar is shared by every handler New builds. Handlers hold a pointer to
	// it, so changing it also affects loggers that already exist and the level can
	// be set after the process has loaded its environment. Its zero value is
	// LevelInfo, which is what slog assumes when HandlerOptions.Level is nil, so
	// the default behaviour is unchanged.
	levelVar = new(slog.LevelVar)

	outputMu sync.Mutex
	output   io.Writer = os.Stdout
)

var handlerOpts = &slog.HandlerOptions{
	Level: levelVar,
	ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
		switch a.Key {
		case slog.LevelKey:
			return slog.Attr{
				Key:   "severity",
				Value: slog.StringValue(levelToGCP(a.Value.String())),
			}

		case slog.MessageKey:
			return slog.String("message", a.Value.String())

		case slog.TimeKey:
			return slog.String("time", a.Value.Time().Format(time.RFC3339Nano))

		default:
			return a
		}
	},
}

func levelToGCP(level string) string {
	switch level {
	case "DEBUG":
		return "DEBUG"
	case "INFO":
		return "INFO"
	case "WARN":
		return "WARNING"
	case "ERROR":
		return "ERROR"
	default:
		return level
	}
}

// ParseLevel maps a level name to a slog.Level. It accepts the names slog uses
// ("debug", "info", "warn", "error") case-insensitively, with an optional numeric
// offset such as "debug+2", plus "warning" for symmetry with the severity values
// this package emits.
func ParseLevel(name string) (slog.Level, error) {
	trimmed := strings.TrimSpace(name)
	if strings.EqualFold(trimmed, "warning") {
		trimmed = "warn"
	}

	var lvl slog.Level
	if err := lvl.UnmarshalText([]byte(trimmed)); err != nil {
		return 0, fmt.Errorf("parsing %s: %w", LevelEnvVar, err)
	}

	return lvl, nil
}

// ConfigureFromEnv applies LevelEnvVar to every logger New builds, including the
// ones it has already built. Call it once the process has loaded its environment,
// since package initialisation runs before any .env file is read. An unset or
// empty variable leaves the level at info; an unparseable one is returned as an
// error and leaves the level untouched.
func ConfigureFromEnv() error {
	raw, ok := os.LookupEnv(LevelEnvVar)
	if !ok || strings.TrimSpace(raw) == "" {
		return nil
	}

	lvl, err := ParseLevel(raw)
	if err != nil {
		return err
	}

	levelVar.Set(lvl)

	return nil
}

// Level reports the minimum level loggers built by New currently emit.
func Level() slog.Level {
	return levelVar.Level()
}

// SetLevel overrides the minimum level for every logger built by New and returns
// a function restoring the previous level. Intended for tests; production code
// should go through ConfigureFromEnv.
func SetLevel(lvl slog.Level) (restore func()) {
	previous := levelVar.Level()
	levelVar.Set(lvl)

	return func() { levelVar.Set(previous) }
}

// SetOutput changes where subsequent calls to New write and returns a function
// restoring the previous destination. Loggers that already exist keep the
// destination they were built with, so callers must set the output before
// constructing whatever they intend to capture. Intended for tests.
func SetOutput(w io.Writer) (restore func()) {
	outputMu.Lock()
	defer outputMu.Unlock()

	previous := output
	output = w

	return func() {
		outputMu.Lock()
		defer outputMu.Unlock()
		output = previous
	}
}

// New creates a new standard logger with a context handler.
func New(name string) *slog.Logger {
	outputMu.Lock()
	w := output
	outputMu.Unlock()

	return slog.New(NewContextHandler(name, slog.NewJSONHandler(w, handlerOpts)))
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
