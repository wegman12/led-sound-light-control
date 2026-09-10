package behavior

import (
	"encoding/json"
	"time"

	"github.com/wegman12/led-sound-light-control/utilities"
)

const (
	flasherDefaultHighDuration = time.Millisecond * 500
	flasherDefaultLowDuration  = time.Millisecond * 500
	flasherDefaultDelay        = time.Millisecond * 1
)

type flasher struct {
	HighDuration utilities.Duration `json:"high_duration"`
	LowDuration  utilities.Duration `json:"low_duration"`
	Delay        utilities.Duration `json:"delay"`
}

func newFlasher(cfg json.RawMessage) (*flasher, error) {
	f := flasher{}
	if cfg != nil {
		err := json.Unmarshal(cfg, &f)
		if err != nil {
			return nil, err
		}
	}

	f.ensureDefaults()

	return &f, nil
}

func (f *flasher) ensureDefaults() {
	utilities.SetValueOrDefault(&f.HighDuration, utilities.Duration(flasherDefaultHighDuration))
	utilities.SetValueOrDefault(&f.LowDuration, utilities.Duration(flasherDefaultLowDuration))
	utilities.SetValueOrDefault(&f.Delay, utilities.Duration(flasherDefaultDelay))

	utilities.PinValueToRange(&f.HighDuration, utilities.Duration(100*time.Millisecond), utilities.Duration(100*time.Minute))
	utilities.PinValueToRange(&f.LowDuration, utilities.Duration(100*time.Millisecond), utilities.Duration(100*time.Minute))
}

func (f *flasher) GetPower(t time.Duration) *float64 {
	tn := float64(t.Nanoseconds()%(int64(f.HighDuration)+int64(f.LowDuration))) / float64(f.HighDuration)
	if tn < 1.0 {
		return nil
	} else {
		v := 0.0
		return &v
	}
}

func (f *flasher) Weight() float64 {
	return 10000000
}
