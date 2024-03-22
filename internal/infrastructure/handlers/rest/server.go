package rest

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"go.openfort.xyz/shield/internal/applications/projectapp"
	"go.openfort.xyz/shield/internal/applications/userapp"
	"go.openfort.xyz/shield/internal/infrastructure/authenticationmgr"
	"go.openfort.xyz/shield/internal/infrastructure/handlers/rest/authmdw"
	"go.openfort.xyz/shield/internal/infrastructure/handlers/rest/projecthdl"
	"go.openfort.xyz/shield/internal/infrastructure/handlers/rest/requestmdw"
	"go.openfort.xyz/shield/internal/infrastructure/handlers/rest/responsemdw"
	"go.openfort.xyz/shield/internal/infrastructure/handlers/rest/userhdl"
	"go.openfort.xyz/shield/pkg/oflog"
)

type Server struct {
	projectApp  *projectapp.ProjectApplication
	userApp     *userapp.UserApplication
	authManager *authenticationmgr.Manager
	server      *http.Server
	logger      *slog.Logger
	config      *Config
}

func New(cfg *Config, projectApp *projectapp.ProjectApplication, userApp *userapp.UserApplication, authManager *authenticationmgr.Manager) *Server {
	return &Server{
		projectApp:  projectApp,
		userApp:     userApp,
		authManager: authManager,
		server:      new(http.Server),
		logger:      slog.New(oflog.NewContextHandler(slog.NewTextHandler(os.Stdout, nil))).WithGroup("rest_server"),
		config:      cfg,
	}
}

func (s *Server) Start(ctx context.Context) error {
	projectHdl := projecthdl.New(s.projectApp)
	userHdl := userhdl.New(s.userApp)
	authMdw := authmdw.New(s.authManager)

	r := mux.NewRouter()
	r.Use(requestmdw.RequestIDMiddleware)
	r.Use(responsemdw.ResponseMiddleware)
	r.HandleFunc("/register", projectHdl.CreateProject).Methods(http.MethodPost)
	p := r.PathPrefix("/project").Subrouter()
	p.Use(authMdw.AuthenticateAPISecret)
	p.HandleFunc("", projectHdl.GetProject).Methods(http.MethodGet)
	p.HandleFunc("/providers", projectHdl.GetProviders).Methods(http.MethodGet)
	p.HandleFunc("/providers", projectHdl.AddProviders).Methods(http.MethodPost)
	p.HandleFunc("/providers/{provider}", projectHdl.GetProvider).Methods(http.MethodGet)
	p.HandleFunc("/providers/{provider}", projectHdl.UpdateProvider).Methods(http.MethodPut)
	p.HandleFunc("/providers/{provider}", projectHdl.DeleteProvider).Methods(http.MethodDelete)

	u := r.PathPrefix("/shares").Subrouter()
	u.Use(authMdw.AuthenticateUser)
	u.HandleFunc("", userHdl.GetShare).Methods(http.MethodGet)
	u.HandleFunc("", userHdl.RegisterShare).Methods(http.MethodPost)

	s.server.Addr = fmt.Sprintf(":%d", s.config.Port)
	s.server.Handler = r

	s.logger.InfoContext(ctx, "starting server", slog.String("address", s.server.Addr))
	return s.server.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
