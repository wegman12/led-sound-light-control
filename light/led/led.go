package led

import (
	"fmt"

	"github.com/wegman12/go-bbhw"
)

type Led interface {
	SetPower(power float64)
	Disable()
	Enable()
	Close()
}

type ledImpl struct {
	pin *bbhw.BBPWMPin

	disabled bool
}

func (l *ledImpl) SetPower(power float64) {
	if l.pin == nil || l.disabled {
		return
	}
	if power > 1.0 {
		power = 1.0
	}
	if power < 0.0 {
		power = 0.0
	}
	bbhw.SetDuty(l.pin, power)
}

func (l *ledImpl) Disable() {
	l.disabled = true
}

func (l *ledImpl) Enable() {
	l.disabled = false
}

func (l *ledImpl) Close() {
	l.SetPower(0.0)
	l.pin.Close()
}

type ledConfig struct {
	chipId int
	pwmId  int
}

func MakeLedColor(t Color) (Led, error) {
	switch t {
	case RedLedColor:
		return makeLed(RedConfig, 0)
	case GreenLedColor:
		return makeLed(GreenConfig, 0)
	case BlueLedColor:
		return makeLed(BlueConfig, 0)
	case WhiteLedColor:
		return makeLed(WhiteConfig, 0)
	default:
		return nil, fmt.Errorf("unknown color type: %v", t)
	}
}

func MakeLedColorWithFrequency(t Color, frequency float64) (Led, error) {
	switch t {
	case RedLedColor:
		return makeLed(RedConfig, frequency)
	case GreenLedColor:
		return makeLed(GreenConfig, frequency)
	case BlueLedColor:
		return makeLed(BlueConfig, frequency)
	case WhiteLedColor:
		return makeLed(WhiteConfig, frequency)
	default:
		return nil, fmt.Errorf("unknown color type: %v", t)
	}
}

func makeLed(cfg ledConfig, frequencyHz float64) (*ledImpl, error) {
	l, err := bbhw.NewPWMChipPWM(cfg.chipId, cfg.pwmId)
	if err != nil {
		return nil, err
	}

	if frequencyHz <= 0 {
		frequencyHz = 4000
	}

	bbhw.SetPWMFreq(l, frequencyHz)
	impl := &ledImpl{
		pin: l,
	}

	impl.SetPower(0.0)
	return impl, nil
}
