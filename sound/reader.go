package sound

import (
	"context"
	"encoding/binary"
	"fmt"
	"sync"
	"time"
)

type bufferPayload struct {
	bytes            [BufferSize * recordSize]byte
	samplingDuration time.Duration
}

type reader struct {
	bufferChannel chan bufferPayload
	sleepTime     time.Duration
}

func (r *reader) start(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		r.readFromIIOFast(ctx)
	}()
}

// readFromIIOFast reads samples from Linux IIO at high speed with decimation
func (r *reader) readFromIIOFast(ctx context.Context) {
	if r.sleepTime <= 0 {
		r.sleepTime = 1 * time.Millisecond
	}
	defer close(r.bufferChannel)

	// Create IIO reader - sample at max speed (~200 kHz)
	// Channel 1 = AIN1 (P9_40)
	iioReader, err := NewFastIIOReader(1, 200000, 1024)
	if err != nil {
		fmt.Printf("Error creating IIO reader: %v\n", err)
		fmt.Println("Make sure:")
		fmt.Println("  1. You're running as root (sudo)")
		fmt.Println("  2. ADC is available at /sys/bus/iio/devices/iio:device0")
		return
	}

	// Start IIO sampling
	if err := iioReader.Start(); err != nil {
		fmt.Printf("Error starting IIO reader: %v\n", err)
		return
	}
	defer iioReader.Stop()

	fmt.Println("IIO fast reader initialized successfully")
	fmt.Printf("Sampling at high speed with decimation to %d Hz\n", SamplingRate)

	// Get actual sample rate
	if actualRate, err := iioReader.GetActualSampleRate(); err == nil {
		fmt.Printf("Actual IIO sample rate: %d Hz\n", actualRate)
	}

	// Create decimator to convert to target sample rate
	decimator := NewAntiAliasDecimator(200000, SamplingRate)

	payload := [BufferSize * recordSize]byte{}
	current := 0
	lastFlush := time.Now()
	lastStatsTime := time.Now()
	totalInputSamples := uint64(0)
	totalOutputSamples := uint64(0)

	done := false

	for !done {
		select {
		case <-ctx.Done():
			fmt.Println("Stopping IIO fast reader")
			done = true
		default:
			// Read high-speed samples from IIO
			samples, err := iioReader.ReadSamples()
			if err != nil {
				fmt.Printf("Error reading IIO samples: %v\n", err)
				done = true
				break
			}

			totalInputSamples += uint64(len(samples))

			// Decimate to target sample rate
			decimatedSamples := decimator.Decimate(samples)
			totalOutputSamples += uint64(len(decimatedSamples))

			// Convert decimated samples to bytes and fill payload buffer
			for _, sample := range decimatedSamples {
				// Store as little-endian 16-bit value
				binary.LittleEndian.PutUint16(payload[current:current+2], sample)
				current += recordSize

				// When buffer is full, send it for processing
				if current >= len(payload) {
					duration := time.Since(lastFlush)
					r.bufferChannel <- bufferPayload{
						bytes:            payload,
						samplingDuration: duration,
					}
					current = 0
					lastFlush = time.Now()
				}
			}

			// Print stats every 5 seconds
			if time.Since(lastStatsTime) > 5*time.Second {
				inputRate := float64(totalInputSamples) / time.Since(lastStatsTime).Seconds()
				outputRate := float64(totalOutputSamples) / time.Since(lastStatsTime).Seconds()
				fmt.Printf("IIO Stats - Input: %.0f Hz, Output: %.0f Hz (target: %d Hz)\n",
					inputRate, outputRate, SamplingRate)
				totalInputSamples = 0
				totalOutputSamples = 0
				lastStatsTime = time.Now()
			}

			time.Sleep(r.sleepTime)
		}
	}
}

