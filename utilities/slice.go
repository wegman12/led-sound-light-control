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

func Filter[T any](values []T, filterFunc func(T) bool) []T {
	back := make([]T, 0, len(values))
	for i := range values {
		if !filterFunc(values[i]) {
			continue
		}
		back = append(back, values[i])
	}
	return back
}

func ApplyMap[TKey comparable, TValue, TOut any](values map[TKey]TValue, applyFunc func(TKey, TValue) TOut) map[TKey]TOut {
	back := make(map[TKey]TOut, len(values))
	for k, v := range values {
		back[k] = applyFunc(k, v)
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
