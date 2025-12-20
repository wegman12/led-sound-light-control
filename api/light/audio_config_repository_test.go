package light

import (
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"
)

func setupTestRepo(t *testing.T) (*AudioConfigRepository, string, func()) {
	t.Helper()

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "audio-config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create logger
	logger := zap.NewNop()

	// Create repository
	repo, err := NewAudioConfigRepository(tempDir, logger)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create repository: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return repo, tempDir, cleanup
}

func TestNewAudioConfigRepository(t *testing.T) {
	repo, tempDir, cleanup := setupTestRepo(t)
	defer cleanup()

	if repo == nil {
		t.Fatal("NewAudioConfigRepository returned nil")
	}

	// Verify default config was created
	defaultPath := filepath.Join(tempDir, "default.json")
	if _, err := os.Stat(defaultPath); os.IsNotExist(err) {
		t.Error("Default config file was not created")
	}

	// Verify active pointer was created
	activePath := filepath.Join(tempDir, "_active.json")
	if _, err := os.Stat(activePath); os.IsNotExist(err) {
		t.Error("Active pointer file was not created")
	}
}

func TestAudioConfigRepository_List(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	// List should return at least the default config
	response, err := repo.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(response.Configs) == 0 {
		t.Error("List returned no configs, expected at least default")
	}

	// Find default config
	found := false
	for _, c := range response.Configs {
		if c.Name == DefaultConfigName {
			found = true
			break
		}
	}
	if !found {
		t.Error("Default config not found in list")
	}

	// Active should be default
	if response.ActiveConfig != DefaultConfigName {
		t.Errorf("Expected active config to be '%s', got '%s'", DefaultConfigName, response.ActiveConfig)
	}
}

func TestAudioConfigRepository_Create(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create a new config
	config := &SavedAudioConfig{
		Name:        "test-config",
		DisplayName: "Test Config",
		Description: "A test configuration",
		Config:      getDefaultAudioTuningConfig(),
	}

	err := repo.Create(config)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify it can be retrieved
	retrieved, err := repo.Get("test-config")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved.DisplayName != "Test Config" {
		t.Errorf("Expected DisplayName 'Test Config', got '%s'", retrieved.DisplayName)
	}

	// Verify timestamps were set
	if retrieved.CreatedAt.IsZero() {
		t.Error("CreatedAt was not set")
	}
	if retrieved.UpdatedAt.IsZero() {
		t.Error("UpdatedAt was not set")
	}
}

func TestAudioConfigRepository_CreateDuplicate(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	config := &SavedAudioConfig{
		Name:        "test-config",
		DisplayName: "Test Config",
		Config:      getDefaultAudioTuningConfig(),
	}

	// First create should succeed
	if err := repo.Create(config); err != nil {
		t.Fatalf("First create failed: %v", err)
	}

	// Second create should fail
	err := repo.Create(config)
	if err == nil {
		t.Error("Expected error when creating duplicate config")
	}
}

func TestAudioConfigRepository_Update(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create a config
	config := &SavedAudioConfig{
		Name:        "test-config",
		DisplayName: "Original Name",
		Description: "Original description",
		Config:      getDefaultAudioTuningConfig(),
	}
	if err := repo.Create(config); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Update the config
	config.DisplayName = "Updated Name"
	config.Description = "Updated description"
	if err := repo.Update("test-config", config); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify the update
	retrieved, err := repo.Get("test-config")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved.DisplayName != "Updated Name" {
		t.Errorf("Expected DisplayName 'Updated Name', got '%s'", retrieved.DisplayName)
	}

	// CreatedAt should be preserved
	if retrieved.CreatedAt.IsZero() {
		t.Error("CreatedAt should be preserved after update")
	}
}

func TestAudioConfigRepository_UpdateNonExistent(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	config := &SavedAudioConfig{
		Name:        "nonexistent",
		DisplayName: "Test",
		Config:      getDefaultAudioTuningConfig(),
	}

	err := repo.Update("nonexistent", config)
	if err == nil {
		t.Error("Expected error when updating non-existent config")
	}
}

func TestAudioConfigRepository_Delete(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create a config
	config := &SavedAudioConfig{
		Name:        "test-config",
		DisplayName: "Test Config",
		Config:      getDefaultAudioTuningConfig(),
	}
	if err := repo.Create(config); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Delete the config
	if err := repo.Delete("test-config"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify it's gone
	_, err := repo.Get("test-config")
	if err == nil {
		t.Error("Expected error when getting deleted config")
	}
}

func TestAudioConfigRepository_DeleteDefault(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	// Should not be able to delete default config
	err := repo.Delete(DefaultConfigName)
	if err == nil {
		t.Error("Expected error when deleting default config")
	}
}

func TestAudioConfigRepository_DeleteActiveConfig(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create and set active
	config := &SavedAudioConfig{
		Name:        "test-config",
		DisplayName: "Test Config",
		Config:      getDefaultAudioTuningConfig(),
	}
	if err := repo.Create(config); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if err := repo.SetActive("test-config"); err != nil {
		t.Fatalf("SetActive failed: %v", err)
	}

	// Should not be able to delete active config
	err := repo.Delete("test-config")
	if err == nil {
		t.Error("Expected error when deleting active config")
	}
}

func TestAudioConfigRepository_GetActive(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	// Initially should return default
	active, err := repo.GetActive()
	if err != nil {
		t.Fatalf("GetActive failed: %v", err)
	}

	if active.Name != DefaultConfigName {
		t.Errorf("Expected active config to be '%s', got '%s'", DefaultConfigName, active.Name)
	}
}

func TestAudioConfigRepository_SetActive(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create a new config
	config := &SavedAudioConfig{
		Name:        "test-config",
		DisplayName: "Test Config",
		Config:      getDefaultAudioTuningConfig(),
	}
	if err := repo.Create(config); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Set it as active
	if err := repo.SetActive("test-config"); err != nil {
		t.Fatalf("SetActive failed: %v", err)
	}

	// Verify it's now active
	active, err := repo.GetActive()
	if err != nil {
		t.Fatalf("GetActive failed: %v", err)
	}

	if active.Name != "test-config" {
		t.Errorf("Expected active config to be 'test-config', got '%s'", active.Name)
	}

	// Also verify via GetActiveName
	activeName, err := repo.GetActiveName()
	if err != nil {
		t.Fatalf("GetActiveName failed: %v", err)
	}
	if activeName != "test-config" {
		t.Errorf("Expected active name to be 'test-config', got '%s'", activeName)
	}
}

func TestAudioConfigRepository_SetActiveNonExistent(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	err := repo.SetActive("nonexistent")
	if err == nil {
		t.Error("Expected error when setting non-existent config as active")
	}
}

func TestValidateName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid simple", "test", false},
		{"valid with hyphen", "test-config", false},
		{"valid with underscore", "test_config", false},
		{"valid with numbers", "config123", false},
		{"valid mixed", "my-config_v2", false},
		{"empty", "", true},
		{"too long", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", true},
		{"starts with underscore", "_reserved", true},
		{"contains space", "test config", true},
		{"contains special char", "test@config", true},
		{"starts with number", "123test", false}, // allowed - e.g., "2024-config"
		{"contains dot", "test.config", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestAudioConfigRepository_InvalidTuningConfig(t *testing.T) {
	repo, _, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create config with invalid tuning parameters
	config := &SavedAudioConfig{
		Name:        "invalid-config",
		DisplayName: "Invalid Config",
		Config: AudioTuningConfig{
			BassCutoff:    100,
			MidHighCutoff: 50, // Invalid: should be > BassCutoff
			TrebleCutoff:  2000,
			Bass:          getDefaultAudioTuningConfig().Bass,
			MidLow:        getDefaultAudioTuningConfig().MidLow,
			MidHigh:       getDefaultAudioTuningConfig().MidHigh,
			Treble:        getDefaultAudioTuningConfig().Treble,
		},
	}

	err := repo.Create(config)
	if err == nil {
		t.Error("Expected error when creating config with invalid tuning parameters")
	}
}

func TestAudioConfigRepository_Persistence(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "audio-config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	logger := zap.NewNop()

	// Create first repository instance
	repo1, err := NewAudioConfigRepository(tempDir, logger)
	if err != nil {
		t.Fatalf("Failed to create first repo: %v", err)
	}

	// Create and set active a config
	config := &SavedAudioConfig{
		Name:        "persistent-config",
		DisplayName: "Persistent Config",
		Config:      getDefaultAudioTuningConfig(),
	}
	if err := repo1.Create(config); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if err := repo1.SetActive("persistent-config"); err != nil {
		t.Fatalf("SetActive failed: %v", err)
	}

	// Create second repository instance (simulating restart)
	repo2, err := NewAudioConfigRepository(tempDir, logger)
	if err != nil {
		t.Fatalf("Failed to create second repo: %v", err)
	}

	// Verify the config and active setting persisted
	active, err := repo2.GetActive()
	if err != nil {
		t.Fatalf("GetActive failed: %v", err)
	}

	if active.Name != "persistent-config" {
		t.Errorf("Expected persisted active config 'persistent-config', got '%s'", active.Name)
	}
}

func TestAudioConfigRepository_FallbackOnMissingActive(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "audio-config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	logger := zap.NewNop()

	// Create repository
	repo, err := NewAudioConfigRepository(tempDir, logger)
	if err != nil {
		t.Fatalf("Failed to create repo: %v", err)
	}

	// Create and set active a config
	config := &SavedAudioConfig{
		Name:        "temp-config",
		DisplayName: "Temp Config",
		Config:      getDefaultAudioTuningConfig(),
	}
	if err := repo.Create(config); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if err := repo.SetActive("temp-config"); err != nil {
		t.Fatalf("SetActive failed: %v", err)
	}

	// Manually delete the config file (simulating corruption)
	os.Remove(filepath.Join(tempDir, "temp-config.json"))

	// GetActive should fall back to default
	active, err := repo.GetActive()
	if err != nil {
		t.Fatalf("GetActive failed: %v", err)
	}

	if active.Name != DefaultConfigName {
		t.Errorf("Expected fallback to default config, got '%s'", active.Name)
	}
}
