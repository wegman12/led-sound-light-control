package remote

import (
	"context"
	"encoding/json"
	"fmt"
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
	case PinkButtonType:
		config := createMixedColorConfig(map[led.Color]float64{
			led.RedLedColor:  1.0,
			led.BlueLedColor: 0.4,
		})
		c.lightController.SendEvent(light.ChangeBehaviorEvent{Config: config})
	case LightBlueButtonType:
		config := createMixedColorConfig(map[led.Color]float64{
			led.BlueLedColor:  1.0,
			led.GreenLedColor: 0.5,
		})
		c.lightController.SendEvent(light.ChangeBehaviorEvent{Config: config})
	case LightGreenButtonType:
		config := createMixedColorConfig(map[led.Color]float64{
			led.GreenLedColor: 1.0,
			led.WhiteLedColor: 0.3,
		})
		c.lightController.SendEvent(light.ChangeBehaviorEvent{Config: config})
	case OrangeButtonType:
		config := createMixedColorConfig(map[led.Color]float64{
			led.RedLedColor:   1.0,
			led.GreenLedColor: 0.4,
		})
		c.lightController.SendEvent(light.ChangeBehaviorEvent{Config: config})
	case LightOrangeButtonType:
		config := createMixedColorConfig(map[led.Color]float64{
			led.RedLedColor:   1.0,
			led.GreenLedColor: 0.6,
		})
		c.lightController.SendEvent(light.ChangeBehaviorEvent{Config: config})
	case GreenBlueButtonType:
		config := createMixedColorConfig(map[led.Color]float64{
			led.GreenLedColor: 1.0,
			led.BlueLedColor:  1.0,
		})
		c.lightController.SendEvent(light.ChangeBehaviorEvent{Config: config})
	case IndigoButtonType:
		config := createMixedColorConfig(map[led.Color]float64{
			led.BlueLedColor: 1.0,
			led.RedLedColor:  0.3,
		})
		c.lightController.SendEvent(light.ChangeBehaviorEvent{Config: config})
	case LightPinkButtonType:
		config := createMixedColorConfig(map[led.Color]float64{
			led.RedLedColor:   0.8,
			led.BlueLedColor:  0.3,
			led.WhiteLedColor: 0.4,
		})
		c.lightController.SendEvent(light.ChangeBehaviorEvent{Config: config})
	case SkyBlueButtonType:
		config := createMixedColorConfig(map[led.Color]float64{
			led.BlueLedColor:  0.8,
			led.GreenLedColor: 0.4,
			led.WhiteLedColor: 0.3,
		})
		c.lightController.SendEvent(light.ChangeBehaviorEvent{Config: config})
	case VioletButtonType:
		config := createMixedColorConfig(map[led.Color]float64{
			led.RedLedColor:  0.6,
			led.BlueLedColor: 1.0,
		})
		c.lightController.SendEvent(light.ChangeBehaviorEvent{Config: config})
	case TealButtonType:
		config := createMixedColorConfig(map[led.Color]float64{
			led.GreenLedColor: 0.7,
			led.BlueLedColor:  1.0,
		})
		c.lightController.SendEvent(light.ChangeBehaviorEvent{Config: config})
	case GoldButtonType:
		config := createMixedColorConfig(map[led.Color]float64{
			led.RedLedColor:   1.0,
			led.GreenLedColor: 0.7,
		})
		c.lightController.SendEvent(light.ChangeBehaviorEvent{Config: config})
	case YellowButtonType:
		config := createMixedColorConfig(map[led.Color]float64{
			led.RedLedColor:   1.0,
			led.GreenLedColor: 1.0,
		})
		c.lightController.SendEvent(light.ChangeBehaviorEvent{Config: config})
	case DarkTealButtonType:
		config := createMixedColorConfig(map[led.Color]float64{
			led.GreenLedColor: 0.5,
			led.BlueLedColor:  0.7,
		})
		c.lightController.SendEvent(light.ChangeBehaviorEvent{Config: config})
	case PurpleButtonType:
		config := createMixedColorConfig(map[led.Color]float64{
			led.RedLedColor:  0.5,
			led.BlueLedColor: 1.0,
		})
		c.lightController.SendEvent(light.ChangeBehaviorEvent{Config: config})
	case LightSkyBlueButtonType:
		config := createMixedColorConfig(map[led.Color]float64{
			led.BlueLedColor:  0.6,
			led.GreenLedColor: 0.3,
			led.WhiteLedColor: 0.5,
		})
		c.lightController.SendEvent(light.ChangeBehaviorEvent{Config: config})
	default:
		log.Printf("No action mapped for button: %s", ButtonNames[button])
	}
}

// createSolidColorConfig creates a ManagerConfig for a solid color at full brightness
func createSolidColorConfig(color led.Color) light.ManagerConfig {
	return createMixedColorConfig(map[led.Color]float64{
		color: 1.0,
	})
}

// createMixedColorConfig creates a ManagerConfig with multiple colors at specified power levels
func createMixedColorConfig(colorMix map[led.Color]float64) light.ManagerConfig {
	behaviors := make([]light.BehaviorConfig, 0, len(colorMix))

	for color, power := range colorMix {
		// Create the JSON config for a fixed behavior at the specified power
		configJSON := json.RawMessage([]byte(`{"power_value": ` + floatToString(power) + `}`))

		// Create the behavior using the factory
		fixedBehavior, err := behavior.CreateBehavior(behavior.FixedBehaviorType, configJSON)
		if err != nil {
			log.Printf("Error creating fixed behavior for color %v: %v", color, err)
			continue
		}

		behaviors = append(behaviors, light.BehaviorConfig{
			Color:    color,
			Behavior: fixedBehavior,
		})
	}

	return light.ManagerConfig{
		Behaviors: behaviors,
	}
}

// floatToString converts a float64 to a string for JSON
func floatToString(f float64) string {
	if f > 1.0 {
		f = 1.0
	}
	if f < 0.0 {
		f = 0.0
	}
	return fmt.Sprintf("%.2f", f)
}
