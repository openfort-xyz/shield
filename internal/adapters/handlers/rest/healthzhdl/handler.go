package healthzhdl

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"go.openfort.xyz/shield/internal/applications/healthzapp"
	domainErrors "go.openfort.xyz/shield/internal/core/domain/errors"
)

type Handler struct {
	app *healthzapp.Application
}

func New(app *healthzapp.Application) *Handler {
	return &Handler{
		app: app,
	}
}

type Status struct {
	Status string  `json:"status"`
	At     string  `json:"at"`
	Checks []Check `json:"checks"`
}

type Check struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

func (h *Handler) Healthz(w http.ResponseWriter, r *http.Request) {
	status := Status{
		Status: "healthy",
		At:     time.Now().UTC().Format(time.RFC3339),
		Checks: []Check{},
	}

	err := h.app.Healthz(r.Context())
	if err != nil {
		status.Status = "unhealthy"
		if errors.Is(err, domainErrors.ErrDatabaseUnavailable) {
			status.Checks = append(status.Checks, Check{
				Name:   "database",
				Status: "unhealthy",
			})
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		status.Checks = append(status.Checks, Check{
			Name:   "database",
			Status: "healthy",
		})
		w.WriteHeader(http.StatusOK)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(status)
}
