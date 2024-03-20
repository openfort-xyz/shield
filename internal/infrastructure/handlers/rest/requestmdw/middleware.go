package requestmdw

import (
	"github.com/google/uuid"
	"go.openfort.xyz/shield/pkg/ofcontext"
	"net/http"
)

func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := ofcontext.WithRequestID(r.Context(), uuid.NewString())
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
