package behavior

import (
	"time"

	"github.com/wegman12/led-sound-light-control/light/led"
	"github.com/wegman12/led-sound-light-control/utilities"
)

type Manager struct {
	behaviors map[led.Color][]Behavior
}

func CreateManager(behaviors map[led.Color][]Behavior) *Manager {
	return &Manager{
		behaviors: behaviors,
	}
}

func (m *Manager) GetPower(t time.Duration) map[led.Color]*float64 {
	back := make(map[led.Color]*float64)
	if m.behaviors == nil {
		return back
	}

	return utilities.ApplyMap(
		m.behaviors,
		func(_ led.Color, behaviors []Behavior) *float64 { return getPowerForColor(t, behaviors) },
	)
}

func getPowerForColor(t time.Duration, behaviors []Behavior) *float64 {
	values := make([]float64, 0, len(behaviors))
	weights := make([]float64, 0, len(behaviors))
	for _, b := range behaviors {
		val := b.GetPower(t)
		if val == nil {
			continue
		}
		weights = append(weights, b.Weight())
		values = append(values, *val*b.Weight())
	}
	if len(values) == 0 {
		return nil
	}
	weightTotal := utilities.Sum(weights)
	if weightTotal <= 0 {
		weightTotal = float64(len(values))
	}
	avg := utilities.Sum(values) / weightTotal
	return &avg
}
