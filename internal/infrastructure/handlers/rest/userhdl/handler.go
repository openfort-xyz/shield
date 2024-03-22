package userhdl

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"

	"go.openfort.xyz/shield/internal/applications/userapp"
	"go.openfort.xyz/shield/internal/infrastructure/handlers/rest/api"
	"go.openfort.xyz/shield/pkg/oflog"
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

	body, err := io.ReadAll(r.Body)
	if err != nil {
		api.RespondWithError(w, api.ErrBadRequestWithMessage("failed to read request body"))
		return
	}

	var req RegisterShareRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		api.RespondWithError(w, api.ErrBadRequestWithMessage("failed to parse request body"))
		return
	}

	err = h.app.RegisterShare(ctx, req.Share)
	if err != nil {
		api.RespondWithError(w, fromApplicationError(err))
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) GetShare(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.InfoContext(ctx, "getting share")

	shr, err := h.app.GetShare(ctx)
	if err != nil {
		api.RespondWithError(w, fromApplicationError(err))
		return
	}

	resp, err := json.Marshal(GetShareResponse{
		Share: shr,
	})
	if err != nil {
		api.RespondWithError(w, api.ErrInternal)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp)
}
