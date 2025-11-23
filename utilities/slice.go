package utilities

func Apply[TIn, TOut any](values []TIn, applyFunc func(TIn) TOut) []TOut {
	back := make([]TOut, len(values))
	for i := range values {
		back[i] = applyFunc(values[i])
	}
	return back
}
