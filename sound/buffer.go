package sound

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type bufferPayload struct {
	bytes            [BufferSize * recordSize]byte
	samplingDuration time.Duration
}

type buffer struct {
	rawChannel    chan []byte
	bufferChannel chan bufferPayload
	sleepTime     time.Duration
}

func (b *buffer) start(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		b.populateBuffer(ctx)
	}()
}

func (b *buffer) populateBuffer(ctx context.Context) {
	defer close(b.bufferChannel)
	if b.sleepTime <= 0 {
		b.sleepTime = 1 * time.Millisecond
	}
	payload := [BufferSize * recordSize]byte{}
	current := 0
	lastFlush := time.Now()
	for {
		select {
		case <-ctx.Done():
			fmt.Println("Stopping buffer")
			return
		case raw := <-b.rawChannel:
			for i := range raw {
				payload[current] = raw[i]
				current++
				if current == len(payload) {
					duration := time.Since(lastFlush)
					b.bufferChannel <- bufferPayload{
						bytes:            payload,
						samplingDuration: duration,
					}
					current = 0
					lastFlush = time.Now()
				}
			}
		default:
			time.Sleep(b.sleepTime)
		}
	}

}
