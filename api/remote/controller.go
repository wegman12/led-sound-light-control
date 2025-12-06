package remote

import (
	"context"
	"log"
	"sync"

	"github.com/wegman12/led-sound-light-control/light"
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
		c.lightController.SendEvent(light.StopEvent{})
	case PauseButtonType:
		c.lightController.SendEvent(light.StopEvent{})
	// TODO: Add more button mappings here
	// For example:
	// case RedButtonType:
	//     config := createFixedColorConfig(led.RedLedColor, 1.0)
	//     c.lightController.SendEvent(light.ChangeBehaviorEvent{
	//         Config: config,
	//     })
	default:
		log.Printf("No action mapped for button: %s", ButtonNames[button])
	}
}
