package sound

import (
	"time"
)

// Decimator converts high-rate samples to a precise lower rate using timing-based selection
type Decimator struct {
	inputRate  int     // Input sample rate (e.g., 200000 Hz)
	outputRate int     // Desired output rate (e.g., 48000 Hz)
	nextTime   float64 // Next sample time in seconds
	startTime  time.Time
}

// NewDecimator creates a decimator for converting from inputRate to outputRate
func NewDecimator(inputRate, outputRate int) *Decimator {
	return &Decimator{
		inputRate:  inputRate,
		outputRate: outputRate,
		nextTime:   0.0,
		startTime:  time.Now(),
	}
}

// Reset resets the decimator timing
func (d *Decimator) Reset() {
	d.nextTime = 0.0
	d.startTime = time.Now()
}

// Decimate takes high-rate samples and returns samples at the target rate
// Uses timing-based selection to maintain precise output rate
func (d *Decimator) Decimate(samples []uint16) []uint16 {
	if len(samples) == 0 {
		return []uint16{}
	}

	// Calculate time per input sample
	inputInterval := 1.0 / float64(d.inputRate)

	output := make([]uint16, 0, len(samples)/4) // Estimate capacity

	// Process each input sample
	for i, sample := range samples {
		currentTime := float64(i) * inputInterval

		// Check if this sample is at or past the next output time
		if currentTime >= d.nextTime {
			output = append(output, sample)
			d.nextTime += inputInterval
		}
	}

	// Adjust nextTime for next batch (accounting for samples consumed)
	batchDuration := float64(len(samples)) * inputInterval
	d.nextTime -= batchDuration

	return output
}

// SimpleDecimator uses basic downsampling (every Nth sample)
// Less precise but simpler - use for testing
type SimpleDecimator struct {
	ratio    int
	position int
}

// NewSimpleDecimator creates a simple decimator that takes every Nth sample
func NewSimpleDecimator(inputRate, outputRate int) *SimpleDecimator {
	ratio := inputRate / outputRate
	if ratio < 1 {
		ratio = 1
	}
	return &SimpleDecimator{
		ratio:    ratio,
		position: 0,
	}
}

// Decimate takes every Nth sample based on the ratio
func (d *SimpleDecimator) Decimate(samples []uint16) []uint16 {
	if len(samples) == 0 {
		return []uint16{}
	}

	output := make([]uint16, 0, len(samples)/d.ratio+1)

	for i, sample := range samples {
		if (d.position+i)%d.ratio == 0 {
			output = append(output, sample)
		}
	}

	d.position = (d.position + len(samples)) % d.ratio

	return output
}

// AntiAliasDecimator uses a simple low-pass filter before decimation
// to prevent aliasing
type AntiAliasDecimator struct {
	ratio      int
	position   int
	filterSize int
	buffer     []uint16
}

// NewAntiAliasDecimator creates a decimator with anti-aliasing filter
func NewAntiAliasDecimator(inputRate, outputRate int) *AntiAliasDecimator {
	ratio := inputRate / outputRate
	if ratio < 1 {
		ratio = 1
	}

	// Use a small averaging window (moving average filter)
	filterSize := ratio / 2
	if filterSize < 1 {
		filterSize = 1
	}

	return &AntiAliasDecimator{
		ratio:      ratio,
		position:   0,
		filterSize: filterSize,
		buffer:     make([]uint16, 0),
	}
}

// Decimate applies low-pass filtering and downsampling
func (d *AntiAliasDecimator) Decimate(samples []uint16) []uint16 {
	if len(samples) == 0 {
		return []uint16{}
	}

	// Append to buffer for filtering
	d.buffer = append(d.buffer, samples...)

	output := make([]uint16, 0, len(samples)/d.ratio+1)

	// Process samples with filtering
	for i := d.filterSize; i < len(d.buffer); i++ {
		if (d.position+i)%d.ratio == 0 {
			// Apply simple moving average filter
			sum := uint32(0)
			for j := 0; j < d.filterSize; j++ {
				sum += uint32(d.buffer[i-j])
			}
			filtered := uint16(sum / uint32(d.filterSize))
			output = append(output, filtered)
		}
	}

	// Keep last filterSize samples for next batch
	if len(d.buffer) > d.filterSize {
		copy(d.buffer, d.buffer[len(d.buffer)-d.filterSize:])
		d.buffer = d.buffer[:d.filterSize]
	}

	d.position = (d.position + len(samples)) % d.ratio

	return output
}
