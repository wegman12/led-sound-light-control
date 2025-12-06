package light

import (
	"encoding/json"
	"io"
	"net/http"

	"go.uber.org/zap"
)

type handler struct {
	controller *Controller
	logger     *zap.Logger
}

func newHandler(controller *Controller, logger *zap.Logger) *handler {
	return &handler{
		controller: controller,
		logger:     logger,
	}
}

func (h *handler) handleRegisterBehavior(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("Received register behavior request")

	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("Error reading request body", zap.Error(err))
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close() // Important to close the body

	var data ManagerConfig
	err = json.Unmarshal(body, &data)
	if err != nil {
		h.logger.Warn("Error unmarshaling JSON", zap.Error(err))
		http.Error(w, "Error unmarshaling JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	h.logger.Info("Registering new behavior", zap.Int("num_behaviors", len(data.Behaviors)))

	h.controller.SendEvent(ChangeBehaviorEvent{
		Config: data,
	})

	w.WriteHeader(http.StatusAccepted)
}

func (h *handler) handleTurnOn(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Received turn on request")
	h.controller.SendEvent(StartEvent{})
	w.WriteHeader(http.StatusAccepted)
}

func (h *handler) handleTurnOff(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Received turn off request")
	h.controller.SendEvent(StopEvent{})
	w.WriteHeader(http.StatusAccepted)
}
