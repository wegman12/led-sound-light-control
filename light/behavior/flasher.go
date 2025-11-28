package behavior

import (
	"context"
	"encoding/json"
	"time"

	"github.com/wegman12/led-sound-light-control/light/led"
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

	l      led.Led
	cancel context.CancelFunc
}

func (f *flasher) Start(ctx context.Context) {
	f.Stop()
	ctx, f.cancel = context.WithCancel(ctx)
	go func() {
		f.flashUntilContextCancelled(ctx)
	}()
}

func newFlasher(l led.Led, cfg json.RawMessage) (*flasher, error) {
	f := flasher{
		l: l,
	}
	if cfg != nil {
		err := json.Unmarshal(cfg, &f)
		if err != nil {
			return nil, err
		}
	}

	return &f, nil
}

func (f *flasher) ensureDefaults() {
	utilities.SetValueOrDefault(&f.HighDuration, utilities.Duration(flasherDefaultHighDuration))
	utilities.SetValueOrDefault(&f.LowDuration, utilities.Duration(flasherDefaultLowDuration))
	utilities.SetValueOrDefault(&f.Delay, utilities.Duration(flasherDefaultDelay))

}

func (f *flasher) flashUntilContextCancelled(ctx context.Context) {

	f.ensureDefaults()
	lastSwitch := time.Now()
	onLow := false
	for {
		select {
		case <-ctx.Done():
			f.l.SetPower(0.0)
			return
		default:
			if onLow && time.Since(lastSwitch) > time.Duration(f.LowDuration) {
				f.l.Enable()
				onLow = false
				lastSwitch = time.Now()
			} else if !onLow && time.Since(lastSwitch) > time.Duration(f.HighDuration) {
				f.l.SetPower(0.0)
				f.l.Disable()
				onLow = true
				lastSwitch = time.Now()
			}
			time.Sleep(time.Duration(f.LowDuration))
		}
	}
}

func (f *flasher) Stop() {
	if f.cancel != nil {
		f.cancel()
	}
}
