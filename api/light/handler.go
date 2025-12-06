package light

import (
	"encoding/json"
	"io"
	"net/http"
)

type handler struct {
	controller *Controller
}

func newHandler(controller *Controller) *handler {
	return &handler{
		controller: controller,
	}
}

func (h *handler) handleRegisterBehavior(w http.ResponseWriter, r *http.Request) {

	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close() // Important to close the body

	var data ManagerConfig
	err = json.Unmarshal(body, &data)
	if err != nil {
		http.Error(w, "Error unmarshaling JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	h.controller.SendEvent(ChangeBehaviorEvent{
		Config: data,
	})

	w.WriteHeader(http.StatusAccepted)
}

func (h *handler) handleTurnOn(w http.ResponseWriter, r *http.Request) {
	h.controller.SendEvent(StartEvent{})
	w.WriteHeader(http.StatusAccepted)
}

func (h *handler) handleTurnOff(w http.ResponseWriter, r *http.Request) {
	h.controller.SendEvent(StopEvent{})
	w.WriteHeader(http.StatusAccepted)
}
