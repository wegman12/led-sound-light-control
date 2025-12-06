package light

import (
	"context"
	"log"
	"sync"
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
}

func NewController(ctx context.Context, wg *sync.WaitGroup) *Controller {
	c := &Controller{
		ctx:          ctx,
		eventChannel: make(chan LightEvent, 100),
		wg:           wg,
	}

	c.wg.Add(1)
	go c.processEvents()

	return c
}

func (c *Controller) processEvents() {
	defer c.wg.Done()
	defer func() {
		if r := recover(); r != nil {
			// Masking panics from go-bbhw library
			log.Printf("Recovered from panic in light controller: %v", r)
		}
	}()

	for {
		select {
		case <-c.ctx.Done():
			if c.manager != nil {
				c.manager.Close()
			}
			return
		case event := <-c.eventChannel:
			if err := c.handleEvent(event); err != nil {
				log.Printf("Error handling light event: %v", err)
			}
		}
	}
}

func (c *Controller) handleEvent(event LightEvent) error {
	switch e := event.(type) {
	case StartEvent:
		if c.manager != nil {
			c.manager.Start(c.ctx)
			c.isRunning = true
		}
	case StopEvent:
		if c.manager != nil {
			c.manager.Stop()
			c.isRunning = false
		}
	case ChangeBehaviorEvent:
		if c.manager == nil {
			var err error
			c.manager, err = NewManager(e.Config)
			if err != nil {
				return err
			}
		} else {
			if err := c.manager.UpdateBehaviors(e.Config); err != nil {
				return err
			}
		}
	case TogglePowerEvent:
		if c.manager != nil {
			if c.isRunning {
				c.manager.Stop()
				c.isRunning = false
			} else {
				c.manager.Start(c.ctx)
				c.isRunning = true
			}
		}
	case TogglePauseEvent:
		if c.manager != nil {
			// Toggle the pause state
			currentPause := c.manager.paused.Load()
			c.manager.paused.Store(!currentPause)
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
