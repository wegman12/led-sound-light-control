package cmd

import (
	"fmt"
	"os"
	"syscall"
	"time"
	"unsafe"

	"github.com/spf13/cobra"
)

const (
	// PRU Shared Memory Configuration
	pruSharedMemAddr = 0x4A310000
	devMem           = "/dev/mem"
)

// audioControlBlock matches the C struct in PRU1 firmware (both ADC and I2S)
type audioControlBlock struct {
	// Configuration section (written by host)
	FFTEnable           uint32
	BassMaxHz           uint32
	MidLowMaxHz         uint32
	MidHighMaxHz        uint32
	SmoothingAlphaX1000 uint32

	// Status section (written by PRU)
	Status          uint32
	TotalSamples    uint32
	BufferCount     uint32
	CurrentBuffer   uint32
	SamplesInBuffer uint32
	ADCTimeouts     uint32 // Also McASP errors in I2S mode
	MissedSamples   uint32
	LastSample      uint32
	MinSample       uint32
	MaxSample       uint32
	FFTCount        uint32
	FFTTimeCycles   uint32
	FFTSkipped      uint32
	Bass            uint32
	MidLow          uint32
	MidHigh         uint32
	Treble          uint32
	BassAvg         uint32
	MidLowAvg       uint32
	MidHighAvg      uint32
	TrebleAvg       uint32
}

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Test PRU1 audio operation",
	Long: `Monitor PRU1 audio status and verify operation.

This command verifies that:
  - PRU1 firmware is loaded and running
  - Shared memory is accessible
  - Audio samples are being collected
  - FFT processing is operational

Supports both ADC firmware (40 kHz) and I2S firmware (48 kHz).
Press Ctrl+C to exit.`,
	Run: runTest,
}

func init() {
	rootCmd.AddCommand(testCmd)
}

// decodeStatusFull returns firmware mode, description, and sample rate from status code
func decodeStatusFull(status uint32) (mode string, desc string, sampleRate uint32) {
	switch status {
	case statusI2SRunning:
		return "I2S", "I2S/McASP firmware (48 kHz)", sampleRateI2S
	case statusADCRunning:
		return "ADC", "ADC audio firmware (40 kHz)", sampleRateADC
	case 0x4D435350: // "MCSP"
		return "I2S", "McASP initializing", sampleRateI2S
	case 0x53414D50: // "SAMP"
		return "I2S", "Sampling active", sampleRateI2S
	case 0x46465450: // "FFTP"
		return "I2S", "FFT processing", sampleRateI2S
	case 0x45525221: // "ERR!"
		return "ERR", "Error occurred", 0
	case 0:
		return "OFF", "Not running", 0
	default:
		return "UNK", fmt.Sprintf("Unknown (0x%08X)", status), 0
	}
}

func runTest(cmd *cobra.Command, args []string) {
	fmt.Println("PRU1 Audio Test")
	fmt.Println("===============")
	fmt.Println()

	// Open /dev/mem (requires root privileges)
	memFile, err := os.OpenFile(devMem, os.O_RDWR|os.O_SYNC, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to open %s (need root): %v\n", devMem, err)
		fmt.Fprintln(os.Stderr, "Try running with: sudo ./audio-util test")
		os.Exit(1)
	}
	defer memFile.Close()

	// Memory map PRU shared memory
	mem, err := syscall.Mmap(
		int(memFile.Fd()),
		pruSharedMemAddr,
		pruSharedMemSize,
		syscall.PROT_READ|syscall.PROT_WRITE,
		syscall.MAP_SHARED,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to mmap PRU memory at 0x%X: %v\n", pruSharedMemAddr, err)
		os.Exit(1)
	}
	defer syscall.Munmap(mem)

	// Get pointer to audio control block
	controlBlock := (*audioControlBlock)(unsafe.Pointer(&mem[audioControlBlockOffset]))

	// Check if PRU1 is running
	if controlBlock.Status == 0 {
		fmt.Fprintln(os.Stderr, "Error: PRU1 firmware not running (status=0)")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Load I2S firmware with:")
		fmt.Fprintln(os.Stderr, "  cd pru/audio && make i2s-deploy-all")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Or ADC firmware with:")
		fmt.Fprintln(os.Stderr, "  cd pru/audio && make deploy-all")
		os.Exit(1)
	}

	// Decode firmware mode
	mode, desc, sampleRate := decodeStatusFull(controlBlock.Status)

	fmt.Printf("Status:      0x%08X (%s)\n", controlBlock.Status, desc)
	fmt.Printf("Mode:        %s\n", mode)
	if sampleRate > 0 {
		fmt.Printf("Sample Rate: %d Hz\n", sampleRate)
	}
	fmt.Printf("FFT Enabled: %v\n", controlBlock.FFTEnable == 1)
	fmt.Printf("Freq Bands:  Bass<%d Hz, MidLow<%d Hz, MidHigh<%d Hz\n",
		controlBlock.BassMaxHz, controlBlock.MidLowMaxHz, controlBlock.MidHighMaxHz)
	fmt.Println()
	fmt.Println("Monitoring audio samples (Ctrl+C to exit)...")
	fmt.Println()

	lastSamples := controlBlock.TotalSamples
	lastFFTCount := controlBlock.FFTCount
	lastTime := time.Now()

	for {
		time.Sleep(1 * time.Second)

		currentSamples := controlBlock.TotalSamples
		currentFFTCount := controlBlock.FFTCount
		currentTime := time.Now()

		elapsed := currentTime.Sub(lastTime).Seconds()

		// Calculate rates
		sampleDelta := currentSamples - lastSamples
		fftDelta := currentFFTCount - lastFFTCount

		sampleRateActual := float64(sampleDelta) / elapsed
		fftRate := float64(fftDelta) / elapsed

		// Display current state
		timestamp := currentTime.Format("15:04:05")
		fmt.Printf("[%s] Samples: %8d (+%5d) | FFTs: %5d (+%2d) | Rate: %.0f Hz | FFT/s: %.1f\n",
			timestamp, currentSamples, sampleDelta, currentFFTCount, fftDelta, sampleRateActual, fftRate)

		// Show frequency bands every 5 seconds
		if int(currentTime.Unix())%5 == 0 {
			fmt.Printf("           Bass: %6d | MidLow: %6d | MidHigh: %6d | Treble: %6d\n",
				controlBlock.Bass, controlBlock.MidLow, controlBlock.MidHigh, controlBlock.Treble)
		}

		// Check for errors
		if controlBlock.ADCTimeouts > 0 {
			if mode == "I2S" {
				fmt.Printf("           WARNING: %d McASP errors\n", controlBlock.ADCTimeouts)
			} else {
				fmt.Printf("           WARNING: %d ADC timeouts\n", controlBlock.ADCTimeouts)
			}
		}
		if controlBlock.MissedSamples > 0 {
			fmt.Printf("           WARNING: %d missed samples\n", controlBlock.MissedSamples)
		}

		lastSamples = currentSamples
		lastFFTCount = currentFFTCount
		lastTime = currentTime
	}
}
