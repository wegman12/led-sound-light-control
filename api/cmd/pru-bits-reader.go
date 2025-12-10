package main

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

const (
	pruSharedMemAddr  = 0x4A310000
	pruSharedMemSize  = 0x4000 // 16KB to include debug bits at 0x3000
	debugBitsOffset   = 0x3000
)

type debugBitsData struct {
	Valid     uint32
	ErrorCode uint32
	Bits      [33]uint8
}

func main() {
	fmt.Println("PRU Captured Bits Reader")
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

	// Get pointer to debug bits data
	debugData := (*debugBitsData)(unsafe.Pointer(&mem[debugBitsOffset]))

	if debugData.Valid == 0 {
		fmt.Println("No captured bits available yet. Press an IR button to capture data.")
		os.Exit(0)
	}

	fmt.Printf("Valid: true\n")
	fmt.Printf("Error Code: 0x%04X (%d)\n", debugData.ErrorCode, debugData.ErrorCode)
	fmt.Println("")

	// Display all 33 bits
	fmt.Println("Captured 33 bits:")
	fmt.Println("Index | Bit | Expected")
	fmt.Println("------+-----+---------")
	for i := 0; i < 33; i++ {
		expected := "?"
		if i == 0 {
			expected = "1 (START)"
		} else if i >= 1 && i <= 8 {
			expected = "0 (HEADER)"
		} else if i >= 9 && i <= 16 {
			expected = "1 (SEPARATOR)"
		} else if i == 32 {
			expected = "1 (STOP)"
		} else {
			expected = "- (data)"
		}

		marker := ""
		if debugData.Bits[i] != 0 && debugData.Bits[i] != 1 {
			marker = " <- INVALID"
		}

		fmt.Printf("%5d | %3d | %s%s\n", i, debugData.Bits[i], expected, marker)
	}

	fmt.Println("")

	// Display as binary string
	fmt.Print("Binary: ")
	for i := 0; i < 33; i++ {
		fmt.Printf("%d", debugData.Bits[i])
	}
	fmt.Println("")

	// Check which validation would fail
	fmt.Println("")
	fmt.Println("Validation checks:")

	// START bit
	if debugData.Bits[0] == 1 {
		fmt.Println("  ✓ START bit is 1")
	} else {
		fmt.Println("  ✗ START bit is NOT 1")
	}

	// HEADER bits (1-8 should be 0)
	headerOk := true
	for i := 1; i <= 8; i++ {
		if debugData.Bits[i] != 0 {
			headerOk = false
			break
		}
	}
	if headerOk {
		fmt.Println("  ✓ HEADER bits (1-8) are all 0")
	} else {
		fmt.Println("  ✗ HEADER bits (1-8) are NOT all 0")
	}

	// SEPARATOR bits (9-16 should be 1)
	separatorOk := true
	for i := 9; i <= 16; i++ {
		if debugData.Bits[i] != 1 {
			separatorOk = false
			break
		}
	}
	if separatorOk {
		fmt.Println("  ✓ SEPARATOR bits (9-16) are all 1")
	} else {
		fmt.Println("  ✗ SEPARATOR bits (9-16) are NOT all 1")
	}

	// STOP bit
	if debugData.Bits[32] == 1 {
		fmt.Println("  ✓ STOP bit is 1")
	} else {
		fmt.Println("  ✗ STOP bit is NOT 1")
	}
}
