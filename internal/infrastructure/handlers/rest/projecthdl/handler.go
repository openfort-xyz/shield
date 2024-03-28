package projecthdl

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"go.openfort.xyz/shield/internal/applications/projectapp"
	"go.openfort.xyz/shield/internal/infrastructure/handlers/rest/api"
	"go.openfort.xyz/shield/pkg/oflog"
)

type Handler struct {
	app    *projectapp.ProjectApplication
	logger *slog.Logger
	parser *parser
}

func New(app *projectapp.ProjectApplication) *Handler {
	return &Handler{
		app:    app,
		logger: slog.New(oflog.NewContextHandler(slog.NewTextHandler(os.Stdout, nil))).WithGroup("project_handler"),
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

	proj, err := h.app.CreateProject(ctx, req.Name)
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
		opts = append(opts, projectapp.WithCustom(req.JWK))
	}

	if req.PublishableKey != "" {
		opts = append(opts, projectapp.WithOpenfort(req.PublishableKey))
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

// AddAllowedOrigin adds an origin to the list of allowed origins
// @Summary Add allowed origin
// @Description Add an origin to the list of allowed origins for a project
// @Tags Project
// @Accept json
// @Produce json
// @Param X-API-Key header string true "API Key"
// @Param X-API-Secret header string true "API Secret"
// @Param addAllowedOriginRequest body AddAllowedOriginRequest true "Add Allowed Origin Request"
// @Success 200 "Origin added successfully"
// @Failure 400 "Bad Request"
// @Failure 500 {object} api.Error "Internal Server Error"
// @Router /project/allowed-origins [post]
func (h *Handler) AddAllowedOrigin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.InfoContext(ctx, "adding allowed origin")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		api.RespondWithError(w, api.ErrBadRequestWithMessage("failed to read request body"))
		return
	}

	var req AddAllowedOriginRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		api.RespondWithError(w, api.ErrBadRequestWithMessage("failed to parse request body"))
		return
	}

	err = h.app.AddAllowedOrigin(ctx, req.Origin)
	if err != nil {
		api.RespondWithError(w, fromApplicationError(err))
		return
	}

	w.WriteHeader(http.StatusOK)
}

// RemoveAllowedOrigin removes an origin from the list of allowed origins
// @Summary Remove allowed origin
// @Description Remove an origin from the list of allowed origins for a project
// @Tags Project
// @Param X-API-Key header string true "API Key"
// @Param X-API-Secret header string true "API Secret"
// @Param origin path string true "Origin"
// @Success 200 "Origin removed successfully"
// @Failure 404 "Origin not found"
// @Failure 500 {object} api.Error "Internal Server Error"
// @Router /project/allowed-origins/{origin} [delete]
func (h *Handler) RemoveAllowedOrigin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.InfoContext(ctx, "removing allowed origin")

	origin := mux.Vars(r)["origin"]
	if origin == "" {
		api.RespondWithError(w, api.ErrBadRequestWithMessage("missing origin"))
		return
	}

	err := h.app.RemoveAllowedOrigin(ctx, origin)
	if err != nil {
		api.RespondWithError(w, fromApplicationError(err))
		return
	}

	w.WriteHeader(http.StatusOK)
}

// GetAllowedOrigins lists all allowed origins of a project
// @Summary List allowed origins
// @Description Get a list of all allowed origins for a project
// @Tags Project
// @Produce json
// @Param X-API-Key header string true "API Key"
// @Param X-API-Secret header string true "API Secret"
// @Success 200 {object} GetAllowedOriginsResponse "Successful response"
// @Failure 500 {object} api.Error "Internal Server Error"
// @Router /project/allowed-origins [get]
func (h *Handler) GetAllowedOrigins(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.InfoContext(ctx, "getting allowed origins")

	origins, err := h.app.GetAllowedOrigins(ctx)
	if err != nil {
		api.RespondWithError(w, fromApplicationError(err))
		return
	}

	resp, err := json.Marshal(h.parser.toGetAllowedOriginsResponse(origins))
	if err != nil {
		api.RespondWithError(w, api.ErrInternal)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp)
}
