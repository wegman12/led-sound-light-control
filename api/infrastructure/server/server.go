package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"
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
	config     ServerConfig
	httpServer *http.Server
}

// NewServer creates a new Server instance with the given configuration
func NewServer(config ServerConfig, ctx context.Context) *Server {
	mux := http.NewServeMux()

	RegisterRoutes(mux, ctx)

	return &Server{
		config: config,
		httpServer: &http.Server{
			Addr:         fmt.Sprintf("%s:%d", config.Host, config.Port),
			Handler:      mux,
			ReadTimeout:  time.Duration(config.ReadTimeout) * time.Second,
			WriteTimeout: time.Duration(config.WriteTimeout) * time.Second,
		},
	}
}

// Start starts the HTTP server and handles graceful shutdown
func (s *Server) Start(ctx context.Context) error {
	// Channel to capture server errors
	serverErrors := make(chan error, 1)

	// Start server in a goroutine
	go func() {
		log.Printf("Starting server on %s", s.httpServer.Addr)
		serverErrors <- s.httpServer.ListenAndServe()
	}()

	return s.waitForContextCancellation(ctx, serverErrors)
}

func (s *Server) waitForContextCancellation(ctx context.Context, serverErrors chan error) error {

	// Wait for either context cancellation or server error
	select {
	case err := <-serverErrors:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("server error: %w", err)
		}
	case <-ctx.Done():
		log.Println("Shutdown signal received, shutting down gracefully...")

		// Create shutdown context with timeout
		shutdownCtx, cancel := context.WithTimeout(
			context.Background(),
			time.Duration(s.config.ShutdownTimeout)*time.Second,
		)
		defer cancel()

		// Attempt graceful shutdown
		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("error during server shutdown: %w", err)
		}

		log.Println("Server stopped gracefully")
	}
	return nil
}
