package remote

import (
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"github.com/wegman12/go-bbhw"
)

type remoteTesterConfig struct {
	gpioPin    uint
	outputFile string
	usePulses  bool
	detector   DetectorConfig
}

var remoteCfg remoteTesterConfig

func MakeRemoteCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "test-remote",
		Short: "Test remote functionality from BBB board",
		Long:  `Uses GPIO port to read values from the IR port`,
		RunE: func(cmd *cobra.Command, args []string) error {

			ctx, _ := signal.NotifyContext(cmd.Context(), os.Interrupt)

			if remoteCfg.usePulses {
				return readPulsesAndExport(ctx)
			} else {

				return readRawDataAndExport(ctx)
			}
		},
	}

	cmd.Flags().UintVarP(&remoteCfg.gpioPin, "pin", "p", 20, "pin that the IR sensor is using")
	cmd.Flags().StringVarP(&remoteCfg.outputFile, "output", "o", "result.bin", "output file to write the result")
	cmd.Flags().BoolVarP(&remoteCfg.usePulses, "use-pulses", "u", false, "use pulses instead of raw data for the read")
	return cmd
}

func readPulsesAndExport(ctx context.Context) error {
	f, err := os.Create(remoteCfg.outputFile)

	if err != nil {
		return err
	}

	defer f.Close()

	pulses := make(chan []DetectionPulse, 100)
	d := newDetector(remoteCfg.gpioPin, DetectorConfig{
		MaximumDurationWait:  0,
		MaximumPulseDuration: 0,
		DelayTime:            0,
	})
	defer d.Close()

	fmt.Println("Reading pulse packets and writing to file")

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		totalPulsePackets := 0
		for pulse := range pulses {
			totalPulsePackets += 1
			pulseLength := int64(len(pulse))
			fmt.Println(pulseLength)
			binary.Write(f, binary.BigEndian, pulseLength)
			for _, p := range pulse {
				high := p.TimeHigh.Microseconds()
				low := p.TimeLow.Microseconds()
				binary.Write(f, binary.BigEndian, high)
				binary.Write(f, binary.BigEndian, low)
			}
		}
		fmt.Println("Pulse packets written:", totalPulsePackets)
		f.Sync()
		wg.Done()
	}()

	d.ReadPulsesUntilContextCancelled(pulses, ctx)

	wg.Wait()
	return nil
}

func readRawDataAndExport(ctx context.Context) error {
	f, err := os.Create(remoteCfg.outputFile)

	if err != nil {
		return err
	}

	defer f.Close()

	pin := bbhw.NewMMappedGPIO(remoteCfg.gpioPin, bbhw.IN)
	defer pin.Close()

	fmt.Println("Reading signal and writing to file")

	writeChannel := make(chan []byte, 1000)
	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		for b := range writeChannel {
			f.Write(b)
		}
		wg.Done()
		f.Sync()
	}()

	buffer := make([]byte, 10000)
	current := 0
	state := false
	stop := false
	start := time.Now()
	for !stop {
		select {
		case <-ctx.Done():
			stop = true
			if current > 0 {
				writeChannel <- buffer[:current]
			}
			close(writeChannel)
		default:
			state, _ = pin.GetState()
			if state {
				buffer[current] = 0
			} else {
				buffer[current] = 1
			}
			current++
			if current >= len(buffer) {
				// Create a new slice with the same length as the original
				copiedSlice := make([]byte, len(buffer))

				// Copy elements from the original slice to the new slice
				copy(copiedSlice, buffer)
				current = 0
				writeChannel <- copiedSlice
			}
			time.Sleep(800 * time.Nanosecond)
		}
	}
	duration := time.Since(start)
	fmt.Printf("Finished in %s\n", duration)
	wg.Wait()
	return nil
}
