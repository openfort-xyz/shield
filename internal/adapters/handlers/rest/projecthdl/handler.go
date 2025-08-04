package projecthdl

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/gorilla/mux"
	"go.openfort.xyz/shield/internal/adapters/handlers/rest/api"
	"go.openfort.xyz/shield/internal/applications/projectapp"
	"go.openfort.xyz/shield/pkg/logger"
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

	proj, err := h.app.CreateProject(ctx, req.Name, opts...)
	if err != nil {
		api.RespondWithError(w, fromApplicationError(err))
		return
	}

	resp, err := json.Marshal(h.parser.toCreateProjectResponse(proj))
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

	sessionID, err := h.app.RegisterEncryptionSession(ctx, req.EncryptionPart)
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

// TODO: Add documentation as soon as we have requests and responses specified

// AddProvider adds provider to a project
// @Summary Add provider
// @Description Add authentication provider to a prject
// @Tags Project
// @Accept json
// @Produce json
// @Param X-API-Key header string true "API Key"
// @Param X-API-Secret header string true "API Secret"
// @Param addProviderV2Request body AddProviderV2Request true "Add Provider v2 Request"
// @Success 200 {object} AddProviderV2Response "Provider added successfully"
// @Failure 400 "Bad Request"
// @Failure 500 {object} api.Error "Internal Server Error"
// @Router /project/v2/providers [post]
func (h *Handler) AddProviderV2(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.InfoContext(ctx, "adding provider")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		api.RespondWithError(w, api.ErrBadRequestWithMessage("failed to read request body"))
		return
	}

	var req AddProviderV2Request
	err = json.Unmarshal(body, &req)
	if err != nil {
		api.RespondWithError(w, api.ErrBadRequestWithMessage("failed to parse request body"))
		return
	}

	// fromAddProviderV2Request returns an option set identical to that of v1
	providers, err := h.app.AddProviders(ctx, h.parser.fromAddProviderV2Request(&req)...)
	if err != nil {
		api.RespondWithError(w, fromApplicationError(err))
		return
	}

	if len(providers) != 1 {
		api.RespondWithError(w, api.ErrInternal)
		return
	}

	// Return the provider's ID
	// V2 only allows the user to register a single provider
	resp, err := json.Marshal(h.parser.toAddProviderV2Response(providers[0]))
	if err != nil {
		api.RespondWithError(w, api.ErrInternal)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp)
}

// GetProvider retrieves the project's auth provider
// @Summary Get the project's auth provider
// @Description Get details of the project's auth provider
// @Tags Project
// @Produce json
// @Param X-API-Key header string true "API Key"
// @Param X-API-Secret header string true "API Secret"
// @Param provider path string true "Provider ID"
// @Success 200 {object} GetProviderV2Response "Successful response"
// @Failure 404 "Provider not found"
// @Failure 500 {object} api.Error "Internal Server Error"
// @Router /project/v2/providers/{provider} [get]
func (h *Handler) GetProviderV2(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.InfoContext(ctx, "getting provider")

	providers, err := h.app.GetProviders(ctx)

	if err != nil {
		api.RespondWithError(w, fromApplicationError(err))
	}

	if len(providers) == 0 {
		// This requires some justification I feel:
		// if we're here it means that the project hasn't set its auth provider yet
		// 404 could be misleading (couldn't we find the already existing provider?
		// is the project what's missing?)
		// And similar arguments could be held for other http codes
		// 200 OK + empty response translates quite well: everything went OK but there's
		// nothing for us to return
		return
	}

	providerID := providers[0].ID

	// Same as in v1 in terms of domain layer
	prov, err := h.app.GetProviderDetail(ctx, providerID)
	if err != nil {
		api.RespondWithError(w, fromApplicationError(err))
		return
	}

	// V2Response returns the same as a Custom Auth response but omitting the auth type
	// (the whole point of V2 providers is hiding the existence of Openfort AP to users)
	resp, err := json.Marshal(h.parser.toGetProviderV2Response(prov))
	if err != nil {
		api.RespondWithError(w, api.ErrInternal)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp)
}

func (h *Handler) UpdateProviderV2(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.InfoContext(ctx, "updating provider")

	providers, err := h.app.GetProviders(ctx)

	if err != nil {
		api.RespondWithError(w, fromApplicationError(err))
	}

	if len(providers) == 0 {
		// Here it's different: this endpoint UPDATES (doesn't UPSERT) auth providers
		// and we cannot update what doesn't exist in the first place
		api.RespondWithError(w, api.ErrMissingAuthProvider)
		return
	}

	providerID := providers[0].ID

	body, err := io.ReadAll(r.Body)
	if err != nil {
		api.RespondWithError(w, api.ErrBadRequestWithMessage("failed to read request body"))
		return
	}

	var req UpdateProviderV2Request
	err = json.Unmarshal(body, &req)
	if err != nil {
		api.RespondWithError(w, api.ErrBadRequestWithMessage("failed to parse request body"))
		return
	}

	var opts []projectapp.ProviderOption
	if req.JWK != "" {
		opts = append(opts, projectapp.WithCustomJWK(req.JWK))
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

func (h *Handler) DeleteProviderV2(w http.ResponseWriter, r *http.Request) {
	api.RespondWithError(w, api.ErrNotImplemented)
}
