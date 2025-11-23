package sound

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/scientificgo/fft"
)

type FrequencyResult struct {
	Magnitudes       [BufferSize]float64
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
				Magnitudes:       convertToFrequency(record.bytes),
				SamplingDuration: record.samplingDuration,
			}
		}
		time.Sleep(p.sleepTime)
	}
}

func convertToFrequency(record [BufferSize * recordSize]byte) [BufferSize]float64 {
	fftInput := make([]complex128, BufferSize)

	for i := 0; i < recordSize*BufferSize; i += recordSize {
		fftInput[i/recordSize] = complex(bytesToFloat64(record[i:i+recordSize]), 0)
	}
	fftResult := fft.Fft(fftInput, false)

	back := [BufferSize]float64{}
	for i := 0; i < len(back); i++ {
		back[i] = math.Sqrt(math.Pow(real(fftResult[i]), 2) + math.Pow(imag(fftResult[i]), 2))
	}
	return back
}

func bytesToFloat64(byteSlice []byte) float64 {
	// For PRU mode: read 16-bit ADC value (2 bytes)
	if len(byteSlice) >= 2 {
		value := binary.LittleEndian.Uint16(byteSlice[0:2])
		return float64(value)
	}

	// Legacy mode: read full uint64
	buf := bytes.NewReader(byteSlice)
	var back uint64
	err := binary.Read(buf, binary.LittleEndian, &back)
	if err != nil {
		fmt.Println("binary.Read failed:", err)
	}
	return float64(back)
}
