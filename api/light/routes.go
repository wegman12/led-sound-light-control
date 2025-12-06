package light

import (
	"context"
	"net/http"
	"sync"

	"go.uber.org/zap"
)

func RegisterRoutes(mux *http.ServeMux, ctx context.Context, wg *sync.WaitGroup, logger *zap.Logger) *Controller {
	logger.Debug("Registering light routes")

	controller := NewController(ctx, wg, logger)
	h := newHandler(controller, logger)
	mux.HandleFunc("POST /api/lights/behavior/register", h.handleRegisterBehavior)
	mux.HandleFunc("POST /api/lights/on", h.handleTurnOn)
	mux.HandleFunc("POST /api/lights/off", h.handleTurnOff)

	logger.Info("Light routes registered")

	return controller
}
