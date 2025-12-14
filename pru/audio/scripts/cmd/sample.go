package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
	"unsafe"

	"github.com/spf13/cobra"
)

const (
	// Control block structure offsets
	offsetStatus      = 0x00
	offsetSampleCount = 0x04
	offsetSampleIndex = 0x08
	offsetADCErrors   = 0x0C
	offsetSamples     = 0x10 // Start of 32-sample array

	// Number of samples in ring buffer
	maxSamples = 32
)

// AudioControlBlock represents the PRU1 audio control block
// Layout: Configuration (written by host) + Status (written by PRU)
type AudioControlBlock struct {
	// === Configuration Section (written by host, read by PRU) ===
	FFTEnable     uint32  // 1 = FFT enabled, 0 = disabled
	BassMaxHz     uint32  // Bass upper frequency boundary (Hz)
	MidLowMaxHz   uint32  // Mid-low upper frequency boundary (Hz)
	MidHighMaxHz  uint32  // Mid-high upper frequency boundary (Hz)

	// === Status Section (written by PRU, read by host) ===
	Status           uint32  // PRU running status
	TotalSamples     uint32  // Total samples collected
	BufferCount      uint32  // Number of completed buffers
	CurrentBuffer    uint32  // 0 = Buffer A, 1 = Buffer B
	SamplesInBuffer  uint32  // Current sample count in active buffer
	ADCTimeouts      uint32  // ADC timeout errors
	MissedSamples    uint32  // Missed samples (overruns)
	LastSample       uint32  // Most recent sample value
	MinSample        uint32  // Minimum sample value
	MaxSample        uint32  // Maximum sample value
	FFTCount         uint32  // Number of FFTs computed
	FFTTimeCycles    uint32  // Last FFT processing time (PRU cycles)
	FFTSkipped       uint32  // FFTs skipped due to timing overrun
	Bass             uint32  // Bass magnitude (0-bass_max_hz)
	MidLow           uint32  // Mid-low magnitude (bass_max_hz-midlow_max_hz)
	MidHigh          uint32  // Mid-high magnitude (midlow_max_hz-midhigh_max_hz)
	Treble           uint32  // Treble magnitude (midhigh_max_hz-Nyquist)
}

// sampleCmd represents the sample command
var sampleCmd = &cobra.Command{
	Use:   "sample",
	Short: "Display real-time ADC samples from PRU1",
	Long: `Reads and displays ADC samples from AIN1 collected by PRU1.
Shows the most recent samples in real-time, updating every second.`,
	Run: runSample,
}

func init() {
	rootCmd.AddCommand(sampleCmd)
}

func runSample(cmd *cobra.Command, args []string) {
	// Open /dev/mem for memory-mapped I/O
	memFile, err := os.OpenFile("/dev/mem", os.O_RDWR|os.O_SYNC, 0)
	if err != nil {
		fmt.Printf("Error opening /dev/mem: %v\n", err)
		fmt.Println("Try running with sudo")
		os.Exit(1)
	}
	defer memFile.Close()

	// Memory map the PRU shared memory
	mem, err := syscall.Mmap(
		int(memFile.Fd()),
		int64(pruSharedMemBase),
		pruSharedMemSize,
		syscall.PROT_READ|syscall.PROT_WRITE,
		syscall.MAP_SHARED,
	)
	if err != nil {
		fmt.Printf("Error mapping memory: %v\n", err)
		os.Exit(1)
	}
	defer syscall.Munmap(mem)

	// Get pointer to audio control block
	controlBlockPtr := unsafe.Pointer(&mem[audioControlBlockOffset])

	// Set up signal handler for clean exit
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Read initial control block to show configuration
	ctrl := (*AudioControlBlock)(controlBlockPtr)

	fmt.Println("PRU1 Audio Sampling Monitor - 40 kHz IEP Timer Mode")
	fmt.Println("====================================================")
	fmt.Println("High-speed ADC sampling from AIN1 at 40 kHz")
	fmt.Printf("FFT: %s | Bands: Bass 0-%d Hz, Mid-Low %d-%d Hz, Mid-High %d-%d Hz, Treble %d-20000 Hz\n",
		map[uint32]string{0: "DISABLED", 1: "ENABLED"}[ctrl.FFTEnable],
		ctrl.BassMaxHz,
		ctrl.BassMaxHz, ctrl.MidLowMaxHz,
		ctrl.MidLowMaxHz, ctrl.MidHighMaxHz,
		ctrl.MidHighMaxHz,
	)
	fmt.Println("Press Ctrl+C to exit")
	fmt.Println()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	lastTotalSamples := uint32(0)
	lastBufferCount := uint32(0)

	for {
		select {
		case <-sigChan:
			fmt.Println("\nExiting...")
			return

		case <-ticker.C:
			// Read control block
			ctrl := (*AudioControlBlock)(controlBlockPtr)

			// Calculate rates
			samplesPerSec := ctrl.TotalSamples - lastTotalSamples
			buffersPerSec := ctrl.BufferCount - lastBufferCount
			lastTotalSamples = ctrl.TotalSamples
			lastBufferCount = ctrl.BufferCount

			// Display status
			statusStr := decodeStatus(ctrl.Status)
			currentBuf := "A"
			if ctrl.CurrentBuffer == 1 {
				currentBuf = "B"
			}

			// Calculate voltage from ADC value (12-bit, 0-4095, 0-1.8V)
			lastVoltage := float64(ctrl.LastSample) * 1.8 / 4095.0
			minVoltage := float64(ctrl.MinSample) * 1.8 / 4095.0
			maxVoltage := float64(ctrl.MaxSample) * 1.8 / 4095.0

			// Calculate FFT timing (cycles @ 200 MHz)
			fftTimeUs := float64(ctrl.FFTTimeCycles) / 200.0  // Convert to microseconds

			fmt.Printf("\r%-10s | Rate: %5d Hz | Buffers: %6d (%d/s) | Buf: %s [%4d/%4d] | Timeouts: %4d | Skipped: %4d",
				statusStr,
				samplesPerSec,
				ctrl.BufferCount,
				buffersPerSec,
				currentBuf,
				ctrl.SamplesInBuffer,
				1024,
				ctrl.ADCTimeouts,
				ctrl.FFTSkipped,
			)

			fmt.Printf("\n")
			fmt.Printf("Last: %4d (%.3fV) | Min: %4d (%.3fV) | Max: %4d (%.3fV) | FFTs: %6d (%.2f ms)",
				ctrl.LastSample, lastVoltage,
				ctrl.MinSample, minVoltage,
				ctrl.MaxSample, maxVoltage,
				ctrl.FFTCount,
				fftTimeUs/1000.0,
			)

			fmt.Printf("\n")
			fmt.Printf("Bass: %10d | Mid-Low: %10d | Mid-High: %10d | Treble: %10d",
				ctrl.Bass,
				ctrl.MidLow,
				ctrl.MidHigh,
				ctrl.Treble,
			)

			// Move cursor up to overwrite previous output (now 2 lines)
			fmt.Print("\033[2A")
		}
	}
}

// decodeStatus converts the status code to a human-readable string
func decodeStatus(status uint32) string {
	switch status {
	case 0x41554431: // "AUD1"
		return "RUNNING"
	case 0x41444349: // "ADCI"
		return "ADC_INIT"
	case 0x49455049: // "IEPI"
		return "IEP_INIT"
	case 0x53414D50: // "SAMP"
		return "SAMPLING"
	case 0x46465450: // "FFTP"
		return "FFT_PROC"
	default:
		return fmt.Sprintf("UNK(0x%08X)", status)
	}
}
