package server

import (
	"context"
	"net/http"
	"sync"

	"github.com/wegman12/led-sound-light-control/health"
	"github.com/wegman12/led-sound-light-control/light"
)

func RegisterRoutes(mux *http.ServeMux, ctx context.Context, wg *sync.WaitGroup) {
	health.RegisterRoutes(mux)
	light.RegisterRoutes(mux, ctx, wg)
}
