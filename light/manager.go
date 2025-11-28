package light

import (
	"context"
	"encoding/json"

	"github.com/wegman12/led-sound-light-control/light/behavior"
	"github.com/wegman12/led-sound-light-control/light/led"
)

type Manager struct {
	leds      map[led.Color]led.Led
	behaviors []behavior.Behavior
}

type ManagerConfig struct {
	Behaviors []BehaviorConfig `json:"behaviors"`
}

type BehaviorConfig struct {
	Behavior string          `json:"behavior"`
	Color    string          `json:"color"`
	Config   json.RawMessage `json:"config"`
}

func NewManager(cfg ManagerConfig) (*Manager, error) {
	leds, err := createLeds()
	if err != nil {
		return nil, err
	}
	behaviors, err := createBehaviors(leds, cfg.Behaviors)
	if err != nil {
		return nil, err
	}
	return &Manager{
		leds:      leds,
		behaviors: behaviors,
	}, nil
}

func (m *Manager) Close() {
	m.Stop()
	for _, l := range m.leds {
		l.Close()
	}
}

func (m *Manager) Stop() {
	for _, b := range m.behaviors {
		b.Stop()
	}
}

func (m *Manager) Start(ctx context.Context) {
	for _, b := range m.behaviors {
		b.Start(ctx)
	}
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

	type behaviorPayload struct {
		color    led.Color
		behavior behavior.BehaviorType
		cfg      json.RawMessage
	}

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
