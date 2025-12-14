package processing

import (
	"github.com/wegman12/led-sound-light-control/utilities"
)

const (
	defaultBassCutoff = 150
	defaultMidCutoff  = 2000
)

type profiler struct {
	bassCutoff float64
	midCutoff  float64
}

func newProfiler(bassCutoff, midCutoff float64) *profiler {

	if bassCutoff <= 0 {
		bassCutoff = defaultBassCutoff
	}
	if midCutoff <= 0 {
		midCutoff = defaultMidCutoff
	}
	return &profiler{
		bassCutoff: bassCutoff,
		midCutoff:  midCutoff,
	}
}

func pinIndexes(maxLength int, bassUpperIndex, midUpperIndex *int) {

	if *bassUpperIndex < 0 {
		*bassUpperIndex = 0
	}
	if *bassUpperIndex >= maxLength {
		*bassUpperIndex = maxLength - 1
	}
	if *midUpperIndex < 0 {
		*midUpperIndex = 0
	}
	if *midUpperIndex >= maxLength {
		*midUpperIndex = maxLength - 1
	}
}

func (p *profiler) GetProfile(binSize float64, frequencies []float64) Profile {
	bassUpperIndex := int(p.bassCutoff / binSize)
	midUpperIndex := int(p.midCutoff / binSize)
	pinIndexes(len(frequencies), &bassUpperIndex, &midUpperIndex)

	return Profile{
		Bass:   utilities.Sum(frequencies[0:bassUpperIndex]),
		Mid:    utilities.Sum(frequencies[bassUpperIndex:midUpperIndex]),
		Treble: utilities.Sum(frequencies[midUpperIndex:]),
	}
}
