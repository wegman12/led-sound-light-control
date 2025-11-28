package sampling

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strconv"

	"go.uber.org/zap"
)

const (
	IIO_DEVICE      = "/sys/bus/iio/devices/iio:device0"
	IIO_BUFFER_PATH = "/dev/iio:device0"
)

// fastIIOReader provides high-speed ADC sampling using Linux IIO buffered mode
type fastIIOReader struct {
	devicePath string
	bufferDev  *os.File
	channel    int // AIN channel (0-7)
	bufferSize int
	buffer     []byte
	samples    []uint16
	isEnabled  bool
	logger     *zap.Logger
}

// newFastIioSampler creates a new IIO Sampler configured for high-speed sampling
func newFastIioSampler(channel int, bufferSize int, logger *zap.Logger) (*fastIIOReader, error) {
	r := &fastIIOReader{
		devicePath: IIO_DEVICE,
		channel:    channel,
		bufferSize: bufferSize,
		isEnabled:  false,
		logger:     logger,
	}

	// Configure the IIO device
	if err := r.configure(); err != nil {
		return nil, fmt.Errorf("failed to configure IIO: %w", err)
	}

	return r, nil
}

func (r *fastIIOReader) configure() error {
	if r.logger == nil {
		r.logger = zap.NewNop()
	}
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

	r.buffer = make([]byte, r.bufferSize*2)
	r.samples = make([]uint16, r.bufferSize)

	return nil
}

// Start begins continuous sampling
func (r *fastIIOReader) Start() error {
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
func (r *fastIIOReader) Stop() error {
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
func (r *fastIIOReader) ReadSamples() ([]uint16, error) {
	if !r.isEnabled || r.bufferDev == nil {
		return nil, fmt.Errorf("Sampler not started")
	}

	// Determine sample size (12-bit ADC typically uses 2 samples per sample)
	sampleSize := 2

	// Read buffer
	n, err := r.bufferDev.Read(r.buffer)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read buffer: %w", err)
	}

	if n == 0 {
		return []uint16{}, nil
	}

	// Parse samples
	numSamples := n / sampleSize

	for i := 0; i < numSamples; i++ {
		offset := i * sampleSize
		// IIO typically uses little-endian 16-bit values
		value := binary.LittleEndian.Uint16(r.buffer[offset : offset+2])
		// Mask to 12 bits
		r.samples[i] = value & 0x0FFF
	}

	return r.samples[:numSamples], nil
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
