package cmd

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"github.com/wegman12/go-bbhw"
)

const frequencyHz = 4000

var ledTester = &cobra.Command{
	Use:   "test-led",
	Short: "Test led functionality from BBB board",
	Long:  `Runs led colors in a test loop`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
	RunE: func(cmd *cobra.Command, args []string) error {

		ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt)
		defer stop() // Ensure the signal handling is stopped on exit

		err := bbhw.LoadOverlayForSysfsPWM()
		if err != nil {
			return err
		}

		redLed, err := bbhw.NewPWMChipPWM(7, 1)
		if err != nil {
			return err
		}
		greenLed, err := bbhw.NewPWMChipPWM(7, 0)
		if err != nil {
			return err
		}

		whiteLed, err := bbhw.NewPWMChipPWM(5, 1)
		if err != nil {
			return err
		}

		blueLed, err := bbhw.NewPWMChipPWM(5, 0)
		if err != nil {
			return err
		}
		defer redLed.Close()
		defer greenLed.Close()
		defer whiteLed.Close()
		defer blueLed.Close()

		wg := &sync.WaitGroup{}
		bbhw.SetPWMFreq(redLed, frequencyHz)
		bbhw.SetPWMFreq(greenLed, frequencyHz)
		bbhw.SetPWMFreq(whiteLed, frequencyHz)
		bbhw.SetPWMFreq(blueLed, frequencyHz)
		startBreathing(redLed, wg, ctx, breathingConfig{delay: 10 * time.Millisecond})
		startBreathing(greenLed, wg, ctx, breathingConfig{delay: 5 * time.Millisecond})
		startBreathing(whiteLed, wg, ctx, breathingConfig{delay: 15 * time.Millisecond})
		startBreathing(blueLed, wg, ctx, breathingConfig{delay: 20 * time.Millisecond})
		wg.Wait()
		return nil
	},
}

type breathingConfig struct {
	delay time.Duration
	step  float64
}

func startBreathing(led *bbhw.BBPWMPin, wg *sync.WaitGroup, ctx context.Context, cfg breathingConfig) {
	wg.Add(1)
	go func() {
		if cfg.delay <= 0 {
			cfg.delay = 50 * time.Millisecond
		}
		if cfg.step == 0 {
			cfg.step = 0.01
		}
		power := 0.0
		for {
			select {
			case <-ctx.Done():
				wg.Done()
				bbhw.SetDuty(led, 0)
				return
			default:
				bbhw.SetDuty(led, power)
				power += cfg.step
				if power > 1.0 {
					power = 1.0
					cfg.step = -cfg.step
				} else if power < 0.0 {
					power = 0.0
					cfg.step = -cfg.step
				}
				time.Sleep(cfg.delay) // Simulate work
			}
		}
	}()
}
