package bodylimitmdw

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/openfort-xyz/shield/internal/adapters/handlers/rest/api"
)

// maxBodyBytes covers the largest legitimate body, a bulk encryption lookup
// of 100 IDs of up to 255 characters each, with room for JSON escaping.
const maxBodyBytes = 256 << 10

var errBodyTooLarge = &api.Error{
	Message: fmt.Sprintf("Request body exceeds %d bytes", maxBodyBytes),
	Code:    "BODY_TOO_LARGE",
	Status:  http.StatusRequestEntityTooLarge,
}

// LimitBody buffers the request body up to maxBodyBytes and answers 413 past
// it. Reading it here, rather than in each handler, gives a body without a
// Content-Length the same 413.
func LimitBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBodyBytes))
		var tooLarge *http.MaxBytesError
		switch {
		case errors.As(err, &tooLarge):
			api.RespondWithError(w, errBodyTooLarge)
			return
		case err != nil:
			api.RespondWithError(w, api.ErrBadRequestWithMessage("failed to read request body"))
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))
		next.ServeHTTP(w, r)
	})
}
