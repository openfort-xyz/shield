package projecthdl

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/openfort-xyz/shield/internal/adapters/handlers/rest/api"
	"github.com/openfort-xyz/shield/internal/applications/projectapp"
	"github.com/openfort-xyz/shield/pkg/logger"
)

// Handler is the REST handler for project operations
type Handler struct {
	app    *projectapp.ProjectApplication
	logger *slog.Logger
	parser *parser
}

// New creates a new project handler
func New(app *projectapp.ProjectApplication) *Handler {
	return &Handler{
		app:    app,
		logger: logger.New("project_handler"),
		parser: newParser(),
	}
}

// CreateProject creates a new project
// @Summary Create a new project
// @Description Create a new project with the provided name
// @Tags Project
// @Accept json
// @Produce json
// @Param createProjectRequest body CreateProjectRequest true "Create Project Request"
// @Success 201 {object} CreateProjectResponse "Project created successfully"
// @Failure 400 {object} api.Error "Bad Request"
// @Failure 500 {object} api.Error "Internal Server Error"
// @Router /register [post]
func (h *Handler) CreateProject(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.InfoContext(ctx, "creating project")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		api.RespondWithError(w, api.ErrBadRequestWithMessage("failed to read request body"))
		return
	}

	var req CreateProjectRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		api.RespondWithError(w, api.ErrBadRequestWithMessage("failed to parse request body"))
		return
	}

	var opts []projectapp.ProjectOption
	if req.GenerateEncryptionKey {
		opts = append(opts, projectapp.WithEncryptionKey())
	}

	enable2fa := false
	if req.Enable2FA != nil {
		enable2fa = *req.Enable2FA
	}

	proj, err := h.app.CreateProject(ctx, req.Name, enable2fa, opts...)
	if err != nil {
		api.RespondWithError(w, fromApplicationError(err))
		return
	}

	// gosec G117: api_key/api_secret are intentionally returned to the caller on project creation.
	resp, err := json.Marshal(h.parser.toCreateProjectResponse(proj)) //nolint:gosec
	if err != nil {
		api.RespondWithError(w, api.ErrInternal)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write(resp)
}

// GetProject retrieves a project
// @Summary Get a project
// @Description Get details of a project
// @Tags Project
// @Produce json
// @Param X-API-Key header string true "API Key"
// @Param X-API-Secret header string true "API Secret"
// @Success 200 {object} GetProjectResponse "Successful response"
// @Failure 404 {object} api.Error "Not Found"
// @Failure 500 {object} api.Error "Internal Server Error"
// @Router /project [get]
func (h *Handler) GetProject(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.InfoContext(ctx, "getting project")

	proj, err := h.app.GetProject(ctx)
	if err != nil {
		api.RespondWithError(w, fromApplicationError(err))
		return
	}

	resp, err := json.Marshal(h.parser.toGetProjectResponse(proj))
	if err != nil {
		api.RespondWithError(w, api.ErrInternal)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp)
}

// DeleteProject permanently deletes the authenticated project and all its data
// @Summary Delete project
// @Description Irreversibly delete the project and ALL its data: providers, users, shares, keychains, encryption parts, notifications and rate limits. The project's API key and secret stop working immediately.
// @Tags Project
// @Param X-API-Key header string true "API Key"
// @Param X-API-Secret header string true "API Secret"
// @Success 204 "Project deleted successfully"
// @Failure 500 {object} api.Error "Internal Server Error"
// @Router /project [delete]
func (h *Handler) DeleteProject(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.InfoContext(ctx, "deleting project")

	err := h.app.DeleteProject(ctx)
	if err != nil {
		api.RespondWithError(w, fromApplicationError(err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// AddProviders adds providers to a project
// @Summary Add providers
// @Description Add one or more providers to a project
// @Tags Project
// @Accept json
// @Produce json
// @Param X-API-Key header string true "API Key"
// @Param X-API-Secret header string true "API Secret"
// @Param addProvidersRequest body AddProvidersRequest true "Add Providers Request"
// @Success 200 {object} AddProvidersResponse "Providers added successfully"
// @Failure 400 "Bad Request"
// @Failure 500 {object} api.Error "Internal Server Error"
// @Router /project/providers [post]
func (h *Handler) AddProviders(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.InfoContext(ctx, "adding providers")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		api.RespondWithError(w, api.ErrBadRequestWithMessage("failed to read request body"))
		return
	}

	var req AddProvidersRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		api.RespondWithError(w, api.ErrBadRequestWithMessage("failed to parse request body"))
		return
	}

	providers, err := h.app.AddProviders(ctx, h.parser.fromAddProvidersRequest(&req)...)
	if err != nil {
		api.RespondWithError(w, fromApplicationError(err))
		return
	}

	resp, err := json.Marshal(h.parser.toAddProvidersResponse(providers))
	if err != nil {
		api.RespondWithError(w, api.ErrInternal)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp)
}

func (h *Handler) RequestOTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.InfoContext(ctx, "generating otp for user")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		api.RespondWithError(w, api.ErrBadRequestWithMessage("failed to read request body"))
		return
	}

	var req GenerateOTPRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		api.RespondWithError(w, api.ErrBadRequestWithMessage("failed to parse request body"))
		return
	}

	if !req.ParametersValid() {
		api.RespondWithError(w, api.ErrBadRequestWithMessage("invalid parameters were passed"))
		return
	}

	err = h.app.GenerateOTP(ctx, req.UserID, req.DangerouslySkipVerification, req.Email, req.Phone)
	if err != nil {
		api.RespondWithError(w, fromApplicationError(err))
		return
	}

	w.WriteHeader(http.StatusOK)
}

// GetProviders lists all providers of a project
// @Summary List providers
// @Description Get a list of all providers associated with a project
// @Tags Project
// @Produce json
// @Param X-API-Key header string true "API Key"
// @Param X-API-Secret header string true "API Secret"
// @Success 200 {object} GetProvidersResponse "Successful response"
// @Failure 500 {object} api.Error "Internal Server Error"
// @Router /project/providers [get]
func (h *Handler) GetProviders(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.InfoContext(ctx, "getting providers")

	providers, err := h.app.GetProviders(ctx)
	if err != nil {
		api.RespondWithError(w, fromApplicationError(err))
		return
	}

	resp, err := json.Marshal(h.parser.toGetProvidersResponse(providers))
	if err != nil {
		api.RespondWithError(w, api.ErrInternal)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp)
}

// GetProvider retrieves a specific provider
// @Summary Get a provider
// @Description Get details of a specific provider
// @Tags Project
// @Produce json
// @Param X-API-Key header string true "API Key"
// @Param X-API-Secret header string true "API Secret"
// @Param provider path string true "Provider ID"
// @Success 200 {object} GetProviderResponse "Successful response"
// @Failure 404 "Provider not found"
// @Failure 500 {object} api.Error "Internal Server Error"
// @Router /project/providers/{provider} [get]
func (h *Handler) GetProvider(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.InfoContext(ctx, "getting provider")

	providerID := mux.Vars(r)["provider"]
	if providerID == "" {
		api.RespondWithError(w, api.ErrMissingProvider)
		return
	}

	prov, err := h.app.GetProviderDetail(ctx, providerID)
	if err != nil {
		api.RespondWithError(w, fromApplicationError(err))
		return
	}

	resp, err := json.Marshal(h.parser.toGetProviderResponse(prov))
	if err != nil {
		api.RespondWithError(w, api.ErrInternal)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp)
}

// UpdateProvider updates a provider's configuration
// @Summary Update a provider
// @Description Update the configuration of a specific provider
// @Tags Project
// @Accept json
// @Param X-API-Key header string true "API Key"
// @Param X-API-Secret header string true "API Secret"
// @Param provider path string true "Provider ID"
// @Param updateProviderRequest body UpdateProviderRequest true "Update Provider Request"
// @Success 200 "Provider updated successfully"
// @Failure 400 "Bad Request"
// @Failure 500 {object} api.Error "Internal Server Error"
// @Router /project/providers/{provider} [put]
func (h *Handler) UpdateProvider(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.InfoContext(ctx, "updating provider")

	providerID := mux.Vars(r)["provider"]
	if providerID == "" {
		api.RespondWithError(w, api.ErrMissingProvider)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		api.RespondWithError(w, api.ErrBadRequestWithMessage("failed to read request body"))
		return
	}

	var req UpdateProviderRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		api.RespondWithError(w, api.ErrBadRequestWithMessage("failed to parse request body"))
		return
	}

	var opts []projectapp.ProviderOption
	if req.JWK != "" {
		opts = append(opts, projectapp.WithCustomJWK(req.JWK))
	}

	if req.PublishableKey != "" {
		opts = append(opts, projectapp.WithOpenfort(req.PublishableKey))
	}

	if req.PEM != "" {
		opts = append(opts, projectapp.WithCustomPEM(req.PEM, h.parser.mapKeyTypeToDomain[req.KeyType]))
	}

	if req.CookieFieldName != nil {
		opts = append(opts, projectapp.WithCustomCookieFieldName(*req.CookieFieldName))
	}

	err = h.app.UpdateProvider(ctx, providerID, opts...)
	if err != nil {
		api.RespondWithError(w, fromApplicationError(err))
		return
	}

	w.WriteHeader(http.StatusOK)
}

// DeleteProvider removes a provider from a project
// @Summary Delete a provider
// @Description Remove a specific provider from a project
// @Tags Project
// @Param X-API-Key header string true "API Key"
// @Param X-API-Secret header string true "API Secret"
// @Param provider path string true "Provider ID"
// @Success 200 "Provider deleted successfully"
// @Failure 404 "Provider not found"
// @Failure 500 {object} api.Error "Internal Server Error"
// @Router /project/providers/{provider} [delete]
func (h *Handler) DeleteProvider(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.InfoContext(ctx, "deleting provider")

	providerID := mux.Vars(r)["provider"]
	if providerID == "" {
		api.RespondWithError(w, api.ErrMissingProvider)
		return
	}

	err := h.app.RemoveProvider(ctx, providerID)
	if err != nil {
		api.RespondWithError(w, fromApplicationError(err))
		return
	}

	w.WriteHeader(http.StatusOK)
}

// EncryptProjectShares encrypts all shares of a project (if not already encrypted)
// @Summary Encrypt project shares
// @Description Encrypt all shares of a project
// @Tags Project
// @Param X-API-Key header string true "API Key"
// @Param X-API-Secret header string true "API Secret"
// @Param encryptBodyRequest body EncryptBodyRequest true "Add Allowed Origin Request"
// @Success 200 "Shares encrypted successfully"
// @Failure 500 {object} api.Error "Internal Server Error"
// @Router /project/encrypt [post]
func (h *Handler) EncryptProjectShares(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.InfoContext(ctx, "encrypting project shares")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		api.RespondWithError(w, api.ErrBadRequestWithMessage("failed to read request body"))
		return
	}

	var req EncryptBodyRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		api.RespondWithError(w, api.ErrBadRequestWithMessage("failed to parse request body"))
		return
	}

	err = h.app.EncryptProjectShares(ctx, req.EncryptionPart)
	if err != nil {
		api.RespondWithError(w, fromApplicationError(err))
		return
	}

	w.WriteHeader(http.StatusOK)
}

// RegisterEncryptionSession registers a session with a one-time encryption key for a project
// @Summary Register encryption session
// @Description Register a session with a one-time encryption key for a project
// @Tags Project
// @Accept json
// @Produce json
// @Param X-API-Key header string true "API Key"
// @Param X-API-Secret header string true "API Secret"
// @Param registerEncryptionSessionRequest body RegisterEncryptionSessionRequest true "Add Allowed Origin Request"
// @Success 200 {object} RegisterEncryptionSessionResponse "Encryption session registered successfully"
// @Failure 400 "Bad Request"
// @Failure 500 {object} api.Error "Internal Server Error"
// @Router /project/encryption-session [post]
func (h *Handler) RegisterEncryptionSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.InfoContext(ctx, "registering encryption session")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		api.RespondWithError(w, api.ErrBadRequestWithMessage("failed to read request body"))
		return
	}

	var req RegisterEncryptionSessionRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		api.RespondWithError(w, api.ErrBadRequestWithMessage("failed to parse request body"))
		return
	}

	sessionID, err := h.app.RegisterEncryptionSession(ctx, req.EncryptionPart, req.UserID, req.OTPCode)
	if err != nil {
		api.RespondWithError(w, fromApplicationError(err))
		return
	}

	resp, err := json.Marshal(RegisterEncryptionSessionResponse{SessionID: sessionID})
	if err != nil {
		api.RespondWithError(w, api.ErrInternal)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp)
}

// RegisterEncryptionKey registers an encryption key for a project
// @Summary Register encryption key
// @Description Register an encryption key for a project
// @Tags Project
// @Accept json
// @Produce json
// @Param X-API-Key header string true "API Key"
// @Param X-API-Secret header string true "API Secret"
// @Success 200 {object} RegisterEncryptionKeyResponse "Encryption key registered successfully"
// @Failure 400 "Bad Request"
// @Failure 500 {object} api.Error "Internal Server Error"
// @Router /project/encryption-key [post]
func (h *Handler) RegisterEncryptionKey(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.InfoContext(ctx, "registering encryption key")

	part, err := h.app.RegisterEncryptionKey(ctx)
	if err != nil {
		api.RespondWithError(w, fromApplicationError(err))
		return
	}

	resp, err := json.Marshal(RegisterEncryptionKeyResponse{EncryptionPart: part})
	if err != nil {
		api.RespondWithError(w, api.ErrInternal)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp)
}

// Enable2FA enables 2FA for a project
// @Summary Enable 2FA
// @Description Enable two-factor authentication for a project. Once enabled, 2FA cannot be disabled.
// @Tags Project
// @Produce json
// @Param X-API-Key header string true "API Key"
// @Param X-API-Secret header string true "API Secret"
// @Success 200 "2FA enabled successfully"
// @Failure 409 {object} api.Error "2FA already enabled"
// @Failure 500 {object} api.Error "Internal Server Error"
// @Router /project/enable-2fa [post]
func (h *Handler) Enable2FA(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.InfoContext(ctx, "enabling 2FA")

	err := h.app.Enable2FA(ctx)
	if err != nil {
		api.RespondWithError(w, fromApplicationError(err))
		return
	}

	w.WriteHeader(http.StatusOK)
}

// PrepareEncryptionKeyRotation generates a replacement encryption key
// @Summary Prepare encryption key rotation
// @Description Verify the current encryption part and generate a replacement key. Returns the project's half of the new key; nothing changes until commit. Preparing again replaces the pending key.
// @Tags Project
// @Accept json
// @Produce json
// @Param X-API-Key header string true "API Key"
// @Param X-API-Secret header string true "API Secret"
// @Param request body RotateEncryptionKeyPrepareRequest true "Current encryption part"
// @Success 200 {object} RotateEncryptionKeyPrepareResponse "New encryption part"
// @Failure 400 {object} api.Error "Bad Request or invalid encryption part"
// @Failure 409 {object} api.Error "Encryption not configured or key not migrated"
// @Failure 500 {object} api.Error "Internal Server Error"
// @Router /project/rotate-encryption-key/prepare [post]
func (h *Handler) PrepareEncryptionKeyRotation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.InfoContext(ctx, "preparing encryption key rotation")

	var req RotateEncryptionKeyPrepareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, api.ErrBadRequestWithMessage("failed to parse request body"))
		return
	}
	if req.EncryptionPart == "" {
		api.RespondWithError(w, api.ErrBadRequestWithMessage("encryption_part is required"))
		return
	}

	newPart, err := h.app.PrepareEncryptionKeyRotation(ctx, req.EncryptionPart)
	if err != nil {
		api.RespondWithError(w, fromApplicationError(err))
		return
	}

	h.respondJSON(w, RotateEncryptionKeyPrepareResponse{EncryptionPart: newPart})
}

// CommitEncryptionKeyRotation re-encrypts the project's shares under the prepared key
// @Summary Commit encryption key rotation
// @Description Re-encrypt every project-entropy share under the prepared key and make it current. Safe to retry: a retry after a successful commit returns already_committed.
// @Tags Project
// @Accept json
// @Produce json
// @Param X-API-Key header string true "API Key"
// @Param X-API-Secret header string true "API Secret"
// @Param request body RotateEncryptionKeyCommitRequest true "Current and new encryption parts"
// @Success 200 {object} RotateEncryptionKeyCommitResponse "committed or already_committed"
// @Failure 400 {object} api.Error "Bad Request or invalid encryption part"
// @Failure 409 {object} api.Error "No pending rotation, rotation expired, encryption not configured or key not migrated"
// @Failure 500 {object} api.Error "Internal Server Error"
// @Router /project/rotate-encryption-key/commit [post]
func (h *Handler) CommitEncryptionKeyRotation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.InfoContext(ctx, "committing encryption key rotation")

	var req RotateEncryptionKeyCommitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, api.ErrBadRequestWithMessage("failed to parse request body"))
		return
	}
	if req.EncryptionPart == "" || req.NewEncryptionPart == "" {
		api.RespondWithError(w, api.ErrBadRequestWithMessage("encryption_part and new_encryption_part are required"))
		return
	}

	committed, err := h.app.CommitEncryptionKeyRotation(ctx, req.EncryptionPart, req.NewEncryptionPart)
	if err != nil {
		api.RespondWithError(w, fromApplicationError(err))
		return
	}

	status := RotationAlreadyCommitted
	if committed {
		status = RotationCommitted
	}
	h.respondJSON(w, RotateEncryptionKeyCommitResponse{Status: status})
}

func (h *Handler) respondJSON(w http.ResponseWriter, body any) {
	resp, err := json.Marshal(body)
	if err != nil {
		api.RespondWithError(w, api.ErrInternal)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp)
}

// PrepareAPIKeyRotation generates a pending API key
// @Summary Prepare API key rotation
// @Description Generate a pending API key. The current key keeps working until the rotation is committed with the pending key. Preparing again replaces the pending key.
// @Tags Project
// @Produce json
// @Param X-API-Key header string true "API Key"
// @Param X-API-Secret header string true "API Secret"
// @Success 200 {object} RotateAPIKeyPrepareResponse "Pending API key"
// @Failure 401 {object} api.Error "Invalid API credentials"
// @Failure 500 {object} api.Error "Internal Server Error"
// @Router /project/rotate-api-key/prepare [post]
func (h *Handler) PrepareAPIKeyRotation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.InfoContext(ctx, "preparing API key rotation")

	apiKey, err := h.app.PrepareAPIKeyRotation(ctx)
	if err != nil {
		api.RespondWithError(w, fromApplicationError(err))
		return
	}

	h.respondJSON(w, RotateAPIKeyPrepareResponse{APIKey: apiKey})
}

// CommitAPIKeyRotation makes the pending API key current
// @Summary Commit API key rotation
// @Description Authenticate with the pending API key from prepare (and the API secret) to make it current; the previous key stops working. Safe to retry: a retry with the now-current key returns status current.
// @Tags Project
// @Produce json
// @Param X-API-Key header string true "Pending API Key"
// @Param X-API-Secret header string true "API Secret"
// @Success 200 {object} RotateAPIKeyCommitResponse "committed or current"
// @Failure 401 {object} api.Error "Invalid API credentials"
// @Failure 409 {object} api.Error "Not authenticated with the pending key, rotation expired, or no pending rotation"
// @Failure 500 {object} api.Error "Internal Server Error"
// @Router /project/rotate-api-key/commit [post]
func (h *Handler) CommitAPIKeyRotation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.InfoContext(ctx, "committing API key rotation")

	apiKey, committed, err := h.app.CommitAPIKeyRotation(ctx)
	if err != nil {
		api.RespondWithError(w, fromApplicationError(err))
		return
	}

	status := RotationCurrent
	if committed {
		status = RotationCommitted
	}
	h.respondJSON(w, RotateAPIKeyCommitResponse{APIKey: apiKey, Status: status})
}
