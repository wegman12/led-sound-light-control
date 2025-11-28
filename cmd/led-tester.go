package cmd

import (
	"encoding/json"
	"os"
	"os/signal"

	"github.com/spf13/cobra"
	"github.com/wegman12/go-bbhw"
	"github.com/wegman12/led-sound-light-control/light/behavior"
	"github.com/wegman12/led-sound-light-control/light/led"
	"github.com/wegman12/led-sound-light-control/utilities"
)

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

		leds, err := createLeds()
		defer func() {
			for _, led := range leds {
				led.Close()
			}
		}()

		behaviors, err := createBehaviors(leds)
		defer func() {
			for _, behavior := range behaviors {
				behavior.Stop()
			}
		}()
		if err != nil {
			return err
		}

		utilities.ForEach(behaviors, func(b behavior.Behavior) { b.Start(ctx) })

		<-ctx.Done()
		return nil
	},
}

func createLeds() (map[led.Color]led.Led, error) {
	leds := make(map[led.Color]led.Led)
	for _, color := range []led.Color{led.RedLedColor, led.GreenLedColor, led.WhiteLedColor, led.BlueLedColor} {
		l, err := led.MakeLedColor(color)
		if err != nil {
			return leds, err
		}
		leds[color] = l
	}
	return leds, nil
}

func createBehaviors(leds map[led.Color]led.Led) ([]behavior.Behavior, error) {
	behaviors := make([]behavior.Behavior, 0)

	type behaviorPayload struct {
		color    led.Color
		behavior behavior.BehaviorType
		cfg      json.RawMessage
	}

	for _, payload := range []behaviorPayload{
		{color: led.RedLedColor, behavior: behavior.FlashingBehaviorType},
		{color: led.RedLedColor, behavior: behavior.BreathingBehaviorType},
	} {
		b, err := behavior.CreateBehavior(leds[payload.color], payload.behavior, payload.cfg)
		if err != nil {
			return behaviors, err
		}
		behaviors = append(behaviors, b)
	}
	return behaviors, nil
}
