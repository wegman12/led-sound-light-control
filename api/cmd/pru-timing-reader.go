package main

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

const (
	pruSharedMemAddr   = 0x4A310000
	pruSharedMemSize   = 0x3000
	timingDataOffset   = 0x2000
	maxTimingSamples   = 512
	pruClockHz         = 200000000 // 200 MHz
	cyclesPerUs        = 200       // 200 cycles per microsecond
)

type timingSample struct {
	State    uint32
	Duration uint32
}

type timingData struct {
	SampleCount uint32
	Complete    uint32
	Samples     [maxTimingSamples]timingSample
}

func main() {
	fmt.Println("PRU Timing Data Reader")
	fmt.Println("======================")
	fmt.Println("")

	// Open /dev/mem
	memFile, err := os.OpenFile("/dev/mem", os.O_RDWR|os.O_SYNC, 0)
	if err != nil {
		fmt.Printf("ERROR: Failed to open /dev/mem: %v\n", err)
		fmt.Println("Are you running with sudo?")
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
		fmt.Printf("ERROR: Failed to mmap PRU memory at 0x%X: %v\n", pruSharedMemAddr, err)
		os.Exit(1)
	}
	defer syscall.Munmap(mem)

	fmt.Printf("✓ Successfully mapped PRU shared memory at 0x%X\n", pruSharedMemAddr)
	fmt.Println("")

	// Get pointer to timing data
	timing := (*timingData)(unsafe.Pointer(&mem[timingDataOffset]))

	fmt.Printf("Timing capture complete: %v\n", timing.Complete != 0)
	fmt.Printf("Sample count: %d\n", timing.SampleCount)
	fmt.Println("")

	if timing.SampleCount == 0 {
		fmt.Println("No timing samples captured yet. Make sure the PRU code is running and press a button.")
		os.Exit(0)
	}

	// Print timing data
	fmt.Println("Timing Samples:")
	fmt.Println("Index | State | Duration (cycles) | Duration (μs) | Duration (ms)")
	fmt.Println("------+-------+-------------------+---------------+-------------")

	totalCycles := uint64(0)
	for i := uint32(0); i < timing.SampleCount && i < maxTimingSamples; i++ {
		sample := timing.Samples[i]
		durationUs := float64(sample.Duration) / float64(cyclesPerUs)
		durationMs := durationUs / 1000.0
		totalCycles += uint64(sample.Duration)

		stateStr := "HIGH"
		if sample.State == 0 {
			stateStr = "LOW "
		}

		fmt.Printf("%5d | %4s  | %17d | %13.2f | %11.3f\n",
			i, stateStr, sample.Duration, durationUs, durationMs)
	}

	fmt.Println("")
	totalUs := float64(totalCycles) / float64(cyclesPerUs)
	totalMs := totalUs / 1000.0
	fmt.Printf("Total duration: %d cycles (%.2f μs / %.3f ms)\n", totalCycles, totalUs, totalMs)
	fmt.Println("")

	// Print bit pattern interpretation (assuming HIGH durations encode bits)
	fmt.Println("Bit pattern interpretation (based on HIGH durations):")
	fmt.Println("(Using 1ms threshold: <1ms = 0, >=1ms = 1)")
	fmt.Println("")

	bitPattern := ""
	for i := uint32(0); i < timing.SampleCount && i < maxTimingSamples; i++ {
		sample := timing.Samples[i]
		if sample.State == 1 { // Only look at HIGH durations
			durationUs := float64(sample.Duration) / float64(cyclesPerUs)
			if durationUs >= 1000.0 {
				bitPattern += "1"
			} else {
				bitPattern += "0"
			}
		}
	}

	fmt.Printf("Bits: %s\n", bitPattern)
	fmt.Printf("Bit count: %d\n", len(bitPattern))
}
