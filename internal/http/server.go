package http

import (
	"context"
	"net/http"
	"time"

	"github.com/edygunawan/go-rest-api/internal/http/router"
	"github.com/edygunawan/golang-pkgs/logger"
)

type Config struct {
	Port string
}

type Server struct {
	httpServer *http.Server
	logger     *logger.Logger
}

func NewServer(cfg Config, log *logger.Logger, deps router.Dependencies) *Server {
	routes := router.NewRouter(deps)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           routes,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	return &Server{
		httpServer: srv,
		logger:     log,
	}
}

func (s *Server) Start() error {
	s.logger.Info("http server started", logger.F("addr", s.httpServer.Addr))
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("http server shutting down")
	return s.httpServer.Shutdown(ctx)
}
