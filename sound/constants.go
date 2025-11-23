package sound

const (
	SamplingRate = 48000 // 48 kHz sampling via PRU
	BufferSize   = 512   // samples (increased for higher sample rate)
	recordSize   = 2     // bytes (16-bit ADC value from PRU)
)
