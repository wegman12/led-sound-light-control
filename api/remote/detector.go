package remote

import (
	"context"
	"fmt"
	"time"

	"github.com/wegman12/go-bbhw"
	"github.com/wegman12/led-sound-light-control/utilities"
)

const (
	detectorDefaultMaximumDurationWait  = 5 * time.Millisecond
	detectorDefaultMaximumPulseDuration = 100 * time.Millisecond
	detectDefaultDelayTime              = 500 * time.Nanosecond
	detectorDefaultMinimumPulseLength   = 5
)

type DetectorConfig struct {
	MaximumDurationWait  time.Duration `json:"maximum_duration_wait"`
	MaximumPulseDuration time.Duration `json:"maximum_pulse_duration"`
	DelayTime            time.Duration `json:"delay_time"`
	MinimumPulseLength   int           `json:"minimum_pulse_length"`
}

type DetectionPulse struct {
	TimeHigh time.Duration `json:"time_high"`
	TimeLow  time.Duration `json:"time_low"`
}

type detector struct {
	pin *bbhw.MMappedGPIO
	cfg DetectorConfig
}

func (cfg *DetectorConfig) setDefaults() {
	if cfg == nil {
		return
	}
	utilities.SetValueOrDefault(&cfg.MaximumDurationWait, detectorDefaultMaximumDurationWait)
	utilities.SetValueOrDefault(&cfg.DelayTime, detectDefaultDelayTime)
	utilities.SetValueOrDefault(&cfg.MaximumPulseDuration, detectorDefaultMaximumPulseDuration)
	utilities.SetValueOrDefault(&cfg.MinimumPulseLength, detectorDefaultMinimumPulseLength)
}

func newDetector(pinNumber uint, cfg DetectorConfig) *detector {
	return &detector{
		pin: bbhw.NewMMappedGPIO(pinNumber, bbhw.IN),
		cfg: cfg,
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

func (d *detector) ReadPulsesUntilContextCancelled(pulseChannel chan []DetectionPulse, ctx context.Context) {
	d.cfg.setDefaults()
	if pulseChannel == nil {
		fmt.Println("pulse channel is nil")
		return
	}
	for {
		select {
		case <-ctx.Done():
			close(pulseChannel)
			return
		default:
			if d.read() == 1 {
				// Read a signal packet
				packet := d.readPulsePacket(ctx)
				if len(packet) > d.cfg.MinimumPulseLength {
					pulseChannel <- packet
				}
			}
			time.Sleep(d.cfg.DelayTime)
		}
	}
}

func (d *detector) readPulsePacket(ctx context.Context) []DetectionPulse {
	pulses := make([]DetectionPulse, 0)
	start := time.Now()
	timedOut := false
	for !timedOut && time.Since(start) < d.cfg.MaximumPulseDuration {
		select {
		case <-ctx.Done():
			return pulses
		default:
			var pulse DetectionPulse
			pulse, timedOut = d.readPulse()
			pulses = append(pulses, pulse)
		}
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
	back.TimeLow, timedOut = d.readWhileValue(0)
	if timedOut {
		return back, true
	}
	return back, false
}

func (d *detector) readWhileValue(value uint) (time.Duration, bool) {
	start := time.Now()
	for d.read() == value {
		if time.Since(start) > d.cfg.MaximumDurationWait {
			return d.cfg.MaximumDurationWait, true
		}
		time.Sleep(d.cfg.DelayTime)
	}
	return time.Since(start), false
}
