package sound

import (
	"encoding/binary"
	"fmt"
	"os"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

const (
	// PRU shared memory physical address
	PRU_SHARED_MEM_ADDR = 0x4A310000
	PRU_SHARED_MEM_SIZE = 0x3000 // 12 KB

	// Buffer configuration
	PRU_BUFFER_SIZE    = 4096 // Number of 16-bit samples
	PRU_CONTROL_OFFSET = 0x2000

	// Memory map file
	DEV_MEM = "/dev/mem"
)

// Control block structure matching PRU firmware
type pruControlBlock struct {
	WriteIndex   uint32
	ReadIndex    uint32
	SampleCount  uint32
	OverrunCount uint32
}

// PRUReader reads high-frequency ADC samples from PRU shared memory
type PRUReader struct {
	memFile       *os.File
	sharedMem     []byte
	sampleBuffer  []uint16
	controlBlock  *pruControlBlock
	lastReadIndex uint32
	mu            sync.Mutex
}

// NewPRUReader creates and initializes a PRU memory reader
func NewPRUReader() (*PRUReader, error) {
	// Open /dev/mem for memory-mapped I/O
	memFile, err := os.OpenFile(DEV_MEM, os.O_RDWR|os.O_SYNC, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s (need root): %w", DEV_MEM, err)
	}

	// Memory map the PRU shared memory region
	mem, err := syscall.Mmap(
		int(memFile.Fd()),
		PRU_SHARED_MEM_ADDR,
		PRU_SHARED_MEM_SIZE,
		syscall.PROT_READ|syscall.PROT_WRITE,
		syscall.MAP_SHARED,
	)
	if err != nil {
		memFile.Close()
		return nil, fmt.Errorf("failed to mmap PRU shared memory: %w", err)
	}

	reader := &PRUReader{
		memFile:      memFile,
		sharedMem:    mem,
		sampleBuffer: make([]uint16, PRU_BUFFER_SIZE),
		controlBlock: (*pruControlBlock)(unsafe.Pointer(&mem[PRU_CONTROL_OFFSET])),
	}

	// Initialize read index to current write index (start from "now")
	reader.lastReadIndex = reader.controlBlock.ReadIndex

	return reader, nil
}

// Close releases PRU memory resources
func (pr *PRUReader) Close() error {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	if pr.sharedMem != nil {
		if err := syscall.Munmap(pr.sharedMem); err != nil {
			return fmt.Errorf("failed to munmap: %w", err)
		}
		pr.sharedMem = nil
	}

	if pr.memFile != nil {
		if err := pr.memFile.Close(); err != nil {
			return fmt.Errorf("failed to close mem file: %w", err)
		}
		pr.memFile = nil
	}

	return nil
}

// ReadSamples reads available samples from the PRU ring buffer
// Returns a slice of ADC values (0-4095) and the number of samples read
func (pr *PRUReader) ReadSamples() ([]uint16, error) {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	if pr.sharedMem == nil {
		return nil, fmt.Errorf("PRU reader is closed")
	}

	// Get current write index from PRU
	writeIndex := pr.controlBlock.WriteIndex
	readIndex := pr.lastReadIndex

	// Calculate number of available samples
	var available uint32
	if writeIndex >= readIndex {
		available = writeIndex - readIndex
	} else {
		// Wrap around case
		available = PRU_BUFFER_SIZE - readIndex + writeIndex
	}

	if available == 0 {
		return []uint16{}, nil
	}

	// Limit to buffer size to prevent reading too much at once
	if available > PRU_BUFFER_SIZE/2 {
		available = PRU_BUFFER_SIZE / 2
	}

	// Read samples from ring buffer
	samples := make([]uint16, available)
	for i := uint32(0); i < available; i++ {
		// Calculate position in ring buffer
		pos := (readIndex + i) % PRU_BUFFER_SIZE

		// Read 16-bit sample from shared memory
		offset := pos * 2 // 2 bytes per sample
		sample := binary.LittleEndian.Uint16(pr.sharedMem[offset : offset+2])
		samples[i] = sample
	}

	// Update read index
	pr.lastReadIndex = (readIndex + available) % PRU_BUFFER_SIZE

	// Update control block read index so PRU knows we've consumed samples
	pr.controlBlock.ReadIndex = pr.lastReadIndex

	return samples, nil
}

// GetStats returns PRU buffer statistics
func (pr *PRUReader) GetStats() (sampleCount, overrunCount uint32) {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	if pr.sharedMem == nil {
		return 0, 0
	}

	return pr.controlBlock.SampleCount, pr.controlBlock.OverrunCount
}

// WaitForSamples blocks until at least minSamples are available or timeout
func (pr *PRUReader) WaitForSamples(minSamples uint32, timeout time.Duration) error {
	start := time.Now()

	for {
		if time.Since(start) > timeout {
			return fmt.Errorf("timeout waiting for samples")
		}

		pr.mu.Lock()
		writeIndex := pr.controlBlock.WriteIndex
		readIndex := pr.lastReadIndex
		pr.mu.Unlock()

		var available uint32
		if writeIndex >= readIndex {
			available = writeIndex - readIndex
		} else {
			available = PRU_BUFFER_SIZE - readIndex + writeIndex
		}

		if available >= minSamples {
			return nil
		}

		time.Sleep(100 * time.Microsecond)
	}
}

// ResetStats resets the sample and overrun counters
func (pr *PRUReader) ResetStats() {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	if pr.sharedMem != nil {
		pr.controlBlock.SampleCount = 0
		pr.controlBlock.OverrunCount = 0
	}
}
