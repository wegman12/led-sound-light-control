package light

import (
	"context"
	"net/http"
)

func RegisterRoutes(mux *http.ServeMux, ctx context.Context) {
	h := newHandler(ctx)
	mux.HandleFunc("POST /api/lights/behavior/register", h.handleRegisterBehavior)
	mux.HandleFunc("POST /api/lights/on", h.handleTurnOn)
	mux.HandleFunc("POST /api/lights/off", h.handleTurnOff)
}
