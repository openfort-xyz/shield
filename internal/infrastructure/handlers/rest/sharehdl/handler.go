package sharehdl

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"go.openfort.xyz/shield/internal/applications/shareapp"
	"go.openfort.xyz/shield/internal/infrastructure/handlers/rest/api"
	"go.openfort.xyz/shield/pkg/logger"
)

type Handler struct {
	app       *shareapp.ShareApplication
	logger    *slog.Logger
	parser    *parser
	validator *validator
}

func New(app *shareapp.ShareApplication) *Handler {
	return &Handler{
		app:       app,
		logger:    logger.New("share_handler"),
		parser:    newParser(),
		validator: newValidator(),
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

	if errV := h.validator.validateShare((*Share)(&req)); errV != nil {
		api.RespondWithError(w, errV)
		return
	}

	share := h.parser.toDomain((*Share)(&req))
	var opts []shareapp.Option
	if req.EncryptionPart != "" {
		opts = append(opts, shareapp.WithEncryptionPart(req.EncryptionPart))
	}
	err = h.app.RegisterShare(ctx, share, opts...)
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
// @Param X-Encryption-Part header string false "Encryption Part"
// @Success 200 {object} GetShareResponse "Successful response"
// @Failure 404 "Description: Not Found"
// @Failure 500 "Description: Internal Server Error"
// @Router /shares [get]
func (h *Handler) GetShare(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.InfoContext(ctx, "getting share")

	var opts []shareapp.Option
	encryptionPart := r.Header.Get(EncryptionPartHeader)
	if encryptionPart != "" {
		opts = append(opts, shareapp.WithEncryptionPart(encryptionPart))
	}

	shr, err := h.app.GetShare(ctx, opts...)
	if err != nil {
		api.RespondWithError(w, fromApplicationError(err))
		return
	}

	resp, err := json.Marshal(GetShareResponse(*h.parser.fromDomain(shr)))
	if err != nil {
		api.RespondWithError(w, api.ErrInternal)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp)
}
