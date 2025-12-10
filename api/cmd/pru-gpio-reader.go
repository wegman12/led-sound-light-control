package main

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

const (
	pruSharedMemAddr  = 0x4A310000
	pruSharedMemSize  = 0x3000
	gpioRawDataOffset = 0x2000
	maxGpioSamples    = 8192
)

type gpioRawData struct {
	SampleCount            uint32
	Complete               uint32
	FirstFullRegisterValue uint32
	Samples                [maxGpioSamples]uint8
}

func main() {
	fmt.Println("PRU Raw GPIO Data Reader")
	fmt.Println("========================")
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

	// Get pointer to GPIO raw data
	gpioData := (*gpioRawData)(unsafe.Pointer(&mem[gpioRawDataOffset]))

	fmt.Printf("Capture complete: %v\n", gpioData.Complete != 0)
	fmt.Printf("Sample count: %d\n", gpioData.SampleCount)
	fmt.Println("")

	if gpioData.SampleCount == 0 {
		fmt.Println("No samples captured yet. Make sure the PRU code is running and press a button.")
		os.Exit(0)
	}

	// Display full GPIO register value for pin verification
	fmt.Printf("DEBUG: First GPIO DATAIN register value: 0x%08X\n", gpioData.FirstFullRegisterValue)
	fmt.Printf("  Bit 20 (our target): %d\n", (gpioData.FirstFullRegisterValue>>20)&1)
	fmt.Printf("  All 32 bits: %032b\n", gpioData.FirstFullRegisterValue)
	fmt.Println("")

	// Analyze the data
	zeros := 0
	ones := 0
	transitions := 0
	lastState := gpioData.Samples[0]

	for i := uint32(0); i < gpioData.SampleCount && i < maxGpioSamples; i++ {
		sample := gpioData.Samples[i]
		if sample == 0 {
			zeros++
		} else {
			ones++
		}

		if i > 0 && sample != lastState {
			transitions++
		}
		lastState = sample
	}

	fmt.Printf("Statistics:\n")
	fmt.Printf("  Total samples: %d\n", gpioData.SampleCount)
	fmt.Printf("  LOW (0) readings: %d (%.1f%%)\n", zeros, float64(zeros)*100/float64(gpioData.SampleCount))
	fmt.Printf("  HIGH (1) readings: %d (%.1f%%)\n", ones, float64(ones)*100/float64(gpioData.SampleCount))
	fmt.Printf("  Transitions: %d\n", transitions)
	fmt.Println("")

	// Print first 200 samples in groups of 50 for readability
	fmt.Println("First 200 samples (0=LOW, 1=HIGH):")
	for start := uint32(0); start < 200 && start < gpioData.SampleCount; start += 50 {
		end := start + 50
		if end > gpioData.SampleCount {
			end = gpioData.SampleCount
		}
		if end > 200 {
			end = 200
		}

		fmt.Printf("%4d-%4d: ", start, end-1)
		for i := start; i < end; i++ {
			fmt.Printf("%d", gpioData.Samples[i])
		}
		fmt.Println()
	}
	fmt.Println("")

	// Find and display runs (consecutive same values)
	fmt.Println("Run-length encoding (first 50 runs):")
	fmt.Println("State | Run Length")
	fmt.Println("------+-----------")

	runCount := 0
	currentState := gpioData.Samples[0]
	runLength := uint32(1)

	for i := uint32(1); i < gpioData.SampleCount && runCount < 50; i++ {
		if gpioData.Samples[i] == currentState {
			runLength++
		} else {
			// End of run
			stateStr := "HIGH"
			if currentState == 0 {
				stateStr = "LOW "
			}
			fmt.Printf("%s  | %d\n", stateStr, runLength)
			runCount++

			// Start new run
			currentState = gpioData.Samples[i]
			runLength = 1
		}
	}

	// Print final run
	if runLength > 0 && runCount < 50 {
		stateStr := "HIGH"
		if currentState == 0 {
			stateStr = "LOW "
		}
		fmt.Printf("%s  | %d\n", stateStr, runLength)
	}
}
