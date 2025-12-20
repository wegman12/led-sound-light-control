package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"os"
	"syscall"
	"time"
	"unsafe"
)

const (
	// PRU Shared Memory - Physical address
	pruSharedMemAddr = 0x4A310000
	pruSharedMemSize = 0x3000 // 12 KB

	// Raw I2S Capture Memory Layout (must match pru1_raw_capture_i2s.c)
	rawControlOffset = 0x0900 // PRU1 audio control offset (after PRU0 space)
	rawBufferOffset  = 0x0A00 // Sample buffer start
	rawBufferSize    = 2048   // Number of int32 samples (~64ms @ 32 kHz)
)

// RawI2SControlBlock matches the C struct in PRU firmware
type RawI2SControlBlock struct {
	EnableCapture    uint32
	Status           uint32
	TotalSamples     uint32
	BufferWriteIndex uint32
	BufferWrapCount  uint32
	McASPErrors      uint32
	LastSample       uint32
	MinSample        uint32
	MaxSample        uint32
	DebugGBLCTL      uint32
	DebugRSTAT       uint32
	DebugACLKRCTL    uint32
	DebugAHCLKRCTL   uint32
	DebugLoopCount   uint32
	Reserved         [50]uint32
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Raw I2S Sample Capture Utility")
		fmt.Println("================================")
		fmt.Println()
		fmt.Println("Usage: raw_i2s_capture <duration_seconds> <output_file>")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  raw_i2s_capture 10 baseline.bin       # Capture 10 seconds of silence")
		fmt.Println("  raw_i2s_capture 30 music_test.bin     # Capture 30 seconds of music")
		fmt.Println()
		fmt.Println("Output Format:")
		fmt.Println("  Binary file with 32-bit signed integers (little-endian)")
		fmt.Println("  Each sample is a 24-bit I2S value sign-extended to 32 bits")
		fmt.Println()
		fmt.Println("Post-Processing:")
		fmt.Println("  Use analyze_i2s_raw.py to analyze the captured data")
		os.Exit(1)
	}

	var duration int
	fmt.Sscanf(os.Args[1], "%d", &duration)
	outputFile := os.Args[2]

	fmt.Printf("Raw I2S Sample Capture\n")
	fmt.Printf("======================\n")
	fmt.Printf("Duration: %d seconds\n", duration)
	fmt.Printf("Output: %s\n\n", outputFile)

	// Open /dev/mem (requires root)
	memFile, err := os.OpenFile("/dev/mem", os.O_RDWR|os.O_SYNC, 0)
	if err != nil {
		fmt.Printf("ERROR: Failed to open /dev/mem (need root): %v\n", err)
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
		fmt.Printf("ERROR: Failed to mmap PRU memory: %v\n", err)
		os.Exit(1)
	}
	defer syscall.Munmap(mem)

	// Get pointers to control block and sample buffer
	ctrl := (*RawI2SControlBlock)(unsafe.Pointer(&mem[rawControlOffset]))
	sampleBuffer := unsafe.Slice((*int32)(unsafe.Pointer(&mem[rawBufferOffset])), rawBufferSize)

	// Verify PRU is running with raw I2S capture firmware
	// "RI2S" = 0x52493253, "CAPT" = 0x43415054
	if ctrl.Status != 0x52493253 && ctrl.Status != 0x43415054 {
		fmt.Printf("ERROR: PRU not running raw I2S capture firmware (status=0x%08X)\n", ctrl.Status)
		fmt.Printf("Expected: 0x52493253 (RI2S) or 0x43415054 (CAPT)\n")
		fmt.Printf("\nLoad firmware with:\n")
		fmt.Printf("  cd ~/led-sound-light-control/pru/audio\n")
		fmt.Printf("  make i2s-raw-deploy-all\n")
		os.Exit(1)
	}

	fmt.Printf("PRU Status: 0x%08X\n", ctrl.Status)
	fmt.Printf("PRU Total Samples: %d\n", ctrl.TotalSamples)
	fmt.Printf("PRU Min: %d, Max: %d, Last: %d\n",
		int32(ctrl.MinSample), int32(ctrl.MaxSample), int32(ctrl.LastSample))
	fmt.Printf("Buffer Size: %d samples\n", rawBufferSize)
	fmt.Printf("McASP Errors: %d\n\n", ctrl.McASPErrors)

	// Open output file with buffered writer
	outFile, err := os.Create(outputFile)
	if err != nil {
		fmt.Printf("ERROR: Failed to create output file: %v\n", err)
		os.Exit(1)
	}
	defer outFile.Close()

	// Use buffered writer to reduce I/O overhead
	writer := bufio.NewWriterSize(outFile, 64*1024) // 64KB buffer
	defer writer.Flush()

	// Capture loop
	startTime := time.Now()
	lastReadIndex := ctrl.BufferWriteIndex
	lastWrapCount := ctrl.BufferWrapCount
	totalWritten := 0

	// Poll interval: 10ms = ~320 samples at 32 kHz, well under 2048 buffer size
	pollInterval := 10 * time.Millisecond

	fmt.Printf("Starting capture...\n\n")

	// Initial delay to let buffer fill
	time.Sleep(pollInterval)

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		<-ticker.C

		elapsed := time.Since(startTime)
		if elapsed.Seconds() >= float64(duration) {
			break
		}

		currentIndex := ctrl.BufferWriteIndex
		currentWrapCount := ctrl.BufferWrapCount

		// Calculate samples to read
		var samplesToRead int
		var startIdx int

		if currentWrapCount > lastWrapCount {
			// Buffer wrapped
			samplesToRead = (rawBufferSize - int(lastReadIndex)) + int(currentIndex)
			startIdx = int(lastReadIndex)

			if currentWrapCount > lastWrapCount+1 {
				fmt.Printf("WARNING: Buffer wrapped %d times, data loss!\n", currentWrapCount-lastWrapCount)
				samplesToRead = rawBufferSize
				startIdx = int(currentIndex)
			}

			if samplesToRead > rawBufferSize {
				samplesToRead = rawBufferSize
				startIdx = int(currentIndex)
			}
		} else {
			// No wrap
			if currentIndex > lastReadIndex {
				samplesToRead = int(currentIndex - lastReadIndex)
				startIdx = int(lastReadIndex)

				if samplesToRead > rawBufferSize {
					samplesToRead = rawBufferSize
					startIdx = int(currentIndex)
				}
			} else {
				samplesToRead = 0
			}
		}

		// Read and write samples
		if samplesToRead > 0 {
			for i := 0; i < samplesToRead; i++ {
				idx := (startIdx + i) % rawBufferSize
				sample := sampleBuffer[idx]

				// Write as little-endian int32
				if err := binary.Write(writer, binary.LittleEndian, sample); err != nil {
					fmt.Printf("ERROR: Failed to write sample: %v\n", err)
					os.Exit(1)
				}
			}

			totalWritten += samplesToRead

			// Progress every second (assuming ~32 kHz sample rate)
			if totalWritten%(32000) < samplesToRead || totalWritten < samplesToRead*2 {
				rate := float64(totalWritten) / elapsed.Seconds()
				fmt.Printf("[%.1fs] %d samples written (%.1f kHz) | McASP errors: %d | Min: %d Max: %d\n",
					elapsed.Seconds(), totalWritten, rate/1000.0,
					ctrl.McASPErrors, int32(ctrl.MinSample), int32(ctrl.MaxSample))
			}
		}

		lastReadIndex = currentIndex
		lastWrapCount = currentWrapCount
	}

	// Flush remaining buffered data
	if err := writer.Flush(); err != nil {
		fmt.Printf("ERROR: Failed to flush output: %v\n", err)
		os.Exit(1)
	}

	// Calculate actual sample rate
	elapsed := time.Since(startTime)
	actualRate := float64(totalWritten) / elapsed.Seconds()

	fmt.Printf("\nCapture complete!\n")
	fmt.Printf("Total samples: %d (%.2f seconds @ %.1f kHz actual rate)\n",
		totalWritten, float64(totalWritten)/actualRate, actualRate/1000.0)
	fmt.Printf("Output file: %s (%.2f MB)\n", outputFile, float64(totalWritten*4)/(1024*1024))
	fmt.Printf("PRU Stats: Min=%d Max=%d McASP_Errors=%d\n",
		int32(ctrl.MinSample), int32(ctrl.MaxSample), ctrl.McASPErrors)
	fmt.Printf("\nTo analyze, copy this file and run:\n")
	fmt.Printf("  python3 analyze_i2s_raw.py %s\n", outputFile)
}
