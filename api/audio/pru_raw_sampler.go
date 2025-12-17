package audio

import (
	"encoding/binary"
	"fmt"
	"os"
	"syscall"
	"time"
	"unsafe"
)

const (
	// Raw Capture Memory Configuration
	rawControlBlockOffset = 0x2000 // 8KB into shared memory (same as audio)
	rawBufferOffset       = 0x2100 // 8KB + 256 bytes
	rawBufferSize         = 4096   // Number of uint16 samples
	rawBufferBytes        = rawBufferSize * 2
)

// RawControlBlock matches the C struct in PRU1 raw capture firmware
type RawControlBlock struct {
	// === Configuration Section (written by host, read by PRU) ===
	EnableCapture uint32 // 1 = capture enabled, 0 = paused

	// === Status Section (written by PRU, read by host) ===
	Status           uint32 // PRU running status
	TotalSamples     uint32 // Total samples collected
	BufferWriteIndex uint32 // Current write position in circular buffer
	BufferWrapCount  uint32 // Number of times buffer has wrapped
	ADCTimeouts      uint32 // ADC timeout errors
	LastSample       uint32 // Most recent sample value
	MinSample        uint32 // Minimum sample value
	MaxSample        uint32 // Maximum sample value

	// Padding to ensure control block is aligned (256 bytes total)
	Reserved [55]uint32
}

// PRURawSampler provides interface to PRU1-based raw audio capture
type PRURawSampler struct {
	memFile      *os.File
	sharedMem    []byte
	controlBlock *RawControlBlock
	sampleBuffer []uint16
}

// NewPRURawSampler creates and initializes a PRU1 raw audio sampler
// Returns error if PRU is not running or memory mapping fails
func NewPRURawSampler() (*PRURawSampler, error) {
	// Open /dev/mem (requires root privileges)
	memFile, err := os.OpenFile(devMem, os.O_RDWR|os.O_SYNC, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s (need root): %w", devMem, err)
	}

	// Memory map PRU shared memory
	mem, err := syscall.Mmap(
		int(memFile.Fd()),
		pruSharedMemAddr,
		pruSharedMemSize,
		syscall.PROT_READ|syscall.PROT_WRITE,
		syscall.MAP_SHARED,
	)
	if err != nil {
		memFile.Close()
		return nil, fmt.Errorf("failed to mmap PRU memory at 0x%X: %w", pruSharedMemAddr, err)
	}

	// Get pointer to control block
	controlBlock := (*RawControlBlock)(unsafe.Pointer(&mem[rawControlBlockOffset]))

	// Verify PRU1 is running (check for valid status code)
	// "RAWC" = 0x52415743
	if controlBlock.Status == 0 {
		syscall.Munmap(mem)
		memFile.Close()
		return nil, fmt.Errorf("PRU1 raw capture firmware not running (status=0). Load firmware with: "+
			"cd pru/audio && make raw-deploy-all")
	}

	// Get pointer to sample buffer
	sampleBuffer := unsafe.Slice((*uint16)(unsafe.Pointer(&mem[rawBufferOffset])), rawBufferSize)

	sampler := &PRURawSampler{
		memFile:      memFile,
		sharedMem:    mem,
		controlBlock: controlBlock,
		sampleBuffer: sampleBuffer,
	}

	return sampler, nil
}

// Close releases the memory mapping and closes the file descriptor
func (s *PRURawSampler) Close() error {
	if s.sharedMem != nil {
		if err := syscall.Munmap(s.sharedMem); err != nil {
			return fmt.Errorf("failed to unmap PRU memory: %w", err)
		}
		s.sharedMem = nil
	}

	if s.memFile != nil {
		if err := s.memFile.Close(); err != nil {
			return fmt.Errorf("failed to close /dev/mem: %w", err)
		}
		s.memFile = nil
	}

	return nil
}

// GetStatus returns the current PRU status and control block information
func (s *PRURawSampler) GetStatus() *RawControlBlock {
	// Return a copy to avoid race conditions
	return &RawControlBlock{
		EnableCapture:    s.controlBlock.EnableCapture,
		Status:           s.controlBlock.Status,
		TotalSamples:     s.controlBlock.TotalSamples,
		BufferWriteIndex: s.controlBlock.BufferWriteIndex,
		BufferWrapCount:  s.controlBlock.BufferWrapCount,
		ADCTimeouts:      s.controlBlock.ADCTimeouts,
		LastSample:       s.controlBlock.LastSample,
		MinSample:        s.controlBlock.MinSample,
		MaxSample:        s.controlBlock.MaxSample,
	}
}

// ReadSamples reads a chunk of samples from the circular buffer
// Returns a copy of the samples to avoid data races
// startIndex: where to start reading (0 to rawBufferSize-1)
// count: number of samples to read
func (s *PRURawSampler) ReadSamples(startIndex, count int) ([]uint16, error) {
	if startIndex < 0 || startIndex >= rawBufferSize {
		return nil, fmt.Errorf("startIndex %d out of range [0, %d)", startIndex, rawBufferSize)
	}
	if count <= 0 || count > rawBufferSize {
		return nil, fmt.Errorf("count %d out of range (0, %d]", count, rawBufferSize)
	}

	samples := make([]uint16, count)

	// Handle circular buffer wrap-around
	for i := 0; i < count; i++ {
		idx := (startIndex + i) % rawBufferSize
		samples[i] = s.sampleBuffer[idx]
	}

	return samples, nil
}

// ReadAllSamples reads the entire circular buffer
func (s *PRURawSampler) ReadAllSamples() []uint16 {
	samples := make([]uint16, rawBufferSize)
	copy(samples, s.sampleBuffer)
	return samples
}

// StreamSamples continuously reads samples and writes them to a file
// duration: how long to record (0 for unlimited)
// outputFile: path to output binary file
// sampleInterval: how often to poll the PRU (recommended: 25ms for ~100ms of buffer)
func (s *PRURawSampler) StreamSamples(duration time.Duration, outputFile string, sampleInterval time.Duration) error {
	// Open output file
	f, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer f.Close()

	// Track the last read position
	// Start from current position to avoid trying to read stale data
	lastReadIndex := s.controlBlock.BufferWriteIndex
	lastWrapCount := s.controlBlock.BufferWrapCount

	// Calculate samples per interval
	// At 40 kHz, in 25ms we get 1000 samples
	// Our buffer holds 4096 samples = ~102ms
	// So polling every 25ms gives us ~25% buffer usage, safe margin

	startTime := time.Now()
	totalSamplesWritten := 0

	// Small delay to let PRU generate some new samples after we start
	time.Sleep(sampleInterval)

	ticker := time.NewTicker(sampleInterval)
	defer ticker.Stop()

	fmt.Printf("Starting raw capture: %s\n", outputFile)
	fmt.Printf("  Sample rate: 40 kHz\n")
	fmt.Printf("  Buffer size: %d samples (~%.1f ms)\n", rawBufferSize, float64(rawBufferSize)/40.0)
	fmt.Printf("  Poll interval: %v\n", sampleInterval)
	if duration > 0 {
		fmt.Printf("  Duration: %v\n", duration)
	} else {
		fmt.Printf("  Duration: unlimited (press Ctrl+C to stop)\n")
	}
	fmt.Println()

	for {
		select {
		case <-ticker.C:
			// Check if we've exceeded the duration
			if duration > 0 && time.Since(startTime) >= duration {
				fmt.Printf("\nCapture complete: %d samples written (%.2f seconds)\n",
					totalSamplesWritten, float64(totalSamplesWritten)/40000.0)
				return nil
			}

			// Read current PRU state
			currentIndex := s.controlBlock.BufferWriteIndex
			currentWrapCount := s.controlBlock.BufferWrapCount

			// Calculate how many samples to read
			var samplesToRead int
			var startIdx int

			if currentWrapCount > lastWrapCount {
				// Buffer wrapped - read from last position to end, then from start to current
				samplesToRead = (rawBufferSize - int(lastReadIndex)) + int(currentIndex)
				startIdx = int(lastReadIndex)

				// Handle potential multiple wraps (shouldn't happen with proper polling)
				if currentWrapCount > lastWrapCount+1 {
					fmt.Printf("WARNING: Buffer wrapped %d times, data loss occurred!\n",
						currentWrapCount-lastWrapCount)
					// Read entire buffer as we missed some data
					samplesToRead = rawBufferSize
					startIdx = int(currentIndex) // Start from where PRU is writing
				}

				// Cap at buffer size to avoid invalid reads
				if samplesToRead > rawBufferSize {
					samplesToRead = rawBufferSize
					startIdx = int(currentIndex) // Read most recent data
				}
			} else {
				// No wrap - simple linear read
				if currentIndex > lastReadIndex {
					samplesToRead = int(currentIndex - lastReadIndex)
					startIdx = int(lastReadIndex)

					// Cap at buffer size (shouldn't happen, but be safe)
					if samplesToRead > rawBufferSize {
						samplesToRead = rawBufferSize
						startIdx = int(currentIndex)
					}
				} else if currentIndex < lastReadIndex {
					// Shouldn't happen without wrap count changing, but handle it
					fmt.Printf("WARNING: Write index decreased without wrap count change!\n")
					samplesToRead = 0
				} else {
					// No new samples
					samplesToRead = 0
				}
			}

			if samplesToRead > 0 {
				// Read the samples
				samples, err := s.ReadSamples(startIdx, samplesToRead)
				if err != nil {
					return fmt.Errorf("failed to read samples: %w", err)
				}

				// Write samples to file as binary (little-endian uint16)
				for _, sample := range samples {
					if err := binary.Write(f, binary.LittleEndian, sample); err != nil {
						return fmt.Errorf("failed to write sample to file: %w", err)
					}
				}

				totalSamplesWritten += samplesToRead

				// Print progress every second
				if totalSamplesWritten%(40000) < samplesToRead {
					elapsed := time.Since(startTime)
					sampleRate := float64(totalSamplesWritten) / elapsed.Seconds()
					fmt.Printf("Progress: %.1fs | %d samples | %.1f kHz actual rate | ADC timeouts: %d\n",
						elapsed.Seconds(),
						totalSamplesWritten,
						sampleRate/1000.0,
						s.controlBlock.ADCTimeouts,
					)
				}
			}

			// Update last read position
			lastReadIndex = currentIndex
			lastWrapCount = currentWrapCount

		}
	}
}
