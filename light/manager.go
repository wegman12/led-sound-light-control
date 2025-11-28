package light

import (
	"context"
	"encoding/json"

	"github.com/wegman12/go-bbhw"
	"github.com/wegman12/led-sound-light-control/light/behavior"
	"github.com/wegman12/led-sound-light-control/light/led"
	"github.com/wegman12/led-sound-light-control/utilities"
)

type Manager struct {
	leds      map[led.Color]led.Led
	behaviors []behavior.Behavior
}

func NewManager(cfg ManagerConfig) (*Manager, error) {

	err := bbhw.LoadOverlayForSysfsPWM()
	if err != nil {
		return nil, err
	}

	leds, err := createLeds()
	if err != nil {
		return &Manager{
			leds: leds,
		}, err
	}
	behaviors, err := createBehaviors(leds, cfg.Behaviors)
	return &Manager{
		leds:      leds,
		behaviors: behaviors,
	}, err
}

func (m *Manager) Close() {
	m.Stop()
	utilities.ForEachValue(m.leds, func(l led.Led) { l.Close() })
}

func (m *Manager) Stop() {
	utilities.ForEach(m.behaviors, func(b behavior.Behavior) { b.Stop() })
}

func (m *Manager) Start(ctx context.Context) {
	utilities.ForEach(m.behaviors, func(b behavior.Behavior) { b.Start(ctx) })
}

func (m *Manager) AddBehavior(ledColor led.Color, behaviorType behavior.BehaviorType, cfg json.RawMessage) error {
	b, err := behavior.CreateBehavior(m.leds[ledColor], behaviorType, cfg)
	if err != nil {
		return err
	}

	m.behaviors = append(m.behaviors, b)
	return nil
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

func createBehaviors(leds map[led.Color]led.Led, cfgs []BehaviorConfig) ([]behavior.Behavior, error) {
	behaviors := make([]behavior.Behavior, 0)

	for _, cfg := range cfgs {
		bt := behavior.LookupBehavior(cfg.Behavior)
		color := led.LookupColor(cfg.Color)

		b, err := behavior.CreateBehavior(leds[color], bt, cfg.Config)
		if err != nil {
			return behaviors, err
		}
		behaviors = append(behaviors, b)
	}

	return behaviors, nil
}
