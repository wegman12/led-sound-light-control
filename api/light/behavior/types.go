package behavior

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type Type int

const (
	BreathingBehaviorType Type = iota
	FlashingBehaviorType
	FixedBehaviorType
	SkipperBehaviorType
	JoinerBehaviorType
	AudioModulatorBehaviorType
)

func LookupBehavior(behaviorName string) Type {
	switch strings.ToLower(behaviorName) {
	case "breathing":
		return BreathingBehaviorType
	case "flashing":
		return FlashingBehaviorType
	case "fixed":
		return FixedBehaviorType
	case "skipper":
		return SkipperBehaviorType
	case "joiner":
		return JoinerBehaviorType
	case "audio_modulator", "audio":
		return AudioModulatorBehaviorType
	default:
		return BreathingBehaviorType
	}
}

type Result struct {
	value *float64
}

type Behavior interface {
	GetPower(t time.Duration) *float64
	Weight() float64
}

// CreateBehavior creates a behavior instance from type and config
// audioProvider can be nil for non-audio behaviors
func CreateBehavior(t Type, cfg json.RawMessage, audioProvider AudioProvider) (Behavior, error) {
	switch t {
	case BreathingBehaviorType:
		return newBreather(cfg)
	case FlashingBehaviorType:
		return newFlasher(cfg)
	case FixedBehaviorType:
		return newFixer(cfg)
	case SkipperBehaviorType:
		return newSkipper(cfg)
	case JoinerBehaviorType:
		return newJoiner(cfg)
	case AudioModulatorBehaviorType:
		if audioProvider == nil {
			return nil, fmt.Errorf("audio_modulator behavior requires AudioProvider")
		}
		return newAudioModulator(cfg, audioProvider)
	default:
		return nil, fmt.Errorf("unknown ActiveBehavior type %d", t)
	}
}
