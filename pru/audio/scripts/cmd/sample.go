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
type AudioControlBlock struct {
	Status      uint32
	SampleCount uint32
	SampleIndex uint32
	ADCErrors   uint32
	Samples     [maxSamples]uint32
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

	fmt.Println("PRU1 Audio Sampling Monitor")
	fmt.Println("============================")
	fmt.Println("Reading ADC samples from AIN1...")
	fmt.Println("Press Ctrl+C to exit")
	fmt.Println()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	lastSampleCount := uint32(0)

	for {
		select {
		case <-sigChan:
			fmt.Println("\nExiting...")
			return

		case <-ticker.C:
			// Read control block
			ctrl := (*AudioControlBlock)(controlBlockPtr)

			// Display status
			statusStr := decodeStatus(ctrl.Status)
			sampleRate := ctrl.SampleCount - lastSampleCount
			lastSampleCount = ctrl.SampleCount

			fmt.Printf("\rStatus: %s | Samples: %d | Rate: %d/s | Errors: %d | Index: %d",
				statusStr,
				ctrl.SampleCount,
				sampleRate,
				ctrl.ADCErrors,
				ctrl.SampleIndex,
			)

			// Display last 8 samples
			fmt.Print(" | Last 8: [")
			startIdx := int(ctrl.SampleIndex) - 8
			if startIdx < 0 {
				startIdx += maxSamples
			}

			for i := 0; i < 8; i++ {
				idx := (startIdx + i) % maxSamples
				sample := ctrl.Samples[idx]

				if i > 0 {
					fmt.Print(", ")
				}
				fmt.Printf("%4d", sample)
			}
			fmt.Print("]")
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
	case 0x41445353: // "ADSS"
		return "SAMPLING"
	default:
		return fmt.Sprintf("UNKNOWN(0x%08X)", status)
	}
}
