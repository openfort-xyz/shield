package api

import (
	"errors"
	"io"
	"net/http"
)

// ReadBody reads the request body. On failure it writes the error response,
// 413 for a body over the server's limit and 400 otherwise, and returns false.
func ReadBody(w http.ResponseWriter, r *http.Request) ([]byte, bool) {
	body, err := io.ReadAll(r.Body)
	var tooLarge *http.MaxBytesError
	switch {
	case errors.As(err, &tooLarge):
		RespondWithError(w, ErrBodyTooLarge(tooLarge.Limit))
		return nil, false
	case err != nil:
		RespondWithError(w, ErrBadRequestWithMessage("failed to read request body"))
		return nil, false
	}
	return body, true
}
