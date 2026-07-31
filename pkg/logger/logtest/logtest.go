// Package logtest captures what the application loggers write, so tests can
// assert on log level and content. Loggers resolve their destination when they
// are built, so a recorder has to be started before the code under test is
// constructed.
package logtest

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/openfort-xyz/shield/pkg/logger"
)

// Record is one decoded log line.
type Record struct {
	Message  string
	Severity string
	Logger   string
	Attrs    map[string]any
}

// Attr returns the string value of an attribute, or "" when it is absent or not
// a string.
func (r Record) Attr(key string) string {
	value, _ := r.Attrs[key].(string)
	return value
}

// Recorder redirects loggers built by logger.New into a buffer and raises the
// level so nothing under test is filtered out before it can be asserted on.
type Recorder struct {
	mu         sync.Mutex
	buf        bytes.Buffer
	restoreOut func()
	restoreLvl func()
}

// Start installs a recorder at the given level. Call Stop, normally deferred, to
// restore the previous destination and level.
func Start(level slog.Level) *Recorder {
	rec := &Recorder{}
	rec.restoreOut = logger.SetOutput(&rec.buf)
	rec.restoreLvl = logger.SetLevel(level)

	return rec
}

// Stop restores the logging configuration Start replaced. It is safe to call
// more than once.
func (r *Recorder) Stop() {
	if r.restoreLvl != nil {
		r.restoreLvl()
		r.restoreLvl = nil
	}

	if r.restoreOut != nil {
		r.restoreOut()
		r.restoreOut = nil
	}
}

// Reset discards everything captured so far, so a table-driven test can reuse
// one recorder across subtests.
func (r *Recorder) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.buf.Reset()
}

// Records decodes everything captured so far. A line that is not valid JSON is
// skipped, since it did not come from a logger this package installed.
func (r *Recorder) Records() []Record {
	r.mu.Lock()
	raw := r.buf.Bytes()
	snapshot := make([]byte, len(raw))
	copy(snapshot, raw)
	r.mu.Unlock()

	var out []Record

	decoder := json.NewDecoder(bytes.NewReader(snapshot))
	for decoder.More() {
		var attrs map[string]any
		if err := decoder.Decode(&attrs); err != nil {
			break
		}

		message, _ := attrs["message"].(string)
		severity, _ := attrs["severity"].(string)
		name, _ := attrs["logger"].(string)

		out = append(out, Record{
			Message:  message,
			Severity: severity,
			Logger:   name,
			Attrs:    attrs,
		})
	}

	return out
}

// Find returns the first record with the given message, and whether one existed.
func (r *Recorder) Find(message string) (Record, bool) {
	for _, rec := range r.Records() {
		if rec.Message == message {
			return rec, true
		}
	}

	return Record{}, false
}

// Severities returns the severity of every captured record, in order.
func (r *Recorder) Severities() []string {
	records := r.Records()
	out := make([]string, 0, len(records))

	for _, rec := range records {
		out = append(out, rec.Severity)
	}

	return out
}
