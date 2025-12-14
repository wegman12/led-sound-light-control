package audio

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// PRUManager manages PRU1 audio sampling and provides streaming sound profiles
type PRUManager struct {
	sampler *PRUSampler
	logger  *zap.Logger
	cancel  context.CancelFunc
	mu      sync.Mutex
}

// NewPRUManager creates a new PRU audio manager
func NewPRUManager(logger *zap.Logger) (*PRUManager, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	sampler, err := NewPRUSampler()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize PRU sampler: %w", err)
	}

	return &PRUManager{
		sampler: sampler,
		logger:  logger,
	}, nil
}

// Start begins streaming sound profiles to the provided channel
// Updates at the specified rate (e.g., 10 * time.Millisecond for 100 Hz)
func (pm *PRUManager) Start(profileChannel chan<- *SoundProfile, updateRate time.Duration, ctx context.Context) error {
	pm.mu.Lock()
	if pm.cancel != nil {
		pm.cancel()
	}
	ctx, pm.cancel = context.WithCancel(ctx)
	pm.mu.Unlock()

	pm.logger.Info("Starting PRU audio manager", zap.Duration("update_rate", updateRate))

	ticker := time.NewTicker(updateRate)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			pm.logger.Info("PRU audio manager stopped")
			return nil

		case <-ticker.C:
			profile, err := pm.sampler.ReadSoundProfile()
			if err != nil {
				pm.logger.Error("Failed to read sound profile", zap.Error(err))
				continue
			}

			select {
			case profileChannel <- profile:
				// Successfully sent profile
			default:
				// Channel full, skip this update
				pm.logger.Debug("Profile channel full, skipping update")
			}
		}
	}
}

// Stop stops the PRU audio manager
func (pm *PRUManager) Stop() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.cancel != nil {
		pm.logger.Info("Stopping PRU audio manager")
		pm.cancel()
		pm.cancel = nil
	}
}

// Close releases PRU resources
func (pm *PRUManager) Close() error {
	pm.Stop()
	if pm.sampler != nil {
		return pm.sampler.Close()
	}
	return nil
}

// GetStatus returns the current PRU status
func (pm *PRUManager) GetStatus() (*PRUStatus, error) {
	return pm.sampler.GetStatus()
}

// SetFFTEnable enables or disables FFT processing
func (pm *PRUManager) SetFFTEnable(enable bool) error {
	return pm.sampler.SetFFTEnable(enable)
}

// SetFrequencyBands updates the frequency band boundaries
func (pm *PRUManager) SetFrequencyBands(bands *FrequencyBands) error {
	return pm.sampler.SetFrequencyBands(bands.BassMax, bands.MidLowMax, bands.MidHighMax)
}

// GetLatestProfile returns the most recent sound profile without starting a stream
func (pm *PRUManager) GetLatestProfile() (*SoundProfile, error) {
	return pm.sampler.ReadSoundProfile()
}
