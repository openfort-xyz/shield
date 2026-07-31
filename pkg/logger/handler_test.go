package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"testing"

	"github.com/openfort-xyz/shield/pkg/contexter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// capture redirects New to a buffer for the duration of the test and returns it.
// The logger must be built after this call, since New resolves the destination
// when it runs.
func capture(t *testing.T) *bytes.Buffer {
	t.Helper()

	buf := &bytes.Buffer{}
	t.Cleanup(SetOutput(buf))

	return buf
}

// records decodes every JSON line written to buf.
func records(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()

	var out []map[string]any

	decoder := json.NewDecoder(bytes.NewReader(buf.Bytes()))
	for decoder.More() {
		var rec map[string]any
		require.NoError(t, decoder.Decode(&rec))
		out = append(out, rec)
	}

	return out
}

func TestDefaultLevelDropsDebug(t *testing.T) {
	buf := capture(t)
	log := New("test")

	log.DebugContext(context.Background(), "dropped")
	log.InfoContext(context.Background(), "kept")

	recs := records(t, buf)
	require.Len(t, recs, 1, "debug should not be emitted at the default level")
	assert.Equal(t, "kept", recs[0]["message"])
}

func TestSetLevelEnablesDebugOnExistingLoggers(t *testing.T) {
	buf := capture(t)
	// Built before the level changes, to prove the handler follows the shared
	// LevelVar rather than a value captured at construction.
	log := New("test")

	t.Cleanup(SetLevel(slog.LevelDebug))

	log.DebugContext(context.Background(), "now visible")

	recs := records(t, buf)
	require.Len(t, recs, 1)
	assert.Equal(t, "now visible", recs[0]["message"])
	assert.Equal(t, "DEBUG", recs[0]["severity"])
}

func TestSetLevelRestores(t *testing.T) {
	before := Level()
	restore := SetLevel(slog.LevelError)
	assert.Equal(t, slog.LevelError, Level())
	restore()
	assert.Equal(t, before, Level())
}

func TestSeverityIsGCPFormatted(t *testing.T) {
	buf := capture(t)
	t.Cleanup(SetLevel(slog.LevelDebug))
	log := New("test")

	ctx := context.Background()
	log.DebugContext(ctx, "d")
	log.InfoContext(ctx, "i")
	log.WarnContext(ctx, "w")
	log.ErrorContext(ctx, "e")

	recs := records(t, buf)
	require.Len(t, recs, 4)
	// WARN is the one that differs: GCP spells it WARNING.
	assert.Equal(t, []any{"DEBUG", "INFO", "WARNING", "ERROR"},
		[]any{recs[0]["severity"], recs[1]["severity"], recs[2]["severity"], recs[3]["severity"]})
}

func TestContextHandlerAddsRequestAndProjectID(t *testing.T) {
	buf := capture(t)
	log := New("share_service")

	ctx := contexter.WithProjectID(context.Background(), "project-1")
	ctx = contexter.WithRequestID(ctx, "request-1")
	log.InfoContext(ctx, "hello")

	recs := records(t, buf)
	require.Len(t, recs, 1)
	assert.Equal(t, "share_service", recs[0]["logger"])
	assert.Equal(t, "project-1", recs[0][ProjectID])
	assert.Equal(t, "request-1", recs[0][RequestID])
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    slog.Level
		wantErr bool
	}{
		{name: "lowercase", input: "debug", want: slog.LevelDebug},
		{name: "uppercase", input: "ERROR", want: slog.LevelError},
		{name: "mixed case", input: "Warn", want: slog.LevelWarn},
		{name: "surrounding space", input: "  info  ", want: slog.LevelInfo},
		{name: "gcp spelling of warn", input: "warning", want: slog.LevelWarn},
		{name: "offset", input: "debug+2", want: slog.LevelDebug + 2},
		{name: "unknown name", input: "verbose", wantErr: true},
		{name: "empty", input: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseLevel(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConfigureFromEnv(t *testing.T) {
	tests := []struct {
		name    string
		set     bool
		value   string
		want    slog.Level
		wantErr bool
	}{
		{name: "unset leaves the default", set: false, want: slog.LevelInfo},
		{name: "empty leaves the default", set: true, value: "", want: slog.LevelInfo},
		{name: "blank leaves the default", set: true, value: "   ", want: slog.LevelInfo},
		{name: "debug", set: true, value: "debug", want: slog.LevelDebug},
		{name: "error", set: true, value: "error", want: slog.LevelError},
		// An unusable value must not silently change the level, otherwise a typo
		// could quietly mute the service.
		{name: "invalid leaves the level untouched", set: true, value: "nope", want: slog.LevelInfo, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(SetLevel(slog.LevelInfo))

			// t.Setenv registers the restore, so unsetting afterwards is still
			// undone when the subtest finishes.
			t.Setenv(LevelEnvVar, tt.value)
			if !tt.set {
				require.NoError(t, os.Unsetenv(LevelEnvVar))
			}

			err := ConfigureFromEnv()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.want, Level())
		})
	}
}
