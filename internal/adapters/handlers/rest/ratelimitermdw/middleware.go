package ratelimitermdw

import (
	"net/http"

	"go.uber.org/ratelimit"
)

type Middleware struct {
	limiter ratelimit.Limiter
}

// New limits requests to limit per second. Zero disables rate limiting.
func New(limit int) *Middleware {
	if limit == 0 {
		return &Middleware{limiter: ratelimit.NewUnlimited()}
	}
	return &Middleware{
		limiter: ratelimit.New(limit),
	}
}

func (m *Middleware) RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.limiter.Take()
		next.ServeHTTP(w, r)
	})
}
