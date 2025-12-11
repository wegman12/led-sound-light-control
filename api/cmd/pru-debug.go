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
	Status       uint32
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
	fmt.Println("Time       | write | read | events | errors | overrun | status  | status_hex")
	fmt.Println("-----------+-------+------+--------+--------+---------+---------+-----------")

	lastStatus := uint32(0)
	lastEvents := uint32(0)

	for {
		time.Sleep(500 * time.Millisecond)

		writeIdx := controlBlock.WriteIndex
		readIdx := controlBlock.ReadIndex
		events := controlBlock.EventCount
		errors := controlBlock.ErrorCount
		overrun := controlBlock.OverrunCount
		status := controlBlock.Status

		timestamp := time.Now().Format("15:04:05")

		// Highlight changes
		statusStr := fmt.Sprintf("%d", status)
		if status != lastStatus {
			statusStr = fmt.Sprintf("\033[32m%d\033[0m", status)  // Green
		}

		eventsStr := fmt.Sprintf("%d", events)
		if events != lastEvents {
			eventsStr = fmt.Sprintf("\033[32m%d\033[0m", events)  // Green
		}

		fmt.Printf("%s | %5d | %4d | %6s | %6d | %7d | %7s | 0x%08X\n",
			timestamp, writeIdx, readIdx, eventsStr, errors, overrun, statusStr, status)

		lastStatus = status
		lastEvents = events
	}
}
