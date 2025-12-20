package pru

import (
	"encoding/binary"
	"fmt"
	"syscall"

	"github.com/spf13/cobra"
)

const (
	debugBitsOffset = 0x1100
)

func MakeMemoryDumpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pru-memory",
		Short: "Dump PRU debug bits memory area",
		Long:  "Dumps raw memory from the PRU debug bits area for structure alignment debugging.",
		RunE:  runMemoryDump,
	}
}

func runMemoryDump(cmd *cobra.Command, args []string) error {
	fmt.Println("PRU Memory Dump Tool - Debug Bits Area")
	fmt.Println("=======================================")
	fmt.Println("")

	mem, err := MapPRUMemory()
	if err != nil {
		return err
	}
	defer syscall.Munmap(mem)

	fmt.Printf("Successfully mapped PRU shared memory at 0x%X\n", PRUSharedMemAddr)
	fmt.Println("")

	fmt.Println("Reading raw memory from debug_bits_data structure:")
	fmt.Println("")

	valid := binary.LittleEndian.Uint32(mem[debugBitsOffset : debugBitsOffset+4])
	fmt.Printf("Valid (offset 0x%04X): 0x%08X (%d)\n", 0, valid, valid)

	errorCode := binary.LittleEndian.Uint32(mem[debugBitsOffset+4 : debugBitsOffset+8])
	fmt.Printf("ErrorCode (offset 0x%04X): 0x%08X (%d)\n", 4, errorCode, errorCode)

	fmt.Println("")
	fmt.Println("Bits[33] starting at offset 8:")
	for i := 0; i < 33; i++ {
		bit := mem[debugBitsOffset+8+i]
		fmt.Printf("  bits[%2d] (offset 0x%04X): %d\n", i, 8+i, bit)
	}

	fmt.Println("")
	fmt.Println("Memory dump of padding area (offsets 41-43):")
	for i := 41; i < 44; i++ {
		val := mem[debugBitsOffset+i]
		fmt.Printf("  offset 0x%04X: 0x%02X (%d)\n", i, val, val)
	}

	fmt.Println("")
	fmt.Println("Durations[33] starting at offset 44 (correct alignment):")
	for i := 0; i < 10; i++ {
		offset := 44 + (i * 4)
		duration := binary.LittleEndian.Uint32(mem[debugBitsOffset+offset : debugBitsOffset+offset+4])
		fmt.Printf("  durations[%2d] (offset 0x%04X): 0x%08X (%d)\n", i, offset, duration, duration)
	}

	fmt.Println("")
	fmt.Println("Durations[33] starting at offset 45 (WRONG - 4 byte padding):")
	for i := 0; i < 10; i++ {
		offset := 45 + (i * 4)
		duration := binary.LittleEndian.Uint32(mem[debugBitsOffset+offset : debugBitsOffset+offset+4])
		fmt.Printf("  durations[%2d] (offset 0x%04X): 0x%08X (%d)\n", i, offset, duration, duration)
	}

	fmt.Println("")
	fmt.Println("Expected test pattern values (written by PRU):")
	fmt.Println("  durations[0] should be 0x11111111 (286331153)")
	fmt.Println("  durations[1] should be 0x22222222 (572662306)")
	fmt.Println("  durations[2] should be 0x33333333 (858993459)")
	fmt.Println("  durations[3] should be 0x44444444 (1145324612)")

	fmt.Println("")
	fmt.Printf("Structure sizes:\n")
	fmt.Printf("  C struct with 3-byte padding: 4 + 4 + 33 + 3 + (33*4) = 176 bytes\n")
	fmt.Printf("  Go struct with 4-byte padding: 4 + 4 + 33 + 4 + (33*4) = 177 bytes (WRONG!)\n")

	return nil
}
