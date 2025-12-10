package remote

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

type Manager struct {
	detector *PRUDetector
	logger   *zap.Logger
}

// NewManager creates a new remote manager using PRU-based IR detection
// Returns error if PRU initialization fails
func NewManager(logger *zap.Logger) (*Manager, error) {
	detector, err := newPRUDetector()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize PRU detector: %w", err)
	}

	return &Manager{
		detector: detector,
		logger:   logger,
	}, nil
}

// Close releases PRU resources
func (m *Manager) Close() error {
	if m.detector != nil {
		return m.detector.Close()
	}
	return nil
}

// ReportButtonPressesUntilContextCancelled polls PRU for button events at 100ms intervals
// Sends detected buttons to the buttonPresses channel until context is cancelled
func (m *Manager) ReportButtonPressesUntilContextCancelled(buttonPresses chan ButtonType, ctx context.Context) {
	defer m.Close()
	defer close(buttonPresses)

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Read available button events from PRU ring buffer
			buttons, err := m.detector.ReadButtons()
			if err != nil {
				// Log error but continue - don't crash on read errors
				fmt.Printf("Error reading PRU buttons: %v\n", err)
				continue
			}

			// Send all detected buttons to channel
			for _, button := range buttons {
				m.logger.Debug(fmt.Sprintf("Detected button press: %s", ButtonNames[button]))
				buttonPresses <- button
			}
		}
	}
}
