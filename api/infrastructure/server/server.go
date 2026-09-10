package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ServerConfig holds the configuration for the HTTP server
type ServerConfig struct {
	Host            string
	Port            int
	ReadTimeout     int
	WriteTimeout    int
	ShutdownTimeout int
}

// Server represents the HTTP server
type Server struct {
	config        ServerConfig
	httpServer    *http.Server
	cleanupEvents *sync.WaitGroup
	logger        *zap.Logger
}

// NewServer creates a new Server instance with the given configuration
// Returns error if PRU initialization or route registration fails
func NewServer(config ServerConfig, ctx context.Context, logger *zap.Logger) (*Server, error) {
	mux := http.NewServeMux()

	wg := &sync.WaitGroup{}
	if err := RegisterRoutes(mux, ctx, wg, logger); err != nil {
		return nil, fmt.Errorf("failed to register routes: %w", err)
	}

	// Wrap mux with CORS middleware
	handler := corsMiddleware(mux)

	logger.Debug("Server initialized",
		zap.String("address", fmt.Sprintf("%s:%d", config.Host, config.Port)),
		zap.Int("read_timeout", config.ReadTimeout),
		zap.Int("write_timeout", config.WriteTimeout),
	)

	return &Server{
		config: config,
		httpServer: &http.Server{
			Addr:         fmt.Sprintf("%s:%d", config.Host, config.Port),
			Handler:      handler,
			ReadTimeout:  time.Duration(config.ReadTimeout) * time.Second,
			WriteTimeout: time.Duration(config.WriteTimeout) * time.Second,
		},
		cleanupEvents: wg,
		logger:        logger,
	}, nil
}

// Start starts the HTTP server and handles graceful shutdown
func (s *Server) Start(ctx context.Context) error {
	// Channel to capture server errors
	serverErrors := make(chan error, 1)

	// Start server in a goroutine
	go func() {
		s.logger.Info("Server listening", zap.String("address", s.httpServer.Addr))
		serverErrors <- s.httpServer.ListenAndServe()
	}()

	return s.waitForContextCancellation(ctx, serverErrors)
}

func (s *Server) waitForContextCancellation(ctx context.Context, serverErrors chan error) error {

	// Wait for either context cancellation or server error
	select {
	case err := <-serverErrors:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Error("Server error", zap.Error(err))
			return fmt.Errorf("server error: %w", err)
		}
	case <-ctx.Done():
		s.logger.Info("Shutdown signal received, shutting down gracefully")

		// Create shutdown context with timeout
		shutdownCtx, cancel := context.WithTimeout(
			context.Background(),
			time.Duration(s.config.ShutdownTimeout)*time.Second,
		)
		defer cancel()

		s.logger.Debug("Shutting down HTTP server", zap.Int("timeout", s.config.ShutdownTimeout))

		// Attempt graceful shutdown
		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			s.logger.Error("Error during server shutdown", zap.Error(err))
			return fmt.Errorf("error during server shutdown: %w", err)
		}

		// Wait for other cleanup events to finish
		if s.cleanupEvents != nil {
			s.logger.Debug("Waiting for cleanup events to complete")
			s.cleanupEvents.Wait()
		}

		s.logger.Info("Server stopped gracefully")
	}
	return nil
}
