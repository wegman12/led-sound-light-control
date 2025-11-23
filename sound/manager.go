package sound

import (
	"context"
	"fmt"
	"sync"
)

type Manager struct {
	ResultsChannel chan FrequencyResult
	cancel         context.CancelFunc
}

func (m *Manager) Start(ctx context.Context) {
	if m.ResultsChannel == nil {
		m.ResultsChannel = make(chan FrequencyResult, BufferSize*10)
	}
	if m.cancel != nil {
		m.cancel()
	}
	ctx, m.cancel = context.WithCancel(ctx)
	defer func() {
		if m.cancel != nil {
			m.cancel()
			m.cancel = nil
		}
	}()

	bufferChannel := make(chan bufferPayload, BufferSize)

	wg := &sync.WaitGroup{}

	r := reader{
		bufferChannel: bufferChannel,
		usePRU:        false,     // PRU mode (requires OCP access)
		useIIOFast:    true,      // Use high-speed IIO with decimation (200kHz -> 48kHz)
	}
	p := processor{
		bufferChannel: bufferChannel,
		resultChannel: m.ResultsChannel,
	}

	p.Start(ctx, wg)
	r.start(ctx, wg)
	wg.Wait()
	fmt.Println("Manager stopped")
}

func (m *Manager) Stop() {
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
}
