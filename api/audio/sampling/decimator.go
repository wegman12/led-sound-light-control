package sampling

// AntiAliasDecimator uses a simple low-pass filter before decimation
// to prevent aliasing
type AntiAliasDecimator struct {
	outputRate int
	buffer     []uint16
}

// NewAntiAliasDecimator creates a decimator with anti-aliasing filter
func NewAntiAliasDecimator(outputRate int) *AntiAliasDecimator {
	ratio := outputRate
	if ratio < 1 {
		ratio = 1
	}

	return &AntiAliasDecimator{
		outputRate: outputRate,
		buffer:     make([]uint16, 0),
	}
}

// Decimate applies low-pass filtering and downsampling
func (d *AntiAliasDecimator) Decimate(inputRate float64, samples []uint16) []uint16 {

	ratio := int(inputRate) / d.outputRate
	if ratio < 1 {
		ratio = 1
	}

	// Use a small averaging window (moving average filter)
	filterSize := ratio / 2
	if filterSize < 1 {
		filterSize = 1
	}

	if len(samples) == 0 {
		return []uint16{}
	}

	// Append to buffer for filtering
	d.buffer = append(d.buffer, samples...)

	output := make([]uint16, 0, len(samples)/ratio+1)

	// Process samples with filtering
	for i := filterSize; i < len(d.buffer); i++ {
		if i%ratio == 0 {
			// Apply simple moving average filter
			sum := uint32(0)
			for j := 0; j < filterSize; j++ {
				sum += uint32(d.buffer[i-j])
			}
			filtered := uint16(sum / uint32(filterSize))
			output = append(output, filtered)
		}
	}

	// Keep last filterSize samples for next batch
	if len(d.buffer) > filterSize {
		copy(d.buffer, d.buffer[len(d.buffer)-filterSize:])
		d.buffer = d.buffer[:filterSize]
	}

	return output
}
