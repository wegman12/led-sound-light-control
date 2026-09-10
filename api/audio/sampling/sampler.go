package sampling

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

type Sampler struct {
	delayBetweenSamples time.Duration
	bufferSize          int
	targetInputRate     float64
	samplingRate        int
	iioSampler          *fastIIOReader
	decimator           *AntiAliasDecimator
	logger              *zap.Logger
}

const defaultBufferSize = 2048

func NewSampler(bufferSize, samplingRate int, targetInputRate float64, delayBetweenSamples time.Duration, logger *zap.Logger) (*Sampler, error) {
	if logger == nil {
		logger = zap.NewNop()
	}
	if delayBetweenSamples <= 0 {
		delayBetweenSamples = 1 * time.Millisecond
	}

	if bufferSize <= 0 {
		bufferSize = defaultBufferSize
	}
	if samplingRate <= 0 {
		samplingRate = 48000
	}
	if targetInputRate <= 0 {
		targetInputRate = 200000
	}
	iioSampler, err := newFastIioSampler(1, bufferSize, logger)
	if err != nil {
		logger.Error("failed to initialize iio Sampler", zap.Error(err))
		logger.Error("make sure:")
		logger.Error("  1. You're running as root (sudo)")
		logger.Error("  2. ADC is available at /sys/bus/iio/devices/iio:device0")
		return nil, err
	}

	decimator := NewAntiAliasDecimator(samplingRate)

	return &Sampler{
		delayBetweenSamples: delayBetweenSamples,
		bufferSize:          bufferSize,
		samplingRate:        samplingRate,
		targetInputRate:     targetInputRate,
		iioSampler:          iioSampler,
		logger:              logger,
		decimator:           decimator,
	}, nil

}

// SampleAudio reads samples from Linux IIO at high speed with decimation
func (s *Sampler) SampleAudio(
	payloadChannel chan Payload,
	ctx context.Context,
) {
	// start IIO sampling
	if err := s.iioSampler.Start(); err != nil {
		fmt.Printf("Error starting IIO Sampler: %v\n", err)
		return
	}
	defer func() {
		err := s.iioSampler.Stop()
		if err != nil {
			s.logger.Error("failed to stop IIO Sampler", zap.Error(err))
		}
	}()

	t := &tracker{
		buffer:             make([]uint16, s.bufferSize),
		current:            0,
		lastStatsTime:      time.Now(),
		lastFlush:          time.Now(),
		totalInputSamples:  0,
		totalOutputSamples: 0,
		samplingRate:       s.samplingRate,
		bufferSize:         s.bufferSize,
		logger:             s.logger,
	}

	done := false

	for !done {
		select {
		case <-ctx.Done():
			s.logger.Debug("Sampler Stopped")
			done = true
		default:
			// Read high-speed samples from IIO
			samples, err := s.iioSampler.ReadSamples()
			if len(samples) == 0 {
				s.logger.Debug("No samples found - continuing")
				break
			}
			if err != nil {
				s.logger.Error("failed to read samples", zap.Error(err))
				done = true
				break
			}

			// Decimate to target sample rate
			decimatedSamples := s.decimator.Decimate(s.targetInputRate, samples)

			t.trackSamples(len(samples), decimatedSamples, payloadChannel)
			t.displayStats(s.targetInputRate)

			time.Sleep(s.delayBetweenSamples)
		}
	}
}

type tracker struct {
	buffer             []uint16
	lastFlush          time.Time
	lastStatsTime      time.Time
	totalInputSamples  uint64
	totalOutputSamples uint64
	current            int
	samplingRate       int
	bufferSize         int
	logger             *zap.Logger
}

func (t *tracker) trackSamples(inputSampleCount int, outputSamples []uint16, channel chan Payload) {
	// Convert decimated outputSamples to outputSamples and fill payload buffer
	t.totalInputSamples += uint64(inputSampleCount)
	t.totalOutputSamples += uint64(len(outputSamples))
	for _, sample := range outputSamples {
		// Store as little-endian 16-bit value
		t.buffer[t.current] = sample
		t.current++

		// When buffer is full, send it for processing
		if t.current >= len(t.buffer) {
			duration := time.Since(t.lastFlush)
			channel <- Payload{
				Samples:          t.buffer,
				SamplingDuration: duration,
				BinSize:          float64(t.samplingRate) / float64(t.bufferSize),
			}
			t.current = 0
			t.lastFlush = time.Now()
		}
	}
}

func (t *tracker) displayStats(actualInputRate float64) {
	// Print stats every 5 seconds
	if time.Since(t.lastStatsTime) > 5*time.Second {
		inputRate := float64(t.totalInputSamples) / time.Since(t.lastStatsTime).Seconds()
		outputRate := float64(t.totalOutputSamples) / time.Since(t.lastStatsTime).Seconds()
		t.logger.Debug(
			"IIO Stats",
			zap.Float64("Input Frequency (Hz)", inputRate),
			zap.Float64("Output Frequency (Hz)", outputRate),
			zap.Int("Sampling Rate (Hz)", t.samplingRate),
		)
		t.totalInputSamples = 0
		t.totalOutputSamples = 0
		t.lastStatsTime = time.Now()
	}
}
