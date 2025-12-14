package cmd

import (
	"fmt"
	"os"
	"syscall"
	"time"
	"unsafe"

	"github.com/spf13/cobra"
)

const (
	// PRU Shared Memory Configuration
	pruSharedMemAddr = 0x4A310000
	devMem           = "/dev/mem"
)

// audioControlBlock matches the C struct in PRU1 firmware
type audioControlBlock struct {
	Status    uint32
	Counter   uint32
	ToggleBit uint32
	Reserved  [13]uint32 // 64 bytes total
}

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Test PRU1 basic operation",
	Long: `Monitor PRU1 status by reading the toggle bit that flips every second.

This command verifies that:
  - PRU1 firmware is loaded and running
  - Shared memory is accessible
  - PRU1 can write to shared memory
  - Go application can read from shared memory

The command will display the toggle bit value and counter every second.
Press Ctrl+C to exit.`,
	Run: runTest,
}

func init() {
	rootCmd.AddCommand(testCmd)
}

func runTest(cmd *cobra.Command, args []string) {
	fmt.Println("PRU1 Audio Test")
	fmt.Println("===============")
	fmt.Println()

	// Open /dev/mem (requires root privileges)
	memFile, err := os.OpenFile(devMem, os.O_RDWR|os.O_SYNC, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to open %s (need root): %v\n", devMem, err)
		fmt.Fprintln(os.Stderr, "Try running with: sudo ./audio-util test")
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
		fmt.Fprintf(os.Stderr, "Error: Failed to mmap PRU memory at 0x%X: %v\n", pruSharedMemAddr, err)
		os.Exit(1)
	}
	defer syscall.Munmap(mem)

	// Get pointer to audio control block
	controlBlock := (*audioControlBlock)(unsafe.Pointer(&mem[audioControlBlockOffset]))

	// Check if PRU1 is running
	if controlBlock.Status == 0 {
		fmt.Fprintln(os.Stderr, "Error: PRU1 firmware not running (status=0)")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Load firmware with:")
		fmt.Fprintln(os.Stderr, "  sudo cp pru/audio/gen/pru1_audio.out /lib/firmware/am335x-pru1-fw")
		fmt.Fprintln(os.Stderr, "  echo 'start' | sudo tee /sys/class/remoteproc/remoteproc2/state")
		os.Exit(1)
	}

	fmt.Printf("PRU1 Status: 0x%08X (", controlBlock.Status)
	// Print status as ASCII if printable
	for i := 0; i < 4; i++ {
		b := byte(controlBlock.Status >> (i * 8))
		if b >= 32 && b <= 126 {
			fmt.Printf("%c", b)
		} else {
			fmt.Printf(".")
		}
	}
	fmt.Println(")")
	fmt.Println()
	fmt.Println("Monitoring PRU1 toggle bit (Ctrl+C to exit)...")
	fmt.Println()

	lastToggle := controlBlock.ToggleBit
	lastCounter := controlBlock.Counter

	for {
		currentToggle := controlBlock.ToggleBit
		currentCounter := controlBlock.Counter

		// Display current state
		timestamp := time.Now().Format("15:04:05")
		fmt.Printf("[%s] Counter: %5d | Toggle: %d", timestamp, currentCounter, currentToggle)

		// Check if toggle changed
		if currentToggle != lastToggle {
			fmt.Print(" ✓ TOGGLED")
		}

		// Check if counter incremented
		if currentCounter != lastCounter {
			delta := currentCounter - lastCounter
			fmt.Printf(" (+%d)", delta)
		}

		fmt.Println()

		lastToggle = currentToggle
		lastCounter = currentCounter

		// Wait 1 second
		time.Sleep(1 * time.Second)
	}
}
