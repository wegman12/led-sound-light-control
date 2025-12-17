package light

import (
	"context"
	"net/http"
	"sync"

	"github.com/wegman12/led-sound-light-control/light/behavior"
	"go.uber.org/zap"
)

func RegisterRoutes(mux *http.ServeMux, ctx context.Context, wg *sync.WaitGroup, audioProvider behavior.AudioProvider, logger *zap.Logger) *Controller {
	logger.Debug("Registering light routes")

	controller := NewController(ctx, wg, audioProvider, logger)
	h := newHandler(controller, logger)
	mux.HandleFunc("POST /api/lights/behavior/register", h.handleRegisterBehavior)
	mux.HandleFunc("POST /api/lights/on", h.handleTurnOn)
	mux.HandleFunc("POST /api/lights/off", h.handleTurnOff)

	// Create audio tuning config manager
	configPath := "/etc/led-sound-light-control/audio_tuning_config.json"
	configManager, err := NewAudioTuningConfigManager(configPath, logger)
	if err != nil {
		logger.Error("Failed to create audio tuning config manager", zap.Error(err))
		// Use a config manager with default config
		configManager = &AudioTuningConfigManager{
			configPath: configPath,
			config:     getDefaultAudioTuningConfig(),
			logger:     logger,
		}
	}

	// Create WebSocket handler for audio tuning
	wsHandler := NewAudioWebSocketHandler(audioProvider, configManager, logger)

	// Register audio control endpoints
	audioHandler := newAudioHandler(ctx, audioProvider, configManager, wsHandler, logger)
	mux.HandleFunc("POST /api/lights/audio/start", audioHandler.handleAudioStart)
	mux.HandleFunc("POST /api/lights/audio/stop", audioHandler.handleAudioStop)
	mux.HandleFunc("GET /api/lights/audio/status", audioHandler.handleAudioStatus)
	mux.HandleFunc("GET /api/lights/audio/config", audioHandler.handleAudioConfig)

	// Register audio tuning endpoints
	mux.HandleFunc("GET /api/audio/tuning/config", audioHandler.handleGetTuningConfig)
	mux.HandleFunc("PUT /api/audio/tuning/config", audioHandler.handleUpdateTuningConfig)
	mux.HandleFunc("POST /api/audio/tuning/config/save", audioHandler.handleSaveTuningConfig)
	mux.HandleFunc("GET /api/audio/tuning/stream", audioHandler.handleTuningWebSocket)

	// Register simulation endpoint
	mux.HandleFunc("POST /api/lights/simulate", h.handleSimulate)

	logger.Info("Light routes registered (including audio control, tuning, and simulation)")

	return controller
}
