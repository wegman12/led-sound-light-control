package behavior

import (
	"encoding/json"
	"time"

	"github.com/wegman12/led-sound-light-control/utilities"
)

const defaultJoinerDuration = time.Millisecond * 100

type JoinedBehavior struct {
	Duration       utilities.Duration
	ActiveBehavior Behavior
}

func (m *JoinedBehavior) UnmarshalJSON(data []byte) error {
	// Anonymous struct to unmarshal known fields and the raw shape data
	var temp struct {
		BehaviorType   string             `json:"behavior_type"`
		Duration       utilities.Duration `json:"duration"`
		ActiveBehavior json.RawMessage    `json:"behavior"`
	}
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	bt := LookupBehavior(temp.BehaviorType)

	b, err := CreateBehavior(bt, temp.ActiveBehavior)
	if err != nil {
		return err
	}
	m.Duration = temp.Duration
	m.ActiveBehavior = b

	return nil
}

type joiner struct {
	Behaviors     []JoinedBehavior
	totalDuration utilities.Duration
}

func newJoiner(cfg json.RawMessage) (*joiner, error) {
	b := joiner{}
	if cfg != nil {
		err := json.Unmarshal(cfg, &b)
		if err != nil {
			return nil, err
		}
	}

	b.ensureDefaults()

	return &b, nil
}

func (j *joiner) ensureDefaults() {
	totalDuration := utilities.Duration(0)
	for i := range j.Behaviors {
		utilities.SetValueOrDefault(&j.Behaviors[i].Duration, utilities.Duration(defaultJoinerDuration))
		totalDuration += j.Behaviors[i].Duration
	}
	j.totalDuration = totalDuration
}

func (j *joiner) GetPower(t time.Duration) *float64 {
	to := t % time.Duration(j.totalDuration)
	bin := time.Duration(0)
	for _, b := range j.Behaviors {
		bin += time.Duration(b.Duration)
		if bin >= to {
			return b.ActiveBehavior.GetPower(to)
		}
	}
	return nil
}

func (j *joiner) Weight() float64 {
	return 1
}
