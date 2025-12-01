package light

import (
	"encoding/json"

	"github.com/wegman12/led-sound-light-control/light/behavior"
	"github.com/wegman12/led-sound-light-control/light/led"
)

type ManagerConfig struct {
	Behaviors []BehaviorConfig `json:"behaviors"`
}

type BehaviorConfig struct {
	Color    led.Color         `json:"color"`
	Behavior behavior.Behavior `json:"config"`
}

func (m *BehaviorConfig) UnmarshalJSON(data []byte) error {
	// Anonymous struct to unmarshal known fields and the raw shape data
	var temp struct {
		BehaviorType string          `json:"behavior_type"`
		Color        string          `json:"color"`
		Behavior     json.RawMessage `json:"config"`
	}
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	m.Color = led.LookupColor(temp.Color)

	bt := behavior.LookupBehavior(temp.BehaviorType)

	b, err := behavior.CreateBehavior(bt, temp.Behavior)
	if err != nil {
		return err
	}
	m.Behavior = b

	return nil
}
