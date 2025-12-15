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
	audioManager  *audio.Manager
	cancelAudio   context.CancelFunc
	mu            sync.Mutex
	isStreaming   bool
	logger        *zap.Logger
}

func newAudioHandler(ctx context.Context, audioProvider behavior.AudioProvider, logger *zap.Logger) *AudioHandler {
	return &AudioHandler{
		ctx:           ctx,
		audioProvider: audioProvider,
		logger:        logger,
	}
}

// AudioStatusResponse contains the current audio streaming status
type AudioStatusResponse struct {
	IsStreaming bool                     `json:"is_streaming"`
	Profile     *behavior.AudioProfile   `json:"profile,omitempty"`
	Message     string                   `json:"message,omitempty"`
}

// AudioBandConfig contains recommended configuration for a frequency band
type AudioBandConfig struct {
	ScalingFactor  float64 `json:"scaling_factor"`
	NoiseThreshold float64 `json:"noise_threshold"`
}

// AudioScalingConfig contains recommended scaling factors from analysis
type AudioScalingConfig struct {
	Bass    AudioBandConfig `json:"bass"`
	MidLow  AudioBandConfig `json:"mid_low"`
	MidHigh AudioBandConfig `json:"mid_high"`
	Treble  AudioBandConfig `json:"treble"`
}

// Default audio configuration (from analysis in analysis/audio/)
const (
	defaultBufferSize           = 2048
	defaultSamplingRate         = 16000
	defaultTargetInputRate      = 8000.0
	defaultBassCutoff           = 250.0
	defaultMidCutoff            = 2000.0
	defaultDelayBetweenSamples  = 100 * time.Microsecond
	defaultDelayBetweenProcess  = 10 * time.Millisecond
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

	h.logger.Info("Starting audio stream to lights")

	// Create audio manager with default configuration
	audioManager, err := audio.NewManager(
		defaultBufferSize,
		defaultSamplingRate,
		defaultTargetInputRate,
		defaultBassCutoff,
		defaultMidCutoff,
		defaultDelayBetweenSamples,
		defaultDelayBetweenProcess,
		h.audioProvider,
		h.logger,
	)
	if err != nil {
		h.logger.Error("Failed to create audio manager", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Failed to initialize audio system: " + err.Error(),
		})
		return
	}

	// Create context for audio streaming
	ctx, cancel := context.WithCancel(h.ctx)
	h.cancelAudio = cancel
	h.audioManager = audioManager

	// Start audio streaming in background
	go func() {
		h.logger.Info("Audio streaming goroutine started")
		audioManager.StreamToLights(ctx)
		h.logger.Info("Audio streaming goroutine stopped")
	}()

	h.isStreaming = true

	h.logger.Info("Audio streaming started successfully")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":     "Audio streaming started",
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

	h.logger.Info("Stopping audio stream")

	// Cancel audio streaming context
	if h.cancelAudio != nil {
		h.cancelAudio()
		h.cancelAudio = nil
	}

	// Stop audio manager
	if h.audioManager != nil {
		h.audioManager.Stop()
		h.audioManager = nil
	}

	h.isStreaming = false

	h.logger.Info("Audio streaming stopped successfully")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":     "Audio streaming stopped",
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
func (h *AudioHandler) handleAudioConfig(w http.ResponseWriter, r *http.Request) {
	// These values are from the audio analysis (analysis/audio/audio_analysis.ipynb)
	// Based on analysis of bagpipes.csv, crazy_frog.csv, and christmas.csv
	config := AudioScalingConfig{
		Bass: AudioBandConfig{
			ScalingFactor:  0.000001,
			NoiseThreshold: 92565436,
		},
		MidLow: AudioBandConfig{
			ScalingFactor:  0.000001,
			NoiseThreshold: 18541891,
		},
		MidHigh: AudioBandConfig{
			ScalingFactor:  0.000002,
			NoiseThreshold: 10913229,
		},
		Treble: AudioBandConfig{
			ScalingFactor:  0.000027,
			NoiseThreshold: 2258769,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(config)
}
