package processing

import (
	"math"

	"github.com/scientificgo/fft"
	"github.com/wegman12/led-sound-light-control/utilities"
)

type frequencyTransformer struct{}

func newTransformer() *frequencyTransformer {
	return &frequencyTransformer{}
}

func calculateMagnitude(result complex128) float64 {
	return math.Sqrt(math.Pow(real(result), 2) + math.Pow(imag(result), 2))
}

func toComplex(v uint16) complex128 {
	return complex(float64(v), 0)
}

func (ft *frequencyTransformer) Transform(buffer []uint16) []float64 {
	fftResult := fft.Fft(utilities.Apply(buffer, toComplex), false)

	return utilities.Apply(fftResult[1:len(buffer)/2], calculateMagnitude)
}
