package processing

import (
	"github.com/wegman12/led-sound-light-control/utilities"
)

const (
	defaultBassCutoff    = 150
	defaultMidHighCutoff = 1000
	defaultTrebleCutoff  = 2000
)

type profiler struct {
	bassCutoff    float64
	midHighCutoff float64
	trebleCutoff  float64
}

func newProfiler(bassCutoff, midHighCutoff, trebleCutoff float64) *profiler {

	if bassCutoff <= 0 {
		bassCutoff = defaultBassCutoff
	}
	if midHighCutoff <= 0 {
		midHighCutoff = defaultMidHighCutoff
	}
	if trebleCutoff <= 0 {
		trebleCutoff = defaultTrebleCutoff
	}
	return &profiler{
		bassCutoff:    bassCutoff,
		midHighCutoff: midHighCutoff,
		trebleCutoff:  trebleCutoff,
	}
}

func pinIndexes(maxLength int, bassUpperIndex, midHighUpperIndex, trebleUpperIndex *int) {

	if *bassUpperIndex < 0 {
		*bassUpperIndex = 0
	}
	if *bassUpperIndex >= maxLength {
		*bassUpperIndex = maxLength - 1
	}
	if *midHighUpperIndex < 0 {
		*midHighUpperIndex = 0
	}
	if *midHighUpperIndex >= maxLength {
		*midHighUpperIndex = maxLength - 1
	}
	if *trebleUpperIndex < 0 {
		*trebleUpperIndex = 0
	}
	if *trebleUpperIndex >= maxLength {
		*trebleUpperIndex = maxLength - 1
	}
}

func (p *profiler) GetProfile(binSize float64, frequencies []float64) Profile {
	bassUpperIndex := int(p.bassCutoff / binSize)
	midHighUpperIndex := int(p.midHighCutoff / binSize)
	trebleUpperIndex := int(p.trebleCutoff / binSize)
	pinIndexes(len(frequencies), &bassUpperIndex, &midHighUpperIndex, &trebleUpperIndex)

	return Profile{
		Bass:    utilities.Sum(frequencies[0:bassUpperIndex]),
		MidLow:  utilities.Sum(frequencies[bassUpperIndex:midHighUpperIndex]),
		MidHigh: utilities.Sum(frequencies[midHighUpperIndex:trebleUpperIndex]),
		Treble:  utilities.Sum(frequencies[trebleUpperIndex:]),
	}
}
