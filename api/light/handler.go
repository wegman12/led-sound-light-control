package light

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
)

type handler struct {
	ctx             context.Context
	manager         *Manager
	currentBehavior *ManagerConfig
}

func newHandler(ctx context.Context) *handler {
	return &handler{
		ctx: ctx,
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

	// Initialize manager on first use
	if h.manager == nil {
		h.manager, err = NewManager(data)
		if err != nil {
			http.Error(w, "Error creating manager: "+err.Error(), http.StatusBadRequest)
			return
		}
	} else {
		// Update behaviors without recreating hardware
		err = h.manager.UpdateBehaviors(data)
		if err != nil {
			http.Error(w, "Error updating behaviors: "+err.Error(), http.StatusBadRequest)
			return
		}
	}

	h.currentBehavior = &data
	w.WriteHeader(http.StatusAccepted)
}

func (h *handler) handleTurnOn(w http.ResponseWriter, r *http.Request) {
	if h.manager == nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("you must register a behavior first"))
		return
	}
	h.manager.Start(h.ctx)
}

func (h *handler) handleTurnOff(w http.ResponseWriter, r *http.Request) {
	if h.manager == nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("you must register a behavior first"))
		return
	}
	h.manager.Stop()
}
