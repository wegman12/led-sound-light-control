package light

import "encoding/json"

type ManagerConfig struct {
	Behaviors []BehaviorConfig `json:"behaviors"`
}

type BehaviorConfig struct {
	Behavior string          `json:"behavior"`
	Color    string          `json:"color"`
	Config   json.RawMessage `json:"config"`
}
