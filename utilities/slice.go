package utilities

import (
	"golang.org/x/exp/constraints"
)

func Apply[TIn, TOut any](values []TIn, applyFunc func(TIn) TOut) []TOut {
	back := make([]TOut, len(values))
	for i := range values {
		back[i] = applyFunc(values[i])
	}
	return back
}
func ForEach[TIn any](values []TIn, forEachFunc func(TIn)) {
	for i := range values {
		forEachFunc(values[i])
	}
}

func ForEachValue[TKey comparable, TValue any](values map[TKey]TValue, forEachFunc func(TValue)) {
	for _, i := range values {
		forEachFunc(i)
	}
}

func Sum[TIn constraints.Ordered](values []TIn) TIn {
	var total TIn
	ForEach(values, func(t TIn) { total += t })
	return total
}
