package userhdl

import (
	"go.openfort.xyz/shield/internal/applications/userapp"
	"go.openfort.xyz/shield/pkg/oflog"
	"log/slog"
	"net/http"
	"os"
)

type Handler struct {
	app    *userapp.UserApplication
	logger *slog.Logger
}

func New(app *userapp.UserApplication) *Handler {
	return &Handler{
		app:    app,
		logger: slog.New(oflog.NewContextHandler(slog.NewTextHandler(os.Stdout, nil))).WithGroup("user_handler"),
	}
}

func (h *Handler) RegisterShare(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.InfoContext(ctx, "registering share")
}

func (h *Handler) GetShare(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.InfoContext(ctx, "getting share")
}
