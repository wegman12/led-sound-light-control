package sound

import (
	"context"
	"sync"
	"time"

	"github.com/wegman12/led-sound-light-control/sound/processing"
	"github.com/wegman12/led-sound-light-control/sound/sampling"
	"go.uber.org/zap"
)

type Manager struct {
	sampler   *sampling.Sampler
	processor *processing.Processor
	cancel    context.CancelFunc
	logger    *zap.Logger
}

func NewManager(bufferSize, samplingRate int, targetInputRate, bassCutoff, midCutoff float64, delayBetweenSamples, delayBetweenProcessing time.Duration, logger *zap.Logger) (*Manager, error) {
	sampler, err := sampling.NewSampler(bufferSize, samplingRate, targetInputRate, delayBetweenSamples, logger)
	if err != nil {
		return nil, err
	}
	processor, err := processing.NewProcessor(delayBetweenProcessing, bassCutoff, midCutoff, logger)
	if err != nil {
		return nil, err
	}
	return &Manager{
		sampler:   sampler,
		processor: processor,
		logger:    logger,
	}, nil
}

func (m *Manager) Start(resultsChannel chan processing.Result, ctx context.Context) {
	if m.logger == nil {
		m.logger = zap.NewNop()
	}
	if resultsChannel == nil {
		resultsChannel = make(chan processing.Result, 10000)
	}
	if m.cancel != nil {
		m.cancel()
	}
	ctx, m.cancel = context.WithCancel(ctx)
	defer m.Stop()

	payloadChannel := make(chan sampling.Payload, 10000)

	wg := &sync.WaitGroup{}

	wg.Add(2)
	go func() {
		m.logger.Debug("Starting processor")
		defer wg.Done()
		m.processor.ProcessResults(payloadChannel, resultsChannel, ctx)
	}()

	go func() {
		m.logger.Debug("Starting sampler")
		defer wg.Done()
		m.sampler.SampleAudio(payloadChannel, ctx)
	}()

	m.logger.Debug("Waiting for channel exit")
	wg.Wait()
	m.logger.Debug("Manager finished")
}

func (m *Manager) Stop() {
	if m.cancel != nil {
		m.logger.Debug("manager stopped - cancelling context")
		m.cancel()
		m.cancel = nil
	}
}
