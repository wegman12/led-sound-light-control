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

func CreateBehavior(t Type, cfg json.RawMessage) (Behavior, error) {
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
	default:
		return nil, fmt.Errorf("unknown ActiveBehavior type %d", t)
	}
}
