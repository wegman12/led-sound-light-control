package sampling

import "time"

type Payload struct {
	Samples          []uint16
	SamplingDuration time.Duration
	BinSize          float64
}
