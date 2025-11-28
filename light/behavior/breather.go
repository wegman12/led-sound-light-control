package behavior

import (
	"context"
	"encoding/json"
	"time"

	"github.com/wegman12/led-sound-light-control/light/led"
	"github.com/wegman12/led-sound-light-control/utilities"
)

const (
	breatherDefaultDelay    = time.Millisecond * 50
	breatherDefaultStep     = 0.01
	breatherMinStep         = 0.001
	breatherMaxStep         = 0.25
	breatherDefaultMaxPower = 1.0
	breatherDefaultMinPower = 0.0
)

type breather struct {
	Delay         time.Duration
	Step          float64
	MaxPowerValue float64
	MinPowerValue float64

	l      led.Led
	cancel context.CancelFunc
}

func newBreather(l led.Led, cfg json.RawMessage) (*breather, error) {
	b := breather{
		l: l,
	}
	if cfg != nil {
		err := json.Unmarshal(cfg, &b)
		if err != nil {
			return nil, err
		}
	}

	return &b, nil
}

func (b *breather) ensureDefaults() {
	utilities.SetValueOrDefault(&b.Delay, breatherDefaultDelay)
	utilities.SetValueOrDefault(&b.Step, breatherDefaultStep)
	utilities.SetValueOrDefault(&b.MaxPowerValue, breatherDefaultMaxPower)
	utilities.SetValueOrDefault(&b.MinPowerValue, breatherDefaultMinPower)

	utilities.PinValueToRange(&b.Step, breatherMinStep, breatherMaxStep)
	utilities.PinValueToRange(&b.MaxPowerValue, 0.55, 1.0)
	utilities.PinValueToRange(&b.MinPowerValue, 0.0, 0.45)
}

func (b *breather) Start(ctx context.Context) {
	b.Stop()
	ctx, b.cancel = context.WithCancel(ctx)
	go func() {
		b.breathUntilContextCancelled(ctx)
	}()
}

func (b *breather) breathUntilContextCancelled(ctx context.Context) {
	b.ensureDefaults()
	power := b.MinPowerValue
	for {
		select {
		case <-ctx.Done():
			b.l.SetPower(0.0)
			return
		default:
			b.l.SetPower(power)
			power += b.Step
			if power > b.MaxPowerValue {
				power = b.MaxPowerValue
				b.Step = -b.Step
			} else if power < b.MinPowerValue {
				power = b.MinPowerValue
				b.Step = -b.Step
			}
			time.Sleep(b.Delay)
		}
	}
}

func (b *breather) Stop() {
	if b.cancel != nil {
		b.cancel()
	}
}
