package remote

import (
	"fmt"
	"os"
	"sync"
	"syscall"
	"unsafe"
)

const (
	// PRU Shared Memory Configuration
	// Physical address of PRU shared memory (from Linux perspective)
	pruSharedMemAddr = 0x4A310000
	pruSharedMemSize = 0x3000 // 12 KB

	// PRU0 IR detector memory layout (must match pru_shared_memory.h)
	ringBufferOffset   = 0x0000 // Ring buffer at start of shared memory
	ringBufferSize     = 256    // 256 events
	controlBlockOffset = 0x0800 // Control block after ring buffer (was 0x1000)

	devMem = "/dev/mem"
)

// pruControlBlock matches the C struct in PRU firmware
type pruControlBlock struct {
	WriteIndex   uint32
	ReadIndex    uint32
	EventCount   uint32
	ErrorCount   uint32
	OverrunCount uint32
	Status       uint32
	ErrorCode    uint32
}

// buttonEvent matches the C struct in PRU firmware (8 bytes)
type buttonEvent struct {
	ButtonType uint8
	Reserved1  uint8
	Reserved2  uint16
	Timestamp  uint32
}

// PRUDetector provides interface to PRU-based IR detection
type PRUDetector struct {
	memFile      *os.File
	sharedMem    []byte
	controlBlock *pruControlBlock
	lastReadIdx  uint32
	mu           sync.Mutex
}

// newPRUDetector creates and initializes a PRU detector
// Returns error if PRU is not running or memory mapping fails
func newPRUDetector() (*PRUDetector, error) {
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
	controlBlock := (*pruControlBlock)(unsafe.Pointer(&mem[controlBlockOffset]))

	// Verify PRU is running
	if controlBlock.Status == 0 {
		syscall.Munmap(mem)
		memFile.Close()
		return nil, fmt.Errorf("PRU firmware not running (status=0). Load firmware with: "+
			"echo 'start' | sudo tee /sys/class/remoteproc/remoteproc1/state")
	}

	detector := &PRUDetector{
		memFile:      memFile,
		sharedMem:    mem,
		controlBlock: controlBlock,
		lastReadIdx:  controlBlock.ReadIndex,
	}

	return detector, nil
}

// Close releases PRU memory mapping and file descriptor
func (pd *PRUDetector) Close() error {
	pd.mu.Lock()
	defer pd.mu.Unlock()

	if pd.sharedMem != nil {
		if err := syscall.Munmap(pd.sharedMem); err != nil {
			return fmt.Errorf("failed to unmap PRU memory: %w", err)
		}
		pd.sharedMem = nil
	}

	if pd.memFile != nil {
		if err := pd.memFile.Close(); err != nil {
			return fmt.Errorf("failed to close %s: %w", devMem, err)
		}
		pd.memFile = nil
	}

	return nil
}

// ReadButtons reads all available button events from the ring buffer
// Returns slice of ButtonType values, or error if detector is closed
func (pd *PRUDetector) ReadButtons() ([]ButtonType, error) {
	pd.mu.Lock()
	defer pd.mu.Unlock()

	if pd.sharedMem == nil {
		return nil, fmt.Errorf("PRU detector closed")
	}

	// Read current write index from PRU
	writeIdx := pd.controlBlock.WriteIndex
	readIdx := pd.lastReadIdx

	// Calculate number of available events
	var available uint32
	if writeIdx >= readIdx {
		available = writeIdx - readIdx
	} else {
		// Handle wraparound
		available = ringBufferSize - readIdx + writeIdx
	}

	if available == 0 {
		return []ButtonType{}, nil
	}

	// Prevent overflow - should never happen but be defensive
	if available > ringBufferSize {
		return nil, fmt.Errorf("ring buffer corruption: available=%d > size=%d", available, ringBufferSize)
	}

	// Read events from ring buffer
	buttons := make([]ButtonType, 0, available)
	for i := uint32(0); i < available; i++ {
		pos := (readIdx + i) % ringBufferSize
		offset := pos * 8 // 8 bytes per event

		// Read button type (first byte of event)
		buttonType := pd.sharedMem[offset]

		// Validate button type is in valid range (0-43)
		if buttonType < 44 {
			buttons = append(buttons, ButtonType(buttonType))
		}
		// Silently skip invalid button types (defense against corruption)
	}

	// Update read index to inform PRU we've consumed these events
	pd.lastReadIdx = (readIdx + available) % ringBufferSize
	pd.controlBlock.ReadIndex = pd.lastReadIdx

	return buttons, nil
}

// GetStats returns current PRU statistics
// Returns (eventCount, errorCount, overrunCount)
func (pd *PRUDetector) GetStats() (eventCount, errorCount, overrunCount uint32) {
	pd.mu.Lock()
	defer pd.mu.Unlock()

	if pd.sharedMem == nil {
		return 0, 0, 0
	}

	return pd.controlBlock.EventCount, pd.controlBlock.ErrorCount, pd.controlBlock.OverrunCount
}

// GetStatus returns PRU running status
// Returns true if PRU firmware is running, false otherwise
func (pd *PRUDetector) GetStatus() bool {
	pd.mu.Lock()
	defer pd.mu.Unlock()

	if pd.sharedMem == nil {
		return false
	}

	return pd.controlBlock.Status != 0
}
