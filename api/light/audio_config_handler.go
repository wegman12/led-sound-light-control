package light

import (
	"encoding/json"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

// AudioConfigHandler manages HTTP endpoints for audio configuration management
type AudioConfigHandler struct {
	repo   *AudioConfigRepository
	logger *zap.Logger
}

// NewAudioConfigHandler creates a new audio configuration handler
func NewAudioConfigHandler(repo *AudioConfigRepository, logger *zap.Logger) *AudioConfigHandler {
	return &AudioConfigHandler{
		repo:   repo,
		logger: logger,
	}
}

// HandleListConfigs handles GET /api/audio/configs
func (h *AudioConfigHandler) HandleListConfigs(w http.ResponseWriter, r *http.Request) {
	response, err := h.repo.List()
	if err != nil {
		h.logger.Error("Failed to list configs", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "Failed to list configurations")
		return
	}

	h.writeJSON(w, http.StatusOK, response)
}

// HandleGetConfig handles GET /api/audio/configs/{name}
func (h *AudioConfigHandler) HandleGetConfig(w http.ResponseWriter, r *http.Request) {
	name := h.extractConfigName(r)
	if name == "" {
		h.writeError(w, http.StatusBadRequest, "Configuration name is required")
		return
	}

	config, err := h.repo.Get(name)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.writeError(w, http.StatusNotFound, err.Error())
			return
		}
		h.logger.Error("Failed to get config", zap.String("name", name), zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "Failed to get configuration")
		return
	}

	h.writeJSON(w, http.StatusOK, config)
}

// CreateConfigRequest is the request body for creating a new configuration
type CreateConfigRequest struct {
	Name        string            `json:"name"`
	DisplayName string            `json:"display_name"`
	Description string            `json:"description"`
	Config      AudioTuningConfig `json:"config"`
}

// HandleCreateConfig handles POST /api/audio/configs
func (h *AudioConfigHandler) HandleCreateConfig(w http.ResponseWriter, r *http.Request) {
	var req CreateConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	config := &SavedAudioConfig{
		Name:        req.Name,
		DisplayName: req.DisplayName,
		Description: req.Description,
		Config:      req.Config,
	}

	if err := h.repo.Create(config); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			h.writeError(w, http.StatusConflict, err.Error())
			return
		}
		if strings.Contains(err.Error(), "invalid") {
			h.writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		h.logger.Error("Failed to create config", zap.String("name", req.Name), zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "Failed to create configuration")
		return
	}

	// Fetch the created config to return it with timestamps
	created, err := h.repo.Get(req.Name)
	if err != nil {
		h.logger.Error("Failed to fetch created config", zap.String("name", req.Name), zap.Error(err))
		h.writeJSON(w, http.StatusCreated, map[string]string{
			"message": "Configuration created successfully",
			"name":    req.Name,
		})
		return
	}

	h.logger.Info("Created configuration", zap.String("name", req.Name))
	h.writeJSON(w, http.StatusCreated, created)
}

// UpdateConfigRequest is the request body for updating a configuration
type UpdateConfigRequest struct {
	DisplayName string            `json:"display_name"`
	Description string            `json:"description"`
	Config      AudioTuningConfig `json:"config"`
}

// HandleUpdateConfig handles PUT /api/audio/configs/{name}
func (h *AudioConfigHandler) HandleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	name := h.extractConfigName(r)
	if name == "" {
		h.writeError(w, http.StatusBadRequest, "Configuration name is required")
		return
	}

	var req UpdateConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	config := &SavedAudioConfig{
		Name:        name,
		DisplayName: req.DisplayName,
		Description: req.Description,
		Config:      req.Config,
	}

	if err := h.repo.Update(name, config); err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.writeError(w, http.StatusNotFound, err.Error())
			return
		}
		if strings.Contains(err.Error(), "invalid") {
			h.writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		h.logger.Error("Failed to update config", zap.String("name", name), zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "Failed to update configuration")
		return
	}

	// Fetch the updated config to return it
	updated, err := h.repo.Get(name)
	if err != nil {
		h.logger.Error("Failed to fetch updated config", zap.String("name", name), zap.Error(err))
		h.writeJSON(w, http.StatusOK, map[string]string{
			"message": "Configuration updated successfully",
			"name":    name,
		})
		return
	}

	h.logger.Info("Updated configuration", zap.String("name", name))
	h.writeJSON(w, http.StatusOK, updated)
}

// HandleDeleteConfig handles DELETE /api/audio/configs/{name}
func (h *AudioConfigHandler) HandleDeleteConfig(w http.ResponseWriter, r *http.Request) {
	name := h.extractConfigName(r)
	if name == "" {
		h.writeError(w, http.StatusBadRequest, "Configuration name is required")
		return
	}

	if err := h.repo.Delete(name); err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.writeError(w, http.StatusNotFound, err.Error())
			return
		}
		if strings.Contains(err.Error(), "cannot delete") {
			h.writeError(w, http.StatusConflict, err.Error())
			return
		}
		h.logger.Error("Failed to delete config", zap.String("name", name), zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "Failed to delete configuration")
		return
	}

	h.logger.Info("Deleted configuration", zap.String("name", name))
	h.writeJSON(w, http.StatusOK, map[string]string{
		"message": "Configuration deleted successfully",
		"name":    name,
	})
}

// HandleGetActiveConfig handles GET /api/audio/configs/active
func (h *AudioConfigHandler) HandleGetActiveConfig(w http.ResponseWriter, r *http.Request) {
	config, err := h.repo.GetActive()
	if err != nil {
		h.logger.Error("Failed to get active config", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "Failed to get active configuration")
		return
	}

	h.writeJSON(w, http.StatusOK, config)
}

// SetActiveConfigRequest is the request body for setting the active configuration
type SetActiveConfigRequest struct {
	ConfigName string `json:"config_name"`
}

// HandleSetActiveConfig handles PUT /api/audio/configs/active
func (h *AudioConfigHandler) HandleSetActiveConfig(w http.ResponseWriter, r *http.Request) {
	var req SetActiveConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if req.ConfigName == "" {
		h.writeError(w, http.StatusBadRequest, "config_name is required")
		return
	}

	if err := h.repo.SetActive(req.ConfigName); err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.writeError(w, http.StatusNotFound, err.Error())
			return
		}
		if strings.Contains(err.Error(), "invalid") {
			h.writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		h.logger.Error("Failed to set active config", zap.String("name", req.ConfigName), zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "Failed to set active configuration")
		return
	}

	// Fetch the active config to return it
	active, err := h.repo.GetActive()
	if err != nil {
		h.logger.Error("Failed to fetch active config", zap.Error(err))
		h.writeJSON(w, http.StatusOK, map[string]string{
			"message":     "Active configuration set successfully",
			"config_name": req.ConfigName,
		})
		return
	}

	h.logger.Info("Set active configuration", zap.String("name", req.ConfigName))
	h.writeJSON(w, http.StatusOK, active)
}

// extractConfigName extracts the config name from the URL path
// Expected format: /api/audio/configs/{name}
func (h *AudioConfigHandler) extractConfigName(r *http.Request) string {
	path := r.URL.Path
	prefix := "/api/audio/configs/"
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	name := strings.TrimPrefix(path, prefix)
	// Remove trailing slash if present
	name = strings.TrimSuffix(name, "/")
	return name
}

// writeJSON writes a JSON response
func (h *AudioConfigHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", zap.Error(err))
	}
}

// writeError writes an error response
func (h *AudioConfigHandler) writeError(w http.ResponseWriter, status int, message string) {
	h.writeJSON(w, status, map[string]string{"error": message})
}
