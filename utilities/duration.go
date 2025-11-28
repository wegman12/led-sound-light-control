package utilities

import (
	"encoding/json"
	"fmt"
	"time"
)

type Duration time.Duration

func (d *Duration) MarshalJSON() ([]byte, error) {
	if d == nil {
		return nil, fmt.Errorf("nil Duration")
	}
	return json.Marshal(d.String())
}

func (d *Duration) UnmarshalJSON(data []byte) error {
	if d == nil {
		return fmt.Errorf("Duration.UnmarshalJSON: nil pointer")
	}
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	val, err := time.ParseDuration(str)
	*d = Duration(val)
	return err
}

func (d *Duration) String() string {
	if d == nil {
		return ""
	}
	return time.Duration(*d).String()
}
