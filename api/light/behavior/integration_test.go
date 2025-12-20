package behavior

import (
	"encoding/json"
	"testing"
	"time"
)

// TestAudioPipelineIntegration tests the full audio-reactive pipeline:
// AudioProvider → Multiple AudioModulator behaviors → Power output
func TestAudioPipelineIntegration(t *testing.T) {
	// Setup: Create AudioProvider
	provider := NewAudioProvider()

	// Create audio modulator behaviors for different frequency bands
	bassConfig := json.RawMessage(`{
		"frequency_band": "bass",
		"min_power_value": 0.1,
		"max_power_value": 1.0,
		"max_magnitude": 1000000,
		"noise_threshold": 92565436,
		"smoothing": 0.3,
		"fallback_power": 0.0
	}`)

	midLowConfig := json.RawMessage(`{
		"frequency_band": "mid-low",
		"min_power_value": 0.1,
		"max_power_value": 1.0,
		"max_magnitude": 1000000,
		"noise_threshold": 18541891,
		"smoothing": 0.3,
		"fallback_power": 0.0
	}`)

	trebleConfig := json.RawMessage(`{
		"frequency_band": "treble",
		"min_power_value": 0.0,
		"max_power_value": 0.8,
		"max_magnitude": 37037,
		"noise_threshold": 2258769,
		"smoothing": 0.1,
		"fallback_power": 0.0
	}`)

	// Create behaviors
	bassBehavior, err := CreateBehavior(AudioModulatorBehaviorType, bassConfig, provider)
	if err != nil {
		t.Fatalf("Failed to create bass behavior: %v", err)
	}

	midLowBehavior, err := CreateBehavior(AudioModulatorBehaviorType, midLowConfig, provider)
	if err != nil {
		t.Fatalf("Failed to create mid-low behavior: %v", err)
	}

	trebleBehavior, err := CreateBehavior(AudioModulatorBehaviorType, trebleConfig, provider)
	if err != nil {
		t.Fatalf("Failed to create treble behavior: %v", err)
	}

	// Test 1: No audio profile (should return fallback values)
	bassPower := bassBehavior.GetPower(0)
	if bassPower == nil {
		t.Error("Expected bass power with fallback, got nil")
	} else if *bassPower != 0.0 {
		t.Errorf("Expected bass fallback power 0.0, got %f", *bassPower)
	}

	// Test 2: Update with strong bass signal
	strongBassProfile := AudioProfile{
		Bass:      150000000.0, // Above noise threshold
		MidLow:    20000000.0,
		MidHigh:   15000000.0,
		Treble:    2000000.0, // Below noise threshold (2258769)
		Timestamp: time.Now(),
	}
	provider.UpdateProfile(strongBassProfile)

	// Allow time for smoothing to take effect
	time.Sleep(10 * time.Millisecond)

	// Verify bass behavior responds
	bassPower = bassBehavior.GetPower(0)
	if bassPower == nil {
		t.Error("Expected bass power, got nil")
	} else if *bassPower < 0.1 {
		t.Errorf("Expected bass power > 0.1 with strong signal, got %f", *bassPower)
	}

	// Verify mid-low behavior responds
	midLowPower := midLowBehavior.GetPower(0)
	if midLowPower == nil {
		t.Error("Expected mid-low power, got nil")
	} else if *midLowPower < 0.1 {
		t.Errorf("Expected mid-low power > 0.1 with signal above noise threshold, got %f", *midLowPower)
	}

	// Verify treble is low (below noise threshold)
	treblePower := trebleBehavior.GetPower(0)
	if treblePower == nil {
		t.Error("Expected treble power, got nil")
	} else if *treblePower > 0.1 {
		t.Errorf("Expected treble power near 0 with weak signal, got %f", *treblePower)
	}

	// Test 3: Update with strong treble signal
	strongTrebleProfile := AudioProfile{
		Bass:      100000000.0,
		MidLow:    20000000.0,
		MidHigh:   15000000.0,
		Treble:    10000000.0, // Above noise threshold
		Timestamp: time.Now(),
	}
	provider.UpdateProfile(strongTrebleProfile)

	time.Sleep(10 * time.Millisecond)

	// Verify treble behavior now responds
	treblePower = trebleBehavior.GetPower(0)
	if treblePower == nil {
		t.Error("Expected treble power, got nil")
	} else if *treblePower < 0.1 {
		t.Errorf("Expected treble power > 0.1 with strong signal, got %f", *treblePower)
	}
}

// TestMultipleAudioModulatorsOnDifferentColors tests multiple audio modulators
// controlling different LED colors simultaneously
func TestMultipleAudioModulatorsOnDifferentColors(t *testing.T) {
	provider := NewAudioProvider()

	// Create RGB audio visualization: Red=Bass, Green=MidLow, Blue=Treble
	configs := map[string]string{
		"red": `{
			"frequency_band": "bass",
			"min_power_value": 0.1,
			"max_power_value": 1.0,
			"max_magnitude": 1000000,
			"noise_threshold": 92565436,
			"smoothing": 0.0
		}`,
		"green": `{
			"frequency_band": "mid-low",
			"min_power_value": 0.1,
			"max_power_value": 1.0,
			"max_magnitude": 1000000,
			"noise_threshold": 18541891,
			"smoothing": 0.0
		}`,
		"blue": `{
			"frequency_band": "treble",
			"min_power_value": 0.0,
			"max_power_value": 1.0,
			"max_magnitude": 37037,
			"noise_threshold": 2258769,
			"smoothing": 0.0
		}`,
	}

	behaviors := make(map[string]Behavior)
	for color, configStr := range configs {
		behavior, err := CreateBehavior(AudioModulatorBehaviorType, json.RawMessage(configStr), provider)
		if err != nil {
			t.Fatalf("Failed to create %s behavior: %v", color, err)
		}
		behaviors[color] = behavior
	}

	// Test scenario: Bass-heavy music (kick drum hit)
	bassHeavyProfile := AudioProfile{
		Bass:      200000000.0, // Strong bass
		MidLow:    25000000.0,  // Moderate mid
		MidHigh:   12000000.0,  // Weak mid-high
		Treble:    2000000.0,   // Very weak treble (below noise threshold)
		Timestamp: time.Now(),
	}
	provider.UpdateProfile(bassHeavyProfile)

	// Check each color responds appropriately
	redPower := behaviors["red"].GetPower(0)
	greenPower := behaviors["green"].GetPower(0)
	bluePower := behaviors["blue"].GetPower(0)

	if redPower == nil || *redPower < 0.5 {
		t.Errorf("Expected strong red (bass) response, got %v", redPower)
	}

	if greenPower == nil || *greenPower < 0.1 {
		t.Errorf("Expected moderate green (mid-low) response, got %v", greenPower)
	}

	if bluePower == nil || *bluePower > 0.1 {
		t.Errorf("Expected weak blue (treble) response, got %v", bluePower)
	}

	// Test scenario: Treble-heavy music (cymbal crash)
	trebleHeavyProfile := AudioProfile{
		Bass:      100000000.0, // Moderate bass
		MidLow:    20000000.0,  // Moderate mid
		MidHigh:   15000000.0,  // Moderate mid-high
		Treble:    20000000.0,  // Very strong treble
		Timestamp: time.Now(),
	}
	provider.UpdateProfile(trebleHeavyProfile)

	bluePower = behaviors["blue"].GetPower(0)
	if bluePower == nil || *bluePower < 0.5 {
		t.Errorf("Expected strong blue (treble) response with cymbal crash, got %v", bluePower)
	}
}

// TestMixedAudioAndTimeBasedBehaviors tests combining audio-reactive behaviors
// with time-based behaviors (e.g., breathing + audio modulation)
func TestMixedAudioAndTimeBasedBehaviors(t *testing.T) {
	provider := NewAudioProvider()

	// Create breathing behavior (time-based)
	breathingConfig := json.RawMessage(`{
		"period_ms": 1000,
		"min_power_value": 0.0,
		"max_power_value": 0.5
	}`)
	breathingBehavior, err := CreateBehavior(BreathingBehaviorType, breathingConfig, nil)
	if err != nil {
		t.Fatalf("Failed to create breathing behavior: %v", err)
	}

	// Create audio modulator behavior
	audioConfig := json.RawMessage(`{
		"frequency_band": "bass",
		"min_power_value": 0.2,
		"max_power_value": 1.0,
		"max_magnitude": 1000000,
		"noise_threshold": 92565436,
		"smoothing": 0.0,
		"fallback_power": 0.0
	}`)
	audioBehavior, err := CreateBehavior(AudioModulatorBehaviorType, audioConfig, provider)
	if err != nil {
		t.Fatalf("Failed to create audio behavior: %v", err)
	}

	// Test: Breathing provides base layer, audio modulates on top
	// In a real system, behaviors would be averaged by the manager

	// At t=0ms, breathing should be at minimum (0.0)
	breathingPower0 := breathingBehavior.GetPower(0)
	if breathingPower0 == nil || *breathingPower0 > 0.1 {
		t.Errorf("Expected breathing at ~0.0 at t=0ms, got %v", breathingPower0)
	}

	// At t=250ms, breathing should be rising (period=1000ms, 1/4 through cycle)
	breathingPower250 := breathingBehavior.GetPower(250 * time.Millisecond)
	if breathingPower250 == nil || *breathingPower250 < 0.1 || *breathingPower250 > 0.4 {
		t.Errorf("Expected breathing at ~0.25 at t=250ms, got %v", breathingPower250)
	}

	// With no audio, audio behavior should return fallback or stay low
	audioPowerNoSignal := audioBehavior.GetPower(0)
	if audioPowerNoSignal == nil {
		t.Error("Expected audio power (even if low), got nil")
	}

	// Update with strong bass
	strongBassProfile := AudioProfile{
		Bass:      150000000.0,
		MidLow:    20000000.0,
		MidHigh:   15000000.0,
		Treble:    3000000.0,
		Timestamp: time.Now(),
	}
	provider.UpdateProfile(strongBassProfile)

	audioPowerWithSignal := audioBehavior.GetPower(0)
	if audioPowerWithSignal == nil || *audioPowerWithSignal < 0.3 {
		t.Errorf("Expected strong audio response, got %v", audioPowerWithSignal)
	}

	// In a complete system, both behaviors would contribute to the final LED power
	// The manager would average them: final_power = (breathing + audio) / 2
}

// TestAudioBehaviorWithNilAudioProvider verifies that creating an AudioModulator
// without an AudioProvider fails as expected
func TestAudioBehaviorWithNilAudioProvider(t *testing.T) {
	config := json.RawMessage(`{
		"frequency_band": "bass",
		"min_power_value": 0.0,
		"max_power_value": 1.0,
		"max_magnitude": 1000000,
		"noise_threshold": 92565436
	}`)

	_, err := CreateBehavior(AudioModulatorBehaviorType, config, nil)
	if err == nil {
		t.Error("Expected error when creating audio_modulator with nil AudioProvider, got nil")
	}
}

// TestAudioBehaviorSmoothingOverTime tests that the smoothing algorithm
// gradually transitions between audio values
func TestAudioBehaviorSmoothingOverTime(t *testing.T) {
	provider := NewAudioProvider()

	// High smoothing factor
	config := json.RawMessage(`{
		"frequency_band": "bass",
		"min_power_value": 0.0,
		"max_power_value": 1.0,
		"max_magnitude": 1000000,
		"noise_threshold": 0,
		"smoothing": 0.9
	}`)

	behavior, err := CreateBehavior(AudioModulatorBehaviorType, config, provider)
	if err != nil {
		t.Fatalf("Failed to create behavior: %v", err)
	}

	// Start with low bass
	provider.UpdateProfile(AudioProfile{
		Bass:      1000000.0,
		MidLow:    1000000.0,
		MidHigh:   1000000.0,
		Treble:    1000000.0,
		Timestamp: time.Now(),
	})

	// Get initial power
	power1 := behavior.GetPower(0)
	if power1 == nil {
		t.Fatal("Expected power1, got nil")
	}

	// Update to high bass
	provider.UpdateProfile(AudioProfile{
		Bass:      100000000.0,
		MidLow:    1000000.0,
		MidHigh:   1000000.0,
		Treble:    1000000.0,
		Timestamp: time.Now(),
	})

	// Get power immediately after update
	power2 := behavior.GetPower(0)
	if power2 == nil {
		t.Fatal("Expected power2, got nil")
	}

	// With high smoothing, power2 should not jump immediately to target
	// It should be closer to power1 than to the target value
	if *power2 > *power1*1.5 {
		t.Errorf("Expected smoothing to prevent large jump, but power jumped from %f to %f", *power1, *power2)
	}

	// Get power again (smoothing should continue)
	power3 := behavior.GetPower(0)
	if power3 == nil {
		t.Fatal("Expected power3, got nil")
	}

	// power3 should be >= power2 (continuing to rise toward target)
	if *power3 < *power2 {
		t.Errorf("Expected power to continue rising with smoothing, got %f -> %f", *power2, *power3)
	}
}

// BenchmarkAudioPipelineThroughput measures the throughput of the audio pipeline
func BenchmarkAudioPipelineThroughput(b *testing.B) {
	provider := NewAudioProvider()

	config := json.RawMessage(`{
		"frequency_band": "bass",
		"min_power_value": 0.0,
		"max_power_value": 1.0,
		"max_magnitude": 1000000,
		"noise_threshold": 92565436,
		"smoothing": 0.3
	}`)

	behavior, err := CreateBehavior(AudioModulatorBehaviorType, config, provider)
	if err != nil {
		b.Fatalf("Failed to create behavior: %v", err)
	}

	profile := AudioProfile{
		Bass:      150000000.0,
		MidLow:    20000000.0,
		MidHigh:   15000000.0,
		Treble:    3000000.0,
		Timestamp: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		provider.UpdateProfile(profile)
		_ = behavior.GetPower(0)
	}
}
