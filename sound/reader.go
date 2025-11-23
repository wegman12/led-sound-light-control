package sound

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

const iioFile = "/dev/iio:device0"

type reader struct {
	rawChannel chan []byte
	sleepTime  time.Duration
}

func (r *reader) start(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		r.readFromIio(ctx)
	}()
}

func (r *reader) readFromIio(ctx context.Context) {
	if r.sleepTime <= 0 {
		r.sleepTime = 1 * time.Millisecond
	}
	defer close(r.rawChannel)
	iioDevice, err := os.Open(iioFile)
	if err != nil {
		fmt.Printf("Error opening iioDevice: %v\n", err)
		return
	}
	defer iioDevice.Close() // Ensure the iioDevice is closed when the function exits

	bufReader := bufio.NewReader(iioDevice)
	slot := make([]byte, BufferSize*recordSize*10)

	done := false

	for !done {
		select {
		case <-ctx.Done():
			fmt.Println("Stopping reader")
			done = true
		default:
			n, readErr := bufReader.Read(slot)
			if readErr == io.EOF {
				done = true
				break
			}
			r.rawChannel <- slot[:n]
			time.Sleep(r.sleepTime)
		}
	}
	return
}
