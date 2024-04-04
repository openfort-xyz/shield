package requestmdw

import (
	"net/http"

	"github.com/google/uuid"
	"go.openfort.xyz/shield/pkg/contexter"
)

const RequestIDHeader = "X-Request-ID"

func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if r.Header.Get(RequestIDHeader) != "" {
			ctx = contexter.WithRequestID(ctx, r.Header.Get(RequestIDHeader))
		} else {
			ctx = contexter.WithRequestID(ctx, uuid.NewString())
		}

		w.Header().Set(RequestIDHeader, contexter.GetRequestID(ctx))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
