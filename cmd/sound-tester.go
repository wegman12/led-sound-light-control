package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"github.com/wegman12/led-sound-light-control/sound"
	"github.com/wegman12/led-sound-light-control/utilities"
)

var soundTester = &cobra.Command{
	Use:   "test-sound",
	Short: "Test sound functionality from BBB board",
	Long:  `Runs sound sampling in a loop and displays the fft results`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
	RunE: doSoundTester,
}

func doSoundTester(cmd *cobra.Command, args []string) error {
	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt)
	defer stop()

	m := sound.Manager{
		ResultsChannel: make(chan sound.FrequencyResult, sound.BufferSize*10),
	}

	wg := &sync.WaitGroup{}

	wg.Add(2)
	go func() {
		defer wg.Done()
		m.Start(ctx)
	}()

	go func() {
		defer wg.Done()
		exportResults(ctx, m.ResultsChannel)
	}()

	wg.Wait()
	return nil
}

func exportResults(ctx context.Context, resultChannel chan sound.FrequencyResult) {
	f, err := os.Create("./sound-test-result.csv")
	if err != nil {
		fmt.Println(err)
		return
	}
	writeHeader(f)
	defer f.Close()
	for {
		select {
		case <-ctx.Done():
			fmt.Println("stopping export")
			return
		case result := <-resultChannel:
			stringResults := []string{fmt.Sprintf("%v", result.SamplingDuration)}
			magnitudes := utilities.Apply(result.Magnitudes[1:], func(v float64) string { return fmt.Sprint(v) })
			stringResults = append(stringResults, magnitudes...)
			_, err = f.WriteString(strings.Join(stringResults, ",") + "\n")
			if err != nil {
				fmt.Println("Failed to print results: " + err.Error())
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
}
func writeHeader(f *os.File) {
	binSize := float64(sound.SamplingRate) / sound.BufferSize
	binDescriptions := make([]string, 0, sound.BufferSize)
	binLower := binSize
	for i := 1; i < sound.BufferSize; i++ {
		binDescriptions = append(binDescriptions, fmt.Sprintf("%0.2f-%0.2f KHZ", binLower/1000, (binLower+binSize)/1000))
		binLower += binSize
	}

	_, err := f.WriteString("Duration (ms)," + strings.Join(binDescriptions, ",") + "\n")
	if err != nil {
		fmt.Println(err)
	}
}
