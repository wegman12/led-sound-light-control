# Audio-Reactive LED Configuration Examples

This directory contains example JSON configurations for audio-reactive lighting behaviors.

## Configuration Files

### 1. bass-driven-red.json
**Description**: Simple bass-driven red LED visualization
**Use Case**: Single color LED that pulses with bass/kick drums
**Features**:
- Red LED brightness modulated by bass frequencies (0-150 Hz)
- Noise threshold filters background noise
- Smoothing factor (0.3) prevents flickering
- Falls back to off when no audio is present

### 2. full-rgb-visualization.json
**Description**: Full RGB audio spectrum visualization
**Use Case**: Multi-color visualization showing all frequency bands
**Features**:
- **Red**: Bass frequencies (0-150 Hz) - kick drums, bass guitar
- **Green**: Mid-low frequencies (150-1000 Hz) - vocals, guitars
- **Blue**: Mid-high frequencies (1000-2000 Hz) - snares, vocals
- **White**: Treble frequencies (2000+ Hz) - cymbals, hi-hats
- Each band has calibrated scaling factors from audio analysis

### 3. mixed-audio-breathing.json
**Description**: Combines audio-reactive and time-based behaviors
**Use Case**: Ambient lighting with audio accents
**Features**:
- Red LED has two behaviors:
  - Bass-reactive modulation (overlays on top)
  - Slow breathing effect (3-second period, base layer)
- Blue LED responds to treble for sparkle effects
- Demonstrates behavior averaging/mixing

### 4. treble-sparkle.json
**Description**: Static color with treble-triggered white sparkle
**Use Case**: Ambient mood lighting with high-frequency accents
**Features**:
- Fixed RGB creates ambient purple color (R:0.3, G:0.2, B:0.4)
- White LED flashes on treble frequencies (cymbals, hi-hats)
- Low smoothing (0.1) allows quick sparkle response

## Usage

### Via HTTP API

```bash
# Start audio streaming
curl -X POST http://localhost:8080/api/lights/audio/start

# Register a configuration
curl -X POST http://localhost:8080/api/lights/behavior/register \
  -H "Content-Type: application/json" \
  -d @examples/audio-reactive/full-rgb-visualization.json

# Turn lights on
curl -X POST http://localhost:8080/api/lights/on

# Check audio status
curl http://localhost:8080/api/lights/audio/status

# Get recommended configuration values
curl http://localhost:8080/api/lights/audio/config
```

### Configuration Parameters

#### Audio Modulator Behavior

```json
{
  "color": "red|green|blue|white",
  "behavior_type": "audio_modulator",
  "config": {
    "frequency_band": "bass|mid-low|mid-high|treble",
    "min_power_value": 0.0,        // Minimum LED brightness (0.0-1.0)
    "max_power_value": 1.0,        // Maximum LED brightness (0.0-1.0)
    "scaling_factor": 0.000001,    // From audio analysis
    "noise_threshold": 92565436,   // Filter background noise
    "smoothing": 0.3,              // 0.0=no smoothing, 1.0=maximum smoothing
    "fallback_power": 0.0          // Brightness when no audio (null = no fallback)
  }
}
```

#### Frequency Bands
- **bass**: 0-150 Hz (kick drums, bass guitar, sub-bass)
- **mid-low**: 150-1000 Hz (bass guitar, lower vocals, rhythm guitars)
- **mid-high**: 1000-2000 Hz (vocals, snare drums, lead guitars)
- **treble**: 2000+ Hz (cymbals, hi-hats, high frequencies)

#### Scaling Factors (from analysis)
Based on analysis of bagpipes.csv, crazy_frog.csv, and christmas.csv:

| Band     | Scaling Factor | Noise Threshold |
|----------|----------------|-----------------|
| Bass     | 0.000001       | 92565436        |
| Mid-Low  | 0.000001       | 18541891        |
| Mid-High | 0.000002       | 10913229        |
| Treble   | 0.000027       | 2258769         |

#### Smoothing Recommendations
- **0.0-0.2**: Fast response, may flicker with noisy audio
- **0.3-0.5**: Balanced, smooth but responsive (recommended)
- **0.6-0.8**: Very smooth, slower response
- **0.9-1.0**: Extremely smooth, may lag behind audio

## Testing

### Manual Testing
```bash
# Test with different music genres
# 1. Play bagpipes.wav → verify bass response
# 2. Play crazy_frog.wav → verify full spectrum
# 3. Play christmas.wav → verify balanced response
# 4. Stop audio → verify fallback behavior
```

### Tuning Tips
1. **Too dim**: Increase `scaling_factor` or decrease `min_power_value`
2. **Too bright**: Decrease `scaling_factor` or decrease `max_power_value`
3. **Flickering**: Increase `smoothing` value
4. **Not responsive**: Decrease `smoothing` or increase `scaling_factor`
5. **Always on**: Decrease `noise_threshold`
6. **Never on**: Increase `scaling_factor` or decrease `noise_threshold`

## Implementation Details

All configurations use the audio-reactive lighting pipeline:

```
PRU/ADC Audio Sampling (40 kHz)
    ↓
FFT Processing (4 frequency bands)
    ↓
AudioProvider (thread-safe cache)
    ↓
AudioModulator Behaviors (per LED color)
    ↓
LED Hardware (PWM control)
```

For more details, see `projects/AUDIO_LIGHT_MODULATION_PLAN.md`.
