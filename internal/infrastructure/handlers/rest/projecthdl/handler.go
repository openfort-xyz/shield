package projecthdl

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"go.openfort.xyz/shield/internal/applications/projectapp"
	"go.openfort.xyz/shield/pkg/oflog"
	"io"
	"log/slog"
	"net/http"
	"os"
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

func (h *Handler) CreateProject(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.InfoContext(ctx, "creating project")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var req CreateProjectRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	proj, err := h.app.CreateProject(ctx, req.Name)
	if err != nil { // TODO parse error
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp, err := json.Marshal(h.parser.toCreateProjectResponse(proj))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write(resp)
}

func (h *Handler) GetProject(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.InfoContext(ctx, "getting project")

	proj, err := h.app.GetProject(ctx)
	if err != nil { // TODO parse error
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp, err := json.Marshal(h.parser.toGetProjectResponse(proj))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func (h *Handler) AddProviders(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.InfoContext(ctx, "adding providers")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var req AddProvidersRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	providers, err := h.app.AddProviders(ctx, h.parser.fromAddProvidersRequest(&req)...)
	if err != nil { // TODO parse error
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp, err := json.Marshal(h.parser.toAddProvidersResponse(providers))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func (h *Handler) GetProviders(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.InfoContext(ctx, "getting providers")

	providers, err := h.app.GetProviders(ctx)
	if err != nil { // TODO parse error
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp, err := json.Marshal(h.parser.toGetProvidersResponse(providers))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func (h *Handler) GetProvider(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.InfoContext(ctx, "getting provider")

	providerID := mux.Vars(r)["provider"]
	if providerID == "" {
		http.Error(w, "provider id is required", http.StatusBadRequest)
		return
	}

	prov, err := h.app.GetProviderDetail(ctx, providerID)
	if err != nil { // TODO parse error
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp, err := json.Marshal(h.parser.toGetProviderResponse(prov))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func (h *Handler) UpdateProvider(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.InfoContext(ctx, "updating provider")

	providerID := mux.Vars(r)["provider"]
	if providerID == "" {
		http.Error(w, "provider id is required", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var req UpdateProviderRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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
	if err != nil { // TODO parse error
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) DeleteProvider(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.InfoContext(ctx, "deleting provider")

	providerID := mux.Vars(r)["provider"]
	if providerID == "" {
		http.Error(w, "provider id is required", http.StatusBadRequest)
		return
	}

	err := h.app.RemoveProvider(ctx, providerID)
	if err != nil { // TODO parse error
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
