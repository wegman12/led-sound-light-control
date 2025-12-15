package behavior_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/wegman12/led-sound-light-control/light"
)

func TestBehavesAsExpected(t *testing.T) {
	payload :=
		`
{
    "behaviors": [
        {
            "behavior_type": "joiner",
            "color": "red",
            "config": {
                "behaviors": [
                    {
                        "behavior_type": "fixed",
                        "duration": "1s",
                        "behavior": {
                            "power_value": 1.0
                        }
                    },
                    {
                        "behavior_type": "fixed",
                        "duration": "1s",
                        "behavior": {
                            "power_value": 0.0
                        }
                    },
                    {
                        "behavior_type": "fixed",
                        "duration": "1s",
                        "behavior": {
                            "power_value": 0.0
                        }
                    }
                ]
            }
        },
        {
            "behavior_type": "joiner",
            "color": "green",
            "config": {
                "behaviors": [
                    {
                        "behavior_type": "fixed",
                        "duration": "1s",
                        "behavior": {
                            "power_value": 0.0
                        }
                    },
                    {
                        "behavior_type": "fixed",
                        "duration": "1s",
                        "behavior": {
                            "power_value": 1.0
                        }
                    },
                    {
                        "behavior_type": "fixed",
                        "duration": "1s",
                        "behavior": {
                            "power_value": 0.0
                        }
                    }
                ]
            }
        },
        {
            "behavior_type": "joiner",
            "color": "blue",
            "config": {
                "behaviors": [
                    {
                        "behavior_type": "fixed",
                        "duration": "1s",
                        "behavior": {
                            "power_value": 0.0
                        }
                    },
                    {
                        "behavior_type": "fixed",
                        "duration": "1s",
                        "behavior": {
                            "power_value": 0.0
                        }
                    },
                    {
                        "behavior_type": "fixed",
                        "duration": "1s",
                        "behavior": {
                            "power_value": 1.0
                        }
                    }
                ]
            }
        }
    ]
}
`

	var cfg light.ManagerConfig
	err := json.Unmarshal([]byte(payload), &cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Behaviors) != 3 {
		t.Fatal("expected 3 behaviors")
	}

	// Create behaviors from config (passing nil for AudioProvider)
	for i := range cfg.Behaviors {
		if err := cfg.Behaviors[i].CreateBehavior(nil); err != nil {
			t.Fatalf("Failed to create behavior: %v", err)
		}
	}

	rp := cfg.Behaviors[0].Behavior.GetPower(500 * time.Millisecond)
	if rp == nil || *rp != 1.0 {
		t.Fatal("expected 1.0")
	}
}
