package main

import (
	"fmt"
	"os"
	"syscall"
	"time"
	"unsafe"
)

const (
	pruSharedMemAddr  = 0x4A310000
	pruSharedMemSize  = 0x3000
	controlBlockOffset = 0x1000
)

type pruControlBlock struct {
	WriteIndex   uint32
	ReadIndex    uint32
	EventCount   uint32
	ErrorCount   uint32
	OverrunCount uint32
	ErrorCode    uint32
}

// Error code constants (matching pru0_ir_detector.c)
const (
	ERROR_NONE           = 0x0000
	ERROR_LEADER_LOW     = 0x0001
	ERROR_LEADER_HIGH    = 0x0002
	ERROR_FIRST_LOW      = 0x0003
	ERROR_DATA_HIGH      = 0x0004
	ERROR_DATA_LOW       = 0x0005
	ERROR_START_BIT      = 0x0006
	ERROR_HEADER_BITS    = 0x0007
	ERROR_SEPARATOR_BITS = 0x0008
	ERROR_NO_MATCH       = 0x0009
)

func errorCodeString(code uint32) string {
	switch code {
	case ERROR_NONE:
		return "NONE"
	case ERROR_LEADER_LOW:
		return "LEADER_LOW"
	case ERROR_LEADER_HIGH:
		return "LEADER_HIGH"
	case ERROR_FIRST_LOW:
		return "FIRST_LOW"
	case ERROR_DATA_HIGH:
		return "DATA_HIGH"
	case ERROR_DATA_LOW:
		return "DATA_LOW"
	case ERROR_START_BIT:
		return "START_BIT"
	case ERROR_HEADER_BITS:
		return "HEADER_BITS"
	case ERROR_SEPARATOR_BITS:
		return "SEPARATOR_BITS"
	case ERROR_NO_MATCH:
		return "NO_MATCH"
	default:
		return fmt.Sprintf("UNKNOWN(0x%04X)", code)
	}
}

func main() {
	fmt.Println("PRU Shared Memory Debug Tool")
	fmt.Println("============================")
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

	// Get pointer to control block - use proper pointer arithmetic for alignment
	controlBlock := (*pruControlBlock)(unsafe.Pointer(uintptr(unsafe.Pointer(&mem[0])) + uintptr(controlBlockOffset)))

	fmt.Println("Monitoring PRU Control Block:")
	fmt.Println("Time       | write | read | events | errors | overrun | error_code")
	fmt.Println("-----------+-------+------+--------+--------+---------+-------------------")

	lastErrorCode := uint32(0)
	lastEvents := uint32(0)
	lastErrors := uint32(0)

	for {
		time.Sleep(500 * time.Millisecond)

		writeIdx := controlBlock.WriteIndex
		readIdx := controlBlock.ReadIndex
		events := controlBlock.EventCount
		errors := controlBlock.ErrorCount
		overrun := controlBlock.OverrunCount
		errorCode := controlBlock.ErrorCode

		timestamp := time.Now().Format("15:04:05")

		// Highlight changes
		errorCodeStr := errorCodeString(errorCode)
		if errorCode != lastErrorCode {
			errorCodeStr = fmt.Sprintf("\033[31m%s\033[0m", errorCodeStr)  // Red for error changes
		}

		eventsStr := fmt.Sprintf("%d", events)
		if events != lastEvents {
			eventsStr = fmt.Sprintf("\033[32m%d\033[0m", events)  // Green for new events
		}

		errorsStr := fmt.Sprintf("%d", errors)
		if errors != lastErrors {
			errorsStr = fmt.Sprintf("\033[33m%d\033[0m", errors)  // Yellow for new errors
		}

		fmt.Printf("%s | %5d | %4d | %6s | %6s | %7d | %s\n",
			timestamp, writeIdx, readIdx, eventsStr, errorsStr, overrun, errorCodeStr)

		lastErrorCode = errorCode
		lastEvents = events
		lastErrors = errors
	}
}
