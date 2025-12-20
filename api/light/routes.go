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

	// Create audio configuration repository
	configRepoPath := "/etc/led-sound-light-control/audio-configs"
	configRepo, err := NewAudioConfigRepository(configRepoPath, logger)
	if err != nil {
		logger.Error("Failed to create audio config repository", zap.Error(err))
		// Continue without repository - handlers will fail gracefully
	}

	// Create audio config handler for new configuration management endpoints
	if configRepo != nil {
		configHandler := NewAudioConfigHandler(configRepo, logger)
		mux.HandleFunc("GET /api/audio/configs", configHandler.HandleListConfigs)
		mux.HandleFunc("POST /api/audio/configs", configHandler.HandleCreateConfig)
		mux.HandleFunc("GET /api/audio/configs/active", configHandler.HandleGetActiveConfig)
		mux.HandleFunc("PUT /api/audio/configs/active", configHandler.HandleSetActiveConfig)
		mux.HandleFunc("GET /api/audio/configs/{name}", configHandler.HandleGetConfig)
		mux.HandleFunc("PUT /api/audio/configs/{name}", configHandler.HandleUpdateConfig)
		mux.HandleFunc("DELETE /api/audio/configs/{name}", configHandler.HandleDeleteConfig)
		logger.Info("Audio configuration management endpoints registered")
	}

	// Create audio tuning config manager (loads from active config in repository)
	var configManager *AudioTuningConfigManager
	if configRepo != nil {
		// Load initial config from repository's active config
		activeConfig, err := configRepo.GetActive()
		if err != nil {
			logger.Error("Failed to get active config from repository", zap.Error(err))
			configManager = &AudioTuningConfigManager{
				configPath: "",
				config:     getDefaultAudioTuningConfig(),
				logger:     logger,
			}
		} else {
			configManager = &AudioTuningConfigManager{
				configPath: "", // Not used when backed by repository
				config:     activeConfig.Config,
				logger:     logger,
			}
			logger.Info("Loaded active configuration from repository", zap.String("name", activeConfig.Name))
		}
	} else {
		// Fallback to legacy single-file config
		configPath := "/etc/led-sound-light-control/audio_tuning_config.json"
		configManager, err = NewAudioTuningConfigManager(configPath, logger)
		if err != nil {
			logger.Error("Failed to create audio tuning config manager", zap.Error(err))
			configManager = &AudioTuningConfigManager{
				configPath: configPath,
				config:     getDefaultAudioTuningConfig(),
				logger:     logger,
			}
		}
	}

	// Create WebSocket handler for audio tuning
	wsHandler := NewAudioWebSocketHandler(audioProvider, configManager, logger)

	// Register audio control endpoints
	audioHandler := newAudioHandler(ctx, audioProvider, configManager, configRepo, wsHandler, logger)
	mux.HandleFunc("POST /api/lights/audio/start", audioHandler.handleAudioStart)
	mux.HandleFunc("POST /api/lights/audio/stop", audioHandler.handleAudioStop)
	mux.HandleFunc("GET /api/lights/audio/status", audioHandler.handleAudioStatus)
	mux.HandleFunc("GET /api/lights/audio/config", audioHandler.handleAudioConfig)

	// Register audio tuning endpoints (these still work with in-memory config)
	mux.HandleFunc("GET /api/audio/tuning/config", audioHandler.handleGetTuningConfig)
	mux.HandleFunc("PUT /api/audio/tuning/config", audioHandler.handleUpdateTuningConfig)
	mux.HandleFunc("POST /api/audio/tuning/config/save", audioHandler.handleSaveTuningConfig)
	mux.HandleFunc("GET /api/audio/tuning/stream", audioHandler.handleTuningWebSocket)

	// Register simulation endpoint
	mux.HandleFunc("POST /api/lights/simulate", h.handleSimulate)

	logger.Info("Light routes registered (including audio control, tuning, configuration, and simulation)")

	return controller
}
