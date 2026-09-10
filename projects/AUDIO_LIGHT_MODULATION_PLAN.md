# Implementation Plan: Audio Stream Modulation for LED Brightness

## Development Guidelines

**IMPORTANT**: This is a multi-phase implementation. To enable easy rollback if issues arise:
- Commit after completing each file or logical unit of work
- Use descriptive commit messages referencing the phase (e.g., "Phase 1: Add AudioProvider interface")
- Test each phase before moving to the next
- If you need to stop and resume in a new context, refer back to this plan

---

## Executive Summary

This plan adds a new `AudioModulator` behavior that modulates LED brightness based on real-time audio frequency analysis (bass, mid-low, mid-high, treble). The implementation extends the existing behavior interface while maintaining backward compatibility with time-based behaviors.

---

## Architecture Overview

### Current State
```
Audio Pipeline (Isolated):
PRU1 → Sampler → Processor → Result (CSV output)

Light Pipeline (Time-based):
Time → Behavior → Power → LED PWM
```

### Proposed State
```
Audio Pipeline (Integrated):
PRU1 → Sampler → Processor → Result → Audio Profile Channel
                                            ↓
Light Pipeline:                             ↓
Time + Audio Profile → Behavior → Power → LED PWM
```

---

## Core Design Principles

1. **Non-breaking**: Existing time-based behaviors continue working unchanged
2. **Optional audio**: Audio modulation is opt-in via new behavior type
3. **Channel-based**: Audio data flows via Go channels (non-blocking)
4. **Configurable mapping**: Flexible frequency band → color mapping
5. **Graceful degradation**: If audio stops, behaviors maintain last known state or fall back to baseline

---

## Implementation Components

### 1. New Audio Profile Provider

**File**: `api/light/behavior/audio_provider.go`

```go
type AudioProfile struct {
    Bass     float64
    MidLow   float64
    MidHigh  float64
    Treble   float64
    Timestamp time.Time
}

type AudioProvider interface {
    GetLatestProfile() *AudioProfile
    Subscribe() <-chan AudioProfile
}

type channelAudioProvider struct {
    latestProfile atomic.Value // *AudioProfile
    subscribers   []chan AudioProfile
    mu            sync.RWMutex
}

func NewAudioProvider() *channelAudioProvider
func (p *channelAudioProvider) UpdateProfile(profile AudioProfile)
func (p *channelAudioProvider) GetLatestProfile() *AudioProfile
```

**Purpose**:
- Provides thread-safe access to latest audio data
- Decouples audio processing from behavior system
- Allows multiple behaviors to read same audio data

---

### 2. Audio Modulator Behavior

**File**: `api/light/behavior/audio_modulator.go`

```go
type AudioModulatorConfig struct {
    FrequencyBand    string   `json:"frequency_band"`    // "bass", "mid-low", "mid-high", "treble"
    MinPowerValue    float64  `json:"min_power_value"`   // 0.0-1.0
    MaxPowerValue    float64  `json:"max_power_value"`   // 0.0-1.0
    ScalingFactor    float64  `json:"scaling_factor"`    // From analysis (e.g., 0.000001)
    NoiseThreshold   float64  `json:"noise_threshold"`   // Filter baseline noise
    Smoothing        float64  `json:"smoothing"`         // 0.0-1.0, exponential smoothing
    FallbackPower    *float64 `json:"fallback_power"`    // When no audio data available
}

type AudioModulator struct {
    config        AudioModulatorConfig
    audioProvider AudioProvider
    lastValue     atomic.Value // *float64
    weight        float64
}

func (a *AudioModulator) GetPower(t time.Duration) *float64 {
    profile := a.audioProvider.GetLatestProfile()
    if profile == nil {
        return a.config.FallbackPower
    }

    // Extract frequency band value
    rawValue := a.extractFrequencyBand(profile)

    // Apply noise threshold
    if rawValue < a.config.NoiseThreshold {
        rawValue = 0
    }

    // Scale to 0-1 range
    scaledValue := rawValue * a.config.ScalingFactor

    // Clamp to min/max
    power := clamp(scaledValue, a.config.MinPowerValue, a.config.MaxPowerValue)

    // Apply smoothing
    if a.config.Smoothing > 0 {
        power = a.smooth(power)
    }

    return &power
}
```

**Key Features**:
- Reads from audio provider (non-blocking)
- Applies scaling factor from analysis (e.g., SNR recommendations)
- Filters noise using threshold
- Smooths transitions to prevent flickering
- Fallback when audio unavailable

**Configuration Example**:
```json
{
  "behavior_type": "audio_modulator",
  "color": "red",
  "config": {
    "frequency_band": "bass",
    "min_power_value": 0.0,
    "max_power_value": 1.0,
    "scaling_factor": 0.000001,
    "noise_threshold": 92565436,
    "smoothing": 0.3,
    "fallback_power": 0.1
  }
}
```

---

### 3. Behavior Type Extension

**File**: `api/light/behavior/types.go`

```diff
const (
    BreathingBehaviorType Type = iota
    FlashingBehaviorType
    FixedBehaviorType
    SkipperBehaviorType
    JoinerBehaviorType
+   AudioModulatorBehaviorType
)

func LookupBehavior(behaviorName string) Type {
    switch strings.ToLower(behaviorName) {
    case "breathing":
        return BreathingBehaviorType
    // ... other cases ...
+   case "audio_modulator", "audio":
+       return AudioModulatorBehaviorType
    default:
        return BreathingBehaviorType
    }
}

-func CreateBehavior(t Type, cfg json.RawMessage) (Behavior, error) {
+func CreateBehavior(t Type, cfg json.RawMessage, audioProvider AudioProvider) (Behavior, error) {
    switch t {
    // ... existing cases ...
+   case AudioModulatorBehaviorType:
+       return newAudioModulator(cfg, audioProvider)
    default:
        return nil, fmt.Errorf("unknown ActiveBehavior type %d", t)
    }
}
```

---

### 4. Audio Manager Integration

**File**: `api/audio/manager.go`

```diff
type Manager struct {
    sampler         sampling.Sampler
    processor       *processing.Processor
    resultChannel   chan processing.Result
+   audioProvider   behavior.AudioProvider  // New field
    logger          *zap.Logger
}

+func (m *Manager) StreamToLights(ctx context.Context) {
+    for {
+        select {
+        case <-ctx.Done():
+            return
+        case result := <-m.resultChannel:
+            profile := behavior.AudioProfile{
+                Bass:     result.Profile.Bass,
+                MidLow:   result.Profile.MidLow,
+                MidHigh:  result.Profile.MidHigh,
+                Treble:   result.Profile.Treble,
+                Timestamp: time.Now(),
+            }
+            m.audioProvider.UpdateProfile(profile)
+        }
+    }
+}
```

---

### 5. Light Controller Extension

**File**: `api/light/controller.go`

```diff
type Controller struct {
    eventChannel  chan event
    manager       *Manager
+   audioProvider behavior.AudioProvider
    logger        *zap.Logger
}

+func NewController(audioProvider behavior.AudioProvider, logger *zap.Logger) *Controller {
    return &Controller{
        eventChannel:  make(chan event, 10),
+       audioProvider: audioProvider,
        logger:        logger,
    }
}
```

---

### 6. New HTTP API Endpoints

**File**: `api/light/routes.go`

```go
// Existing routes remain unchanged
POST /api/lights/behavior/register
POST /api/lights/on
POST /api/lights/off

// New audio-specific routes
POST /api/lights/audio/start    → Start audio streaming to lights
POST /api/lights/audio/stop     → Stop audio streaming
GET  /api/lights/audio/status   → Get current audio profile
GET  /api/lights/audio/config   → Get recommended scaling factors
```

**Handler Implementation**: `api/light/audio_handler.go`

```go
func handleAudioStart(controller *Controller, audioManager *audio.Manager) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Start audio sampling and streaming
        ctx := context.Background()
        go audioManager.StreamToLights(ctx)
        // ...
    }
}

func handleAudioConfig(w http.ResponseWriter, r *http.Request) {
    // Return recommended config from analysis
    config := AudioScalingConfig{
        Bass: AudioBandConfig{
            ScalingFactor:  0.000001,
            NoiseThreshold: 92565436,
        },
        // ... other bands from analysis
    }
    json.NewEncoder(w).Encode(config)
}
```

---

### 7. Processing Profile Extension

**File**: `api/audio/processing/profile.go`

Currently outputs only 3 bands (Bass, Mid, Treble). Extend to 4 bands to match analysis:

```diff
type Profile struct {
    Bass    float64
-   Mid     float64
+   MidLow  float64
+   MidHigh float64
    Treble  float64
}
```

Update profiler.go cutoff logic to split mid into mid-low and mid-high based on frequency ranges from analysis.

---

## Configuration Examples

### Example 1: Bass-driven Red LEDs

```json
{
  "behaviors": [
    {
      "behavior_type": "audio_modulator",
      "color": "red",
      "config": {
        "frequency_band": "bass",
        "min_power_value": 0.1,
        "max_power_value": 1.0,
        "scaling_factor": 0.000001,
        "noise_threshold": 92565436,
        "smoothing": 0.2,
        "fallback_power": 0.1
      }
    }
  ]
}
```

### Example 2: Full RGB Audio Visualization

```json
{
  "behaviors": [
    {
      "behavior_type": "audio_modulator",
      "color": "red",
      "config": {
        "frequency_band": "bass",
        "scaling_factor": 0.000001,
        "noise_threshold": 92565436,
        "smoothing": 0.3
      }
    },
    {
      "behavior_type": "audio_modulator",
      "color": "green",
      "config": {
        "frequency_band": "mid-low",
        "scaling_factor": 0.000001,
        "noise_threshold": 18541891,
        "smoothing": 0.3
      }
    },
    {
      "behavior_type": "audio_modulator",
      "color": "blue",
      "config": {
        "frequency_band": "mid-high",
        "scaling_factor": 0.000002,
        "noise_threshold": 10913229,
        "smoothing": 0.3
      }
    },
    {
      "behavior_type": "audio_modulator",
      "color": "white",
      "config": {
        "frequency_band": "treble",
        "scaling_factor": 0.000027,
        "noise_threshold": 2258769,
        "smoothing": 0.2
      }
    }
  ]
}
```

### Example 3: Mixed Audio + Time-based Behaviors

```json
{
  "behaviors": [
    {
      "behavior_type": "audio_modulator",
      "color": "red",
      "config": {
        "frequency_band": "bass",
        "scaling_factor": 0.000001,
        "smoothing": 0.3
      }
    },
    {
      "behavior_type": "breathing",
      "color": "red",
      "config": {
        "Duration": "2s",
        "min_power_value": 0.0,
        "max_power_value": 0.3
      }
    }
  ]
}
```

This combines audio-reactive bass with a gentle breathing effect (weighted average).

---

## Implementation Steps

### Phase 1: Audio Provider Foundation
1. Create `audio_provider.go` with thread-safe profile storage
2. Add `AudioProvider` interface and implementation
3. Unit tests for concurrent access patterns

**Files to create/modify**:
- `api/light/behavior/audio_provider.go` (new)
- `api/light/behavior/audio_provider_test.go` (new)

**Commit after**: Each file is complete and tests pass

---

### Phase 2: Audio Modulator Behavior
1. Implement `AudioModulator` struct and config
2. Add frequency band extraction logic
3. Implement scaling, threshold, and smoothing
4. Add to behavior factory in `types.go`
5. Unit tests with mock audio provider

**Files to create/modify**:
- `api/light/behavior/audio_modulator.go` (new)
- `api/light/behavior/audio_modulator_test.go` (new)
- `api/light/behavior/types.go` (modify - add enum + factory)

**Commit after**: Each file is complete and tests pass

---

### Phase 3: Audio-Light Integration
1. Modify `audio.Manager` to accept `AudioProvider`
2. Add `StreamToLights()` method to audio manager
3. Update `light.Controller` to accept and pass `AudioProvider`
4. Modify behavior manager to support audio provider

**Files to modify**:
- `api/audio/manager.go`
- `api/light/controller.go`
- `api/light/behavior/manager.go`
- `api/light/models.go` (add audio modulator config unmarshaling)

**Commit after**: Each logical unit (e.g., audio manager changes, then controller changes)

---

### Phase 4: HTTP API
1. Create new audio handlers
2. Add routes for audio start/stop/status/config
3. Update route registration

**Files to create/modify**:
- `api/light/audio_handler.go` (new)
- `api/light/routes.go` (modify - add audio routes)
- `api/infrastructure/server/routes.go` (modify - register audio handlers)

**Commit after**: Each file is complete

---

### Phase 5: Processing Enhancement
1. Split Profile.Mid into MidLow and MidHigh
2. Update profiler cutoff logic
3. Update existing CLI commands to use new structure

**Files to modify**:
- `api/audio/processing/profile.go`
- `api/audio/processing/profiler.go`
- `api/audio/cmd.go` (CLI commands)

**Commit after**: Each file is complete and existing functionality still works

---

### Phase 6: Configuration & Documentation
1. Add analysis-based scaling config endpoint
2. Create example configurations
3. Integration tests

**Files to create/modify**:
- `api/light/audio_config.go` (new - recommended settings)
- Integration tests

**Commit after**: Complete phase

---

## Testing Strategy

### Unit Tests
- `AudioProvider`: Concurrent read/write scenarios
- `AudioModulator`:
  - Scaling and clamping logic
  - Noise threshold filtering
  - Smoothing algorithm
  - Fallback behavior
- Behavior factory with audio provider injection

### Integration Tests
1. Audio pipeline → AudioProvider → AudioModulator → LED output
2. Multiple audio modulators on different colors
3. Mixed audio + time-based behaviors
4. Audio stream start/stop

### Manual Testing Scenarios
1. Play bagpipes.wav → verify red LED responds to bass
2. Play crazy_frog.wav → verify RGB visualization
3. Stop audio → verify graceful fallback
4. Network latency → verify smoothing prevents flicker

---

## Performance Considerations

### Timing Analysis
- Audio sampling: ~40 Hz (25ms period)
- LED update loop: 1250 Hz (800μs period)
- **Implication**: LED updates ~31 times per audio sample

**Solution**: Cache audio profile in `AudioProvider` with atomic access

### Memory
- Audio profile: ~40 bytes
- Minimal overhead: One shared profile for all behaviors

### CPU
- AudioModulator.GetPower() is O(1) - just reads cached profile
- No FFT in lighting thread - that's in audio manager

---

## Backward Compatibility

✅ **Existing behaviors unchanged**: Breathing, flashing, fixed, skipper, joiner continue working
✅ **Optional audio**: System works without audio streaming
✅ **API unchanged**: Existing `/api/lights/*` endpoints remain identical
✅ **Configuration compatible**: Old behavior configs still valid

---

## Future Enhancements

### Phase 7+ (Future)
1. **Beat detection behavior**: Trigger flashes on beat
2. **Frequency range behaviors**: Custom Hz ranges (not just bass/mid/treble)
3. **Audio-driven patterns**: Change pattern based on audio characteristics
4. **Multi-song learning**: Adapt scaling factors based on song genre
5. **Web UI**: Visual configuration of frequency → color mappings

---

## Summary of Files

### New Files (6)
1. `api/light/behavior/audio_provider.go` - Audio data distribution
2. `api/light/behavior/audio_modulator.go` - Audio-reactive behavior
3. `api/light/audio_handler.go` - HTTP handlers for audio control
4. `api/light/audio_config.go` - Recommended scaling configuration
5. `api/light/behavior/audio_provider_test.go` - Unit tests
6. `api/light/behavior/audio_modulator_test.go` - Unit tests

### Modified Files (10)
1. `api/light/behavior/types.go` - Add AudioModulatorBehaviorType
2. `api/light/behavior/manager.go` - Accept AudioProvider
3. `api/light/controller.go` - Inject AudioProvider
4. `api/light/routes.go` - Register audio routes
5. `api/light/models.go` - Audio config unmarshaling
6. `api/audio/manager.go` - Add StreamToLights method
7. `api/audio/processing/profile.go` - Split Mid into MidLow/MidHigh
8. `api/audio/processing/profiler.go` - Update frequency cutoffs
9. `api/audio/cmd.go` - Update for new profile structure
10. `api/infrastructure/server/routes.go` - Wire up dependencies

---

## Current Status

**Last Updated**: 2025-12-14

**Phase Status**:
- [ ] Phase 1: Audio Provider Foundation
- [ ] Phase 2: Audio Modulator Behavior
- [ ] Phase 3: Audio-Light Integration
- [ ] Phase 4: HTTP API
- [ ] Phase 5: Processing Enhancement
- [ ] Phase 6: Configuration & Documentation

**Notes**: Update this section as work progresses to track current state.

---

## Recommended Scaling Factors (from Analysis)

Based on analysis of bagpipes.csv, crazy_frog.csv, and christmas.csv:

| Band | Scaling Factor | Noise Threshold |
|------|----------------|-----------------|
| Bass | 0.000001 | 92565436 |
| Mid-Low | 0.000001 | 18541891 |
| Mid-High | 0.000002 | 10913229 |
| Treble | 0.000027 | 2258769 |

These values scale the raw magnitude values to 0-255 range and filter baseline noise.
