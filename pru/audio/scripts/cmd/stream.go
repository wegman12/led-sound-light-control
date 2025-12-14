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

// streamCmd represents the stream command
var streamCmd = &cobra.Command{
	Use:   "stream",
	Short: "Stream real-time sound profile data from PRU1",
	Long: `Displays live frequency band analysis (bass, mid-low, mid-high, treble)
from PRU1 audio sampling. Shows both sum and average magnitude values
with visual bar graphs and update statistics.`,
	Run: runStream,
}

func init() {
	rootCmd.AddCommand(streamCmd)
}

func runStream(cmd *cobra.Command, args []string) {
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

	fmt.Println("PRU1 Sound Profile Stream")
	fmt.Println("=========================")
	fmt.Printf("Configuration: FFT %s | Sample Rate: 40 kHz\n",
		map[uint32]string{0: "DISABLED", 1: "ENABLED"}[ctrl.FFTEnable])
	fmt.Printf("Frequency Bands: Bass 0-%d Hz | Mid-Low %d-%d Hz | Mid-High %d-%d Hz | Treble %d-20k Hz\n",
		ctrl.BassMaxHz,
		ctrl.BassMaxHz, ctrl.MidLowMaxHz,
		ctrl.MidLowMaxHz, ctrl.MidHighMaxHz,
		ctrl.MidHighMaxHz)
	fmt.Println("Press Ctrl+C to exit")
	fmt.Println()

	ticker := time.NewTicker(250 * time.Millisecond) // 4 updates per second
	defer ticker.Stop()

	lastFFTCount := uint32(0)
	startTime := time.Now()

	for {
		select {
		case <-sigChan:
			fmt.Println("\nExiting...")
			return

		case <-ticker.C:
			// Read control block
			ctrl := (*AudioControlBlock)(controlBlockPtr)

			// Calculate FFT rate
			fftDelta := ctrl.FFTCount - lastFFTCount
			lastFFTCount = ctrl.FFTCount
			fftPerSec := float64(fftDelta) * 4.0 // 4 updates per second

			// Calculate elapsed time
			elapsed := time.Since(startTime)

			// Display header
			fmt.Printf("\rFFTs: %6d | Rate: %.1f/s | Elapsed: %s | Status: %s",
				ctrl.FFTCount,
				fftPerSec,
				formatDuration(elapsed),
				decodeStatus(ctrl.Status),
			)

			// Display sum values with bar graphs
			fmt.Printf("\n")
			fmt.Printf("Sum  | Bass: %10d %s\n", ctrl.Bass, makeBar(ctrl.Bass, 200000000))
			fmt.Printf("     | MLow: %10d %s\n", ctrl.MidLow, makeBar(ctrl.MidLow, 100000000))
			fmt.Printf("     | MHgh: %10d %s\n", ctrl.MidHigh, makeBar(ctrl.MidHigh, 100000000))
			fmt.Printf("     | Trbl: %10d %s\n", ctrl.Treble, makeBar(ctrl.Treble, 500000000))

			// Display average values with bar graphs
			fmt.Printf("\n")
			fmt.Printf("Avg  | Bass: %10d %s\n", ctrl.BassAvg, makeBar(ctrl.BassAvg, 50000000))
			fmt.Printf("     | MLow: %10d %s\n", ctrl.MidLowAvg, makeBar(ctrl.MidLowAvg, 5000000))
			fmt.Printf("     | MHgh: %10d %s\n", ctrl.MidHighAvg, makeBar(ctrl.MidHighAvg, 5000000))
			fmt.Printf("     | Trbl: %10d %s\n", ctrl.TrebleAvg, makeBar(ctrl.TrebleAvg, 2000000))

			// Move cursor up to overwrite previous output (9 lines total)
			fmt.Print("\033[9A")
		}
	}
}

// makeBar creates a visual bar graph
// value: current value
// scale: value for full bar (40 characters)
func makeBar(value uint32, scale uint32) string {
	const maxWidth = 40

	if scale == 0 {
		return ""
	}

	width := int((uint64(value) * maxWidth) / uint64(scale))
	if width > maxWidth {
		width = maxWidth
	}

	bar := ""
	for i := 0; i < width; i++ {
		bar += "█"
	}

	return bar
}

// formatDuration formats a duration as HH:MM:SS
func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}
