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
	usePRU        bool // If true, use PRU reader; if false, use IIO (legacy)
}

func (r *reader) start(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		if r.usePRU {
			r.readFromPRU(ctx)
		} else {
			r.readFromIIOLegacy(ctx)
		}
	}()
}

// readFromPRU reads samples from PRU shared memory
func (r *reader) readFromPRU(ctx context.Context) {
	if r.sleepTime <= 0 {
		r.sleepTime = 1 * time.Millisecond
	}
	defer close(r.bufferChannel)

	// Initialize PRU reader
	pruReader, err := NewPRUReader()
	if err != nil {
		fmt.Printf("Error opening PRU shared memory: %v\n", err)
		fmt.Println("Make sure:")
		fmt.Println("  1. You're running as root (sudo)")
		fmt.Println("  2. PRU firmware is loaded")
		fmt.Println("  3. Device tree overlay is enabled")
		return
	}
	defer pruReader.Close()

	fmt.Println("PRU reader initialized successfully")

	payload := [BufferSize * recordSize]byte{}
	current := 0
	lastFlush := time.Now()
	lastStatsTime := time.Now()

	done := false

	for !done {
		select {
		case <-ctx.Done():
			fmt.Println("Stopping PRU reader")
			done = true
		default:
			// Read available samples from PRU
			samples, err := pruReader.ReadSamples()
			if err != nil {
				fmt.Printf("Error reading PRU samples: %v\n", err)
				done = true
				break
			}

			// Convert samples to bytes and fill payload buffer
			for _, sample := range samples {
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
				sampleCount, overrunCount := pruReader.GetStats()
				if overrunCount > 0 {
					fmt.Printf("PRU Stats - Total samples: %d, Overruns: %d (buffer too slow!)\n",
						sampleCount, overrunCount)
				}
				lastStatsTime = time.Now()
			}

			time.Sleep(r.sleepTime)
		}
	}
}

// readFromIIOLegacy is the old IIO-based reader (kept for fallback)
func (r *reader) readFromIIOLegacy(ctx context.Context) {
	fmt.Println("IIO reader not implemented in PRU mode")
	fmt.Println("Please use PRU mode by setting reader.usePRU = true")
	close(r.bufferChannel)
}
