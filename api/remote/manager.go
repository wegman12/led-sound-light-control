package remote

import (
	"context"
	"time"

	"github.com/wegman12/led-sound-light-control/utilities"
)

type Manager struct {
	sensorPort int
}

func (m *Manager) ReportButtonPressesUntilContextCancelled(buttonPresses chan ButtonType, ctx context.Context) {
	pulses := make(chan []DetectionPulse, 100)
	d := newDetector(remoteCfg.gpioPin, DetectorConfig{
		MaximumDurationWait:  0,
		MaximumPulseDuration: 0,
		DelayTime:            0,
	})
	defer d.Close()
	go func() {
		dr := newDurationReducer(time.Duration(0))
		for pulse := range pulses {
			code := utilities.Apply(pulse, dr.isLowDuration)
			buttonType := matchCode(code)
			if buttonType != nil {
				buttonPresses <- *buttonType
			}
		}
	}()

	d.ReadPulsesUntilContextCancelled(pulses, ctx)
}
