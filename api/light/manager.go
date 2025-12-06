package light

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"time"

	"github.com/wegman12/go-bbhw"
	"github.com/wegman12/led-sound-light-control/light/behavior"
	"github.com/wegman12/led-sound-light-control/light/led"
	"github.com/wegman12/led-sound-light-control/utilities"
)

type Manager struct {
	leds            map[led.Color]led.Led
	behaviorManager *behavior.Manager

	startTime time.Time
	cancel    context.CancelFunc
	paused    atomic.Bool
}

const (
	defaultNextCycleDelay = 10 * time.Millisecond
)

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
	behaviors, err := createBehaviors(cfg.Behaviors)
	return &Manager{
		leds:            leds,
		behaviorManager: behavior.CreateManager(behaviors),
	}, err
}

func (m *Manager) UpdateBehaviors(cfg ManagerConfig) error {
	behaviors, err := createBehaviors(cfg.Behaviors)
	if err != nil {
		return err
	}
	m.paused.Store(true)
	m.behaviorManager = behavior.CreateManager(behaviors)
	m.paused.Store(false)
	return nil
}

func (m *Manager) Close() {
	m.Stop()
	utilities.ForEachValue(m.leds, func(l led.Led) { l.Close() })
}

func (m *Manager) Stop() {
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
}

func (m *Manager) Start(ctx context.Context) {
	m.Stop()
	ctx, m.cancel = context.WithCancel(ctx)
	m.startTime = time.Now()
	go func() {
		defer func() {
			if r := recover(); r != nil {
				// Masking panics from go-bbhw library
			}
		}()
		m.SetLedPowerUntilContextCancelled(ctx)
	}()
}

func (m *Manager) SetLedPowerUntilContextCancelled(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			for _, l := range m.leds {
				l.SetPower(0.0)
			}
			return
		default:
			isPaused := m.paused.Load()
			if isPaused {
				break
			}
			powers := m.behaviorManager.GetPower(time.Since(m.startTime))
			for c, p := range powers {
				if p == nil {
					continue
				}
				m.leds[c].SetPower(*p)
			}
		}
		time.Sleep(defaultNextCycleDelay)
	}

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

func createBehaviors(cfgs []BehaviorConfig) (map[led.Color][]behavior.Behavior, error) {
	// Normalize config to ensure all LED colors are represented
	normalizedCfgs := normalizeConfig(cfgs)

	behaviors := make(map[led.Color][]behavior.Behavior)

	for _, cfg := range normalizedCfgs {
		behaviors[cfg.Color] = append(behaviors[cfg.Color], cfg.Behavior)
	}

	return behaviors, nil
}

// normalizeConfig ensures all LED colors are represented in the configuration
// For any color not specified, adds a fixed behavior with 0 power
func normalizeConfig(cfgs []BehaviorConfig) []BehaviorConfig {
	allColors := []led.Color{led.RedLedColor, led.GreenLedColor, led.BlueLedColor, led.WhiteLedColor}

	// Track which colors are already in the config
	existingColors := make(map[led.Color]bool)
	for _, cfg := range cfgs {
		existingColors[cfg.Color] = true
	}

	// Start with the provided configs
	normalized := make([]BehaviorConfig, len(cfgs))
	copy(normalized, cfgs)

	// Add zero-power behaviors for missing colors
	for _, color := range allColors {
		if !existingColors[color] {
			// Create a fixed behavior with 0 power using the factory
			configJSON := json.RawMessage(`{"power_value": 0.0}`)
			zeroBehavior, err := behavior.CreateBehavior(behavior.FixedBehaviorType, configJSON)
			if err != nil {
				// If we can't create the zero behavior, skip this color
				// This shouldn't happen but we handle it gracefully
				continue
			}
			normalized = append(normalized, BehaviorConfig{
				Color:    color,
				Behavior: zeroBehavior,
			})
		}
	}

	return normalized
}
