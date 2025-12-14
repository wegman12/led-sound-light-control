package audio

import "time"

// SoundProfile represents the current audio frequency analysis
type SoundProfile struct {
	Timestamp  time.Time `json:"timestamp"`
	FFTCount   uint32    `json:"fft_count"`
	FFTRate    float64   `json:"fft_rate"`     // FFTs per second
	FFTTimeMs  float64   `json:"fft_time_ms"`  // FFT processing time in milliseconds

	// Magnitude sums for each frequency band
	BassSum    uint32 `json:"bass_sum"`     // Bass total magnitude (0-bass_max_hz)
	MidLowSum  uint32 `json:"mid_low_sum"`  // Mid-low total magnitude
	MidHighSum uint32 `json:"mid_high_sum"` // Mid-high total magnitude
	TrebleSum  uint32 `json:"treble_sum"`   // Treble total magnitude

	// Average magnitudes per bin for each frequency band
	BassAvg    uint32 `json:"bass_avg"`     // Bass average magnitude per bin
	MidLowAvg  uint32 `json:"mid_low_avg"`  // Mid-low average magnitude per bin
	MidHighAvg uint32 `json:"mid_high_avg"` // Mid-high average magnitude per bin
	TrebleAvg  uint32 `json:"treble_avg"`   // Treble average magnitude per bin
}

// PRUStatus represents the PRU1 audio sampling status
type PRUStatus struct {
	Status        uint32 `json:"status"`          // PRU status code
	TotalSamples  uint32 `json:"total_samples"`   // Total samples collected
	BufferCount   uint32 `json:"buffer_count"`    // Number of buffers processed
	ADCTimeouts   uint32 `json:"adc_timeouts"`    // ADC timeout errors
	FFTSkipped    uint32 `json:"fft_skipped"`     // FFTs skipped due to timing
	FFTEnabled    bool   `json:"fft_enabled"`     // FFT processing enabled
	BassMaxHz     uint32 `json:"bass_max_hz"`     // Bass frequency boundary
	MidLowMaxHz   uint32 `json:"mid_low_max_hz"`  // Mid-low frequency boundary
	MidHighMaxHz  uint32 `json:"mid_high_max_hz"` // Mid-high frequency boundary
	SampleRateHz  uint32 `json:"sample_rate_hz"`  // Sampling rate (40 kHz)
}

// FrequencyBands represents configurable frequency band boundaries
type FrequencyBands struct {
	BassMax    uint32 `json:"bass_max"`     // Bass upper limit (Hz)
	MidLowMax  uint32 `json:"mid_low_max"`  // Mid-low upper limit (Hz)
	MidHighMax uint32 `json:"mid_high_max"` // Mid-high upper limit (Hz)
}
