package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/wegman12/led-sound-light-control/sound"
)

var pruTestCmd = &cobra.Command{
	Use:   "pru-test",
	Short: "Test PRU ADC sampling",
	Long:  `Test the PRU ADC reader by displaying real-time sample statistics`,
	Run:   runPRUTest,
}

func init() {
	rootCmd.AddCommand(pruTestCmd)
}

func runPRUTest(cmd *cobra.Command, args []string) {
	fmt.Println("PRU ADC Sampling Test")
	fmt.Println("=====================")
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println()

	// Create PRU reader
	reader, err := sound.NewPRUReader()
	if err != nil {
		fmt.Printf("ERROR: Failed to initialize PRU reader: %v\n", err)
		fmt.Println("\nMake sure you:")
		fmt.Println("  1. Run this as root: sudo ./bin/led-sound-light-control pru-test")
		fmt.Println("  2. PRU firmware is loaded (check scripts/check_pru_status.sh)")
		fmt.Println("  3. Device tree overlay is enabled")
		return
	}
	defer reader.Close()

	fmt.Println("✓ PRU reader initialized successfully")
	fmt.Println()

	// Reset stats
	reader.ResetStats()

	// Sampling statistics
	startTime := time.Now()
	lastPrintTime := time.Now()
	var lastSampleCount uint32

	for {
		// Read samples
		samples, err := reader.ReadSamples()
		if err != nil {
			fmt.Printf("ERROR: %v\n", err)
			return
		}

		// Calculate statistics every second
		if time.Since(lastPrintTime) >= 1*time.Second {
			elapsed := time.Since(startTime)
			sampleCount, overrunCount := reader.GetStats()
			samplesPerSecond := float64(sampleCount-lastSampleCount) / time.Since(lastPrintTime).Seconds()
			lastSampleCount = sampleCount

			// Calculate min, max, average from recent samples
			if len(samples) > 0 {
				var min, max uint16 = 4095, 0
				var sum uint64
				for _, sample := range samples {
					if sample < min {
						min = sample
					}
					if sample > max {
						max = sample
					}
					sum += uint64(sample)
				}
				avg := float64(sum) / float64(len(samples))

				// Convert to voltage (0-1.8V for BeagleBone Black)
				minV := float64(min) * 1.8 / 4095.0
				maxV := float64(max) * 1.8 / 4095.0
				avgV := avg * 1.8 / 4095.0

				fmt.Printf("[%6.1fs] Rate: %7.0f Hz | Samples: %9d | Overruns: %4d | ADC: [%4d - %4d] (%.3fV - %.3fV) Avg: %.3fV\n",
					elapsed.Seconds(), samplesPerSecond, sampleCount, overrunCount,
					min, max, minV, maxV, avgV)
			} else {
				fmt.Printf("[%6.1fs] Rate: %7.0f Hz | Samples: %9d | Overruns: %4d | No samples available\n",
					elapsed.Seconds(), samplesPerSecond, sampleCount, overrunCount)
			}

			lastPrintTime = time.Now()

			// Warning if overruns detected
			if overrunCount > 0 {
				fmt.Println("  ⚠ WARNING: Buffer overruns detected! Reading too slowly.")
			}
		}

		// Small sleep to avoid busy waiting
		time.Sleep(10 * time.Millisecond)
	}
}
