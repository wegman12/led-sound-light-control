package light

import (
	"context"
	"sync"

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
			c.logger.Warn("Cannot start lights: no manager configured")
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
			c.manager, err = NewManager(e.Config)
			if err != nil {
				c.logger.Error("Failed to create light manager", zap.Error(err))
				return err
			}
			c.logger.Info("Light manager created successfully")
		} else {
			c.logger.Debug("Updating existing light manager behaviors")
			if err := c.manager.UpdateBehaviors(e.Config); err != nil {
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
			c.logger.Warn("Cannot toggle power: no manager configured")
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
