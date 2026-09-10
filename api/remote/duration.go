package remote

import "time"

const (
	defaultLowDurationMark = 1000 * time.Microsecond
)

type durationReducer struct {
	lowDurationMark time.Duration
}

func newDurationReducer(lowDurationMark time.Duration) *durationReducer {
	if lowDurationMark <= 0 {
		lowDurationMark = defaultLowDurationMark
	}
	return &durationReducer{lowDurationMark: lowDurationMark}
}

func (dr *durationReducer) isLowDuration(pulse DetectionPulse) int {
	if pulse.TimeLow < dr.lowDurationMark {
		return 0
	}
	return 1
}
