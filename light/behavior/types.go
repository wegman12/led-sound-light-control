package behavior

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/wegman12/led-sound-light-control/light/led"
)

type BehaviorType int

const (
	BreathingBehaviorType BehaviorType = iota
	FlashingBehaviorType
	FixedBehaviorType
	SkipperBehaviorType
)

func LookupBehavior(behaviorName string) BehaviorType {
	switch strings.ToLower(behaviorName) {
	case "breathing":
		return BreathingBehaviorType
	case "flashing":
		return FlashingBehaviorType
	case "fixed":
		return FixedBehaviorType
	case "skipper":
		return SkipperBehaviorType
	default:
		return BreathingBehaviorType
	}
}

type Behavior interface {
	Start(ctx context.Context)
	Stop()
}

func CreateBehavior(l led.Led, t BehaviorType, cfg json.RawMessage) (Behavior, error) {
	switch t {
	case BreathingBehaviorType:
		return newBreather(l, cfg)
	case FlashingBehaviorType:
		return newFlasher(l, cfg)
	case FixedBehaviorType:
		return nil, fmt.Errorf("fixed behavior has not been implemented yet")
	case SkipperBehaviorType:
		return nil, fmt.Errorf("skipper behavior has not been implemented yet")
	default:
		return nil, fmt.Errorf("unknown behavior type %d", t)
	}
}
