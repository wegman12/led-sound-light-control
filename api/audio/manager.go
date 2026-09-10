package audio

import (
	"context"
	"sync"
	"time"

	"github.com/wegman12/led-sound-light-control/audio/processing"
	"github.com/wegman12/led-sound-light-control/audio/sampling"
	"github.com/wegman12/led-sound-light-control/light/behavior"
	"go.uber.org/zap"
)

type Manager struct {
	sampler       *sampling.Sampler
	processor     *processing.Processor
	audioProvider behavior.AudioProvider
	cancel        context.CancelFunc
	logger        *zap.Logger
}

func NewManager(bufferSize, samplingRate int, targetInputRate, bassCutoff, midHighCutoff, trebleCutoff float64, delayBetweenSamples, delayBetweenProcessing time.Duration, audioProvider behavior.AudioProvider, logger *zap.Logger) (*Manager, error) {
	sampler, err := sampling.NewSampler(bufferSize, samplingRate, targetInputRate, delayBetweenSamples, logger)
	if err != nil {
		return nil, err
	}
	processor, err := processing.NewProcessor(delayBetweenProcessing, bassCutoff, midHighCutoff, trebleCutoff, logger)
	if err != nil {
		return nil, err
	}
	return &Manager{
		sampler:       sampler,
		processor:     processor,
		audioProvider: audioProvider,
		logger:        logger,
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

// StreamToLights streams audio processing results to the LED system via AudioProvider
// This method starts audio sampling and processing, then forwards results to the AudioProvider
func (m *Manager) StreamToLights(ctx context.Context) {
	if m.audioProvider == nil {
		m.logger.Warn("StreamToLights called but no AudioProvider configured")
		return
	}

	m.logger.Info("Starting audio stream to lights")

	// Create results channel for audio processing
	resultsChannel := make(chan processing.Result, 100)

	// Start audio processing in background
	go m.Start(resultsChannel, ctx)

	// Stream results to AudioProvider
	for {
		select {
		case <-ctx.Done():
			m.logger.Info("Audio streaming to lights stopped")
			return
		case result := <-resultsChannel:
			// Convert processing.Profile to behavior.AudioProfile
			profile := behavior.AudioProfile{
				Bass:      result.Profile.Bass,
				MidLow:    result.Profile.MidLow,
				MidHigh:   result.Profile.MidHigh,
				Treble:    result.Profile.Treble,
				Timestamp: time.Now(),
			}

			// Update the audio provider with the new profile
			m.audioProvider.UpdateProfile(profile)

			m.logger.Debug("Updated audio profile",
				zap.Float64("bass", profile.Bass),
				zap.Float64("mid_low", profile.MidLow),
				zap.Float64("mid_high", profile.MidHigh),
				zap.Float64("treble", profile.Treble),
			)
		}
	}
}
