package light

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/wegman12/led-sound-light-control/audio"
	"github.com/wegman12/led-sound-light-control/light/behavior"
	"go.uber.org/zap"
)

// AudioHandler manages HTTP endpoints for audio-reactive lighting control
type AudioHandler struct {
	ctx           context.Context
	audioProvider behavior.AudioProvider
	configManager *AudioTuningConfigManager
	configRepo    *AudioConfigRepository
	wsHandler     *AudioWebSocketHandler
	pruManager    *audio.PRUManager
	cancelAudio   context.CancelFunc
	mu            sync.Mutex
	isStreaming   bool
	logger        *zap.Logger
}

func newAudioHandler(
	ctx context.Context,
	audioProvider behavior.AudioProvider,
	configManager *AudioTuningConfigManager,
	configRepo *AudioConfigRepository,
	wsHandler *AudioWebSocketHandler,
	logger *zap.Logger,
) *AudioHandler {
	return &AudioHandler{
		ctx:           ctx,
		audioProvider: audioProvider,
		configManager: configManager,
		configRepo:    configRepo,
		wsHandler:     wsHandler,
		logger:        logger,
	}
}

// AudioStatusResponse contains the current audio streaming status
type AudioStatusResponse struct {
	IsStreaming bool                   `json:"is_streaming"`
	Profile     *behavior.AudioProfile `json:"profile,omitempty"`
	Message     string                 `json:"message,omitempty"`
}

// AudioBandConfig contains recommended configuration for a frequency band
type AudioBandConfig struct {
	MaxMagnitude   float64 `json:"max_magnitude"`
	NoiseThreshold float64 `json:"noise_threshold"`
}

// AudioScalingConfig contains recommended scaling factors from analysis
type AudioScalingConfig struct {
	Bass    AudioBandConfig `json:"bass"`
	MidLow  AudioBandConfig `json:"mid_low"`
	MidHigh AudioBandConfig `json:"mid_high"`
	Treble  AudioBandConfig `json:"treble"`
}

// Default audio configuration for PRU audio streaming
// PRU samples at ~40 Hz, we read at 25ms intervals (40 Hz)
const (
	defaultPRUUpdateRate = 25 * time.Millisecond // Read PRU audio data at 40 Hz
)

// handleAudioStart starts audio streaming to lights
func (h *AudioHandler) handleAudioStart(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.isStreaming {
		h.logger.Warn("Audio streaming already active")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Audio streaming is already active",
		})
		return
	}

	h.logger.Info("Starting PRU audio stream to lights")

	// Create PRU audio manager
	pruManager, err := audio.NewPRUManager(h.logger)
	if err != nil {
		h.logger.Error("Failed to create PRU audio manager", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Failed to initialize PRU audio system: " + err.Error(),
		})
		return
	}

	h.pruManager = pruManager

	// Push current config values to PRU to ensure sync with website
	config := h.configManager.GetConfig()
	bands := &audio.FrequencyBands{
		BassMax:    uint32(config.BassCutoff),
		MidLowMax:  uint32(config.MidHighCutoff),
		MidHighMax: uint32(config.TrebleCutoff),
	}
	if err := h.pruManager.SetFrequencyBands(bands); err != nil {
		h.logger.Error("Failed to set initial PRU frequency bands", zap.Error(err))
		// Continue anyway - PRU will use its defaults
	} else {
		h.logger.Info("Initialized PRU frequency bands from config",
			zap.Uint32("bass_max", bands.BassMax),
			zap.Uint32("midlow_max", bands.MidLowMax),
			zap.Uint32("midhigh_max", bands.MidHighMax),
		)
	}

	// Create context for audio streaming
	ctx, cancel := context.WithCancel(h.ctx)
	h.cancelAudio = cancel

	// Create channel for PRU sound profiles
	profileChannel := make(chan *audio.SoundProfile, 100)

	// Start PRU audio manager in background
	go func() {
		h.logger.Info("PRU audio manager goroutine started")
		err := h.pruManager.Start(profileChannel, defaultPRUUpdateRate, ctx)
		if err != nil {
			h.logger.Error("PRU audio manager error", zap.Error(err))
		}
		h.logger.Info("PRU audio manager goroutine stopped")
	}()

	// Start goroutine to convert PRU profiles to AudioProvider profiles
	go func() {
		h.logger.Info("Audio profile conversion goroutine started")
		for {
			select {
			case <-ctx.Done():
				h.logger.Info("Audio profile conversion stopped")
				return
			case profile := <-profileChannel:
				// Convert PRU SoundProfile to behavior AudioProfile
				// Use the Avg values (average magnitude per bin)
				audioProfile := behavior.AudioProfile{
					Bass:      float64(profile.BassSum),
					MidLow:    float64(profile.MidLowSum),
					MidHigh:   float64(profile.MidHighSum),
					Treble:    float64(profile.TrebleSum),
					Timestamp: time.Now(),
				}

				// Update the audio provider
				h.audioProvider.UpdateProfile(audioProfile)

				h.logger.Debug("Updated audio profile from PRU",
					zap.Float64("bass", audioProfile.Bass),
					zap.Float64("mid_low", audioProfile.MidLow),
					zap.Float64("mid_high", audioProfile.MidHigh),
					zap.Float64("treble", audioProfile.Treble),
				)
			}
		}
	}()

	h.isStreaming = true

	h.logger.Info("PRU audio streaming started successfully")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":      "Audio streaming started",
		"is_streaming": true,
	})
}

// handleAudioStop stops audio streaming
func (h *AudioHandler) handleAudioStop(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.isStreaming {
		h.logger.Warn("Audio streaming not active")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Audio streaming is not active",
		})
		return
	}

	h.logger.Info("Stopping PRU audio stream")

	// Cancel audio streaming context
	if h.cancelAudio != nil {
		h.cancelAudio()
		h.cancelAudio = nil
	}

	// Stop and close PRU manager
	if h.pruManager != nil {
		h.pruManager.Stop()
		if err := h.pruManager.Close(); err != nil {
			h.logger.Error("Error closing PRU manager", zap.Error(err))
		}
		h.pruManager = nil
	}

	h.isStreaming = false

	h.logger.Info("PRU audio streaming stopped successfully")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":      "Audio streaming stopped",
		"is_streaming": false,
	})
}

// handleAudioStatus returns the current audio streaming status
func (h *AudioHandler) handleAudioStatus(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	streaming := h.isStreaming
	h.mu.Unlock()

	response := AudioStatusResponse{
		IsStreaming: streaming,
	}

	// Get current audio profile if available
	if profile := h.audioProvider.GetLatestProfile(); profile != nil {
		response.Profile = profile
		response.Message = "Audio data available"
	} else if streaming {
		response.Message = "Streaming active, waiting for audio data"
	} else {
		response.Message = "Audio streaming not active"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleAudioConfig returns recommended audio scaling configuration
// DEPRECATED: Use handleGetTuningConfig instead
func (h *AudioHandler) handleAudioConfig(w http.ResponseWriter, r *http.Request) {
	// These values are from the audio analysis (analysis/audio/audio_analysis.ipynb)
	// Based on analysis of bagpipes.csv, crazy_frog.csv, and christmas.csv
	config := AudioScalingConfig{
		Bass: AudioBandConfig{
			MaxMagnitude:   1000000,
			NoiseThreshold: 92565436,
		},
		MidLow: AudioBandConfig{
			MaxMagnitude:   1000000,
			NoiseThreshold: 18541891,
		},
		MidHigh: AudioBandConfig{
			MaxMagnitude:   500000,
			NoiseThreshold: 10913229,
		},
		Treble: AudioBandConfig{
			MaxMagnitude:   37037,
			NoiseThreshold: 2258769,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(config)
}

// handleGetTuningConfig returns the current audio tuning configuration
func (h *AudioHandler) handleGetTuningConfig(w http.ResponseWriter, r *http.Request) {
	config := h.configManager.GetConfig()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(config)
}

// handleUpdateTuningConfig updates the audio tuning configuration
func (h *AudioHandler) handleUpdateTuningConfig(w http.ResponseWriter, r *http.Request) {
	var config AudioTuningConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		h.logger.Error("Failed to decode tuning config", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid request body: " + err.Error(),
		})
		return
	}

	// Update configuration
	if err := h.configManager.UpdateConfig(config); err != nil {
		h.logger.Error("Failed to update tuning config", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid configuration: " + err.Error(),
		})
		return
	}

	// Push frequency band changes to PRU if streaming is active
	h.mu.Lock()
	if h.pruManager != nil {
		bands := &audio.FrequencyBands{
			BassMax:    uint32(config.BassCutoff),
			MidLowMax:  uint32(config.MidHighCutoff),
			MidHighMax: uint32(config.TrebleCutoff),
		}
		if err := h.pruManager.SetFrequencyBands(bands); err != nil {
			h.logger.Error("Failed to update PRU frequency bands", zap.Error(err))
			// Don't fail the request, just log the error
		} else {
			h.logger.Info("Updated PRU frequency bands",
				zap.Uint32("bass_max", bands.BassMax),
				zap.Uint32("midlow_max", bands.MidLowMax),
				zap.Uint32("midhigh_max", bands.MidHighMax),
			)
		}
	}
	h.mu.Unlock()

	// Notify WebSocket clients of config update
	h.wsHandler.NotifyConfigUpdate(config)

	h.logger.Info("Audio tuning configuration updated")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Configuration updated successfully",
		"config":  config,
	})
}

// handleSaveTuningConfig saves the current tuning configuration
// If repository is available, saves to the active config in repository
// Otherwise falls back to legacy file-based save
func (h *AudioHandler) handleSaveTuningConfig(w http.ResponseWriter, r *http.Request) {
	config := h.configManager.GetConfig()

	// If we have a repository, update the active config there
	if h.configRepo != nil {
		activeName, err := h.configRepo.GetActiveName()
		if err != nil {
			h.logger.Error("Failed to get active config name", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Failed to get active configuration: " + err.Error(),
			})
			return
		}

		// Get the current saved config to preserve metadata
		savedConfig, err := h.configRepo.Get(activeName)
		if err != nil {
			h.logger.Error("Failed to get active config", zap.String("name", activeName), zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Failed to get active configuration: " + err.Error(),
			})
			return
		}

		// Update the tuning config while preserving metadata
		savedConfig.Config = config
		if err := h.configRepo.Update(activeName, savedConfig); err != nil {
			h.logger.Error("Failed to save config to repository", zap.String("name", activeName), zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Failed to save configuration: " + err.Error(),
			})
			return
		}

		h.logger.Info("Audio tuning configuration saved to repository", zap.String("config", activeName))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message":     "Configuration saved successfully",
			"config_name": activeName,
			"config":      config,
		})
		return
	}

	// Fallback to legacy file-based save
	if err := h.configManager.SaveConfig(); err != nil {
		h.logger.Error("Failed to save tuning config", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Failed to save configuration: " + err.Error(),
		})
		return
	}

	h.logger.Info("Audio tuning configuration saved to file")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Configuration saved successfully",
		"config":  config,
	})
}

// handleTuningWebSocket handles WebSocket connections for audio tuning
func (h *AudioHandler) handleTuningWebSocket(w http.ResponseWriter, r *http.Request) {
	h.wsHandler.HandleWebSocket(w, r)
}
