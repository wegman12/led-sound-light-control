package utilities

import (
	"go.uber.org/zap"
)

// NewLogger creates a new zap logger instance
// If development is true, creates a development logger with debug level
// Otherwise creates a production logger with info level
func NewLogger(development bool) (*zap.Logger, error) {
	var logger *zap.Logger
	var err error

	if development {
		config := zap.NewDevelopmentConfig()
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
		logger, err = config.Build()
	} else {
		config := zap.NewProductionConfig()
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
		logger, err = config.Build()
	}

	if err != nil {
		return nil, err
	}

	return logger, nil
}
