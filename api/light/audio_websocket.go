package light

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/wegman12/led-sound-light-control/light/behavior"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for development
		// In production, you should validate the origin
		return true
	},
}

// AudioStreamData contains both raw and processed audio data
type AudioStreamData struct {
	Timestamp time.Time              `json:"timestamp"`
	Raw       RawAudioData           `json:"raw"`
	Processed ProcessedAudioData     `json:"processed"`
	Config    AudioTuningConfig      `json:"config"`
}

// RawAudioData contains raw audio profile values
type RawAudioData struct {
	Bass    float64 `json:"bass"`
	MidLow  float64 `json:"mid_low"`
	MidHigh float64 `json:"mid_high"`
	Treble  float64 `json:"treble"`
}

// ProcessedAudioData contains processed LED power values after tuning
type ProcessedAudioData struct {
	Bass    float64 `json:"bass"`
	MidLow  float64 `json:"mid_low"`
	MidHigh float64 `json:"mid_high"`
	Treble  float64 `json:"treble"`
}

// AudioWebSocketHandler manages WebSocket connections for audio streaming
type AudioWebSocketHandler struct {
	audioProvider behavior.AudioProvider
	configManager *AudioTuningConfigManager
	logger        *zap.Logger
	clients       map[*websocket.Conn]bool
	clientsMu     sync.RWMutex
	broadcast     chan AudioStreamData
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewAudioWebSocketHandler creates a new WebSocket handler
func NewAudioWebSocketHandler(
	audioProvider behavior.AudioProvider,
	configManager *AudioTuningConfigManager,
	logger *zap.Logger,
) *AudioWebSocketHandler {
	ctx, cancel := context.WithCancel(context.Background())

	handler := &AudioWebSocketHandler{
		audioProvider: audioProvider,
		configManager: configManager,
		logger:        logger,
		clients:       make(map[*websocket.Conn]bool),
		broadcast:     make(chan AudioStreamData, 100),
		ctx:           ctx,
		cancel:        cancel,
	}

	// Start broadcast goroutine
	go handler.broadcastLoop()

	return handler
}

// HandleWebSocket handles WebSocket connections
func (h *AudioWebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("Failed to upgrade to WebSocket", zap.Error(err))
		return
	}

	// Register client
	h.clientsMu.Lock()
	h.clients[conn] = true
	clientCount := len(h.clients)
	h.clientsMu.Unlock()

	h.logger.Info("WebSocket client connected", zap.Int("total_clients", clientCount))

	// Send initial configuration
	config := h.configManager.GetConfig()
	initialData := AudioStreamData{
		Timestamp: time.Now(),
		Config:    config,
	}
	if err := conn.WriteJSON(initialData); err != nil {
		h.logger.Error("Failed to send initial config", zap.Error(err))
	}

	// Handle client disconnection
	defer func() {
		h.clientsMu.Lock()
		delete(h.clients, conn)
		clientCount := len(h.clients)
		h.clientsMu.Unlock()
		conn.Close()
		h.logger.Info("WebSocket client disconnected", zap.Int("total_clients", clientCount))
	}()

	// Keep connection alive and handle client messages (for future use)
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				h.logger.Error("WebSocket error", zap.Error(err))
			}
			break
		}
	}
}

// broadcastLoop continuously broadcasts audio data to all connected clients
func (h *AudioWebSocketHandler) broadcastLoop() {
	ticker := time.NewTicker(25 * time.Millisecond) // 40 Hz update rate
	defer ticker.Stop()

	for {
		select {
		case <-h.ctx.Done():
			return
		case <-ticker.C:
			// Get latest audio profile
			profile := h.audioProvider.GetLatestProfile()
			if profile == nil {
				continue
			}

			// Get current tuning config
			config := h.configManager.GetConfig()

			// Calculate processed values (LED power) for each band
			processedData := h.calculateProcessedValues(profile, config)

			// Create stream data
			data := AudioStreamData{
				Timestamp: profile.Timestamp,
				Raw: RawAudioData{
					Bass:    profile.Bass,
					MidLow:  profile.MidLow,
					MidHigh: profile.MidHigh,
					Treble:  profile.Treble,
				},
				Processed: processedData,
				Config:    config,
			}

			// Broadcast to all clients
			h.clientsMu.RLock()
			for client := range h.clients {
				err := client.WriteJSON(data)
				if err != nil {
					h.logger.Error("Failed to send data to client", zap.Error(err))
					// Don't remove client here, let the read loop handle it
				}
			}
			h.clientsMu.RUnlock()
		}
	}
}

// calculateProcessedValues calculates the processed LED power values for each band
func (h *AudioWebSocketHandler) calculateProcessedValues(
	profile *behavior.AudioProfile,
	config AudioTuningConfig,
) ProcessedAudioData {
	return ProcessedAudioData{
		Bass:    h.processBandValue(profile.Bass, config.Bass),
		MidLow:  h.processBandValue(profile.MidLow, config.MidLow),
		MidHigh: h.processBandValue(profile.MidHigh, config.MidHigh),
		Treble:  h.processBandValue(profile.Treble, config.Treble),
	}
}

// processBandValue processes a single band value according to tuning parameters
// This mirrors the logic in AudioModulator.GetPower() but without smoothing
func (h *AudioWebSocketHandler) processBandValue(rawValue float64, bandConfig AudioTuningBandConfig) float64 {
	// Apply noise threshold
	if rawValue < bandConfig.NoiseThreshold {
		return bandConfig.MinPowerValue
	}

	// Subtract noise floor
	adjustedValue := rawValue - bandConfig.NoiseThreshold

	// Scale to 0-1 range
	scaledValue := adjustedValue * bandConfig.ScalingFactor

	// Clamp to [0, 1]
	if scaledValue < 0 {
		scaledValue = 0
	} else if scaledValue > 1 {
		scaledValue = 1
	}

	// Map to min-max range
	powerValue := bandConfig.MinPowerValue + scaledValue*(bandConfig.MaxPowerValue-bandConfig.MinPowerValue)

	return powerValue
}

// NotifyConfigUpdate sends updated configuration to all connected clients
func (h *AudioWebSocketHandler) NotifyConfigUpdate(config AudioTuningConfig) {
	data := AudioStreamData{
		Timestamp: time.Now(),
		Config:    config,
	}

	h.clientsMu.RLock()
	for client := range h.clients {
		err := client.WriteJSON(data)
		if err != nil {
			h.logger.Error("Failed to send config update to client", zap.Error(err))
		}
	}
	h.clientsMu.RUnlock()
}

// Close stops the broadcast loop and closes all connections
func (h *AudioWebSocketHandler) Close() {
	h.cancel()

	h.clientsMu.Lock()
	for client := range h.clients {
		client.Close()
	}
	h.clients = make(map[*websocket.Conn]bool)
	h.clientsMu.Unlock()
}
