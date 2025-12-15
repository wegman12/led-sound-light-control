package light

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/wegman12/led-sound-light-control/light/behavior"
	"github.com/wegman12/led-sound-light-control/light/led"
)

// SimulatorConfig contains the configuration for LED simulation
type SimulatorConfig struct {
	// InputCSVPath is the path to the audio CSV file (PRU format)
	InputCSVPath string `json:"input_csv_path"`

	// BehaviorConfig is the LED behavior configuration to simulate
	BehaviorConfig ManagerConfig `json:"behavior_config"`

	// OutputCSVPath is where to write the LED power simulation results
	OutputCSVPath string `json:"output_csv_path"`
}

// SimulationResult represents a single timestep of LED simulation
type SimulationResult struct {
	Timestamp float64 `json:"timestamp"` // seconds
	Red       float64 `json:"red"`       // power 0.0-1.0
	Green     float64 `json:"green"`     // power 0.0-1.0
	Blue      float64 `json:"blue"`      // power 0.0-1.0
	White     float64 `json:"white"`     // power 0.0-1.0
}

// AudioCSVRow represents a row from the PRU audio CSV file
type AudioCSVRow struct {
	Sample      int
	Timestamp   float64 // seconds
	FFTCount    int
	FFTRate     float64
	FFTTimeMs   float64
	BassSum     int64
	MidLowSum   int64
	MidHighSum  int64
	TrebleSum   int64
	BassAvg     int64
	MidLowAvg   int64
	MidHighAvg  int64
	TrebleAvg   int64
}

// parseAudioCSVRow parses a CSV row into an AudioCSVRow struct
func parseAudioCSVRow(record []string) (*AudioCSVRow, error) {
	if len(record) < 13 {
		return nil, fmt.Errorf("invalid CSV row: expected 13 columns, got %d", len(record))
	}

	sample, err := strconv.Atoi(record[0])
	if err != nil {
		return nil, fmt.Errorf("invalid sample number: %w", err)
	}

	timestamp, err := strconv.ParseFloat(record[1], 64)
	if err != nil {
		return nil, fmt.Errorf("invalid timestamp: %w", err)
	}

	fftCount, err := strconv.Atoi(record[2])
	if err != nil {
		return nil, fmt.Errorf("invalid FFT count: %w", err)
	}

	fftRate, err := strconv.ParseFloat(record[3], 64)
	if err != nil {
		return nil, fmt.Errorf("invalid FFT rate: %w", err)
	}

	fftTimeMs, err := strconv.ParseFloat(record[4], 64)
	if err != nil {
		return nil, fmt.Errorf("invalid FFT time: %w", err)
	}

	bassSum, err := strconv.ParseInt(record[5], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid bass sum: %w", err)
	}

	midLowSum, err := strconv.ParseInt(record[6], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid mid-low sum: %w", err)
	}

	midHighSum, err := strconv.ParseInt(record[7], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid mid-high sum: %w", err)
	}

	trebleSum, err := strconv.ParseInt(record[8], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid treble sum: %w", err)
	}

	bassAvg, err := strconv.ParseInt(record[9], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid bass avg: %w", err)
	}

	midLowAvg, err := strconv.ParseInt(record[10], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid mid-low avg: %w", err)
	}

	midHighAvg, err := strconv.ParseInt(record[11], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid mid-high avg: %w", err)
	}

	trebleAvg, err := strconv.ParseInt(record[12], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid treble avg: %w", err)
	}

	return &AudioCSVRow{
		Sample:     sample,
		Timestamp:  timestamp,
		FFTCount:   fftCount,
		FFTRate:    fftRate,
		FFTTimeMs:  fftTimeMs,
		BassSum:    bassSum,
		MidLowSum:  midLowSum,
		MidHighSum: midHighSum,
		TrebleSum:  trebleSum,
		BassAvg:    bassAvg,
		MidLowAvg:  midLowAvg,
		MidHighAvg: midHighAvg,
		TrebleAvg:  trebleAvg,
	}, nil
}

// SimulateLEDs runs LED simulation on audio CSV data
func SimulateLEDs(config SimulatorConfig) ([]SimulationResult, error) {
	// Open and parse the audio CSV file
	audioData, err := readAudioCSV(config.InputCSVPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read audio CSV: %w", err)
	}

	if len(audioData) == 0 {
		return nil, fmt.Errorf("audio CSV file is empty")
	}

	// Create an audio provider that we'll update for each timestep
	audioProvider := behavior.NewAudioProvider()

	// Create behaviors from the configuration
	behaviors, err := createBehaviors(config.BehaviorConfig.Behaviors, audioProvider)
	if err != nil {
		return nil, fmt.Errorf("failed to create behaviors: %w", err)
	}

	behaviorManager := behavior.CreateManager(behaviors)

	// Process each audio data point and simulate LED output
	results := make([]SimulationResult, 0, len(audioData))

	for _, audioRow := range audioData {
		// Update the audio provider with current audio data
		profile := behavior.AudioProfile{
			Bass:    float64(audioRow.BassAvg),
			MidLow:  float64(audioRow.MidLowAvg),
			MidHigh: float64(audioRow.MidHighAvg),
			Treble:  float64(audioRow.TrebleAvg),
		}
		audioProvider.UpdateProfile(profile)

		// Get power values for each LED color
		// We use 0 for time.Duration since audio modulator doesn't use it
		powers := behaviorManager.GetPower(0)

		result := SimulationResult{
			Timestamp: audioRow.Timestamp,
			Red:       getPowerValue(powers, led.RedLedColor),
			Green:     getPowerValue(powers, led.GreenLedColor),
			Blue:      getPowerValue(powers, led.BlueLedColor),
			White:     getPowerValue(powers, led.WhiteLedColor),
		}

		results = append(results, result)
	}

	return results, nil
}

// SimulateLEDsAndWrite runs simulation and writes results to CSV file
func SimulateLEDsAndWrite(config SimulatorConfig) error {
	results, err := SimulateLEDs(config)
	if err != nil {
		return err
	}

	return writeSimulationResults(config.OutputCSVPath, results)
}

// readAudioCSV reads and parses a PRU audio CSV file
func readAudioCSV(filepath string) ([]AudioCSVRow, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)

	// Skip header row
	_, err = reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV header: %w", err)
	}

	var rows []AudioCSVRow

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read CSV row: %w", err)
		}

		row, err := parseAudioCSVRow(record)
		if err != nil {
			return nil, fmt.Errorf("failed to parse CSV row: %w", err)
		}

		rows = append(rows, *row)
	}

	return rows, nil
}

// writeSimulationResults writes simulation results to a CSV file
func writeSimulationResults(filepath string, results []SimulationResult) error {
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	// Write CSV header
	_, err = file.WriteString("timestamp,red,green,blue,white\n")
	if err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Write each result
	for _, result := range results {
		line := fmt.Sprintf("%.6f,%.6f,%.6f,%.6f,%.6f\n",
			result.Timestamp,
			result.Red,
			result.Green,
			result.Blue,
			result.White,
		)
		if _, err := file.WriteString(line); err != nil {
			return fmt.Errorf("failed to write result: %w", err)
		}
	}

	return nil
}

// getPowerValue extracts power value for a given color, defaulting to 0.0
func getPowerValue(powers map[led.Color]*float64, color led.Color) float64 {
	if powers == nil {
		return 0.0
	}
	power := powers[color]
	if power == nil {
		return 0.0
	}
	return *power
}

// SimulatorRequest is the HTTP request body for simulation
type SimulatorRequest struct {
	// AudioCSVContent is the content of the PRU audio CSV file
	AudioCSVContent string `json:"audio_csv_content"`

	// BehaviorConfig is the LED behavior configuration
	BehaviorConfig ManagerConfig `json:"behavior_config"`
}

// SimulatorResponse is the HTTP response body for simulation
type SimulatorResponse struct {
	Results []SimulationResult `json:"results"`
	Message string             `json:"message,omitempty"`
}

// SimulateFromRequest runs simulation from HTTP request data
func SimulateFromRequest(req SimulatorRequest) (*SimulatorResponse, error) {
	// Write CSV content to temporary file
	tmpFile, err := os.CreateTemp("", "audio-sim-*.csv")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	_, err = tmpFile.WriteString(req.AudioCSVContent)
	if err != nil {
		return nil, fmt.Errorf("failed to write temp file: %w", err)
	}
	tmpFile.Close()

	// Run simulation
	config := SimulatorConfig{
		InputCSVPath:   tmpFile.Name(),
		BehaviorConfig: req.BehaviorConfig,
		OutputCSVPath:  "", // Not used, we return results directly
	}

	results, err := SimulateLEDs(config)
	if err != nil {
		return nil, fmt.Errorf("simulation failed: %w", err)
	}

	return &SimulatorResponse{
		Results: results,
		Message: fmt.Sprintf("Simulation completed: %d timesteps processed", len(results)),
	}, nil
}

// ParseSimulatorRequestFromJSON parses a JSON request body
func ParseSimulatorRequestFromJSON(data []byte) (*SimulatorRequest, error) {
	var req SimulatorRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("failed to parse JSON request: %w", err)
	}

	// Validate that we have audio data
	if strings.TrimSpace(req.AudioCSVContent) == "" {
		return nil, fmt.Errorf("audio_csv_content is required")
	}

	// Validate that we have behaviors
	if len(req.BehaviorConfig.Behaviors) == 0 {
		return nil, fmt.Errorf("behavior_config.behaviors is required")
	}

	return &req, nil
}
