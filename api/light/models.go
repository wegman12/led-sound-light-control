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
	Color        led.Color         `json:"color"`
	BehaviorType behavior.Type     `json:"-"` // Not serialized, populated during unmarshal
	RawConfig    json.RawMessage   `json:"config"`
	Behavior     behavior.Behavior `json:"-"` // Created later with AudioProvider
}

func (m *BehaviorConfig) UnmarshalJSON(data []byte) error {
	// Anonymous struct to unmarshal known fields and the raw shape data
	var temp struct {
		BehaviorType string          `json:"behavior_type"`
		Color        string          `json:"color"`
		Config       json.RawMessage `json:"config"`
	}
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	m.Color = led.LookupColor(temp.Color)
	m.BehaviorType = behavior.LookupBehavior(temp.BehaviorType)
	m.RawConfig = temp.Config

	return nil
}

// CreateBehavior creates the behavior instance with the given AudioProvider
func (m *BehaviorConfig) CreateBehavior(audioProvider behavior.AudioProvider) error {
	b, err := behavior.CreateBehavior(m.BehaviorType, m.RawConfig, audioProvider)
	if err != nil {
		return err
	}
	m.Behavior = b
	return nil
}
