package light

import (
	"context"
	"net/http"
	"sync"
	"time"
)

func RegisterRoutes(mux *http.ServeMux, ctx context.Context, wg *sync.WaitGroup) {
	h := newHandler(ctx)
	mux.HandleFunc("POST /api/lights/behavior/register", h.handleRegisterBehavior)
	mux.HandleFunc("POST /api/lights/on", h.handleTurnOn)
	mux.HandleFunc("POST /api/lights/off", h.handleTurnOff)

	wg.Add(1)

	// Cleanup handler resources
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				// Masking panics from go-bbhw library
			}
		}()
		for {
			select {
			case <-ctx.Done():
				h.manager.Close()
			default:
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()
}
