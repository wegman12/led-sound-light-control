package processing

import (
	"context"
	"slices"
	"time"

	"github.com/wegman12/led-sound-light-control/audio/sampling"
	"go.uber.org/zap"
)

type Processor struct {
	delayBetweenProcessing time.Duration
	transformer            *frequencyTransformer
	profiler               *profiler
	logger                 *zap.Logger
}

func NewProcessor(
	delayBetweenProcessing time.Duration,
	bassCutoff float64,
	midCutoff float64,
	logger *zap.Logger,
) (*Processor, error) {
	if delayBetweenProcessing == 0 {
		delayBetweenProcessing = 10 * time.Millisecond
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Processor{
		delayBetweenProcessing: delayBetweenProcessing,
		logger:                 logger,
		profiler:               newProfiler(bassCutoff, midCutoff),
		transformer:            newTransformer(),
	}, nil
}

func (p *Processor) ensureDefaults() {
	if p.delayBetweenProcessing <= 0 {
		p.delayBetweenProcessing = 10 * time.Millisecond
	}
	if p.logger == nil {
		p.logger = zap.NewNop()
	}
	if p.profiler == nil {
		p.profiler = newProfiler(0, 0)
	}
	if p.transformer == nil {
		p.transformer = newTransformer()
	}
}

func (p *Processor) ProcessResults(
	bufferChannel chan sampling.Payload,
	resultChannel chan Result,
	ctx context.Context,
) {
	if p.delayBetweenProcessing <= 0 {
		p.delayBetweenProcessing = 10 * time.Millisecond
	}
	if p.logger == nil {
		p.logger = zap.NewNop()
	}
	for {
		select {
		case <-ctx.Done():
			p.logger.Debug("Processor stopped", zap.Error(ctx.Err()))
			return
		case record := <-bufferChannel:
			frequencies := p.transformer.Transform(record.Samples)
			profile := p.profiler.GetProfile(record.BinSize, frequencies)
			resultChannel <- Result{
				Magnitudes:       frequencies,
				SamplingDuration: record.SamplingDuration,
				SignalStrength:   float64(slices.Max(record.Samples)),
				Profile:          profile,
			}
		}
		time.Sleep(p.delayBetweenProcessing)
	}
}
