package bodylimitmdw

import (
	"net/http"
)

// maxBodyBytes covers the largest legitimate body, a bulk encryption lookup
// of 100 IDs of up to 255 characters each, with room for JSON escaping.
const maxBodyBytes = 256 << 10

// LimitBody caps the request body at maxBodyBytes without reading it, so
// authentication still answers first. Handlers read the body with
// api.ReadBody, which answers 413 past the cap.
func LimitBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		next.ServeHTTP(w, r)
	})
}
