package nedirect

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/omcgo/omcgo/internal/core/appconfig"
	"go.uber.org/zap"
)

// Server is an independent HTTP server for NE Direct connections (CMCC only).
// It uses net/http stdlib, not Gin, and only starts if enabled in config.
type Server struct {
	cfg        appconfig.NEDirectConfig
	httpServer *http.Server
	handler    *Handler
	logger     *zap.Logger
}

// NewServer creates a new NE Direct Server.
func NewServer(cfg appconfig.NEDirectConfig, handler *Handler, logger *zap.Logger) *Server {
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	return &Server{
		cfg:     cfg,
		handler: handler,
		logger:  logger,
		httpServer: &http.Server{
			Addr:         addr,
			Handler:      mux,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  120 * time.Second,
		},
	}
}

// Start begins listening for NE Direct connections in a background goroutine.
func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.httpServer.Addr)
	if err != nil {
		return fmt.Errorf("ne-direct listen %s: %w", s.httpServer.Addr, err)
	}

	s.logger.Info("ne-direct server starting", zap.String("addr", s.httpServer.Addr))

	go func() {
		if err := s.httpServer.Serve(ln); err != http.ErrServerClosed {
			s.logger.Error("ne-direct server error", zap.Error(err))
		}
	}()

	return nil
}

// Shutdown gracefully stops the NE Direct server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("ne-direct server shutting down")
	return s.httpServer.Shutdown(ctx)
}
