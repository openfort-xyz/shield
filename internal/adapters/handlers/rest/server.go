package rest

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"go.openfort.xyz/shield/pkg/prometheus"

	"go.openfort.xyz/shield/internal/adapters/handlers/rest/healthzhdl"
	"go.openfort.xyz/shield/internal/applications/healthzapp"

	"go.openfort.xyz/shield/internal/core/ports/factories"
	"go.openfort.xyz/shield/internal/core/ports/services"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
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
	projectApp            *projectapp.ProjectApplication
	shareApp              *shareapp.ShareApplication
	healthzApp            *healthzapp.Application
	server                *http.Server
	logger                *slog.Logger
	config                *Config
	authenticationFactory factories.AuthenticationFactory
	identityFactory       factories.IdentityFactory
	userService           services.UserService
}

// New creates a new REST server
func New(cfg *Config, projectApp *projectapp.ProjectApplication, shareApp *shareapp.ShareApplication, authenticationFactory factories.AuthenticationFactory, identityFactory factories.IdentityFactory, userService services.UserService, healthzApp *healthzapp.Application) *Server {
	return &Server{
		projectApp:            projectApp,
		shareApp:              shareApp,
		healthzApp:            healthzApp,
		server:                new(http.Server),
		logger:                logger.New("rest_server"),
		config:                cfg,
		authenticationFactory: authenticationFactory,
		identityFactory:       identityFactory,
		userService:           userService,
	}
}

// Start starts the REST server
func (s *Server) Start(ctx context.Context) error {
	healthzHdl := healthzhdl.New(s.healthzApp)
	projectHdl := projecthdl.New(s.projectApp)
	shareHdl := sharehdl.New(s.shareApp)
	authMdw := authmdw.New(s.authenticationFactory, s.identityFactory, s.userService)
	rateLimiterMdw := ratelimitermdw.New(s.config.RPS)

	r := mux.NewRouter()
	r.Use(rateLimiterMdw.RateLimitMiddleware)

	r.Handle("/metrics", prometheus.ExposeHTTP())
	r.Use(prometheus.Metrics)

	r.Use(requestmdw.RequestIDMiddleware)
	r.Use(responsemdw.ResponseMiddleware)
	r.HandleFunc("/healthz", healthzHdl.Healthz).Methods(http.MethodGet)
	r.HandleFunc("/register", projectHdl.CreateProject).Methods(http.MethodPost)
	// This endpoint only lists the available share storage methods, so it does not require authentication
	r.HandleFunc("/storage-methods", shareHdl.GetShareStorageMethods).Methods(http.MethodGet)
	p := r.PathPrefix("/project").Subrouter()
	p.Use(authMdw.AuthenticateAPISecret)
	p.HandleFunc("", projectHdl.GetProject).Methods(http.MethodGet)
	p.HandleFunc("/providers", projectHdl.GetProviders).Methods(http.MethodGet)
	p.HandleFunc("/providers", projectHdl.AddProviders).Methods(http.MethodPost)
	p.HandleFunc("/providers/{provider}", projectHdl.GetProvider).Methods(http.MethodGet)
	p.HandleFunc("/providers/{provider}", projectHdl.UpdateProvider).Methods(http.MethodPut)
	p.HandleFunc("/providers/{provider}", projectHdl.DeleteProvider).Methods(http.MethodDelete)
	p.HandleFunc("/encrypt", projectHdl.EncryptProjectShares).Methods(http.MethodPost)
	p.HandleFunc("/encryption-session", projectHdl.RegisterEncryptionSession).Methods(http.MethodPost)
	p.HandleFunc("/encryption-key", projectHdl.RegisterEncryptionKey).Methods(http.MethodPost)

	u := r.PathPrefix("/shares").Subrouter()
	u.Use(authMdw.AuthenticateUser)
	u.HandleFunc("", shareHdl.GetShare).Methods(http.MethodGet)
	u.HandleFunc("/encryption", shareHdl.GetShareEncryption).Methods(http.MethodGet)
	u.HandleFunc("", shareHdl.RegisterShare).Methods(http.MethodPost)
	u.HandleFunc("", shareHdl.DeleteShare).Methods(http.MethodDelete)
	u.HandleFunc("/{reference}", shareHdl.DeleteShare).Methods(http.MethodDelete)
	u.HandleFunc("", shareHdl.UpdateShare).Methods(http.MethodPut)
	k := r.PathPrefix("/keychain").Subrouter()
	k.Use(authMdw.AuthenticateUser)
	k.HandleFunc("", shareHdl.Keychain).Methods(http.MethodGet)

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
			authmdw.EncryptionSessionHeader,
			authmdw.RequestIDHeader,
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
