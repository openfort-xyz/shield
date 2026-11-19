// Package deadlinemdw lets named routes run past the server-wide WriteTimeout.
package deadlinemdw

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/openfort-xyz/shield/pkg/logger"
)

// ExtendWriteDeadline sets the connection write deadline to now+d for each
// route whose mux name is a key of routes. Without it, net/http drops the
// response of a handler that outlives WriteTimeout and closes the connection.
//
// Register it before any middleware whose ResponseWriter wrapper lacks
// Unwrap (metrics.HTTPMiddleware): http.ResponseController cannot reach the
// connection through such a wrapper.
func ExtendWriteDeadline(routes map[string]time.Duration) mux.MiddlewareFunc {
	log := logger.New("deadline_middleware")
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if route := mux.CurrentRoute(r); route != nil {
				if d, ok := routes[route.GetName()]; ok {
					if err := http.NewResponseController(w).SetWriteDeadline(time.Now().Add(d)); err != nil {
						log.WarnContext(r.Context(), "could not extend write deadline",
							slog.String("route", route.GetName()), logger.Error(err))
					}
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
