package audio

import (
	"fmt"
	"os"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

const (
	// PRU Shared Memory Configuration
	// Physical address of PRU shared memory (from Linux perspective)
	pruSharedMemAddr = 0x4A310000
	pruSharedMemSize = 0x3000 // 12 KB (was incorrectly 64KB)

	// PRU1 audio control block offset (must match pru_shared_memory.h)
	audioControlBlockOffset = 0x0900 // After PRU0 control block (was 0x2000)

	devMem = "/dev/mem"
)

// AudioControlBlock matches the C struct in PRU1 audio firmware
// Layout: Configuration (written by host) + Status (written by PRU)
type AudioControlBlock struct {
	// === Configuration Section (written by host, read by PRU) ===
	FFTEnable           uint32 // 1 = FFT enabled, 0 = disabled
	BassMaxHz           uint32 // Bass upper frequency boundary (Hz)
	MidLowMaxHz         uint32 // Mid-low upper frequency boundary (Hz)
	MidHighMaxHz        uint32 // Mid-high upper frequency boundary (Hz)
	SmoothingAlphaX1000 uint32 // Temporal smoothing factor (0-1000, where 1000 = 1.0)

	// === Status Section (written by PRU, read by host) ===
	Status           uint32 // PRU running status
	TotalSamples     uint32 // Total samples collected
	BufferCount      uint32 // Number of completed buffers
	CurrentBuffer    uint32 // 0 = Buffer A, 1 = Buffer B
	SamplesInBuffer  uint32 // Current sample count in active buffer
	ADCTimeouts      uint32 // ADC timeout errors
	MissedSamples    uint32 // Missed samples (overruns)
	LastSample       uint32 // Most recent sample value
	MinSample        uint32 // Minimum sample value
	MaxSample        uint32 // Maximum sample value
	FFTCount         uint32 // Number of FFTs computed
	FFTTimeCycles    uint32 // Last FFT processing time (PRU cycles)
	FFTSkipped       uint32 // FFTs skipped due to timing overrun
	Bass             uint32 // Bass magnitude sum (0-bass_max_hz)
	MidLow           uint32 // Mid-low magnitude sum (bass_max_hz-midlow_max_hz)
	MidHigh          uint32 // Mid-high magnitude sum (midlow_max_hz-midhigh_max_hz)
	Treble           uint32 // Treble magnitude sum (midhigh_max_hz-Nyquist)
	BassAvg          uint32 // Bass average magnitude per bin
	MidLowAvg        uint32 // Mid-low average magnitude per bin
	MidHighAvg       uint32 // Mid-high average magnitude per bin
	TrebleAvg        uint32 // Treble average magnitude per bin
}

// PRUSampler provides interface to PRU1-based audio sampling and FFT
type PRUSampler struct {
	memFile      *os.File
	sharedMem    []byte
	controlBlock *AudioControlBlock
	mu           sync.Mutex
	lastFFTCount uint32
	lastReadTime time.Time
}

// NewPRUSampler creates and initializes a PRU1 audio sampler
// Returns error if PRU is not running or memory mapping fails
func NewPRUSampler() (*PRUSampler, error) {
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
	controlBlock := (*AudioControlBlock)(unsafe.Pointer(&mem[audioControlBlockOffset]))

	// Verify PRU1 is running (check for valid status code)
	if controlBlock.Status == 0 {
		syscall.Munmap(mem)
		memFile.Close()
		return nil, fmt.Errorf("PRU1 audio firmware not running (status=0). Load firmware with: "+
			"sudo cp pru/audio/gen/pru1_audio.out /lib/firmware/am335x-pru1-fw && "+
			"echo 'start' | sudo tee /sys/class/remoteproc/remoteproc2/state")
	}

	sampler := &PRUSampler{
		memFile:      memFile,
		sharedMem:    mem,
		controlBlock: controlBlock,
		lastFFTCount: controlBlock.FFTCount,
		lastReadTime: time.Now(),
	}

	return sampler, nil
}

// Close releases PRU memory mapping and file descriptor
func (ps *PRUSampler) Close() error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if ps.sharedMem != nil {
		if err := syscall.Munmap(ps.sharedMem); err != nil {
			return fmt.Errorf("failed to unmap PRU memory: %w", err)
		}
		ps.sharedMem = nil
	}

	if ps.memFile != nil {
		if err := ps.memFile.Close(); err != nil {
			return fmt.Errorf("failed to close /dev/mem: %w", err)
		}
		ps.memFile = nil
	}

	return nil
}

// ReadSoundProfile reads the current sound profile from PRU1
// Returns SoundProfile with frequency band magnitudes
func (ps *PRUSampler) ReadSoundProfile() (*SoundProfile, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if ps.controlBlock == nil {
		return nil, fmt.Errorf("PRU sampler not initialized")
	}

	// Read control block atomically
	ctrl := *ps.controlBlock

	// Calculate FFT rate
	now := time.Now()
	elapsed := now.Sub(ps.lastReadTime).Seconds()
	fftDelta := ctrl.FFTCount - ps.lastFFTCount
	fftRate := 0.0
	if elapsed > 0 {
		fftRate = float64(fftDelta) / elapsed
	}

	ps.lastFFTCount = ctrl.FFTCount
	ps.lastReadTime = now

	profile := &SoundProfile{
		Timestamp:  now,
		FFTCount:   ctrl.FFTCount,
		FFTRate:    fftRate,
		FFTTimeMs:  float64(ctrl.FFTTimeCycles) / 200000.0, // PRU runs at 200 MHz
		BassSum:    ctrl.Bass,
		MidLowSum:  ctrl.MidLow,
		MidHighSum: ctrl.MidHigh,
		TrebleSum:  ctrl.Treble,
		BassAvg:    ctrl.BassAvg,
		MidLowAvg:  ctrl.MidLowAvg,
		MidHighAvg: ctrl.MidHighAvg,
		TrebleAvg:  ctrl.TrebleAvg,
	}

	return profile, nil
}

// GetStatus returns the current PRU1 status
func (ps *PRUSampler) GetStatus() (*PRUStatus, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if ps.controlBlock == nil {
		return nil, fmt.Errorf("PRU sampler not initialized")
	}

	ctrl := *ps.controlBlock

	status := &PRUStatus{
		Status:          ctrl.Status,
		TotalSamples:    ctrl.TotalSamples,
		BufferCount:     ctrl.BufferCount,
		ADCTimeouts:     ctrl.ADCTimeouts,
		FFTSkipped:      ctrl.FFTSkipped,
		FFTEnabled:      ctrl.FFTEnable == 1,
		BassMaxHz:       ctrl.BassMaxHz,
		MidLowMaxHz:     ctrl.MidLowMaxHz,
		MidHighMaxHz:    ctrl.MidHighMaxHz,
		SampleRateHz:    40000, // Fixed at 40 kHz
	}

	return status, nil
}

// SetFFTEnable enables or disables FFT processing
func (ps *PRUSampler) SetFFTEnable(enable bool) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if ps.controlBlock == nil {
		return fmt.Errorf("PRU sampler not initialized")
	}

	if enable {
		ps.controlBlock.FFTEnable = 1
	} else {
		ps.controlBlock.FFTEnable = 0
	}

	return nil
}

// SetFrequencyBands updates the frequency band boundaries
func (ps *PRUSampler) SetFrequencyBands(bassMax, midLowMax, midHighMax uint32) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if ps.controlBlock == nil {
		return fmt.Errorf("PRU sampler not initialized")
	}

	// Validate ranges
	if bassMax < 10 || bassMax > 1000 {
		return fmt.Errorf("bassMax out of range (10-1000 Hz): %d", bassMax)
	}
	if midLowMax < 100 || midLowMax > 5000 {
		return fmt.Errorf("midLowMax out of range (100-5000 Hz): %d", midLowMax)
	}
	if midHighMax < 500 || midHighMax > 10000 {
		return fmt.Errorf("midHighMax out of range (500-10000 Hz): %d", midHighMax)
	}

	ps.controlBlock.BassMaxHz = bassMax
	ps.controlBlock.MidLowMaxHz = midLowMax
	ps.controlBlock.MidHighMaxHz = midHighMax

	return nil
}
