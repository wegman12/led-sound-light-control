package light

import (
	"context"
	"net/http"
	"sync"

	"github.com/wegman12/led-sound-light-control/light/behavior"
	"go.uber.org/zap"
)

func RegisterRoutes(mux *http.ServeMux, ctx context.Context, wg *sync.WaitGroup, audioProvider behavior.AudioProvider, logger *zap.Logger) *Controller {
	logger.Debug("Registering light routes")

	controller := NewController(ctx, wg, audioProvider, logger)
	h := newHandler(controller, logger)
	mux.HandleFunc("POST /api/lights/behavior/register", h.handleRegisterBehavior)
	mux.HandleFunc("POST /api/lights/on", h.handleTurnOn)
	mux.HandleFunc("POST /api/lights/off", h.handleTurnOff)

	logger.Info("Light routes registered")

	return controller
}
