// Migration script to convert scaling_factor to max_magnitude in config files
// Run with: go run migrate_scaling_to_magnitude.go /etc/led-sound-light-control/audio-configs
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Old config structure
type OldBandConfig struct {
	ScalingFactor  float64 `json:"scaling_factor"`
	NoiseThreshold float64 `json:"noise_threshold"`
	MinPowerValue  float64 `json:"min_power_value"`
	MaxPowerValue  float64 `json:"max_power_value"`
	Smoothing      float64 `json:"smoothing"`
}

type OldTuningConfig struct {
	BassCutoff    float64       `json:"bass_cutoff"`
	MidHighCutoff float64       `json:"mid_high_cutoff"`
	TrebleCutoff  float64       `json:"treble_cutoff"`
	Bass          OldBandConfig `json:"bass"`
	MidLow        OldBandConfig `json:"mid_low"`
	MidHigh       OldBandConfig `json:"mid_high"`
	Treble        OldBandConfig `json:"treble"`
}

type OldSavedConfig struct {
	Name        string          `json:"name"`
	DisplayName string          `json:"display_name"`
	Description string          `json:"description"`
	CreatedAt   string          `json:"created_at"`
	UpdatedAt   string          `json:"updated_at"`
	Config      OldTuningConfig `json:"config"`
}

// New config structure
type NewBandConfig struct {
	MaxMagnitude   float64 `json:"max_magnitude"`
	NoiseThreshold float64 `json:"noise_threshold"`
	MinPowerValue  float64 `json:"min_power_value"`
	MaxPowerValue  float64 `json:"max_power_value"`
	Smoothing      float64 `json:"smoothing"`
}

type NewTuningConfig struct {
	BassCutoff    float64       `json:"bass_cutoff"`
	MidHighCutoff float64       `json:"mid_high_cutoff"`
	TrebleCutoff  float64       `json:"treble_cutoff"`
	Bass          NewBandConfig `json:"bass"`
	MidLow        NewBandConfig `json:"mid_low"`
	MidHigh       NewBandConfig `json:"mid_high"`
	Treble        NewBandConfig `json:"treble"`
}

type NewSavedConfig struct {
	Name        string          `json:"name"`
	DisplayName string          `json:"display_name"`
	Description string          `json:"description"`
	CreatedAt   string          `json:"created_at"`
	UpdatedAt   string          `json:"updated_at"`
	Config      NewTuningConfig `json:"config"`
}

func convertBand(old OldBandConfig) NewBandConfig {
	maxMagnitude := 1.0 / old.ScalingFactor
	if old.ScalingFactor == 0 {
		maxMagnitude = 1000000 // Default fallback
	}
	return NewBandConfig{
		MaxMagnitude:   maxMagnitude,
		NoiseThreshold: old.NoiseThreshold,
		MinPowerValue:  old.MinPowerValue,
		MaxPowerValue:  old.MaxPowerValue,
		Smoothing:      old.Smoothing,
	}
}

func migrateConfig(oldConfig OldSavedConfig) NewSavedConfig {
	return NewSavedConfig{
		Name:        oldConfig.Name,
		DisplayName: oldConfig.DisplayName,
		Description: oldConfig.Description,
		CreatedAt:   oldConfig.CreatedAt,
		UpdatedAt:   oldConfig.UpdatedAt,
		Config: NewTuningConfig{
			BassCutoff:    oldConfig.Config.BassCutoff,
			MidHighCutoff: oldConfig.Config.MidHighCutoff,
			TrebleCutoff:  oldConfig.Config.TrebleCutoff,
			Bass:          convertBand(oldConfig.Config.Bass),
			MidLow:        convertBand(oldConfig.Config.MidLow),
			MidHigh:       convertBand(oldConfig.Config.MidHigh),
			Treble:        convertBand(oldConfig.Config.Treble),
		},
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: migrate_scaling_to_magnitude <config-directory>")
		fmt.Println("Example: migrate_scaling_to_magnitude /etc/led-sound-light-control/audio-configs")
		os.Exit(1)
	}

	configDir := os.Args[1]

	// Find all JSON files except _active.json
	entries, err := os.ReadDir(configDir)
	if err != nil {
		fmt.Printf("Error reading directory %s: %v\n", configDir, err)
		os.Exit(1)
	}

	migratedCount := 0
	skippedCount := 0

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()
		if !strings.HasSuffix(filename, ".json") || filename == "_active.json" {
			continue
		}

		filepath := filepath.Join(configDir, filename)

		// Read file
		data, err := os.ReadFile(filepath)
		if err != nil {
			fmt.Printf("Error reading %s: %v\n", filepath, err)
			continue
		}

		// Check if already migrated (has max_magnitude instead of scaling_factor)
		if strings.Contains(string(data), "max_magnitude") {
			fmt.Printf("Skipping %s (already migrated)\n", filename)
			skippedCount++
			continue
		}

		// Parse old config
		var oldConfig OldSavedConfig
		if err := json.Unmarshal(data, &oldConfig); err != nil {
			fmt.Printf("Error parsing %s: %v\n", filepath, err)
			continue
		}

		// Migrate
		newConfig := migrateConfig(oldConfig)

		// Write back
		newData, err := json.MarshalIndent(newConfig, "", "  ")
		if err != nil {
			fmt.Printf("Error marshaling %s: %v\n", filepath, err)
			continue
		}

		if err := os.WriteFile(filepath, newData, 0644); err != nil {
			fmt.Printf("Error writing %s: %v\n", filepath, err)
			continue
		}

		fmt.Printf("Migrated %s\n", filename)
		fmt.Printf("  Bass: scaling_factor %.9f -> max_magnitude %.0f\n",
			oldConfig.Config.Bass.ScalingFactor, newConfig.Config.Bass.MaxMagnitude)
		fmt.Printf("  MidLow: scaling_factor %.9f -> max_magnitude %.0f\n",
			oldConfig.Config.MidLow.ScalingFactor, newConfig.Config.MidLow.MaxMagnitude)
		fmt.Printf("  MidHigh: scaling_factor %.9f -> max_magnitude %.0f\n",
			oldConfig.Config.MidHigh.ScalingFactor, newConfig.Config.MidHigh.MaxMagnitude)
		fmt.Printf("  Treble: scaling_factor %.9f -> max_magnitude %.0f\n",
			oldConfig.Config.Treble.ScalingFactor, newConfig.Config.Treble.MaxMagnitude)
		migratedCount++
	}

	fmt.Printf("\nMigration complete: %d migrated, %d skipped\n", migratedCount, skippedCount)
}
