package pru

import (
	"fmt"
	"syscall"
	"unsafe"

	"github.com/spf13/cobra"
)

const (
	timingDataOffset = 0x2000
	maxTimingSamples = 512
	pruClockHz       = 200000000 // 200 MHz
	cyclesPerUs      = 200       // 200 cycles per microsecond
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

func MakeTimingReaderCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pru-timing",
		Short: "Read timing data from PRU",
		Long:  "Reads and analyzes timing data (state transitions and durations) captured by the PRU.",
		RunE:  runTimingReader,
	}
}

func runTimingReader(cmd *cobra.Command, args []string) error {
	fmt.Println("PRU Timing Data Reader")
	fmt.Println("======================")
	fmt.Println("")

	mem, err := MapPRUMemory()
	if err != nil {
		return err
	}
	defer syscall.Munmap(mem)

	fmt.Printf("Successfully mapped PRU shared memory at 0x%X\n", PRUSharedMemAddr)
	fmt.Println("")

	timing := (*timingData)(unsafe.Pointer(uintptr(unsafe.Pointer(&mem[0])) + uintptr(timingDataOffset)))

	fmt.Printf("Timing capture complete: %v\n", timing.Complete != 0)
	fmt.Printf("Sample count: %d\n", timing.SampleCount)
	fmt.Println("")

	if timing.SampleCount == 0 {
		fmt.Println("No timing samples captured yet. Make sure the PRU code is running and press a button.")
		return nil
	}

	fmt.Println("Timing Samples:")
	fmt.Println("Index | State | Duration (cycles) | Duration (us) | Duration (ms)")
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
	fmt.Printf("Total duration: %d cycles (%.2f us / %.3f ms)\n", totalCycles, totalUs, totalMs)
	fmt.Println("")

	fmt.Println("Bit pattern interpretation (based on HIGH durations):")
	fmt.Println("(Using 1ms threshold: <1ms = 0, >=1ms = 1)")
	fmt.Println("")

	bitPattern := ""
	for i := uint32(0); i < timing.SampleCount && i < maxTimingSamples; i++ {
		sample := timing.Samples[i]
		if sample.State == 1 {
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

	return nil
}
