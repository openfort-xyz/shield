package usrhdl

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/openfort-xyz/shield/internal/adapters/handlers/rest/api"
	"github.com/openfort-xyz/shield/internal/core/ports/services"
	"github.com/openfort-xyz/shield/pkg/contexter"
	"github.com/openfort-xyz/shield/pkg/logger"
)

type Handler struct {
	userService services.UserService
	logger      *slog.Logger
}

func New(userService services.UserService) *Handler {
	return &Handler{
		userService: userService,
		logger:      logger.New("user_handler"),
	}
}

type CreateUserRequest struct {
	ExternalUserID string `json:"external_user_id"`
	ProviderID     string `json:"provider_id"`
}

type CreateUserResponse struct {
	UserID string `json:"user_id"`
}

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.InfoContext(ctx, "creating user")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		api.RespondWithError(w, api.ErrBadRequestWithMessage("failed to read request body"))
		return
	}

	var req CreateUserRequest
	if err := json.Unmarshal(body, &req); err != nil {
		api.RespondWithError(w, api.ErrBadRequestWithMessage("failed to parse request body"))
		return
	}

	if req.ExternalUserID == "" {
		api.RespondWithError(w, api.ErrBadRequestWithMessage("external_user_id is required"))
		return
	}

	if req.ProviderID == "" {
		api.RespondWithError(w, api.ErrBadRequestWithMessage("provider_id is required"))
		return
	}

	projectID := contexter.GetProjectID(ctx)

	usr, err := h.userService.GetOrCreate(ctx, projectID, req.ExternalUserID, req.ProviderID)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to create user", slog.String("error", err.Error()))
		api.RespondWithError(w, api.ErrInternal)
		return
	}

	resp, err := json.Marshal(CreateUserResponse{UserID: usr.ID})
	if err != nil {
		api.RespondWithError(w, api.ErrInternal)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write(resp)
}
