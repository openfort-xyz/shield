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

// RegisterShare registers a new share
// @Summary Register new share
// @Description Register a new share for the user
// @Tags Share
// @Accept json
// @Produce json
// @Param X-API-Key header string true "API Key"
// @Param Authorization header string true "Bearer token"
// @Param X-Auth-Provider header string true "Auth Provider"
// @Param X-Openfort-Provider header string false "Openfort Provider"
// @Param X-Openfort-Token-Type header string false "Openfort Token Type"
// @Param registerShareRequest body RegisterShareRequest true "Register Share Request"
// @Success 201 "Description: Share registered successfully"
// @Failure 400 {object} api.Error "Bad Request"
// @Failure 404 {object} api.Error "Not Found"
// @Failure 500 {object} api.Error "Internal Server Error"
// @Router /shares/register [post]
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

	if req.Secret == "" {
		api.RespondWithError(w, api.ErrBadRequestWithMessage("secret is required"))
		return
	}

	var parameters *userapp.EncryptionParameters
	if req.Salt != "" || req.Iterations != 0 || req.Length != 0 || req.Digest != "" {
		parameters = &userapp.EncryptionParameters{
			Salt:       req.Salt,
			Iterations: req.Iterations,
			Length:     req.Length,
			Digest:     req.Digest,
		}
	}

	err = h.app.RegisterShare(ctx, req.Secret, req.UserEntropy, parameters)
	if err != nil {
		api.RespondWithError(w, fromApplicationError(err))
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// GetShare gets a share
// @Summary Get share
// @Description Get a share for the user
// @Tags Share
// @Accept json
// @Produce json
// @Param X-API-Key header string true "API Key"
// @Param Authorization header string true "Bearer token"
// @Param X-Auth-Provider header string true "Auth Provider"
// @Param X-Openfort-Provider header string false "Openfort Provider"
// @Param X-Openfort-Token-Type header string false "Openfort Token Type"
// @Success 200 {object} GetShareResponse "Successful response"
// @Failure 404 "Description: Not Found"
// @Failure 500 "Description: Internal Server Error"
// @Router /shares [get]
func (h *Handler) GetShare(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.InfoContext(ctx, "getting share")

	shr, err := h.app.GetShare(ctx)
	if err != nil {
		api.RespondWithError(w, fromApplicationError(err))
		return
	}

	resp, err := json.Marshal(GetShareResponse{
		Secret:      shr.Data,
		UserEntropy: shr.UserEntropy,
		Salt:        shr.Salt,
		Iterations:  shr.Iterations,
		Length:      shr.Length,
		Digest:      shr.Digest,
	})
	if err != nil {
		api.RespondWithError(w, api.ErrInternal)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp)
}
