package remote

import (
	"context"
	"time"

	"github.com/wegman12/go-bbhw"
	"github.com/wegman12/led-sound-light-control/utilities"
)

const (
	detectorDefaultMaximumDurationWait  = 1 * time.Millisecond
	detectorDefaultMaximumPulseDuration = 10 * time.Millisecond
	detectDefaultDelayTime              = 1 * time.Microsecond
)

type DetectorConfig struct {
	MaximumDurationWait  time.Duration `json:"maximum_duration_wait"`
	MaximumPulseDuration time.Duration `json:"maximum_pulse_duration"`
	DelayTime            time.Duration `json:"delay_time"`
}

type DetectionPulse struct {
	TimeHigh time.Duration `json:"time_high"`
	TimeLow  time.Duration `json:"time_low"`
}

type detector struct {
	pin          *bbhw.MMappedGPIO
	cfg          DetectorConfig
	pulseChannel chan []DetectionPulse
}

func (cfg DetectorConfig) setDefaults() {
	utilities.SetValueOrDefault(&cfg.MaximumDurationWait, detectorDefaultMaximumDurationWait)
	utilities.SetValueOrDefault(&cfg.DelayTime, detectDefaultDelayTime)
	utilities.SetValueOrDefault(&cfg.MaximumPulseDuration, detectorDefaultMaximumPulseDuration)
}

func newDetector(pinNumber uint, pulseChannel chan []DetectionPulse, cfg DetectorConfig) *detector {
	return &detector{
		pin:          bbhw.NewMMappedGPIO(pinNumber, bbhw.IN),
		cfg:          cfg,
		pulseChannel: pulseChannel,
	}
}

func (d *detector) read() uint {
	state, _ := d.pin.GetState()
	if state {
		return 0
	} else {
		return 1
	}
}

func (d *detector) Close() {
	d.pin.Close()
}

func (d *detector) ReadPulsesUntilContextCancelled(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if d.read() == 1 {
				// Read a signal packet
				packet := d.readPulsePacket()
				if len(packet) > 0 {
					d.pulseChannel <- packet
				}
			}
			time.Sleep(d.cfg.DelayTime)
		}
		for d.read() == 0 {
			// wait until high
			time.Sleep(d.cfg.DelayTime)
		}
	}
}

func (d *detector) readPulsePacket() []DetectionPulse {
	pulses := make([]DetectionPulse, 0)
	start := time.Now()
	timedOut := false
	for !timedOut && time.Since(start) < d.cfg.MaximumDurationWait {
		var pulse DetectionPulse
		pulse, timedOut = d.readPulse()
		pulses = append(pulses, pulse)
	}

	return pulses
}

func (d *detector) readPulse() (DetectionPulse, bool) {
	var timedOut bool
	back := DetectionPulse{}
	back.TimeHigh, timedOut = d.readWhileValue(1)
	if timedOut {
		return back, true
	}
	back.TimeLow, timedOut = d.readWhileValue(2)
	if timedOut {
		return back, true
	}
	return back, false
}

func (d *detector) readWhileValue(value uint) (time.Duration, bool) {
	start := time.Now()
	for d.read() == value {
		if time.Since(start) > d.cfg.DelayTime {
			return d.cfg.DelayTime, false
		}
		time.Sleep(d.cfg.DelayTime)
	}
	return time.Since(start), true
}
