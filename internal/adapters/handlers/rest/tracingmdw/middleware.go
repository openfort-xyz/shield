// Package tracingmdw renames the active server span to the iFrame flow name
// (e.g. "embedded.create") when the client sends `x-openfort-flow-name`.
// This gives Grafana a useful trace label in the search summary, since
// Jaeger-style trace name = root span's operation name.
//
// Must run after `otelmux.Middleware` so a span is already on the request
// context. Safe to run on every request — absent/invalid headers are no-ops.
package tracingmdw

import (
	"net/http"

	"go.opentelemetry.io/otel/trace"
)

const FlowNameHeader = "X-Openfort-Flow-Name"

// Iframe-sourced flow attributes. Shield itself doesn't read them today, but
// they're named here (and allow-listed in CORS) so the browser doesn't block
// the request before the api/castle side can attach them as span attributes.
const UserIDHeader = "X-Openfort-User-Id"
const ChainIDHeader = "X-Openfort-Chain-Id"

// Bound guard so a malicious client can't balloon span names in storage.
const maxFlowNameLen = 64

func FlowNameMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := r.Header.Get(FlowNameHeader)
		if name != "" && len(name) <= maxFlowNameLen {
			if span := trace.SpanFromContext(r.Context()); span.IsRecording() {
				span.SetName(name)
			}
		}
		next.ServeHTTP(w, r)
	})
}
