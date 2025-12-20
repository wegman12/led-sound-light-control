package pru

import (
	"fmt"
	"syscall"
	"unsafe"

	"github.com/spf13/cobra"
)

type debugBitsData struct {
	Valid     uint32
	ErrorCode uint32
	Bits      [36]uint8  // 36 bytes (33 actual bits + 3 padding) for natural 4-byte alignment
	Durations [33]uint32 // LOW pulse durations in cycles
}

const (
	threshold1ms     = 200000 // 1ms at 200 MHz
	cyclesPerMicro   = 200
)

func MakeBitsReaderCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pru-bits",
		Short: "Read captured IR bits from PRU",
		Long:  "Reads and validates captured IR bits from the PRU, showing bit patterns and validation checks.",
		RunE:  runBitsReader,
	}
}

func runBitsReader(cmd *cobra.Command, args []string) error {
	fmt.Println("PRU Captured Bits Reader")
	fmt.Println("========================")
	fmt.Println("")

	mem, err := MapPRUMemory()
	if err != nil {
		return err
	}
	defer syscall.Munmap(mem)

	fmt.Printf("Successfully mapped PRU shared memory at 0x%X\n", PRUSharedMemAddr)
	fmt.Println("")

	debugData := (*debugBitsData)(unsafe.Pointer(uintptr(unsafe.Pointer(&mem[0])) + uintptr(debugBitsOffset)))

	if debugData.Valid == 0 {
		fmt.Println("No captured bits available yet. Press an IR button to capture data.")
		return nil
	}

	fmt.Printf("Valid: true\n")
	fmt.Printf("Error Code: 0x%04X (%d)\n", debugData.ErrorCode, debugData.ErrorCode)
	fmt.Println("")

	fmt.Println("Captured 33 bits:")
	fmt.Println("Index | Bit | Duration (cycles) | Duration (us) | Expected | vs Threshold")
	fmt.Println("------+-----+-------------------+---------------+----------+--------------")

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

		durationUs := float64(debugData.Durations[i]) / float64(cyclesPerMicro)

		thresholdInfo := ""
		if debugData.Durations[i] > 0 {
			if debugData.Durations[i] < threshold1ms {
				thresholdInfo = "< 1ms (SHORT)"
			} else {
				thresholdInfo = ">= 1ms (LONG)"
			}
		}

		fmt.Printf("%5d | %3d | %17d | %13.1f | %s | %s%s\n",
			i, debugData.Bits[i], debugData.Durations[i], durationUs, expected, thresholdInfo, marker)
	}

	fmt.Println("")

	fmt.Print("Binary: ")
	for i := 0; i < 33; i++ {
		fmt.Printf("%d", debugData.Bits[i])
	}
	fmt.Println("")

	fmt.Println("")
	fmt.Println("Validation checks:")

	if debugData.Bits[0] == 1 {
		fmt.Println("  [OK] START bit is 1")
	} else {
		fmt.Println("  [FAIL] START bit is NOT 1")
	}

	headerOk := true
	for i := 1; i <= 8; i++ {
		if debugData.Bits[i] != 0 {
			headerOk = false
			break
		}
	}
	if headerOk {
		fmt.Println("  [OK] HEADER bits (1-8) are all 0")
	} else {
		fmt.Println("  [FAIL] HEADER bits (1-8) are NOT all 0")
	}

	separatorOk := true
	for i := 9; i <= 16; i++ {
		if debugData.Bits[i] != 1 {
			separatorOk = false
			break
		}
	}
	if separatorOk {
		fmt.Println("  [OK] SEPARATOR bits (9-16) are all 1")
	} else {
		fmt.Println("  [FAIL] SEPARATOR bits (9-16) are NOT all 1")
	}

	if debugData.Bits[32] == 1 {
		fmt.Println("  [OK] STOP bit is 1")
	} else {
		fmt.Println("  [FAIL] STOP bit is NOT 1")
	}

	return nil
}
