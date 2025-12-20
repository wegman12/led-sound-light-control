package light

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"go.uber.org/zap"
)

// AudioTuningBandConfig contains tuning parameters for a single frequency band
type AudioTuningBandConfig struct {
	ScalingFactor  float64 `json:"scaling_factor"`
	NoiseThreshold float64 `json:"noise_threshold"`
	MinPowerValue  float64 `json:"min_power_value"`
	MaxPowerValue  float64 `json:"max_power_value"`
	Smoothing      float64 `json:"smoothing"`
}

// AudioTuningConfig contains all audio tuning parameters
type AudioTuningConfig struct {
	// Frequency band cutoffs (Hz)
	BassCutoff    float64 `json:"bass_cutoff"`     // 0-BassCutoff Hz
	MidHighCutoff float64 `json:"mid_high_cutoff"` // BassCutoff-MidHighCutoff Hz (mid-low is implicit)
	TrebleCutoff  float64 `json:"treble_cutoff"`   // MidHighCutoff-TrebleCutoff Hz (mid-high), TrebleCutoff+ (treble)

	// Per-band tuning parameters
	Bass    AudioTuningBandConfig `json:"bass"`
	MidLow  AudioTuningBandConfig `json:"mid_low"`
	MidHigh AudioTuningBandConfig `json:"mid_high"`
	Treble  AudioTuningBandConfig `json:"treble"`
}

// AudioTuningConfigManager manages loading and saving of audio tuning configuration
type AudioTuningConfigManager struct {
	configPath string
	config     AudioTuningConfig
	mu         sync.RWMutex
	logger     *zap.Logger
}

// NewAudioTuningConfigManager creates a new config manager
func NewAudioTuningConfigManager(configPath string, logger *zap.Logger) (*AudioTuningConfigManager, error) {
	manager := &AudioTuningConfigManager{
		configPath: configPath,
		logger:     logger,
		config:     getDefaultAudioTuningConfig(),
	}

	// Try to load config from file, but don't fail if it doesn't exist
	if err := manager.LoadConfig(); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load audio tuning config: %w", err)
		}
		// File doesn't exist, use defaults
		logger.Info("Audio tuning config file not found, using defaults", zap.String("path", configPath))
	} else {
		logger.Info("Loaded audio tuning config from file", zap.String("path", configPath))
	}

	return manager, nil
}

// getDefaultAudioTuningConfig returns the default audio tuning configuration
// Updated based on I2S microphone optimization analysis
// See: analysis/audio/i2s_capture_data/optimization_results_v2/optimization_report.txt
func getDefaultAudioTuningConfig() AudioTuningConfig {
	return AudioTuningConfig{
		BassCutoff:    100,  // Bass: 0-100 Hz (sub-bass and low bass)
		MidHighCutoff: 500,  // Mid-Low: 100-500 Hz (mid-bass, warmth)
		TrebleCutoff:  2000, // Mid-High: 500-2000 Hz (clarity, presence), Treble: 2000+ Hz
		Bass: AudioTuningBandConfig{
			ScalingFactor:  0.000001,
			NoiseThreshold: 92565436,
			MinPowerValue:  0.0,
			MaxPowerValue:  1.0,
			Smoothing:      0.3,
		},
		MidLow: AudioTuningBandConfig{
			ScalingFactor:  0.000001,
			NoiseThreshold: 18541891,
			MinPowerValue:  0.0,
			MaxPowerValue:  1.0,
			Smoothing:      0.3,
		},
		MidHigh: AudioTuningBandConfig{
			ScalingFactor:  0.000002,
			NoiseThreshold: 10913229,
			MinPowerValue:  0.0,
			MaxPowerValue:  1.0,
			Smoothing:      0.3,
		},
		Treble: AudioTuningBandConfig{
			ScalingFactor:  0.000027,
			NoiseThreshold: 2258769,
			MinPowerValue:  0.0,
			MaxPowerValue:  1.0,
			Smoothing:      0.3,
		},
	}
}

// LoadConfig loads the configuration from the file
func (m *AudioTuningConfigManager) LoadConfig() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return err
	}

	var config AudioTuningConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate configuration
	if err := m.validateConfig(&config); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	m.config = config
	return nil
}

// SaveConfig saves the current configuration to the file
func (m *AudioTuningConfigManager) SaveConfig() error {
	m.mu.RLock()
	config := m.config
	m.mu.RUnlock()

	// Validate before saving
	if err := m.validateConfig(&config); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Marshal config to JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(m.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write to file
	if err := os.WriteFile(m.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	m.logger.Info("Saved audio tuning config to file", zap.String("path", m.configPath))
	return nil
}

// GetConfig returns a copy of the current configuration
func (m *AudioTuningConfigManager) GetConfig() AudioTuningConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

// UpdateConfig updates the configuration
func (m *AudioTuningConfigManager) UpdateConfig(config AudioTuningConfig) error {
	// Validate before updating
	if err := m.validateConfig(&config); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	m.mu.Lock()
	m.config = config
	m.mu.Unlock()

	m.logger.Info("Updated audio tuning config in memory")
	return nil
}

// validateConfig validates the configuration parameters
func (m *AudioTuningConfigManager) validateConfig(config *AudioTuningConfig) error {
	// Validate cutoff frequencies
	if config.BassCutoff <= 0 {
		return fmt.Errorf("bass_cutoff must be positive, got %f", config.BassCutoff)
	}
	if config.MidHighCutoff <= config.BassCutoff {
		return fmt.Errorf("mid_high_cutoff (%f) must be greater than bass_cutoff (%f)",
			config.MidHighCutoff, config.BassCutoff)
	}
	if config.TrebleCutoff <= config.MidHighCutoff {
		return fmt.Errorf("treble_cutoff (%f) must be greater than mid_high_cutoff (%f)",
			config.TrebleCutoff, config.MidHighCutoff)
	}

	// Validate each band
	bands := map[string]AudioTuningBandConfig{
		"bass":     config.Bass,
		"mid_low":  config.MidLow,
		"mid_high": config.MidHigh,
		"treble":   config.Treble,
	}

	for name, band := range bands {
		if err := validateBandConfig(name, &band); err != nil {
			return err
		}
	}

	return nil
}

// validateBandConfig validates a single band configuration
func validateBandConfig(name string, config *AudioTuningBandConfig) error {
	if config.ScalingFactor <= 0 {
		return fmt.Errorf("%s: scaling_factor must be positive, got %f", name, config.ScalingFactor)
	}
	if config.NoiseThreshold < 0 {
		return fmt.Errorf("%s: noise_threshold must be non-negative, got %f", name, config.NoiseThreshold)
	}
	if config.MinPowerValue < 0 || config.MinPowerValue > 1 {
		return fmt.Errorf("%s: min_power_value must be between 0 and 1, got %f", name, config.MinPowerValue)
	}
	if config.MaxPowerValue < 0 || config.MaxPowerValue > 1 {
		return fmt.Errorf("%s: max_power_value must be between 0 and 1, got %f", name, config.MaxPowerValue)
	}
	if config.MaxPowerValue < config.MinPowerValue {
		return fmt.Errorf("%s: max_power_value (%f) must be >= min_power_value (%f)",
			name, config.MaxPowerValue, config.MinPowerValue)
	}
	if config.Smoothing < 0 || config.Smoothing > 1 {
		return fmt.Errorf("%s: smoothing must be between 0 and 1, got %f", name, config.Smoothing)
	}
	return nil
}
