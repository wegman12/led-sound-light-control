// I2S Signal Monitor - Result Reader
// Reads PRU monitoring results from shared memory
package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"syscall"
	"time"
)

const (
	PRU_SHARED_MEM_ADDR = 0x4A310000
	RESULTS_SIZE        = 28 // 7 uint32 fields
	PRU_REMOTEPROC_PATH = "/sys/class/remoteproc/remoteproc1/state"
)

type MonitorResults struct {
	WSTransitions  uint32
	SCKTransitions uint32
	SDTransitions  uint32
	SampleCount    uint32
	WSHighTime     uint32
	WSLowTime      uint32
	Status         uint32
}

func main() {
	fmt.Println("PRU I2S Signal Monitor - Results Reader")
	fmt.Println("========================================")

	// Stop PRU if running
	fmt.Println("Stopping PRU0...")
	if err := os.WriteFile(PRU_REMOTEPROC_PATH, []byte("stop"), 0644); err != nil {
		fmt.Printf("Warning: Could not stop PRU: %v\n", err)
	}
	time.Sleep(500 * time.Millisecond)

	// Load new firmware
	fmt.Println("Loading I2S monitor firmware...")
	if err := os.WriteFile(PRU_REMOTEPROC_PATH, []byte("start"), 0644); err != nil {
		fmt.Printf("Error starting PRU: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("PRU monitoring for 1 second...")
	time.Sleep(1200 * time.Millisecond)

	// Open /dev/mem to read PRU shared memory
	f, err := os.OpenFile("/dev/mem", os.O_RDONLY|os.O_SYNC, 0)
	if err != nil {
		fmt.Printf("Error opening /dev/mem: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	// Map PRU shared memory
	mem, err := syscall.Mmap(
		int(f.Fd()),
		int64(PRU_SHARED_MEM_ADDR),
		RESULTS_SIZE,
		syscall.PROT_READ,
		syscall.MAP_SHARED,
	)
	if err != nil {
		fmt.Printf("Error mapping memory: %v\n", err)
		os.Exit(1)
	}
	defer syscall.Munmap(mem)

	// Read results
	results := MonitorResults{
		WSTransitions:  binary.LittleEndian.Uint32(mem[0:4]),
		SCKTransitions: binary.LittleEndian.Uint32(mem[4:8]),
		SDTransitions:  binary.LittleEndian.Uint32(mem[8:12]),
		SampleCount:    binary.LittleEndian.Uint32(mem[12:16]),
		WSHighTime:     binary.LittleEndian.Uint32(mem[16:20]),
		WSLowTime:      binary.LittleEndian.Uint32(mem[20:24]),
		Status:         binary.LittleEndian.Uint32(mem[24:28]),
	}

	// Display results
	fmt.Println("\nMonitoring Results:")
	fmt.Println("==================")
	fmt.Printf("Status: %s\n", map[uint32]string{0: "Completed", 1: "Running"}[results.Status])
	fmt.Printf("Samples taken: %d\n", results.SampleCount)
	fmt.Println()

	fmt.Println("Signal Activity:")
	fmt.Printf("  WS (P9_29) transitions:  %d\n", results.WSTransitions)
	fmt.Printf("  SCK (P9_31) transitions: %d\n", results.SCKTransitions)
	fmt.Printf("  SD (P9_30) transitions:  %d\n", results.SDTransitions)
	fmt.Println()

	// Analyze results
	fmt.Println("Analysis:")
	if results.WSTransitions > 0 {
		// Calculate frequency: transitions / 2 (full cycle) / time (1 second)
		wsFreq := float64(results.WSTransitions) / 2.0
		fmt.Printf("  ✓ WS Clock detected: ~%.0f Hz (expected: 48000 Hz)\n", wsFreq)

		// Calculate WS period
		if results.WSHighTime > 0 {
			wsHighUs := float64(results.WSHighTime) * 1000 / 200.0  // Convert cycles to µs
			wsLowUs := float64(results.WSLowTime) * 1000 / 200.0
			fmt.Printf("    WS high time: %.1f µs\n", wsHighUs)
			fmt.Printf("    WS low time: %.1f µs\n", wsLowUs)
		}
	} else {
		fmt.Println("  ✗ NO WS clock activity detected on P9_29")
		fmt.Println("    → McASP may not be generating frame sync")
	}

	if results.SCKTransitions > 0 {
		sckFreq := float64(results.SCKTransitions) / 2.0
		fmt.Printf("  ✓ SCK Clock detected: ~%.0f Hz (expected: 3072000 Hz)\n", sckFreq)
	} else {
		fmt.Println("  ✗ NO SCK clock activity detected on P9_31")
		fmt.Println("    → McASP may not be generating bit clock")
	}

	if results.SDTransitions > 0 {
		fmt.Printf("  ✓ Data signal activity detected on P9_30 (%d transitions)\n", results.SDTransitions)
	} else {
		fmt.Println("  ✗ NO data signal activity on P9_30")
		if results.SCKTransitions == 0 {
			fmt.Println("    → No clocks present, so no data expected")
		} else {
			fmt.Println("    → Clocks present but microphone not sending data")
		}
	}

	fmt.Println()
	fmt.Println("Conclusion:")
	if results.WSTransitions == 0 && results.SCKTransitions == 0 {
		fmt.Println("  → Device tree pin configuration may be incorrect")
		fmt.Println("  → OR McASP is not starting (driver issue)")
		fmt.Println("  → Check: dmesg | grep mcasp")
	} else if results.SDTransitions == 0 {
		fmt.Println("  → Clocks are present but microphone not responding")
		fmt.Println("  → Check: INMP441 power, wiring, L/R pin connection")
	} else {
		fmt.Println("  → All signals present! Check ALSA configuration.")
	}
}
