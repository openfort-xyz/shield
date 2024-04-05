package ratelimitermdw

import (
	"net/http"

	"go.uber.org/ratelimit"
)

type Middleware struct {
	limiter ratelimit.Limiter
}

func New(limit int) *Middleware {
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
