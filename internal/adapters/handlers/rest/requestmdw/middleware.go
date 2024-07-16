package requestmdw

import (
	"net/http"

	"go.openfort.xyz/shield/pkg/random"

	"go.openfort.xyz/shield/pkg/contexter"
)

const RequestIDHeader = "X-Request-ID"

func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if r.Header.Get(RequestIDHeader) != "" {
			ctx = contexter.WithRequestID(ctx, r.Header.Get(RequestIDHeader))
		} else {
			requestID, _ := random.UUIDv7()
			ctx = contexter.WithRequestID(ctx, requestID)
		}

		w.Header().Set(RequestIDHeader, contexter.GetRequestID(ctx))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
