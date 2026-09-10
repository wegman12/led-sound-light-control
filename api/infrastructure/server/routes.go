package server

import (
	"context"
	"net/http"
	"sync"

	"github.com/wegman12/led-sound-light-control/health"
	"github.com/wegman12/led-sound-light-control/light"
	"github.com/wegman12/led-sound-light-control/light/behavior"
	"github.com/wegman12/led-sound-light-control/remote"
	"go.uber.org/zap"
)

const (
	defaultRemoteGPIOPin = 20
)

func RegisterRoutes(mux *http.ServeMux, ctx context.Context, wg *sync.WaitGroup, logger *zap.Logger) error {
	logger.Debug("Registering routes")

	health.RegisterRoutes(mux)

	// Create shared AudioProvider for audio-reactive lighting
	audioProvider := behavior.NewAudioProvider()
	logger.Debug("Created AudioProvider for audio-reactive lighting")

	// Create and register light controller (with audio support)
	lightController := light.RegisterRoutes(mux, ctx, wg, audioProvider, logger)

	// Create and start remote controller (PRU-based)
	remoteController, err := remote.NewController(ctx, defaultRemoteGPIOPin, lightController, wg, logger)
	if err != nil {
		logger.Fatal("Failed to initialize PRU-based remote controller", zap.Error(err))
		return err
	}
	remoteController.Start()

	logger.Info("Routes registered successfully (with PRU remote control)")
	return nil
}
