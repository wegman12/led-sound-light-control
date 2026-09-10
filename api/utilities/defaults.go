package utilities

import (
	"golang.org/x/exp/constraints"
)

func SetValueOrDefault[T comparable](v *T, defaultValue T) {
	if v == nil {
		return
	}
	var baseline T
	if *v == baseline {
		*v = defaultValue
	}
}

func PinValueToRange[T constraints.Ordered](v *T, lowerLimit T, upperLimit T) {
	if v == nil {
		return
	}
	if lowerLimit > upperLimit {
		old := lowerLimit
		lowerLimit = upperLimit
		upperLimit = old
	}

	if *v < lowerLimit {
		*v = lowerLimit
	}
	if *v > upperLimit {
		*v = upperLimit
	}
}
