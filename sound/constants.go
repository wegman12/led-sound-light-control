package sound

const (
	HighSpeedRate = 200000 // 185 kHz high-speed IIO sampling
	SamplingRate  = 48000  // 48 kHz target rate after decimation
	BufferSize    = 1024   // samples (increased for higher sample rate)
)
