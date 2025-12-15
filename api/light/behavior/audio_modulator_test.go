package behavior

import (
	"encoding/json"
	"math"
	"testing"
	"time"
)

// mockAudioProvider is a simple mock for testing
type mockAudioProvider struct {
	profile *AudioProfile
}

func (m *mockAudioProvider) GetLatestProfile() *AudioProfile {
	return m.profile
}

func (m *mockAudioProvider) UpdateProfile(profile AudioProfile) {
	m.profile = &profile
}

func (m *mockAudioProvider) Subscribe() <-chan AudioProfile {
	return make(chan AudioProfile)
}

func TestNewAudioModulator_ValidConfig(t *testing.T) {
	provider := &mockAudioProvider{}
	config := AudioModulatorConfig{
		FrequencyBand:  "bass",
		MinPowerValue:  0.0,
		MaxPowerValue:  1.0,
		ScalingFactor:  0.000001,
		NoiseThreshold: 1000000,
		Smoothing:      0.3,
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	modulator, err := newAudioModulator(configJSON, provider)
	if err != nil {
		t.Fatalf("newAudioModulator failed: %v", err)
	}

	if modulator == nil {
		t.Fatal("newAudioModulator returned nil")
	}

	if modulator.Weight() != 1.0 {
		t.Errorf("Expected default weight 1.0, got %f", modulator.Weight())
	}
}

func TestNewAudioModulator_InvalidFrequencyBand(t *testing.T) {
	provider := &mockAudioProvider{}
	config := AudioModulatorConfig{
		FrequencyBand:  "invalid",
		MinPowerValue:  0.0,
		MaxPowerValue:  1.0,
		ScalingFactor:  0.000001,
		NoiseThreshold: 1000000,
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	_, err = newAudioModulator(configJSON, provider)
	if err == nil {
		t.Error("Expected error for invalid frequency band")
	}
}

func TestNewAudioModulator_InvalidPowerRange(t *testing.T) {
	provider := &mockAudioProvider{}

	tests := []struct {
		name string
		min  float64
		max  float64
	}{
		{"min < 0", -0.1, 1.0},
		{"max > 1", 0.0, 1.1},
		{"min > max", 0.8, 0.2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := AudioModulatorConfig{
				FrequencyBand:  "bass",
				MinPowerValue:  tt.min,
				MaxPowerValue:  tt.max,
				ScalingFactor:  0.000001,
				NoiseThreshold: 1000000,
			}

			configJSON, _ := json.Marshal(config)
			_, err := newAudioModulator(configJSON, provider)
			if err == nil {
				t.Errorf("Expected error for invalid power range")
			}
		})
	}
}

func TestAudioModulator_GetPower_NoAudioData(t *testing.T) {
	provider := &mockAudioProvider{profile: nil}
	fallback := 0.5

	config := AudioModulatorConfig{
		FrequencyBand:  "bass",
		MinPowerValue:  0.0,
		MaxPowerValue:  1.0,
		ScalingFactor:  0.000001,
		NoiseThreshold: 1000000,
		FallbackPower:  &fallback,
	}

	configJSON, _ := json.Marshal(config)
	modulator, err := newAudioModulator(configJSON, provider)
	if err != nil {
		t.Fatalf("Failed to create modulator: %v", err)
	}

	power := modulator.GetPower(0)
	if power == nil {
		t.Fatal("GetPower returned nil when fallback was configured")
	}

	if *power != fallback {
		t.Errorf("Expected fallback power %f, got %f", fallback, *power)
	}
}

func TestAudioModulator_GetPower_BassExtraction(t *testing.T) {
	provider := &mockAudioProvider{
		profile: &AudioProfile{
			Bass:      100000.0, // Raw value
			MidLow:    50000.0,
			MidHigh:   30000.0,
			Treble:    10000.0,
			Timestamp: time.Now(),
		},
	}

	config := AudioModulatorConfig{
		FrequencyBand:  "bass",
		MinPowerValue:  0.0,
		MaxPowerValue:  1.0,
		ScalingFactor:  0.000001, // 100000 * 0.000001 = 0.1
		NoiseThreshold: 1000,     // Below bass value
		Smoothing:      0.0,
	}

	configJSON, _ := json.Marshal(config)
	modulator, _ := newAudioModulator(configJSON, provider)

	power := modulator.GetPower(0)
	if power == nil {
		t.Fatal("GetPower returned nil")
	}

	expected := 0.1 // 100000 * 0.000001 = 0.1
	if math.Abs(*power-expected) > 0.001 {
		t.Errorf("Expected power ~%f, got %f", expected, *power)
	}
}

func TestAudioModulator_GetPower_AllBands(t *testing.T) {
	profile := &AudioProfile{
		Bass:      100000.0,
		MidLow:    50000.0,
		MidHigh:   30000.0,
		Treble:    10000.0,
		Timestamp: time.Now(),
	}

	tests := []struct {
		band          string
		expectedValue float64
	}{
		{"bass", 0.1},       // 100000 * 0.000001
		{"mid-low", 0.05},   // 50000 * 0.000001
		{"mid-high", 0.03},  // 30000 * 0.000001
		{"treble", 0.01},    // 10000 * 0.000001
	}

	for _, tt := range tests {
		t.Run(tt.band, func(t *testing.T) {
			provider := &mockAudioProvider{profile: profile}
			config := AudioModulatorConfig{
				FrequencyBand:  tt.band,
				MinPowerValue:  0.0,
				MaxPowerValue:  1.0,
				ScalingFactor:  0.000001,
				NoiseThreshold: 1000,
				Smoothing:      0.0,
			}

			configJSON, _ := json.Marshal(config)
			modulator, _ := newAudioModulator(configJSON, provider)

			power := modulator.GetPower(0)
			if power == nil {
				t.Fatal("GetPower returned nil")
			}

			if math.Abs(*power-tt.expectedValue) > 0.001 {
				t.Errorf("Expected power ~%f, got %f", tt.expectedValue, *power)
			}
		})
	}
}

func TestAudioModulator_NoiseThreshold(t *testing.T) {
	provider := &mockAudioProvider{
		profile: &AudioProfile{
			Bass:      5000.0, // Below threshold
			Timestamp: time.Now(),
		},
	}

	config := AudioModulatorConfig{
		FrequencyBand:  "bass",
		MinPowerValue:  0.0,
		MaxPowerValue:  1.0,
		ScalingFactor:  0.000001,
		NoiseThreshold: 10000.0, // Above bass value
		Smoothing:      0.0,
	}

	configJSON, _ := json.Marshal(config)
	modulator, _ := newAudioModulator(configJSON, provider)

	power := modulator.GetPower(0)
	if power == nil {
		t.Fatal("GetPower returned nil")
	}

	// Should be filtered to 0 by noise threshold
	if *power != 0.0 {
		t.Errorf("Expected power 0.0 (filtered by threshold), got %f", *power)
	}
}

func TestAudioModulator_Clamping(t *testing.T) {
	tests := []struct {
		name          string
		rawValue      float64
		scalingFactor float64
		minPower      float64
		maxPower      float64
		expected      float64
	}{
		{"below min", 1000.0, 0.00001, 0.5, 1.0, 0.5},   // 0.01 clamped to 0.5
		{"above max", 200000.0, 0.000001, 0.0, 0.1, 0.1}, // 0.2 clamped to 0.1
		{"within range", 50000.0, 0.000001, 0.0, 1.0, 0.05}, // 0.05 unchanged
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &mockAudioProvider{
				profile: &AudioProfile{
					Bass:      tt.rawValue,
					Timestamp: time.Now(),
				},
			}

			config := AudioModulatorConfig{
				FrequencyBand:  "bass",
				MinPowerValue:  tt.minPower,
				MaxPowerValue:  tt.maxPower,
				ScalingFactor:  tt.scalingFactor,
				NoiseThreshold: 0,
				Smoothing:      0.0,
			}

			configJSON, _ := json.Marshal(config)
			modulator, _ := newAudioModulator(configJSON, provider)

			power := modulator.GetPower(0)
			if power == nil {
				t.Fatal("GetPower returned nil")
			}

			if math.Abs(*power-tt.expected) > 0.001 {
				t.Errorf("Expected power ~%f, got %f", tt.expected, *power)
			}
		})
	}
}

func TestAudioModulator_Smoothing(t *testing.T) {
	provider := &mockAudioProvider{
		profile: &AudioProfile{
			Bass:      100000.0, // Scales to 0.1
			Timestamp: time.Now(),
		},
	}

	config := AudioModulatorConfig{
		FrequencyBand:  "bass",
		MinPowerValue:  0.0,
		MaxPowerValue:  1.0,
		ScalingFactor:  0.000001,
		NoiseThreshold: 0,
		Smoothing:      0.5, // 50% smoothing
	}

	configJSON, _ := json.Marshal(config)
	modulator, _ := newAudioModulator(configJSON, provider)

	// First call - no previous value
	power1 := modulator.GetPower(0)
	if power1 == nil {
		t.Fatal("First GetPower returned nil")
	}

	expected1 := 0.1
	if math.Abs(*power1-expected1) > 0.001 {
		t.Errorf("First call: expected %f, got %f", expected1, *power1)
	}

	// Update to new value
	provider.profile.Bass = 200000.0 // Scales to 0.2

	// Second call - should be smoothed
	power2 := modulator.GetPower(0)
	if power2 == nil {
		t.Fatal("Second GetPower returned nil")
	}

	// Smoothed = (1-0.5)*0.2 + 0.5*0.1 = 0.1 + 0.05 = 0.15
	expected2 := 0.15
	if math.Abs(*power2-expected2) > 0.001 {
		t.Errorf("Second call: expected ~%f, got %f", expected2, *power2)
	}
}

func TestAudioModulator_CustomWeight(t *testing.T) {
	provider := &mockAudioProvider{
		profile: &AudioProfile{Bass: 100000000.0},
	}

	customWeight := 2.5
	config := AudioModulatorConfig{
		FrequencyBand:  "bass",
		MinPowerValue:  0.0,
		MaxPowerValue:  1.0,
		ScalingFactor:  0.000001,
		NoiseThreshold: 0,
		BehaviorWeight: customWeight,
	}

	configJSON, _ := json.Marshal(config)
	modulator, _ := newAudioModulator(configJSON, provider)

	if modulator.Weight() != customWeight {
		t.Errorf("Expected weight %f, got %f", customWeight, modulator.Weight())
	}
}

func TestNormalizeFrequencyBand(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"bass", "bass"},
		{"Bass", "bass"},
		{"BASS", "bass"},
		{"mid-low", "mid-low"},
		{"mid_low", "mid-low"},
		{"midlow", "mid-low"},
		{"MidLow", "mid-low"},
		{"mid-high", "mid-high"},
		{"mid_high", "mid-high"},
		{"MidHigh", "mid-high"},
		{"treble", "treble"},
		{"Treble", "treble"},
		{"invalid", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeFrequencyBand(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeFrequencyBand(%q) = %q, expected %q",
					tt.input, result, tt.expected)
			}
		})
	}
}

func TestClamp(t *testing.T) {
	tests := []struct {
		name     string
		value    float64
		min      float64
		max      float64
		expected float64
	}{
		{"below min", -0.5, 0.0, 1.0, 0.0},
		{"above max", 1.5, 0.0, 1.0, 1.0},
		{"within range", 0.5, 0.0, 1.0, 0.5},
		{"at min", 0.0, 0.0, 1.0, 0.0},
		{"at max", 1.0, 0.0, 1.0, 1.0},
		{"NaN", math.NaN(), 0.0, 1.0, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := clamp(tt.value, tt.min, tt.max)
			if math.IsNaN(result) {
				t.Errorf("clamp returned NaN for input %v", tt.value)
			}
			if result != tt.expected {
				t.Errorf("clamp(%f, %f, %f) = %f, expected %f",
					tt.value, tt.min, tt.max, result, tt.expected)
			}
		})
	}
}

func BenchmarkAudioModulator_GetPower(b *testing.B) {
	provider := &mockAudioProvider{
		profile: &AudioProfile{
			Bass:      100000.0,
			MidLow:    50000.0,
			MidHigh:   30000.0,
			Treble:    10000.0,
			Timestamp: time.Now(),
		},
	}

	config := AudioModulatorConfig{
		FrequencyBand:  "bass",
		MinPowerValue:  0.0,
		MaxPowerValue:  1.0,
		ScalingFactor:  0.000001,
		NoiseThreshold: 1000,
		Smoothing:      0.3,
	}

	configJSON, _ := json.Marshal(config)
	modulator, _ := newAudioModulator(configJSON, provider)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = modulator.GetPower(time.Duration(i))
	}
}
