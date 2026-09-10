package behavior

import (
	"sync"
	"sync/atomic"
	"time"
)

// AudioProfile represents the current audio frequency analysis
type AudioProfile struct {
	Bass      float64
	MidLow    float64
	MidHigh   float64
	Treble    float64
	Timestamp time.Time
}

// AudioProvider provides access to the latest audio profile data
type AudioProvider interface {
	// GetLatestProfile returns the most recent audio profile, or nil if no data available
	GetLatestProfile() *AudioProfile

	// UpdateProfile updates the current audio profile (called by audio manager)
	UpdateProfile(profile AudioProfile)

	// Subscribe returns a channel that receives audio profile updates
	// Note: Currently not implemented, reserved for future use
	Subscribe() <-chan AudioProfile
}

// channelAudioProvider is a thread-safe implementation of AudioProvider
type channelAudioProvider struct {
	latestProfile atomic.Value // stores *AudioProfile
	mu            sync.RWMutex
	subscribers   []chan AudioProfile
}

// NewAudioProvider creates a new AudioProvider instance
func NewAudioProvider() AudioProvider {
	provider := &channelAudioProvider{
		subscribers: make([]chan AudioProfile, 0),
	}
	// Initialize with nil profile
	provider.latestProfile.Store((*AudioProfile)(nil))
	return provider
}

// GetLatestProfile returns the most recent audio profile
// Returns nil if no audio data has been received yet
func (p *channelAudioProvider) GetLatestProfile() *AudioProfile {
	val := p.latestProfile.Load()
	if val == nil {
		return nil
	}
	profile, ok := val.(*AudioProfile)
	if !ok {
		return nil
	}
	return profile
}

// UpdateProfile atomically updates the current audio profile
// This method is thread-safe and can be called from the audio processing goroutine
func (p *channelAudioProvider) UpdateProfile(profile AudioProfile) {
	// Make a copy to avoid race conditions
	profileCopy := &AudioProfile{
		Bass:      profile.Bass,
		MidLow:    profile.MidLow,
		MidHigh:   profile.MidHigh,
		Treble:    profile.Treble,
		Timestamp: profile.Timestamp,
	}

	// Atomically store the new profile
	p.latestProfile.Store(profileCopy)

	// Notify subscribers (if any)
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, ch := range p.subscribers {
		select {
		case ch <- *profileCopy:
			// Successfully sent
		default:
			// Channel full, skip this update
		}
	}
}

// Subscribe returns a channel that receives audio profile updates
// Note: Currently returns a dummy channel for interface compatibility
// Full pub/sub implementation can be added in future if needed
func (p *channelAudioProvider) Subscribe() <-chan AudioProfile {
	// Create a buffered channel to prevent blocking
	ch := make(chan AudioProfile, 10)

	p.mu.Lock()
	p.subscribers = append(p.subscribers, ch)
	p.mu.Unlock()

	return ch
}
