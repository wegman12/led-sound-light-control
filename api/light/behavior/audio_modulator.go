package behavior

import (
	"encoding/json"
	"fmt"
	"math"
	"sync/atomic"
	"time"
)

// AudioModulatorConfig defines configuration for audio-reactive LED behavior
type AudioModulatorConfig struct {
	// FrequencyBand specifies which audio band to respond to
	// Valid values: "bass", "mid-low", "mid_low", "mid-high", "mid_high", "treble"
	FrequencyBand string `json:"frequency_band"`

	// MinPowerValue is the minimum LED power output (0.0-1.0)
	MinPowerValue float64 `json:"min_power_value"`

	// MaxPowerValue is the maximum LED power output (0.0-1.0)
	MaxPowerValue float64 `json:"max_power_value"`

	// ScalingFactor converts raw audio magnitude to 0-1 range
	// Recommended values from analysis:
	//   Bass: 0.000001, Mid-Low: 0.000001, Mid-High: 0.000002, Treble: 0.000027
	ScalingFactor float64 `json:"scaling_factor"`

	// NoiseThreshold filters out background noise (raw magnitude units)
	// Values below this threshold are treated as zero
	// Recommended values from analysis:
	//   Bass: 92565436, Mid-Low: 18541891, Mid-High: 10913229, Treble: 2258769
	NoiseThreshold float64 `json:"noise_threshold"`

	// Smoothing applies exponential smoothing to prevent flicker (0.0-1.0)
	// Higher values = more smoothing but slower response
	// 0.0 = no smoothing, 1.0 = maximum smoothing
	// Recommended: 0.2-0.4 for most use cases
	Smoothing float64 `json:"smoothing"`

	// FallbackPower is the power level when no audio data is available
	// If nil, the behavior returns nil (LED unchanged)
	FallbackPower *float64 `json:"fallback_power,omitempty"`

	// Weight for behavior averaging (default: 1.0)
	BehaviorWeight float64 `json:"weight,omitempty"`
}

// AudioModulator is a behavior that modulates LED brightness based on audio frequency bands
type AudioModulator struct {
	config        AudioModulatorConfig
	audioProvider AudioProvider
	lastValue     atomic.Value // stores *float64
	weight        float64
}

// newAudioModulator creates a new AudioModulator from JSON configuration
func newAudioModulator(cfg json.RawMessage, audioProvider AudioProvider) (*AudioModulator, error) {
	var config AudioModulatorConfig
	if err := json.Unmarshal(cfg, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal audio modulator config: %w", err)
	}

	// Validate configuration
	if err := validateAudioModulatorConfig(&config); err != nil {
		return nil, err
	}

	// Apply defaults
	if config.BehaviorWeight == 0 {
		config.BehaviorWeight = 1.0
	}

	modulator := &AudioModulator{
		config:        config,
		audioProvider: audioProvider,
		weight:        config.BehaviorWeight,
	}

	// Initialize lastValue to nil
	modulator.lastValue.Store((*float64)(nil))

	return modulator, nil
}

// validateAudioModulatorConfig validates the configuration parameters
func validateAudioModulatorConfig(config *AudioModulatorConfig) error {
	// Validate frequency band
	band := normalizeFrequencyBand(config.FrequencyBand)
	if band == "" {
		return fmt.Errorf("invalid frequency_band: %s (must be bass, mid-low, mid-high, or treble)",
			config.FrequencyBand)
	}

	// Validate power range
	if config.MinPowerValue < 0.0 || config.MinPowerValue > 1.0 {
		return fmt.Errorf("min_power_value must be between 0.0 and 1.0, got %f", config.MinPowerValue)
	}
	if config.MaxPowerValue < 0.0 || config.MaxPowerValue > 1.0 {
		return fmt.Errorf("max_power_value must be between 0.0 and 1.0, got %f", config.MaxPowerValue)
	}
	if config.MinPowerValue > config.MaxPowerValue {
		return fmt.Errorf("min_power_value (%f) cannot be greater than max_power_value (%f)",
			config.MinPowerValue, config.MaxPowerValue)
	}

	// Validate scaling factor
	if config.ScalingFactor < 0 {
		return fmt.Errorf("scaling_factor must be non-negative, got %f", config.ScalingFactor)
	}

	// Validate smoothing
	if config.Smoothing < 0.0 || config.Smoothing > 1.0 {
		return fmt.Errorf("smoothing must be between 0.0 and 1.0, got %f", config.Smoothing)
	}

	// Validate noise threshold
	if config.NoiseThreshold < 0 {
		return fmt.Errorf("noise_threshold must be non-negative, got %f", config.NoiseThreshold)
	}

	return nil
}

// normalizeFrequencyBand converts various band name formats to a standard form
func normalizeFrequencyBand(band string) string {
	switch band {
	case "bass", "Bass", "BASS":
		return "bass"
	case "mid-low", "mid_low", "midlow", "MidLow", "Mid-Low", "MID_LOW":
		return "mid-low"
	case "mid-high", "mid_high", "midhigh", "MidHigh", "Mid-High", "MID_HIGH":
		return "mid-high"
	case "treble", "Treble", "TREBLE":
		return "treble"
	default:
		return ""
	}
}

// GetPower returns the LED power level based on current audio data
// Implements the Behavior interface
func (a *AudioModulator) GetPower(t time.Duration) *float64 {
	// Get latest audio profile
	profile := a.audioProvider.GetLatestProfile()
	if profile == nil {
		// No audio data available - use fallback
		return a.config.FallbackPower
	}

	// Extract the appropriate frequency band value
	rawValue := a.extractFrequencyBand(profile)

	// Apply noise threshold
	if rawValue < a.config.NoiseThreshold {
		rawValue = 0.0
	}

	// Scale to 0-1 range
	scaledValue := rawValue * a.config.ScalingFactor

	// Clamp to configured min/max range
	power := clamp(scaledValue, a.config.MinPowerValue, a.config.MaxPowerValue)

	// Apply exponential smoothing if configured
	if a.config.Smoothing > 0.0 {
		power = a.applySmoothing(power)
	} else {
		// Store current value even without smoothing
		a.lastValue.Store(&power)
	}

	return &power
}

// Weight returns the behavior's weight for averaging
// Implements the Behavior interface
func (a *AudioModulator) Weight() float64 {
	return a.weight
}

// extractFrequencyBand extracts the configured frequency band from the audio profile
func (a *AudioModulator) extractFrequencyBand(profile *AudioProfile) float64 {
	band := normalizeFrequencyBand(a.config.FrequencyBand)
	switch band {
	case "bass":
		return profile.Bass
	case "mid-low":
		return profile.MidLow
	case "mid-high":
		return profile.MidHigh
	case "treble":
		return profile.Treble
	default:
		// Should never happen due to validation, but return bass as default
		return profile.Bass
	}
}

// applySmoothing applies exponential moving average smoothing
// Formula: smoothed = (1 - alpha) * current + alpha * previous
// where alpha = smoothing factor
func (a *AudioModulator) applySmoothing(currentValue float64) float64 {
	// Get previous value
	lastVal := a.lastValue.Load()
	if lastVal == nil {
		// First value - no smoothing needed
		a.lastValue.Store(&currentValue)
		return currentValue
	}

	previousValue, ok := lastVal.(*float64)
	if !ok || previousValue == nil {
		// Invalid previous value - use current
		a.lastValue.Store(&currentValue)
		return currentValue
	}

	// Apply exponential smoothing
	// Higher smoothing factor = more weight on previous value = smoother but slower
	alpha := a.config.Smoothing
	smoothed := (1-alpha)*currentValue + alpha*(*previousValue)

	// Store smoothed value for next iteration
	a.lastValue.Store(&smoothed)

	return smoothed
}

// clamp restricts a value to a given range
func clamp(value, min, max float64) float64 {
	if math.IsNaN(value) {
		return min
	}
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
