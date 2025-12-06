package server

import (
	"context"
	"net/http"
	"sync"

	"github.com/wegman12/led-sound-light-control/health"
	"github.com/wegman12/led-sound-light-control/light"
	"github.com/wegman12/led-sound-light-control/remote"
	"go.uber.org/zap"
)

const (
	defaultRemoteGPIOPin = 20
)

func RegisterRoutes(mux *http.ServeMux, ctx context.Context, wg *sync.WaitGroup, logger *zap.Logger) {
	logger.Debug("Registering routes")

	health.RegisterRoutes(mux)

	// Create and register light controller
	lightController := light.RegisterRoutes(mux, ctx, wg, logger)

	// Create and start remote controller
	remoteController := remote.NewController(ctx, defaultRemoteGPIOPin, lightController, wg, logger)
	remoteController.Start()

	logger.Info("Routes registered successfully")
}
