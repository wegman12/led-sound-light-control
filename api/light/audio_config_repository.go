package light

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

const (
	// DefaultConfigName is the name of the built-in default configuration
	DefaultConfigName = "default"
	// ActiveConfigFileName is the name of the file that stores the active config pointer
	ActiveConfigFileName = "_active.json"
	// ConfigFileExtension is the extension for config files
	ConfigFileExtension = ".json"
)

// SavedAudioConfig represents a saved audio configuration with metadata
type SavedAudioConfig struct {
	Name        string            `json:"name"`
	DisplayName string            `json:"display_name"`
	Description string            `json:"description"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	Config      AudioTuningConfig `json:"config"`
}

// ActiveConfigPointer stores the name of the currently active configuration
type ActiveConfigPointer struct {
	ConfigName string    `json:"config_name"`
	SetAt      time.Time `json:"set_at"`
}

// ConfigListResponse is the response for listing configurations
type ConfigListResponse struct {
	Configs      []SavedAudioConfigSummary `json:"configs"`
	ActiveConfig string                    `json:"active_config"`
}

// SavedAudioConfigSummary is a summary of a saved config (without the full config data)
type SavedAudioConfigSummary struct {
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// AudioConfigRepository manages multiple audio configurations on disk
type AudioConfigRepository struct {
	basePath string
	mu       sync.RWMutex
	logger   *zap.Logger
}

// NewAudioConfigRepository creates a new audio configuration repository
func NewAudioConfigRepository(basePath string, logger *zap.Logger) (*AudioConfigRepository, error) {
	repo := &AudioConfigRepository{
		basePath: basePath,
		logger:   logger,
	}

	// Ensure the directory exists
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Ensure default configuration exists
	if err := repo.ensureDefaultExists(); err != nil {
		return nil, fmt.Errorf("failed to ensure default config: %w", err)
	}

	// Ensure active config pointer exists
	if err := repo.ensureActiveExists(); err != nil {
		return nil, fmt.Errorf("failed to ensure active config pointer: %w", err)
	}

	logger.Info("Audio config repository initialized", zap.String("path", basePath))
	return repo, nil
}

// ensureDefaultExists creates the default configuration if it doesn't exist
func (r *AudioConfigRepository) ensureDefaultExists() error {
	defaultPath := r.configPath(DefaultConfigName)
	if _, err := os.Stat(defaultPath); os.IsNotExist(err) {
		defaultConfig := SavedAudioConfig{
			Name:        DefaultConfigName,
			DisplayName: "Default",
			Description: "Built-in default configuration",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Config:      getDefaultAudioTuningConfig(),
		}
		if err := r.writeConfig(defaultPath, &defaultConfig); err != nil {
			return fmt.Errorf("failed to create default config: %w", err)
		}
		r.logger.Info("Created default configuration")
	}
	return nil
}

// ensureActiveExists creates the active config pointer if it doesn't exist
func (r *AudioConfigRepository) ensureActiveExists() error {
	activePath := filepath.Join(r.basePath, ActiveConfigFileName)
	if _, err := os.Stat(activePath); os.IsNotExist(err) {
		// Check if we need to migrate from old single-file config
		if err := r.migrateFromLegacy(); err != nil {
			r.logger.Warn("Failed to migrate legacy config, using default", zap.Error(err))
		}

		// Set default as active if no active pointer exists
		if _, err := os.Stat(activePath); os.IsNotExist(err) {
			pointer := ActiveConfigPointer{
				ConfigName: DefaultConfigName,
				SetAt:      time.Now(),
			}
			if err := r.writeActivePointer(&pointer); err != nil {
				return fmt.Errorf("failed to create active pointer: %w", err)
			}
			r.logger.Info("Set default configuration as active")
		}
	}
	return nil
}

// migrateFromLegacy migrates the old single-file configuration to the new repository
func (r *AudioConfigRepository) migrateFromLegacy() error {
	legacyPath := "/etc/led-sound-light-control/audio_tuning_config.json"
	if _, err := os.Stat(legacyPath); os.IsNotExist(err) {
		return nil // No legacy config to migrate
	}

	r.logger.Info("Migrating legacy audio tuning config", zap.String("path", legacyPath))

	// Read legacy config
	data, err := os.ReadFile(legacyPath)
	if err != nil {
		return fmt.Errorf("failed to read legacy config: %w", err)
	}

	var legacyConfig AudioTuningConfig
	if err := json.Unmarshal(data, &legacyConfig); err != nil {
		return fmt.Errorf("failed to parse legacy config: %w", err)
	}

	// Create migrated config
	migratedConfig := SavedAudioConfig{
		Name:        "migrated",
		DisplayName: "Migrated",
		Description: "Configuration migrated from legacy single-file format",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Config:      legacyConfig,
	}

	// Save migrated config
	migratedPath := r.configPath("migrated")
	if err := r.writeConfig(migratedPath, &migratedConfig); err != nil {
		return fmt.Errorf("failed to save migrated config: %w", err)
	}

	// Set migrated as active
	pointer := ActiveConfigPointer{
		ConfigName: "migrated",
		SetAt:      time.Now(),
	}
	if err := r.writeActivePointer(&pointer); err != nil {
		return fmt.Errorf("failed to set migrated as active: %w", err)
	}

	r.logger.Info("Successfully migrated legacy configuration")
	return nil
}

// configPath returns the file path for a configuration name
func (r *AudioConfigRepository) configPath(name string) string {
	return filepath.Join(r.basePath, name+ConfigFileExtension)
}

// writeConfig writes a configuration to disk
func (r *AudioConfigRepository) writeConfig(path string, config *SavedAudioConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	return nil
}

// readConfig reads a configuration from disk
func (r *AudioConfigRepository) readConfig(path string) (*SavedAudioConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var config SavedAudioConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	return &config, nil
}

// writeActivePointer writes the active config pointer to disk
func (r *AudioConfigRepository) writeActivePointer(pointer *ActiveConfigPointer) error {
	path := filepath.Join(r.basePath, ActiveConfigFileName)
	data, err := json.MarshalIndent(pointer, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal active pointer: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write active pointer: %w", err)
	}
	return nil
}

// readActivePointer reads the active config pointer from disk
func (r *AudioConfigRepository) readActivePointer() (*ActiveConfigPointer, error) {
	path := filepath.Join(r.basePath, ActiveConfigFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var pointer ActiveConfigPointer
	if err := json.Unmarshal(data, &pointer); err != nil {
		return nil, fmt.Errorf("failed to parse active pointer: %w", err)
	}
	return &pointer, nil
}

// ValidateName validates a configuration name
func ValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	if len(name) > 50 {
		return fmt.Errorf("name must be 50 characters or less")
	}
	if strings.HasPrefix(name, "_") {
		return fmt.Errorf("name cannot start with underscore (reserved)")
	}
	// Allow alphanumeric, hyphens, and underscores
	validName := regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)
	if !validName.MatchString(name) {
		return fmt.Errorf("name must contain only letters, numbers, hyphens, and underscores")
	}
	return nil
}

// List returns all saved configurations
func (r *AudioConfigRepository) List() (*ConfigListResponse, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entries, err := os.ReadDir(r.basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config directory: %w", err)
	}

	var configs []SavedAudioConfigSummary
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ConfigFileExtension) {
			continue
		}
		if name == ActiveConfigFileName {
			continue
		}

		configName := strings.TrimSuffix(name, ConfigFileExtension)
		config, err := r.readConfig(filepath.Join(r.basePath, name))
		if err != nil {
			r.logger.Warn("Failed to read config", zap.String("name", configName), zap.Error(err))
			continue
		}

		configs = append(configs, SavedAudioConfigSummary{
			Name:        config.Name,
			DisplayName: config.DisplayName,
			Description: config.Description,
			CreatedAt:   config.CreatedAt,
			UpdatedAt:   config.UpdatedAt,
		})
	}

	// Get active config name
	activeConfig := DefaultConfigName
	if pointer, err := r.readActivePointer(); err == nil {
		activeConfig = pointer.ConfigName
	}

	return &ConfigListResponse{
		Configs:      configs,
		ActiveConfig: activeConfig,
	}, nil
}

// Get returns a specific configuration by name
func (r *AudioConfigRepository) Get(name string) (*SavedAudioConfig, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if err := ValidateName(name); err != nil {
		return nil, fmt.Errorf("invalid name: %w", err)
	}

	config, err := r.readConfig(r.configPath(name))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("configuration '%s' not found", name)
		}
		return nil, fmt.Errorf("failed to read configuration: %w", err)
	}

	return config, nil
}

// Create creates a new configuration
func (r *AudioConfigRepository) Create(config *SavedAudioConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := ValidateName(config.Name); err != nil {
		return fmt.Errorf("invalid name: %w", err)
	}

	// Check if config already exists
	configPath := r.configPath(config.Name)
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("configuration '%s' already exists", config.Name)
	}

	// Validate the tuning config
	if err := validateAudioTuningConfig(&config.Config); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Set timestamps
	now := time.Now()
	config.CreatedAt = now
	config.UpdatedAt = now

	// Write config
	if err := r.writeConfig(configPath, config); err != nil {
		return err
	}

	r.logger.Info("Created configuration", zap.String("name", config.Name))
	return nil
}

// Update updates an existing configuration
func (r *AudioConfigRepository) Update(name string, config *SavedAudioConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := ValidateName(name); err != nil {
		return fmt.Errorf("invalid name: %w", err)
	}

	// Check if config exists
	configPath := r.configPath(name)
	existingConfig, err := r.readConfig(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("configuration '%s' not found", name)
		}
		return fmt.Errorf("failed to read existing configuration: %w", err)
	}

	// Validate the tuning config
	if err := validateAudioTuningConfig(&config.Config); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Preserve original name and created timestamp
	config.Name = name
	config.CreatedAt = existingConfig.CreatedAt
	config.UpdatedAt = time.Now()

	// Write config
	if err := r.writeConfig(configPath, config); err != nil {
		return err
	}

	r.logger.Info("Updated configuration", zap.String("name", name))
	return nil
}

// Delete deletes a configuration
func (r *AudioConfigRepository) Delete(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := ValidateName(name); err != nil {
		return fmt.Errorf("invalid name: %w", err)
	}

	// Prevent deletion of default config
	if name == DefaultConfigName {
		return fmt.Errorf("cannot delete the default configuration")
	}

	// Check if this is the active config
	pointer, err := r.readActivePointer()
	if err == nil && pointer.ConfigName == name {
		return fmt.Errorf("cannot delete the active configuration; set a different configuration as active first")
	}

	// Check if config exists
	configPath := r.configPath(name)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("configuration '%s' not found", name)
	}

	// Delete config
	if err := os.Remove(configPath); err != nil {
		return fmt.Errorf("failed to delete configuration: %w", err)
	}

	r.logger.Info("Deleted configuration", zap.String("name", name))
	return nil
}

// GetActive returns the currently active configuration
func (r *AudioConfigRepository) GetActive() (*SavedAudioConfig, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	pointer, err := r.readActivePointer()
	if err != nil {
		if os.IsNotExist(err) {
			// Return default if no active pointer
			return r.readConfig(r.configPath(DefaultConfigName))
		}
		return nil, fmt.Errorf("failed to read active pointer: %w", err)
	}

	config, err := r.readConfig(r.configPath(pointer.ConfigName))
	if err != nil {
		if os.IsNotExist(err) {
			// Active config was deleted, fall back to default
			r.logger.Warn("Active config not found, falling back to default",
				zap.String("active", pointer.ConfigName))
			return r.readConfig(r.configPath(DefaultConfigName))
		}
		return nil, fmt.Errorf("failed to read active configuration: %w", err)
	}

	return config, nil
}

// SetActive sets the active configuration
func (r *AudioConfigRepository) SetActive(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := ValidateName(name); err != nil {
		return fmt.Errorf("invalid name: %w", err)
	}

	// Check if config exists
	configPath := r.configPath(name)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("configuration '%s' not found", name)
	}

	// Update active pointer
	pointer := ActiveConfigPointer{
		ConfigName: name,
		SetAt:      time.Now(),
	}
	if err := r.writeActivePointer(&pointer); err != nil {
		return err
	}

	r.logger.Info("Set active configuration", zap.String("name", name))
	return nil
}

// GetActiveName returns just the name of the active configuration
func (r *AudioConfigRepository) GetActiveName() (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	pointer, err := r.readActivePointer()
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfigName, nil
		}
		return "", fmt.Errorf("failed to read active pointer: %w", err)
	}
	return pointer.ConfigName, nil
}

// validateAudioTuningConfig validates an AudioTuningConfig
// This is a package-level function to avoid duplication with AudioTuningConfigManager
func validateAudioTuningConfig(config *AudioTuningConfig) error {
	// Validate cutoff frequencies
	if config.BassCutoff <= 0 {
		return fmt.Errorf("bass_cutoff must be positive, got %f", config.BassCutoff)
	}
	if config.MidHighCutoff <= config.BassCutoff {
		return fmt.Errorf("mid_high_cutoff (%f) must be greater than bass_cutoff (%f)",
			config.MidHighCutoff, config.BassCutoff)
	}
	if config.TrebleCutoff <= config.MidHighCutoff {
		return fmt.Errorf("treble_cutoff (%f) must be greater than mid_high_cutoff (%f)",
			config.TrebleCutoff, config.MidHighCutoff)
	}

	// Validate each band
	bands := map[string]AudioTuningBandConfig{
		"bass":     config.Bass,
		"mid_low":  config.MidLow,
		"mid_high": config.MidHigh,
		"treble":   config.Treble,
	}

	for name, band := range bands {
		if err := validateBandConfig(name, &band); err != nil {
			return err
		}
	}

	return nil
}
