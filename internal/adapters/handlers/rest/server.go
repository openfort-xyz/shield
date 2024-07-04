package rest

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"go.openfort.xyz/shield/internal/adapters/authenticationmgr"
	"go.openfort.xyz/shield/internal/adapters/handlers/rest/authmdw"
	"go.openfort.xyz/shield/internal/adapters/handlers/rest/projecthdl"
	"go.openfort.xyz/shield/internal/adapters/handlers/rest/ratelimitermdw"
	"go.openfort.xyz/shield/internal/adapters/handlers/rest/requestmdw"
	"go.openfort.xyz/shield/internal/adapters/handlers/rest/responsemdw"
	"go.openfort.xyz/shield/internal/adapters/handlers/rest/sharehdl"
	"go.openfort.xyz/shield/internal/applications/projectapp"
	"go.openfort.xyz/shield/internal/applications/shareapp"
	"go.openfort.xyz/shield/pkg/logger"
)

// Server is the REST server for the shield API
type Server struct {
	projectApp  *projectapp.ProjectApplication
	shareApp    *shareapp.ShareApplication
	authManager *authenticationmgr.Manager
	server      *http.Server
	logger      *slog.Logger
	config      *Config
}

// New creates a new REST server
func New(cfg *Config, projectApp *projectapp.ProjectApplication, shareApp *shareapp.ShareApplication, authManager *authenticationmgr.Manager) *Server {
	return &Server{
		projectApp:  projectApp,
		shareApp:    shareApp,
		authManager: authManager,
		server:      new(http.Server),
		logger:      logger.New("rest_server"),
		config:      cfg,
	}
}

// Start starts the REST server
func (s *Server) Start(ctx context.Context) error {
	projectHdl := projecthdl.New(s.projectApp)
	shareHdl := sharehdl.New(s.shareApp)
	authMdw := authmdw.New(s.authManager)
	rateLimiterMdw := ratelimitermdw.New(s.config.RPS)

	r := mux.NewRouter()
	r.Use(rateLimiterMdw.RateLimitMiddleware)
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
	p.HandleFunc("/encrypt", projectHdl.EncryptProjectShares).Methods(http.MethodPost)
	p.HandleFunc("/encryption-key", projectHdl.RegisterEncryptionKey).Methods(http.MethodPost)

	u := r.PathPrefix("/shares").Subrouter()
	u.Use(authMdw.AuthenticateUser)
	u.HandleFunc("", shareHdl.GetShare).Methods(http.MethodGet)
	u.HandleFunc("", shareHdl.RegisterShare).Methods(http.MethodPost)
	u.HandleFunc("", shareHdl.DeleteShare).Methods(http.MethodDelete)

	a := r.PathPrefix("/admin").Subrouter()
	a.Use(authMdw.AuthenticateAPISecret)
	a.Use(authMdw.PreRegisterUser)
	a.HandleFunc("/preregister", shareHdl.RegisterShare).Methods(http.MethodPost)

	extraHeaders := strings.Split(s.config.CORSExtraAllowedHeaders, ",")
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowedHeaders: append([]string{
			authmdw.AccessControlAllowOriginHeader,
			authmdw.TokenHeader,
			responsemdw.ContentTypeHeader,
			authmdw.APIKeyHeader,
			authmdw.APISecretHeader,
			authmdw.AuthProviderHeader,
			authmdw.OpenfortProviderHeader,
			authmdw.OpenfortTokenTypeHeader,
			authmdw.EncryptionPartHeader,
		}, extraHeaders...),
		MaxAge: s.config.CORSMaxAge,
	}).Handler(r)

	s.server.Addr = fmt.Sprintf(":%d", s.config.Port)
	s.server.Handler = c
	s.server.ReadTimeout = s.config.ReadTimeout
	s.server.WriteTimeout = s.config.WriteTimeout
	s.server.IdleTimeout = s.config.IdleTimeout

	s.logger.InfoContext(ctx, "starting server", slog.String("address", s.server.Addr))
	return s.server.ListenAndServe()
}

// Stop stops the REST server gracefully
func (s *Server) Stop(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
