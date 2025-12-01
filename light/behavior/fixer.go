package behavior

import (
	"encoding/json"
	"time"

	"github.com/wegman12/led-sound-light-control/utilities"
)

type fixer struct {
	PowerValue float64 `json:"power_value"`
}

func newFixer(cfg json.RawMessage) (*fixer, error) {
	f := fixer{}
	if cfg != nil {
		err := json.Unmarshal(cfg, &f)
		if err != nil {
			return nil, err
		}
	}

	f.ensureDefaults()

	return &f, nil
}

func (f *fixer) ensureDefaults() {
	utilities.PinValueToRange(&f.PowerValue, 0.0, 1.0)
}

func (f *fixer) GetPower(_ time.Duration) *float64 {
	cpy := f.PowerValue
	return &cpy
}

func (f *fixer) Weight() float64 {
	return 1
}
