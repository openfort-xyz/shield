package rest

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/openfort-xyz/shield/internal/adapters/handlers/rest/healthzhdl"
	"github.com/openfort-xyz/shield/internal/applications/healthzapp"

	"github.com/gorilla/mux"
	metrics "github.com/openfort-xyz/metrics"
	"github.com/openfort-xyz/shield/internal/adapters/handlers/rest/authmdw"
	"github.com/openfort-xyz/shield/internal/adapters/handlers/rest/projecthdl"
	"github.com/openfort-xyz/shield/internal/adapters/handlers/rest/ratelimitermdw"
	"github.com/openfort-xyz/shield/internal/adapters/handlers/rest/requestmdw"
	"github.com/openfort-xyz/shield/internal/adapters/handlers/rest/responsemdw"
	"github.com/openfort-xyz/shield/internal/adapters/handlers/rest/sharehdl"
	"github.com/openfort-xyz/shield/internal/adapters/handlers/rest/tracingmdw"
	"github.com/openfort-xyz/shield/internal/adapters/handlers/rest/usrhdl"
	"github.com/openfort-xyz/shield/internal/applications/projectapp"
	"github.com/openfort-xyz/shield/internal/applications/shareapp"
	"github.com/openfort-xyz/shield/internal/core/ports/factories"
	"github.com/openfort-xyz/shield/internal/core/ports/services"
	"github.com/openfort-xyz/shield/pkg/logger"
	"github.com/rs/cors"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
)

// Server is the REST server for the shield API
type Server struct {
	projectApp            *projectapp.ProjectApplication
	shareApp              *shareapp.ShareApplication
	healthzApp            *healthzapp.Application
	server                *http.Server
	metricsServer         *metrics.Server
	logger                *slog.Logger
	config                *Config
	authenticationFactory factories.AuthenticationFactory
	identityFactory       factories.IdentityFactory
	userService           services.UserService
	projectService        services.ProjectService
}

// New creates a new REST server
func New(cfg *Config,
	projectApp *projectapp.ProjectApplication,
	shareApp *shareapp.ShareApplication,
	authenticationFactory factories.AuthenticationFactory,
	identityFactory factories.IdentityFactory,
	userService services.UserService,
	healthzApp *healthzapp.Application,
	projectService services.ProjectService) *Server {
	return &Server{
		projectApp:            projectApp,
		shareApp:              shareApp,
		healthzApp:            healthzApp,
		server:                new(http.Server),
		metricsServer:         metrics.NewServer(cfg.MetricsPort),
		logger:                logger.New("rest_server"),
		config:                cfg,
		authenticationFactory: authenticationFactory,
		identityFactory:       identityFactory,
		userService:           userService,
		projectService:        projectService,
	}
}

// Start starts the REST server
func (s *Server) Start(ctx context.Context) error {
	healthzHdl := healthzhdl.New(s.healthzApp)
	projectHdl := projecthdl.New(s.projectApp)
	shareHdl := sharehdl.New(s.shareApp)
	userHdl := usrhdl.New(s.userService)
	authMdw := authmdw.New(s.authenticationFactory, s.identityFactory, s.userService, s.projectService)
	rateLimiterMdw := ratelimitermdw.New(s.config.RPS)

	r := mux.NewRouter()
	// Tracing first so the span wraps rate-limit/metrics/request-id/response work,
	// and so trace context is extracted from incoming W3C headers before anything
	// downstream reads the request. FlowNameMiddleware runs immediately after
	// so the rename lands on the otelmux-created span.
	r.Use(otelmux.Middleware("shield"))
	r.Use(tracingmdw.FlowNameMiddleware)
	r.Use(rateLimiterMdw.RateLimitMiddleware)

	r.Use(metrics.HTTPMiddleware)

	r.Use(requestmdw.RequestIDMiddleware)
	r.Use(responsemdw.ResponseMiddleware)
	r.HandleFunc("/healthz", healthzHdl.Healthz).Methods(http.MethodGet)
	r.HandleFunc("/register", projectHdl.CreateProject).Methods(http.MethodPost)
	// This endpoint only lists the available share storage methods, so it does not require authentication
	r.HandleFunc("/storage-methods", shareHdl.GetShareStorageMethods).Methods(http.MethodGet)
	p := r.PathPrefix("/project").Subrouter()
	p.Use(authMdw.AuthenticateAPISecret)
	p.HandleFunc("", projectHdl.GetProject).Methods(http.MethodGet)
	p.HandleFunc("/reset-api-secret", projectHdl.ResetAPISecret).Methods(http.MethodPost)
	p.HandleFunc("/otp", projectHdl.RequestOTP).Methods(http.MethodPost)
	p.HandleFunc("/providers", projectHdl.GetProviders).Methods(http.MethodGet)
	p.HandleFunc("/providers", projectHdl.AddProviders).Methods(http.MethodPost)
	p.HandleFunc("/providers/{provider}", projectHdl.GetProvider).Methods(http.MethodGet)
	p.HandleFunc("/providers/{provider}", projectHdl.UpdateProvider).Methods(http.MethodPut)
	p.HandleFunc("/providers/{provider}", projectHdl.DeleteProvider).Methods(http.MethodDelete)
	p.HandleFunc("/encrypt", projectHdl.EncryptProjectShares).Methods(http.MethodPost)
	p.HandleFunc("/encryption-session", projectHdl.RegisterEncryptionSession).Methods(http.MethodPost)
	p.HandleFunc("/encryption-key", projectHdl.RegisterEncryptionKey).Methods(http.MethodPost)
	p.HandleFunc("/enable-2fa", projectHdl.Enable2FA).Methods(http.MethodPost)

	usr := r.PathPrefix("/user").Subrouter()
	usr.Use(authMdw.AuthenticateAPISecret)
	usr.HandleFunc("", userHdl.CreateUser).Methods(http.MethodPost)

	u := r.PathPrefix("/shares").Subrouter()
	u.Use(authMdw.AuthenticateUser)
	u.HandleFunc("", shareHdl.GetShare).Methods(http.MethodGet)
	u.HandleFunc("/{reference}", shareHdl.GetShareByReference).Methods(http.MethodGet)

	u.HandleFunc("", shareHdl.RegisterShare).Methods(http.MethodPost)
	u.HandleFunc("", shareHdl.DeleteShare).Methods(http.MethodDelete)
	u.HandleFunc("/{reference}", shareHdl.DeleteShare).Methods(http.MethodDelete)
	u.HandleFunc("", shareHdl.UpdateShare).Methods(http.MethodPut)
	k := r.PathPrefix("/keychain").Subrouter()
	k.Use(authMdw.AuthenticateUser)
	k.HandleFunc("", shareHdl.Keychain).Methods(http.MethodGet)

	e := r.PathPrefix("/shares/encryption").Subrouter()
	e.Use(authMdw.AuthenticateAPISecret)
	e.HandleFunc("", shareHdl.GetShareEncryption).Methods(http.MethodGet)
	e.HandleFunc("/reference/bulk", shareHdl.GetSharesEncryptionForReferences).Methods(http.MethodPost)
	e.HandleFunc("/user/bulk", shareHdl.GetSharesEncryptionForUsers).Methods(http.MethodPost)

	m := r.PathPrefix("/shares/migration").Subrouter()
	m.Use(authMdw.AuthenticateAPISecret)
	m.HandleFunc("/export/{reference}", shareHdl.ExportShare).Methods(http.MethodGet)
	m.HandleFunc("/import", shareHdl.ImportShare).Methods(http.MethodPost)

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
			// W3C Trace Context — sent by the iFrame so shield-side spans
			// join the same trace as the api/castle path of the flow.
			"traceparent",
			// Human-readable flow name (e.g. "embedded.create") used to
			// rename the server root span — see tracingmdw.FlowNameMiddleware.
			tracingmdw.FlowNameHeader,
			// Flow attributes attached to the iframe-root span by the api.
			// Shield doesn't consume them but must allow-list them for CORS.
			tracingmdw.UserIDHeader,
			tracingmdw.ChainIDHeader,
		}, extraHeaders...),
		MaxAge: s.config.CORSMaxAge,
	}).Handler(r)

	s.server.Addr = fmt.Sprintf(":%d", s.config.Port)
	s.server.Handler = c
	s.server.ReadTimeout = s.config.ReadTimeout
	s.server.WriteTimeout = s.config.WriteTimeout
	s.server.IdleTimeout = s.config.IdleTimeout

	// Start the metrics server
	// Ideally, this server is not meant to be exposed to the public internet
	// and its /metrics endpoint must only be consumed by prometheus
	// or any other monitoring system
	// so no authz is required
	// Default port is 9100 and can be configured via METRICS_PORT env var
	// (look how Config is defined in config.go and used when instantiating the server)
	go func() {
		s.logger.InfoContext(ctx, "starting metrics server", slog.Int("port", s.config.MetricsPort))
		if err := s.metricsServer.Start(ctx); err != nil && err != http.ErrServerClosed {
			s.logger.ErrorContext(ctx, "failed to start metrics server", slog.Any("error", err))
		}
	}()

	s.logger.InfoContext(ctx, "starting server", slog.String("address", s.server.Addr))
	return s.server.ListenAndServe()
}

// Stop stops the REST server gracefully
func (s *Server) Stop(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
