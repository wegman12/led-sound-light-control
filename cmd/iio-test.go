package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/wegman12/led-sound-light-control/sound"
)

var iioTestCmd = &cobra.Command{
	Use:   "iio-test",
	Short: "Test Linux IIO high-speed ADC sampling",
	Long:  `Test the Linux IIO interface for high-speed buffered ADC sampling`,
	Run:   runIIOTest,
}

func init() {
	rootCmd.AddCommand(iioTestCmd)
}

func runIIOTest(cmd *cobra.Command, args []string) {
	fmt.Println("Linux IIO High-Speed ADC Test")
	fmt.Println("==============================")
	fmt.Println()

	// Check if IIO is available
	if err := sound.CheckIIOAvailable(); err != nil {
		fmt.Printf("ERROR: %v\n", err)
		fmt.Println("\nTroubleshooting:")
		fmt.Println("1. Check if ADC is enabled:")
		fmt.Println("   ls -la /sys/bus/iio/devices/iio:device0/")
		fmt.Println("2. Check if you need root access:")
		fmt.Println("   sudo ./bin/led-sound-light-control iio-test")
		return
	}

	fmt.Println("✓ IIO device found")

	// Create reader for AIN1 (channel 1) at 48 kHz
	reader, err := sound.NewFastIIOReader(1, 48000, 512)
	if err != nil {
		fmt.Printf("ERROR: Failed to create IIO reader: %v\n", err)
		return
	}

	fmt.Println("✓ IIO reader configured")

	// Try to get actual sample rate
	if rate, err := reader.GetActualSampleRate(); err == nil {
		fmt.Printf("✓ Configured sample rate: %d Hz\n", rate)
	}

	fmt.Println()
	fmt.Println("Starting continuous sampling...")
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println()

	// Start sampling
	if err := reader.Start(); err != nil {
		fmt.Printf("ERROR: Failed to start sampling: %v\n", err)
		return
	}
	defer reader.Stop()

	fmt.Println("✓ Sampling started")
	fmt.Println()

	startTime := time.Now()
	lastPrintTime := time.Now()
	totalSamples := uint64(0)
	lastTotalSamples := uint64(0)

	for {
		// Read samples
		samples, err := reader.ReadSamples()
		if err != nil {
			fmt.Printf("ERROR: %v\n", err)
			return
		}

		totalSamples += uint64(len(samples))

		// Print statistics every second
		if time.Since(lastPrintTime) >= 1*time.Second {
			elapsed := time.Since(startTime)
			samplesThisSecond := totalSamples - lastTotalSamples
			samplesPerSecond := float64(samplesThisSecond) / time.Since(lastPrintTime).Seconds()
			lastTotalSamples = totalSamples

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

				fmt.Printf("[%6.1fs] Rate: %8.0f Hz | Total: %9d | Batch: %4d | ADC: [%4d - %4d] | Voltage: [%.3fV - %.3fV] Avg: %.3fV\n",
					elapsed.Seconds(), samplesPerSecond, totalSamples, len(samples),
					min, max, minV, maxV, avgV)
			} else {
				fmt.Printf("[%6.1fs] Rate: %8.0f Hz | Total: %9d | No samples in this batch\n",
					elapsed.Seconds(), samplesPerSecond, totalSamples)
			}

			lastPrintTime = time.Now()
		}

		// Small sleep to avoid busy waiting
		time.Sleep(1 * time.Millisecond)
	}
}
