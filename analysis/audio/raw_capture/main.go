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

	// Raw Capture Memory Layout (must match pru1_raw_capture.c)
	rawControlOffset = 0x0000 // Start at beginning of shared memory
	rawBufferOffset  = 0x0100 // Start buffer after 256-byte control block
	rawBufferSize    = 6016   // Number of uint16 samples (~150ms @ 40kHz)
)

// RawControlBlock matches the C struct in PRU firmware
type RawControlBlock struct {
	EnableCapture    uint32
	Status           uint32
	TotalSamples     uint32
	BufferWriteIndex uint32
	BufferWrapCount  uint32
	ADCTimeouts      uint32
	LastSample       uint32
	MinSample        uint32
	MaxSample        uint32
	Reserved         [55]uint32
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: raw_capture <duration_seconds> <output_file>")
		fmt.Println("Example: raw_capture 10 baseline.bin")
		os.Exit(1)
	}

	var duration int
	fmt.Sscanf(os.Args[1], "%d", &duration)
	outputFile := os.Args[2]

	fmt.Printf("Raw ADC Sample Capture\n")
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
	ctrl := (*RawControlBlock)(unsafe.Pointer(&mem[rawControlOffset]))
	sampleBuffer := unsafe.Slice((*uint16)(unsafe.Pointer(&mem[rawBufferOffset])), rawBufferSize)

	// Verify PRU is running
	if ctrl.Status == 0 {
		fmt.Printf("ERROR: PRU not running (status=0)\n")
		fmt.Printf("Load firmware: cd ~/led-sound-light-control/pru-audio && sudo cp gen/pru1_raw_capture.out /lib/firmware/am335x-pru1-fw\n")
		os.Exit(1)
	}

	fmt.Printf("PRU Status: 0x%08X\n", ctrl.Status)
	fmt.Printf("PRU Total Samples: %d\n", ctrl.TotalSamples)
	fmt.Printf("PRU Min: %d, Max: %d, Last: %d\n", ctrl.MinSample, ctrl.MaxSample, ctrl.LastSample)
	fmt.Printf("Buffer Size: %d samples (~%.1f ms @ 40 kHz)\n\n", rawBufferSize, float64(rawBufferSize)/40.0)

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
	pollInterval := 10 * time.Millisecond // 10ms = 400 samples, well under 6016 buffer size

	fmt.Printf("Starting capture...\n\n")

	// Initial delay
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

				// Write as little-endian uint16 to buffered writer
				if err := binary.Write(writer, binary.LittleEndian, sample); err != nil {
					fmt.Printf("ERROR: Failed to write sample: %v\n", err)
					os.Exit(1)
				}
			}

			totalWritten += samplesToRead

			// Progress every second
			if totalWritten%(40000) < samplesToRead || totalWritten < samplesToRead*2 {
				rate := float64(totalWritten) / elapsed.Seconds()
				fmt.Printf("[%.1fs] %d samples written (%.1f kHz) | ADC timeouts: %d | Min: %d Max: %d\n",
					elapsed.Seconds(), totalWritten, rate/1000.0,
					ctrl.ADCTimeouts, ctrl.MinSample, ctrl.MaxSample)
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

	fmt.Printf("\nCapture complete!\n")
	fmt.Printf("Total samples: %d (%.2f seconds @ 40 kHz)\n", totalWritten, float64(totalWritten)/40000.0)
	fmt.Printf("Output file: %s (%.2f MB)\n", outputFile, float64(totalWritten*2)/(1024*1024))
	fmt.Printf("PRU Stats: Min=%d Max=%d Timeouts=%d\n", ctrl.MinSample, ctrl.MaxSample, ctrl.ADCTimeouts)
}
