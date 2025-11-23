package sound

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

const (
	IIO_DEVICE      = "/sys/bus/iio/devices/iio:device0"
	IIO_BUFFER_PATH = "/dev/iio:device0"
)

// FastIIOReader provides high-speed ADC sampling using Linux IIO buffered mode
type FastIIOReader struct {
	devicePath string
	bufferDev  *os.File
	channel    int // AIN channel (0-7)
	sampleRate int
	bufferSize int
	isEnabled  bool
}

// NewFastIIOReader creates a new IIO reader configured for high-speed sampling
func NewFastIIOReader(channel int, sampleRate int, bufferSize int) (*FastIIOReader, error) {
	reader := &FastIIOReader{
		devicePath: IIO_DEVICE,
		channel:    channel,
		sampleRate: sampleRate,
		bufferSize: bufferSize,
		isEnabled:  false,
	}

	// Configure the IIO device
	if err := reader.configure(); err != nil {
		return nil, fmt.Errorf("failed to configure IIO: %w", err)
	}

	return reader, nil
}

func (r *FastIIOReader) configure() error {
	// Enable the specific channel
	scanElementPath := fmt.Sprintf("%s/scan_elements/in_voltage%d_en", r.devicePath, r.channel)
	if err := writeFile(scanElementPath, "1"); err != nil {
		return fmt.Errorf("failed to enable channel %d: %w", r.channel, err)
	}

	// Set buffer length
	bufferLengthPath := fmt.Sprintf("%s/buffer/length", r.devicePath)
	if err := writeFile(bufferLengthPath, strconv.Itoa(r.bufferSize)); err != nil {
		return fmt.Errorf("failed to set buffer length: %w", err)
	}

	// Try to set sampling frequency (may not be supported on all kernels)
	samplingFreqPath := fmt.Sprintf("%s/sampling_frequency", r.devicePath)
	if err := writeFile(samplingFreqPath, strconv.Itoa(r.sampleRate)); err != nil {
		// Non-fatal, some kernels don't support this
		fmt.Printf("Note: Could not set sampling frequency: %v\n", err)
	}

	return nil
}

// Start begins continuous sampling
func (r *FastIIOReader) Start() error {
	if r.isEnabled {
		return fmt.Errorf("already started")
	}

	// Enable buffer
	bufferEnablePath := fmt.Sprintf("%s/buffer/enable", r.devicePath)
	if err := writeFile(bufferEnablePath, "1"); err != nil {
		return fmt.Errorf("failed to enable buffer: %w", err)
	}

	// Open the buffer device for reading
	bufferDev, err := os.Open(IIO_BUFFER_PATH)
	if err != nil {
		// Try to disable buffer on error
		writeFile(bufferEnablePath, "0")
		return fmt.Errorf("failed to open buffer device: %w", err)
	}

	r.bufferDev = bufferDev
	r.isEnabled = true

	return nil
}

// Stop halts sampling and closes resources
func (r *FastIIOReader) Stop() error {
	if !r.isEnabled {
		return nil
	}

	// Close buffer device
	if r.bufferDev != nil {
		r.bufferDev.Close()
		r.bufferDev = nil
	}

	// Disable buffer
	bufferEnablePath := fmt.Sprintf("%s/buffer/enable", r.devicePath)
	if err := writeFile(bufferEnablePath, "0"); err != nil {
		return fmt.Errorf("failed to disable buffer: %w", err)
	}

	// Disable channel
	scanElementPath := fmt.Sprintf("%s/scan_elements/in_voltage%d_en", r.devicePath, r.channel)
	if err := writeFile(scanElementPath, "0"); err != nil {
		return fmt.Errorf("failed to disable channel: %w", err)
	}

	r.isEnabled = false
	return nil
}

// ReadSamples reads available samples from the IIO buffer
// Returns slice of 12-bit ADC values (0-4095)
func (r *FastIIOReader) ReadSamples() ([]uint16, error) {
	if !r.isEnabled || r.bufferDev == nil {
		return nil, fmt.Errorf("reader not started")
	}

	// Determine sample size (12-bit ADC typically uses 2 samples per sample)
	sampleSize := 2

	// Read buffer
	buf := make([]byte, r.bufferSize*sampleSize)
	n, err := r.bufferDev.Read(buf)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read buffer: %w", err)
	}

	if n == 0 {
		return []uint16{}, nil
	}

	// Parse samples
	numSamples := n / sampleSize
	samples := make([]uint16, numSamples)

	for i := 0; i < numSamples; i++ {
		offset := i * sampleSize
		// IIO typically uses little-endian 16-bit values
		value := binary.LittleEndian.Uint16(buf[offset : offset+2])
		// Mask to 12 bits
		samples[i] = value & 0x0FFF
	}

	return samples, nil
}

// GetActualSampleRate attempts to read the actual configured sample rate
func (r *FastIIOReader) GetActualSampleRate() (int, error) {
	samplingFreqPath := fmt.Sprintf("%s/sampling_frequency", r.devicePath)
	data, err := os.ReadFile(samplingFreqPath)
	if err != nil {
		return 0, err
	}

	rate, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, err
	}

	return rate, nil
}

// Helper function to write to sysfs files
func writeFile(path string, value string) error {
	return os.WriteFile(path, []byte(value), 0644)
}

// CheckIIOAvailable checks if IIO device is available
func CheckIIOAvailable() error {
	if _, err := os.Stat(IIO_DEVICE); os.IsNotExist(err) {
		return fmt.Errorf("IIO device not found at %s", IIO_DEVICE)
	}

	if _, err := os.Stat(IIO_BUFFER_PATH); os.IsNotExist(err) {
		return fmt.Errorf("IIO buffer device not found at %s - may need to enable buffer first", IIO_BUFFER_PATH)
	}

	return nil
}
