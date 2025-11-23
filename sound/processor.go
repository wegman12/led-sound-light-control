package sound

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/scientificgo/fft"
)

type FrequencyResult struct {
	Magnitudes       [BufferSize/2 - 1]float64
	SamplingDuration time.Duration
}
type processor struct {
	bufferChannel chan bufferPayload
	resultChannel chan FrequencyResult
	sleepTime     time.Duration
}

func (p *processor) Start(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		p.extractFrequencies(ctx)
	}()
}

func (p *processor) extractFrequencies(ctx context.Context) {
	if p.sleepTime <= 0 {
		p.sleepTime = 10 * time.Millisecond
	}
	defer close(p.resultChannel)
	for {
		select {
		case <-ctx.Done():
			fmt.Println("Stopping processor")
			return
		case record := <-p.bufferChannel:
			p.resultChannel <- FrequencyResult{
				Magnitudes:       convertToFrequency(record.samples),
				SamplingDuration: record.samplingDuration,
			}
		}
		time.Sleep(p.sleepTime)
	}
}

func convertToFrequency(record [BufferSize]uint16) [BufferSize/2 - 1]float64 {
	fftInput := make([]complex128, BufferSize)

	for i := 0; i < BufferSize; i += 1 {
		fftInput[i] = complex(float64(record[i]), 0)
	}
	fftResult := fft.Fft(fftInput, false)

	back := [BufferSize/2 - 1]float64{}
	// Since input are real values, only the first half of frequencies are meaningful (imaginary mirrored in second half)
	// Also drop the DC 0 frequency as that has high noise
	for i := 1; i < len(back)/2; i++ {
		back[i-1] = math.Sqrt(math.Pow(real(fftResult[i]), 2) + math.Pow(imag(fftResult[i]), 2))
	}
	return back
}
