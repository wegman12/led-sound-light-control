package light

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/wegman12/led-sound-light-control/light/behavior"
	"github.com/wegman12/led-sound-light-control/light/led"
	"go.uber.org/zap"
)

type EventType int

const (
	StartEventType EventType = iota
	StopEventType
	ChangeBehaviorEventType
	TogglePowerEventType
	TogglePauseEventType
)

type LightEvent interface {
	Type() EventType
}

type StartEvent struct{}

func (e StartEvent) Type() EventType {
	return StartEventType
}

type StopEvent struct{}

func (e StopEvent) Type() EventType {
	return StopEventType
}

type ChangeBehaviorEvent struct {
	Config ManagerConfig
}

func (e ChangeBehaviorEvent) Type() EventType {
	return ChangeBehaviorEventType
}

type TogglePowerEvent struct{}

func (e TogglePowerEvent) Type() EventType {
	return TogglePowerEventType
}

type TogglePauseEvent struct{}

func (e TogglePauseEvent) Type() EventType {
	return TogglePauseEventType
}

type Controller struct {
	ctx          context.Context
	manager      *Manager
	eventChannel chan LightEvent
	wg           *sync.WaitGroup
	isRunning    bool
	logger       *zap.Logger
}

func NewController(ctx context.Context, wg *sync.WaitGroup, logger *zap.Logger) *Controller {
	c := &Controller{
		ctx:          ctx,
		eventChannel: make(chan LightEvent, 100),
		wg:           wg,
		logger:       logger,
	}

	logger.Debug("Light controller initialized")

	c.wg.Add(1)
	go c.processEvents()

	return c
}

func (c *Controller) processEvents() {
	defer c.wg.Done()
	defer func() {
		if r := recover(); r != nil {
			// Masking panics from go-bbhw library
			c.logger.Warn("Recovered from panic in light controller", zap.Any("panic", r))
		}
	}()

	c.logger.Debug("Light controller event processing started")

	for {
		select {
		case <-c.ctx.Done():
			c.logger.Info("Light controller shutting down")
			if c.manager != nil {
				c.manager.Close()
			}
			return
		case event := <-c.eventChannel:
			c.logger.Debug("Processing light event", zap.String("event_type", eventTypeName(event.Type())))
			if err := c.handleEvent(event); err != nil {
				c.logger.Error("Error handling light event", zap.Error(err), zap.String("event_type", eventTypeName(event.Type())))
			}
		}
	}
}

func eventTypeName(t EventType) string {
	switch t {
	case StartEventType:
		return "start"
	case StopEventType:
		return "stop"
	case ChangeBehaviorEventType:
		return "change_behavior"
	case TogglePowerEventType:
		return "toggle_power"
	case TogglePauseEventType:
		return "toggle_pause"
	default:
		return "unknown"
	}
}

func (c *Controller) handleEvent(event LightEvent) error {
	switch e := event.(type) {
	case StartEvent:
		if c.manager != nil {
			c.logger.Info("Starting lights")
			c.manager.Start(c.ctx)
			c.isRunning = true
		} else {
			c.logger.Info("No behavior configured, creating default red behavior")
			defaultConfig := createDefaultConfig()
			var err error
			// TODO: Phase 3 - Pass actual AudioProvider instead of nil
		c.manager, err = NewManager(defaultConfig, nil)
			if err != nil {
				c.logger.Error("Failed to create default manager", zap.Error(err))
				return err
			}
			c.logger.Info("Starting lights with default behavior")
			c.manager.Start(c.ctx)
			c.isRunning = true
		}
	case StopEvent:
		if c.manager != nil {
			c.logger.Info("Stopping lights")
			c.manager.Stop()
			c.isRunning = false
		}
	case ChangeBehaviorEvent:
		c.logger.Info("Changing light behavior", zap.Int("num_behaviors", len(e.Config.Behaviors)))
		if c.manager == nil {
			c.logger.Debug("Creating new light manager")
			var err error
			// TODO: Phase 3 - Pass actual AudioProvider instead of nil
			c.manager, err = NewManager(e.Config, nil)
			if err != nil {
				c.logger.Error("Failed to create light manager", zap.Error(err))
				return err
			}
			c.logger.Info("Light manager created successfully")
		} else {
			c.logger.Debug("Updating existing light manager behaviors")
			// TODO: Phase 3 - Pass actual AudioProvider instead of nil
			if err := c.manager.UpdateBehaviors(e.Config, nil); err != nil {
				c.logger.Error("Failed to update light behaviors", zap.Error(err))
				return err
			}
			c.logger.Info("Light behaviors updated successfully")
		}
	case TogglePowerEvent:
		if c.manager != nil {
			if c.isRunning {
				c.logger.Info("Toggling power: turning off")
				c.manager.Stop()
				c.isRunning = false
			} else {
				c.logger.Info("Toggling power: turning on")
				c.manager.Start(c.ctx)
				c.isRunning = true
			}
		} else {
			c.logger.Info("No behavior configured, creating default red behavior and turning on")
			defaultConfig := createDefaultConfig()
			var err error
			// TODO: Phase 3 - Pass actual AudioProvider instead of nil
		c.manager, err = NewManager(defaultConfig, nil)
			if err != nil {
				c.logger.Error("Failed to create default manager", zap.Error(err))
				return err
			}
			c.logger.Info("Starting lights with default behavior")
			c.manager.Start(c.ctx)
			c.isRunning = true
		}
	case TogglePauseEvent:
		if c.manager != nil {
			currentPause := c.manager.paused.Load()
			newPause := !currentPause
			c.logger.Info("Toggling pause state", zap.Bool("paused", newPause))
			c.manager.paused.Store(newPause)
		} else {
			c.logger.Warn("Cannot toggle pause: no manager configured")
		}
	}
	return nil
}

func (c *Controller) SendEvent(event LightEvent) {
	select {
	case c.eventChannel <- event:
	case <-c.ctx.Done():
		// Context cancelled, don't block
	}
}

// createDefaultConfig creates a default configuration with Red LED at full power
func createDefaultConfig() ManagerConfig {
	return ManagerConfig{
		Behaviors: []BehaviorConfig{
			{
				Color:    led.RedLedColor,
				Behavior: mustCreateFixedBehavior(1.0),
			},
		},
	}
}

// mustCreateFixedBehavior creates a fixed behavior with the given power value
// Panics if creation fails, which should never happen with valid power values
func mustCreateFixedBehavior(power float64) behavior.Behavior {
	configJSON := json.RawMessage([]byte(`{"power_value": ` + floatToString(power) + `}`))
	fixedBehavior, err := behavior.CreateBehavior(behavior.FixedBehaviorType, configJSON, nil)
	if err != nil {
		panic("failed to create fixed behavior: " + err.Error())
	}
	return fixedBehavior
}

// floatToString converts a float64 to a string for JSON
func floatToString(f float64) string {
	if f >= 1.0 {
		return "1.0"
	}
	if f <= 0.0 {
		return "0.0"
	}
	// For values between 0 and 1, format with 2 decimal places
	return fmt.Sprintf("%.2f", f)
}
