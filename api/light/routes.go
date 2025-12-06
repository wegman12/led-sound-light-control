package light

import (
	"context"
	"net/http"
	"sync"
)

func RegisterRoutes(mux *http.ServeMux, ctx context.Context, wg *sync.WaitGroup) *Controller {
	controller := NewController(ctx, wg)
	h := newHandler(controller)
	mux.HandleFunc("POST /api/lights/behavior/register", h.handleRegisterBehavior)
	mux.HandleFunc("POST /api/lights/on", h.handleTurnOn)
	mux.HandleFunc("POST /api/lights/off", h.handleTurnOff)

	return controller
}
