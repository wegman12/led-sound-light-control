package behavior

import (
	"encoding/json"
	"time"

	"github.com/wegman12/led-sound-light-control/utilities"
)

const (
	skipperDefaultMinPower = 0.0
	skipperDefaultMaxPower = 1.0
	skipperDefaultDuration = 1 * time.Second
)

type skipper struct {
	Duration      utilities.Duration `json:"duration"`
	MaxPowerValue float64            `json:"max_power_value"`
	MinPowerValue float64            `json:"min_power_value"`
}

func newSkipper(cfg json.RawMessage) (*skipper, error) {
	b := skipper{}
	if cfg != nil {
		err := json.Unmarshal(cfg, &b)
		if err != nil {
			return nil, err
		}
	}

	b.ensureDefaults()

	return &b, nil
}

func (b *skipper) ensureDefaults() {
	utilities.SetValueOrDefault(&b.Duration, utilities.Duration(skipperDefaultDuration))
	utilities.SetValueOrDefault(&b.MaxPowerValue, skipperDefaultMaxPower)
	utilities.SetValueOrDefault(&b.MinPowerValue, skipperDefaultMinPower)

	utilities.PinValueToRange(&b.Duration, utilities.Duration(100*time.Millisecond), utilities.Duration(100*time.Minute))
	utilities.PinValueToRange(&b.MaxPowerValue, 0.55, 1.0)
	utilities.PinValueToRange(&b.MinPowerValue, 0.0, 0.45)
}

func (b *skipper) GetPower(t time.Duration) *float64 {
	tn := float64(int64(t)%int64(b.Duration)) / float64(b.Duration)
	next := b.MinPowerValue + tn*(b.MaxPowerValue-b.MinPowerValue)
	return &next
}

func (b *skipper) Weight() float64 {
	return 1
}
