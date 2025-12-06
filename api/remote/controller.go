package remote

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"github.com/wegman12/led-sound-light-control/light"
	"github.com/wegman12/led-sound-light-control/light/behavior"
	"github.com/wegman12/led-sound-light-control/light/led"
)

type Controller struct {
	ctx             context.Context
	manager         *Manager
	lightController *light.Controller
	wg              *sync.WaitGroup
}

func NewController(ctx context.Context, gpioPin uint, lightController *light.Controller, wg *sync.WaitGroup) *Controller {
	return &Controller{
		ctx:             ctx,
		manager:         NewManager(gpioPin),
		lightController: lightController,
		wg:              wg,
	}
}

func (c *Controller) Start() {
	buttonPresses := make(chan ButtonType, 100)

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.manager.ReportButtonPressesUntilContextCancelled(buttonPresses, c.ctx)
	}()

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		for {
			select {
			case <-c.ctx.Done():
				return
			case button := <-buttonPresses:
				c.handleButtonPress(button)
			}
		}
	}()
}

func (c *Controller) handleButtonPress(button ButtonType) {
	log.Printf("Button pressed: %s", ButtonNames[button])

	// Map button presses to light events
	switch button {
	case PowerButtonType:
		c.lightController.SendEvent(light.TogglePowerEvent{})
	case PauseButtonType:
		c.lightController.SendEvent(light.TogglePauseEvent{})
	case RedButtonType:
		config := createSolidColorConfig(led.RedLedColor)
		c.lightController.SendEvent(light.ChangeBehaviorEvent{
			Config: config,
		})
	case GreenButtonType:
		config := createSolidColorConfig(led.GreenLedColor)
		c.lightController.SendEvent(light.ChangeBehaviorEvent{
			Config: config,
		})
	case BlueButtonType:
		config := createSolidColorConfig(led.BlueLedColor)
		c.lightController.SendEvent(light.ChangeBehaviorEvent{
			Config: config,
		})
	case WhiteButtonType:
		config := createSolidColorConfig(led.WhiteLedColor)
		c.lightController.SendEvent(light.ChangeBehaviorEvent{
			Config: config,
		})
	default:
		log.Printf("No action mapped for button: %s", ButtonNames[button])
	}
}

// createSolidColorConfig creates a ManagerConfig for a solid color at full brightness
func createSolidColorConfig(color led.Color) light.ManagerConfig {
	// Create the JSON config for a fixed behavior at full power
	configJSON := json.RawMessage(`{"power_value": 1.0}`)

	// Create the behavior using the factory
	fixedBehavior, err := behavior.CreateBehavior(behavior.FixedBehaviorType, configJSON)
	if err != nil {
		log.Printf("Error creating fixed behavior: %v", err)
		return light.ManagerConfig{}
	}

	return light.ManagerConfig{
		Behaviors: []light.BehaviorConfig{
			{
				Color:    color,
				Behavior: fixedBehavior,
			},
		},
	}
}
