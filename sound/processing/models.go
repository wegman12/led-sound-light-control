package processing

import "time"

type Profile struct {
	Bass   float64
	Mid    float64
	Treble float64
}

type Result struct {
	Magnitudes       []float64
	SamplingDuration time.Duration
	SignalStrength   float64
	Profile          Profile
}
