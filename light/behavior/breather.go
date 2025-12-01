package behavior

import (
	"encoding/json"
	"time"

	"github.com/wegman12/led-sound-light-control/utilities"
)

const (
	breatherDefaultMinPower = 0.0
	breatherDefaultMaxPower = 1.0
	breatherDefaultDuration = 1 * time.Second
)

type breather struct {
	Duration      utilities.Duration `json:"duration"`
	MaxPowerValue float64            `json:"max_power_value"`
	MinPowerValue float64            `json:"min_power_value"`
}

func newBreather(cfg json.RawMessage) (*breather, error) {
	b := breather{}
	if cfg != nil {
		err := json.Unmarshal(cfg, &b)
		if err != nil {
			return nil, err
		}
	}

	b.ensureDefaults()

	return &b, nil
}

func (b *breather) ensureDefaults() {
	utilities.SetValueOrDefault(&b.Duration, utilities.Duration(breatherDefaultDuration))
	utilities.SetValueOrDefault(&b.MaxPowerValue, breatherDefaultMaxPower)
	utilities.SetValueOrDefault(&b.MinPowerValue, breatherDefaultMinPower)

	utilities.PinValueToRange(&b.Duration, utilities.Duration(100*time.Millisecond), utilities.Duration(100*time.Minute))
	utilities.PinValueToRange(&b.MaxPowerValue, 0.55, 1.0)
	utilities.PinValueToRange(&b.MinPowerValue, 0.0, 0.45)
}

func (b *breather) GetPower(t time.Duration) *float64 {
	tn := float64(int64(t)%int64(2*b.Duration)) / float64(b.Duration)
	next := 0.0
	if tn < 1 {
		next = b.MinPowerValue + tn*(b.MaxPowerValue-b.MinPowerValue)
	} else {
		next = b.MaxPowerValue - (tn-1)*(b.MaxPowerValue-b.MinPowerValue)
	}
	return &next
}

func (b *breather) Weight() float64 {
	return 1
}
